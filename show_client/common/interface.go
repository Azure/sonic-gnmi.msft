package common

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	log "github.com/golang/glog"
	natural "github.com/maruel/natural"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

const (
	Alias                      = "alias"
	VlanSubInterfaceSeparator  = '.'
	defaultCountersDBSeparator = ":"
)

var (
	countersDBSeparator string = defaultCountersDBSeparator
	countersOnce        sync.Once
)

func CountersDBSeparator() string {
	countersOnce.Do(func() {
		sep, err := sdc.GetTableKeySeparator("COUNTERS_DB", "")
		if err != nil {
			log.Warningf("Failed to get table key separator for COUNTERS DB: %v\nUsing the default separator '%s'.", err, defaultCountersDBSeparator)
			return
		}
		countersDBSeparator = sep
	})
	return countersDBSeparator
}

func NatsortInterfaces(interfaces []string) []string {
	// Naturally sort the port list
	sort.Sort(natural.StringSlice(interfaces))
	return interfaces
}

func RemapAliasToPortName(portData map[string]interface{}) map[string]interface{} {
	aliasMap := sdc.AliasToPortNameMap()
	remapped := make(map[string]interface{})

	needRemap := false

	for key := range portData {
		if _, isAlias := aliasMap[key]; isAlias {
			needRemap = true
			break
		}
	}

	if !needRemap { // Not an alias keyed map, no-op
		return portData
	}

	for alias, val := range portData {
		if portName, ok := aliasMap[alias]; ok {
			remapped[portName] = val
		}
	}
	return remapped
}

func RemapAliasToPortNameForQueues(queueData map[string]interface{}) map[string]interface{} {
	aliasMap := sdc.AliasToPortNameMap()
	remapped := make(map[string]interface{})
	sep := CountersDBSeparator()

	for key, val := range queueData {
		port, queueIdx, found := strings.Cut(key, sep)
		if !found {
			log.Warningf("Ignoring the invalid queue '%v'", key)
			continue
		}
		if sonicPortName, ok := aliasMap[port]; ok {
			remapped[sonicPortName+sep+queueIdx] = val
		} else {
			remapped[key] = val
		}
	}

	return remapped
}

func GetNameForInterfaceAlias(intfAlias string) string {
	aliasMap := sdc.AliasToPortNameMap()
	if name, ok := aliasMap[intfAlias]; ok {
		return name
	} else {
		return ""
	}
}

func GetInterfaceNamingMode(namingMode string) string {
	if namingMode != "" {
		return namingMode
	}
	return "default"
}

// GetInterfaceNameForDisplay returns alias when SONIC_CLI_IFACE_MODE=alias; otherwise the name.
// It also preserves VLAN sub-interface suffix like Ethernet0.100.
func GetInterfaceNameForDisplay(name string, namingMode string) string {
	if name == "" {
		return name
	}
	if interfaceNamingMode := GetInterfaceNamingMode(namingMode); interfaceNamingMode != Alias {
		return name
	}

	nameToAlias := sdc.PortToAliasNameMap()

	base, suffix := name, ""
	if i := strings.IndexByte(name, VlanSubInterfaceSeparator); i >= 0 {
		base, suffix = name[:i], name[i:] // keep .<vlan>
	}

	if alias, ok := nameToAlias[base]; ok {
		return alias + suffix
	}
	return name
}

// TryConvertInterfaceNameFromAlias tries to convert an interface alias to its interface name.
// If naming mode is "alias", attempts conversion; if conversion fails, returns error.
func TryConvertInterfaceNameFromAlias(interfaceName string, namingMode string) (string, error) {
	if GetInterfaceNamingMode(namingMode) == Alias {
		alias := interfaceName
		aliasMap := sdc.AliasToPortNameMap()
		if itfName, ok := aliasMap[alias]; ok {
			interfaceName = itfName
		}

		// AliasToName should return "" if not found
		if interfaceName == "" || interfaceName == alias {
			return "", fmt.Errorf("Cannot find interface name for alias %s", alias)
		}
	}
	return interfaceName, nil
}
