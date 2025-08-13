package ipinterfaces

import (
	"fmt"
	"net"
)

const (
	AddressFamilyIPv4 = "ipv4"
	AddressFamilyIPv6 = "ipv6"
	DisplayAll        = "all"
	DisplayExternal   = "frontend"
)

// IPInterfaceDetail holds all the consolidated information for a network interface.
type IPInterfaceDetail struct {
	Name        string            `json:"name"`
	IPAddresses []IPAddressDetail `json:"ip_addresses"`
	AdminStatus string            `json:"admin_status"`
	OperStatus  string            `json:"oper_status"`
	Master      string            `json:"master,omitempty"`
}

// GetIPInterfaces returns IP interface details for the selected namespaces.
// addressFamily: "ipv4" or "ipv6" (required)
// namespaceOption: nil means auto-selection.
// displayOption: nil means use default (DisplayAll).
func GetIPInterfaces(addressFamily string, namespaceOption *string, displayOption *string) ([]IPInterfaceDetail, error) {
	if addressFamily != AddressFamilyIPv4 && addressFamily != AddressFamilyIPv6 {
		return nil, fmt.Errorf("unsupported address family: %s", addressFamily)
	}

	var displayVal string
	if displayOption == nil {
		displayVal = DisplayAll
	} else {
		displayVal = *displayOption
	}

	nsList, err := resolveNamespaceSelection(namespaceOption, displayVal)
	if err != nil {
		return nil, err
	}

	// Python ipintutil always appends the default (host) namespace for multi-ASIC after selection
	if isMulti, err := IsMultiASIC(); err == nil && isMulti {
		found := false
		for _, ns := range nsList {
			if ns == defaultNamespace {
				found = true
				break
			}
		}
		if !found {
			nsList = append(nsList, defaultNamespace)
		}
	}

	interfaceMap := make(map[string]*IPInterfaceDetail)
	for _, ns := range nsList {
		interfacesInNs, err := getInterfacesInNamespace(ns, addressFamily)
		if err != nil {
			fmt.Printf("Warning: could not get interfaces for namespace '%s': %v\n", ns, err)
			continue
		}
		for _, iface := range interfacesInNs {
			if _, ok := interfaceMap[iface.Name]; !ok {
				// Shallow copy
				copy := iface
				interfaceMap[iface.Name] = &copy
				continue
			}
			// Merge IP addresses (avoid duplicates)
			existing := interfaceMap[iface.Name]
			for _, ipd := range iface.IPAddresses {
				if !ipAddressExists(existing.IPAddresses, ipd.Address) {
					existing.IPAddresses = append(existing.IPAddresses, ipd)
				}
			}
		}
	}

	all := make([]IPInterfaceDetail, 0, len(interfaceMap))
	for _, v := range interfaceMap {
		all = append(all, *v)
	}

	if err := enrichWithBGPData(all); err != nil {
		fmt.Printf("Warning: failed to enrich with BGP data: %v\n", err)
	}
	return all, nil
}

func ipAddressExists(list []IPAddressDetail, addr string) bool {
	for _, a := range list {
		if a.Address == addr {
			return true
		}
	}
	return false
}

// resolveNamespaceSelection builds namespace list.
// - Single ASIC: always [defaultNamespace]
// - Multi ASIC + explicit namespace (pointer not nil): validate & return specified namespace
// - Multi ASIC + auto (pointer nil): return namespaces per display
func resolveNamespaceSelection(namespaceOption *string, displayVal string) ([]string, error) {
	isMultiASIC, err := IsMultiASIC()
	if err != nil {
		return nil, err
	}

	if !isMultiASIC { // single ASIC
		if namespaceOption != nil && *namespaceOption != "" && *namespaceOption != defaultNamespace {
			return nil, fmt.Errorf("unknown namespace %s", *namespaceOption)
		}
		return []string{defaultNamespace}, nil
	}

	namespacesByRole, err := GetAllNamespaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get namespaces: %w", err)
	}

	if namespaceOption != nil {
		ns := *namespaceOption
		if !containsString(namespacesByRole.Frontend, ns) && !containsString(namespacesByRole.Backend, ns) && !containsString(namespacesByRole.Fabric, ns) {
			return nil, fmt.Errorf("unknown namespace %s", ns)
		}
		return []string{ns}, nil
	}

	var nsList []string
	if displayVal == DisplayAll {
		nsList = append(nsList, namespacesByRole.Frontend...)
		nsList = append(nsList, namespacesByRole.Backend...)
		nsList = append(nsList, namespacesByRole.Fabric...)
	} else {
		nsList = append(nsList, namespacesByRole.Frontend...)
	}
	return nsList, nil
}

func enrichWithBGPData(interfaces []IPInterfaceDetail) error {
	bgpNeighbors, err := getBGPNeighborsFromDB("")
	if err != nil {
		fmt.Printf("Warning: failed to get BGP neighbors from default namespace: %v\n", err)
		return nil
	}
	for i := range interfaces {
		iface := &interfaces[i]
		for j := range iface.IPAddresses {
			ipDetail := &iface.IPAddresses[j]
			addr, _, err := net.ParseCIDR(ipDetail.Address)
			if err != nil {
				continue
			}
			ipStr := addr.String()
			if neighborInfo, ok := bgpNeighbors[ipStr]; ok {
				ipDetail.BGPNeighborIP = neighborInfo.NeighborIP
				ipDetail.BGPNeighborName = neighborInfo.Name
			}
		}
	}
	return nil
}
