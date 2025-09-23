package common

import (
	"fmt"
	"sort"
	"strconv"
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

func IsValidPhysicalPort(iface string) (bool, error) {
	queries := [][]string{
		{"APPL_DB", "PORT_TABLE"},
	}
	portTable, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to pull data for queries %v, got err %v", queries, err)
		return false, err
	}
	role := GetFieldValueString(portTable, iface, DefaultMissingCounterValue, "role")
	return IsFrontPanelPort(iface, role), nil
}

func IsRoleInternal(role string) bool {
	return role != DefaultMissingCounterValue && (role == InternalPort || role == InbandPort || role == RecircPort || role == DpuConnectPort)
}

func IsFrontPanelPort(iface string, role string) bool {
	if !strings.HasPrefix(iface, SonicInterfacePrefixes["Ethernet-FrontPanel"]) {
		return false
	}
	if strings.HasPrefix(iface, SonicInterfacePrefixes["Ethernet-Backplane"]) || strings.HasPrefix(iface, SonicInterfacePrefixes["Ethernet-Inband"]) || strings.HasPrefix(iface, SonicInterfacePrefixes["Ethernet-Recirc"]) {
		return false
	}
	if strings.Contains(iface, ".") {
		return false
	}
	return !IsRoleInternal(role)
}

type PortMappingRetriever struct {
	logicalToPhysical map[string][]int
	physicalToLogic   map[int][]string
	err               error
}

// This funtion is used to get all ports on device, and then, returns two maps -- logic ports to physical ports and physical ports to logic ports
// To get all ports, the function is https://github.com/sonic-net/sonic-buildimage/blob/master/src/sonic-config-engine/portconfig.py#L171
// We can see from the code that, first, we try to get all ports from config db, if the connection is not available, we will use other methods to get ports
func (pmr *PortMappingRetriever) ReadPorttabMappings() {
	logicalToPhysical := make(map[string][]int)
	physicalToLogic := make(map[int][]string)
	logical := []string{}

	queries := [][]string{
		{"CONFIG_DB", "PORT"},
	}
	portTable, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to pull data for queries %v, got err %v", queries, err)
		pmr.err = err
		return
	}
	for iface := range portTable {
		if IsFrontPanelPort(iface, GetFieldValueString(portTable, iface, DefaultMissingCounterValue, "role")) {
			logical = append(logical, iface)
		}
	}

	sort.Sort(natural.StringSlice(logical))

	for _, intfName := range logical {
		fpPortIndex := 1
		if v, ok := portTable[intfName].(map[string]interface{}); ok {
			if idx, exists := v["index"]; exists {
				if indexStr, ok := idx.(string); ok {
					if val, err := strconv.Atoi(indexStr); err == nil {
						fpPortIndex = val
					}
				}
			}
			logicalToPhysical[intfName] = []int{fpPortIndex}
		}

		if _, ok := physicalToLogic[fpPortIndex]; !ok {
			physicalToLogic[fpPortIndex] = []string{intfName}
		} else {
			physicalToLogic[fpPortIndex] = append(physicalToLogic[fpPortIndex], intfName)
		}
	}

	pmr.logicalToPhysical = logicalToPhysical
	pmr.physicalToLogic = physicalToLogic
	pmr.err = nil
}

func GetLogicalToPhysical(pmr *PortMappingRetriever, logicalPort string) []int {
	if pmr.err != nil {
		return nil
	}
	return pmr.logicalToPhysical[logicalPort]
}

func GetPhysicalToLogic(pmr *PortMappingRetriever, physicalPort int) []string {
	if pmr.err != nil {
		return nil
	}
	return pmr.physicalToLogic[physicalPort]
}

func GetFirstSubPort(pmr *PortMappingRetriever, logicalPort string) string {
	physicalPort := GetLogicalToPhysical(pmr, logicalPort)
	if len(physicalPort) != 0 {
		logicalPortList := GetPhysicalToLogic(pmr, physicalPort[0])
		if len(logicalPortList) != 0 {
			return logicalPortList[0]
		}
	}
	return ""
}

func MergeMaps(a, b map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range a {
		result[k] = v
	}
	for k, v := range b {
		result[k] = v
	}
	return result
}
