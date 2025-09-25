package ipinterfaces

import (
	"fmt"
	"net"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
)

const (
	AddressFamilyIPv4 = "ipv4"
	AddressFamilyIPv6 = "ipv6"
	DisplayAll        = "all"
	DisplayExternal   = "frontend"
)

// GetIPInterfaces returns IP interface details for the selected namespaces.
// addressFamily: "ipv4" or "ipv6" (required)
// opts: may be nil.
func GetIPInterfaces(addressFamily string, opts *GetInterfacesOptions) ([]IPInterfaceDetail, error) {
	if addressFamily != AddressFamilyIPv4 && addressFamily != AddressFamilyIPv6 {
		return nil, fmt.Errorf("unsupported address family: %s", addressFamily)
	}

	nsList, err := resolveNamespaceSelection(opts)
	if err != nil {
		return nil, err
	}

	// Log raw option inputs (may be nil) and effective display after defaults applied for better debugging.
	nsOptVal := "<nil>"
	dispOptVal := "<nil>"
	if opts != nil {
		if opts.Namespace != nil {
			nsOptVal = *opts.Namespace
		}
		if opts.Display != nil {
			dispOptVal = *opts.Display
		}
	}
	log.Infof(
		"GetIPInterfaces(family=%s namespace_opt=%s display_opt=%s) from namespaces '%v':",
		addressFamily, nsOptVal, dispOptVal, nsList,
	)

	interfaceMap := make(map[string]*IPInterfaceDetail)
	for _, ns := range nsList {
		interfacesInNs, err := getInterfacesInNamespace(ns, addressFamily)
		if err != nil {
			log.Warningf("could not get interfaces for namespace '%s': %v", ns, err)
			continue
		}
		log.V(6).Infof("Fetched %d interfaces in namespace %s", len(interfacesInNs), ns)
		for _, iface := range interfacesInNs {
			if shouldSkipInterface(iface.Name, opts) {
				// Placeholder: currently always false. TODO implement display-based filtering.
				continue
			}
			if _, ok := interfaceMap[iface.Name]; !ok {
				// Shallow copy
				copy := iface
				interfaceMap[iface.Name] = &copy
				continue
			}
			// Merge IP addresses (avoid duplicates)
			existing := interfaceMap[iface.Name]
			for _, ipd := range iface.IPAddresses {
				exists := false
				for _, a := range existing.IPAddresses {
					if a.Address == ipd.Address {
						exists = true
						break
					}
				}
				if !exists {
					existing.IPAddresses = append(existing.IPAddresses, ipd)
				}
			}
		}
	}

	all := make([]IPInterfaceDetail, 0, len(interfaceMap))
	for _, v := range interfaceMap {
		all = append(all, *v)
	}
	log.Infof("Aggregated %d interfaces across namespaces", len(all))

	if err := enrichWithBGPData(all); err != nil {
		log.Warningf("failed to enrich with BGP data: %v", err)
	}
	return all, nil
}

// resolveNamespaceSelection builds namespace list.
// - Single ASIC: always [defaultNamespace]
// - Multi ASIC + explicit namespace (pointer not nil): validate & return specified namespace
// - Multi ASIC + auto (pointer nil): return namespaces per display
func resolveNamespaceSelection(opts *GetInterfacesOptions) ([]string, error) {
	var namespaceOption *string
	var displayOption *string
	if opts != nil {
		namespaceOption = opts.Namespace
		displayOption = opts.Display
	}

	isMultiASIC := common.IsMultiAsic()

	if !isMultiASIC { // single ASIC
		if namespaceOption != nil && *namespaceOption != defaultNamespace {
			return nil, fmt.Errorf("unknown namespace %s", *namespaceOption)
		}
		return []string{defaultNamespace}, nil
	}

	namespacesByRole, err := GetAllNamespaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get namespaces: %w", err)
	}

	var nsList []string
	if namespaceOption != nil { // explicit namespace
		ns := *namespaceOption
		if !common.ContainsString(namespacesByRole.Frontend, ns) && !common.ContainsString(namespacesByRole.Backend, ns) && !common.ContainsString(namespacesByRole.Fabric, ns) {
			return nil, fmt.Errorf("unknown namespace %s", ns)
		}
		nsList = []string{ns}
	} else {
		if displayOption == nil || *displayOption == DisplayAll {
			nsList = append(nsList, namespacesByRole.Frontend...)
			nsList = append(nsList, namespacesByRole.Backend...)
			nsList = append(nsList, namespacesByRole.Fabric...)
		} else {
			nsList = append(nsList, namespacesByRole.Frontend...)
		}
	}
	// For multi-ASIC, emulate ipintutil behavior: always include default namespace if not present.
	foundDefault := false
	for _, ns := range nsList {
		if ns == defaultNamespace {
			foundDefault = true
			break
		}
	}
	if !foundDefault {
		nsList = append(nsList, defaultNamespace)
	}
	return nsList, nil
}

func enrichWithBGPData(interfaces []IPInterfaceDetail) error {
	bgpNeighbors, err := getBGPNeighborsFromDB(defaultNamespace)
	if err != nil {
		log.Warningf("failed to get BGP neighbors from default namespace: %v", err)
		return nil
	}
	log.V(6).Infof("Enriching interfaces with %d BGP neighbors from default namespace", len(bgpNeighbors))
	// Dump BGP neighbor map keys for debugging correlation issues
	for k, info := range bgpNeighbors {
		log.V(6).Infof("Dump BGP_NEIGHBOR map: local_addr=%s -> neighbor_ip=%s name=%s", k, info.NeighborIP, info.Name)
	}
	for i := range interfaces {
		iface := &interfaces[i]
		for j := range iface.IPAddresses {
			ipDetail := &iface.IPAddresses[j]
			addr, _, err := net.ParseCIDR(ipDetail.Address)
			if err != nil {
				log.V(6).Infof("Skipping unparsable address %q for interface %s", ipDetail.Address, iface.Name)
				continue
			}
			ipStr := addr.String()
			if neighborInfo, ok := bgpNeighbors[ipStr]; ok {
				ipDetail.BGPNeighborIP = neighborInfo.NeighborIP
				ipDetail.BGPNeighborName = neighborInfo.Name
				log.V(6).Infof("Matched %s -> neighbor %s (%s)", ipStr, neighborInfo.NeighborIP, neighborInfo.Name)
			}
		}
	}
	return nil
}

// shouldSkipInterface is a stub mirroring python skip_ip_intf_display intent.
// Since multi-ASIC/display filtering isn't required yet, it always returns false.
// TODO: implement filtering (internal ports, PortChannels, loopbacks, management, veth) when needed.
func shouldSkipInterface(name string, opts *GetInterfacesOptions) bool { return false }
