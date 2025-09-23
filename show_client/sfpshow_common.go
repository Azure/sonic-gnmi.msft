package show_client

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	log "github.com/golang/glog"
	natural "github.com/maruel/natural"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
)

const TransceiverStatusNotApplicable = "Transceiver status info not applicable"

func isTransceiverCmis(sfpInfoDict map[string]interface{}) bool {
	if sfpInfoDict == nil {
		return false
	}
	_, ok := sfpInfoDict["cmis_rev"]
	return ok
}

func isTransceiverCCmis(sfpInfoDict map[string]interface{}) bool {
	if sfpInfoDict == nil {
		return false
	}
	_, ok := sfpInfoDict["supported_max_tx_power"]
	return ok
}

var CmisDataMap = common.MergeMaps(QsfpDataMap, QsfpCmisDeltaDataMap)
var CCmisDataMap = common.MergeMaps(CmisDataMap, CCmisDeltaDataMap)

func getTransceiverDataMap(sfpInfoDict map[string]interface{}) map[string]string {
	if sfpInfoDict == nil {
		return QsfpDataMap
	}
	isSfpCmis := isTransceiverCmis(sfpInfoDict)
	isSfpCCmis := isTransceiverCCmis(sfpInfoDict)

	if isSfpCCmis {
		return CCmisDataMap
	} else if isSfpCmis {
		return CmisDataMap
	} else {
		return QsfpDataMap
	}
}

func convertApplicationAdvertisementToOutputString(sfpInfoDict map[string]interface{}) map[string]interface{} {
	key := "application_advertisement"

	output := make(map[string]interface{})
	output[QsfpDataMap[key]] = ""

	appAdvStr, ok := sfpInfoDict[key].(string)
	if !ok || appAdvStr == "" {
		output[QsfpDataMap[key]] = "N/A"
		return output
	}

	appAdvStr = strings.ReplaceAll(appAdvStr, "'", "\"")
	re := regexp.MustCompile(`(\{|,)\s*(\d+)\s*:`)
	appAdvStr = re.ReplaceAllString(appAdvStr, `$1 "$2":`)

	var appAdvDict map[string]interface{}
	if err := json.Unmarshal([]byte(appAdvStr), &appAdvDict); err != nil {
		output[QsfpDataMap[key]] = appAdvStr
		return output
	}
	if len(appAdvDict) == 0 {
		output[QsfpDataMap[key]] = "N/A"
		return output
	}

	lines := []string{}
	for _, item := range appAdvDict {
		if dict, ok := item.(map[string]interface{}); ok {
			var elements []string
			if v, ok := dict["host_electrical_interface_id"].(string); ok && v != "" {
				elements = []string{v}
			} else {
				continue
			}

			hostAssignOptions := "Unknown"
			if val, ok := dict["host_lane_assignment_options"].(float64); ok {
				hostAssignOptions = fmt.Sprintf("0x%x", int(val))
			}
			elements = append(elements, fmt.Sprintf("Host Assign (%s)", hostAssignOptions))

			mediaID := "Unknown"
			if val, ok := dict["module_media_interface_id"].(string); ok && val != "" {
				mediaID = val
			}
			elements = append(elements, mediaID)

			mediaAssignOptions := "Unknown"
			if val, ok := dict["media_lane_assignment_options"].(float64); ok {
				mediaAssignOptions = fmt.Sprintf("0x%x", int(val))
			}
			elements = append(elements, fmt.Sprintf("Media Assign (%s)", mediaAssignOptions))

			lines = append(lines, strings.Join(elements, " - "))
		}
	}
	output[QsfpDataMap[key]] = lines
	return output
}

func getDataMapSortKey(keys []string, dataMap map[string]string) []string {
	sort.Slice(keys, func(i, j int) bool {
		ki, iKnown := dataMap[keys[i]]
		kj, jKnown := dataMap[keys[j]]

		if iKnown && !jKnown {
			return true
		}
		if !iKnown && jKnown {
			return false
		}
		if iKnown && jKnown {
			return ki < kj
		}
		return keys[i] < keys[j]
	})
	return keys
}

func convertSfpInfoToOutputString(sfpInfoDict map[string]interface{}, sfpFirmwareInfoDict map[string]interface{}) map[string]interface{} {
	output := make(map[string]interface{})
	isSfpCmis := isTransceiverCmis(sfpInfoDict)
	dataMap := getTransceiverDataMap(sfpInfoDict)

	combinedDict := make(map[string]interface{})
	for k, v := range sfpInfoDict {
		combinedDict[k] = v
	}
	if len(sfpFirmwareInfoDict) != 0 {
		for k, v := range sfpFirmwareInfoDict {
			combinedDict[k] = v
		}
	}

	keys := make([]string, 0, len(combinedDict))
	for k := range combinedDict {
		keys = append(keys, k)
	}

	sortedKeys := getDataMapSortKey(keys, dataMap)

	for _, key := range sortedKeys {
		if key == "cable_type" {
			if cableType, ok := sfpInfoDict["cable_type"].(string); ok {
				output[cableType] = sfpInfoDict["cable_length"]
			}
		} else if key == "cable_length" {
		} else if key == "specification_compliance" && !isSfpCmis {
			if sfpInfoDict["type"] == "QSFP-DD Double Density 8X Pluggable Transceiver" {
				output[QsfpDataMap[key]] = sfpInfoDict[key]
			} else {
				output[QsfpDataMap[key]] = ""

				specComplianceDict := make(map[string]interface{})
				specStr, ok := sfpInfoDict["specification_compliance"]
        
				if ok && specStr != "" {
					if s, ok := specStr.(string); ok && s != "" {
						if err := json.Unmarshal([]byte(s), &specComplianceDict); err != nil {
							output[QsfpDataMap[key]] = "N/A"
						} else {
							keys := make([]string, 0, len(specComplianceDict))
							for k := range specComplianceDict {
								keys = append(keys, k)
							}
							sort.Sort(natural.StringSlice(keys))

							m := make(map[string]interface{})
							for _, k := range keys {
								m[k] = specComplianceDict[k]
							}
							output[QsfpDataMap[key]] = m
						}
					} else {
						output[QsfpDataMap[key]] = "N/A"
					}
				}
			}
		} else if key == "application_advertisement" {
			applicationOutput := convertApplicationAdvertisementToOutputString(sfpInfoDict)
			for key, value := range applicationOutput {
				output[key] = value
			}
		} else if key == "active_firmware" || key == "inactive_firmware" {
			val := "N/A"
			if v, ok := sfpFirmwareInfoDict[key]; ok {
				val = fmt.Sprintf("%v", v)
			}
			output[dataMap[key]] = val
		} else if strings.HasPrefix(key, "e1_") || strings.HasPrefix(key, "e2_") {
			if v, ok := sfpFirmwareInfoDict[key]; ok {
				output[dataMap[key]] = v
			}
		} else {
			displayName := key

			if v, ok := dataMap[key]; ok && v != "" {
				displayName = v

				value := "N/A"
				if v, ok := sfpInfoDict[key]; ok {
					value = fmt.Sprintf("%v", v)
				} else if len(sfpFirmwareInfoDict) != 0 {
					if v, ok := sfpFirmwareInfoDict[key]; ok {
						value = fmt.Sprintf("%v", v)
					}
				}
				output[displayName] = value
			}
		}
	}
	return output
}

func formatDictValueToString(sortedKeyTable []string, domInfoDict map[string]interface{}, domValueMap map[string]string, domUnitMap map[string]string) map[string]interface{} {
	output := make(map[string]interface{})

	for _, key := range sortedKeyTable {
		val := domInfoDict[key]
		if val == nil {
			continue
		}

		var value string
		if str, ok := val.(string); ok {
			if str == "N/A" {
				continue
			}
			value = str
		} else {
			value = fmt.Sprintf("%v", val)
		}

		units := ""
		if value != "Unknown" && !strings.HasSuffix(value, domUnitMap[key]) {
			units = domUnitMap[key]
		}
		output[domValueMap[key]] = fmt.Sprintf("%v%v", value, units)
	}
	return output
}

func convertDomToOutputString(sfpType string, isSfpCmis bool, domInfoDict map[string]interface{}) map[string]interface{} {
	outputDom := make(map[string]interface{})

	if strings.HasPrefix(sfpType, "QSFP") || strings.HasPrefix(sfpType, "OSFP") {
		outputDom["ChannelMonitorValues"] = ""
		sortedKeyTable := []string{}
		var domMap map[string]string

		if isSfpCmis {
			sortedKeyTable = make([]string, 0, len(CmisDomChannelMonitorMap))
			for k := range CmisDomChannelMonitorMap {
				sortedKeyTable = append(sortedKeyTable, k)
			}
			sort.Sort(natural.StringSlice(sortedKeyTable))
			outputChannel := formatDictValueToString(sortedKeyTable, domInfoDict, CmisDomChannelMonitorMap, QsfpDdDomValueUnitMap)
			outputDom["ChannelMonitorValues"] = outputChannel
		} else {
			sortedKeyTable = make([]string, 0, len(QsfpDomChannelMonitorMap))
			for k := range QsfpDomChannelMonitorMap {
				sortedKeyTable = append(sortedKeyTable, k)
			}
			sort.Sort(natural.StringSlice(sortedKeyTable))
			outputChannel := formatDictValueToString(sortedKeyTable, domInfoDict, QsfpDomChannelMonitorMap, DomValueUnitMap)
			outputDom["ChannelMonitorValues"] = outputChannel
		}

		if isSfpCmis {
			domMap = SfpDomChannelThresholdMap
		} else {
			domMap = QsfpDomChannelThresholdMap
		}

		outputDom["ChannelThresholdValues"] = ""
		sortedKeyTable = make([]string, 0, len(domMap))
		for k := range domMap {
			sortedKeyTable = append(sortedKeyTable, k)
		}
		sort.Sort(natural.StringSlice(sortedKeyTable))
		outputChannelThreshold := formatDictValueToString(sortedKeyTable, domInfoDict, domMap, DomChannelThresholdUnitMap)
		outputDom["ChannelThresholdValues"] = outputChannelThreshold

		outputDom["ModuleMonitorValues"] = ""
		sortedKeyTable = make([]string, 0, len(DomModuleMonitorMap))
		for k := range DomModuleMonitorMap {
			sortedKeyTable = append(sortedKeyTable, k)
		}
		sort.Sort(natural.StringSlice(sortedKeyTable))
		outputModule := formatDictValueToString(sortedKeyTable, domInfoDict, DomModuleMonitorMap, DomValueUnitMap)
		outputDom["ModuleMonitorValues"] = outputModule

		outputDom["ModuleThresholdValues"] = ""
		sortedKeyTable = make([]string, 0, len(DomModuleThresholdMap))
		for k := range DomModuleThresholdMap {
			sortedKeyTable = append(sortedKeyTable, k)
		}
		sort.Sort(natural.StringSlice(sortedKeyTable))
		outputModuleThreshold := formatDictValueToString(sortedKeyTable, domInfoDict, DomModuleThresholdMap, DomModuleThresholdUnitMap)
		outputDom["ModuleThresholdValues"] = outputModuleThreshold
	} else {
		outputDom["MonitorData"] = ""
		sortedKeyTable := make([]string, 0, len(SfpDomChannelMonitorMap))
		for k := range SfpDomChannelMonitorMap {
			sortedKeyTable = append(sortedKeyTable, k)
		}
		sort.Sort(natural.StringSlice(sortedKeyTable))
		outputChannel := formatDictValueToString(sortedKeyTable, domInfoDict, SfpDomChannelMonitorMap, DomValueUnitMap)
		outputDom["MonitorData"] = outputChannel

		sortedKeyTable = make([]string, 0, len(DomModuleMonitorMap))
		for k := range DomModuleMonitorMap {
			sortedKeyTable = append(sortedKeyTable, k)
		}
		sort.Sort(natural.StringSlice(sortedKeyTable))
		outputModule := formatDictValueToString(sortedKeyTable, domInfoDict, DomModuleMonitorMap, DomValueUnitMap)

		monitorData, ok := outputDom["MonitorData"].(map[string]interface{})
		if !ok {
			monitorData = make(map[string]interface{})
			outputDom["MonitorData"] = monitorData
		}

		for key, value := range outputModule {
			monitorData[key] = value
		}

		outputDom["ThresholdData"] = ""
		sortedKeyTable = make([]string, 0, len(DomModuleThresholdMap))
		for k := range DomModuleThresholdMap {
			sortedKeyTable = append(sortedKeyTable, k)
		}
		sort.Sort(natural.StringSlice(sortedKeyTable))
		outputModuleThreshold := formatDictValueToString(sortedKeyTable, domInfoDict, DomModuleThresholdMap, DomModuleThresholdUnitMap)
		outputDom["ThresholdData"] = outputModuleThreshold

		sortedKeyTable = make([]string, 0, len(SfpDomChannelThresholdMap))
		for k := range SfpDomChannelThresholdMap {
			sortedKeyTable = append(sortedKeyTable, k)
		}
		sort.Sort(natural.StringSlice(sortedKeyTable))
		outputChannelThreshold := formatDictValueToString(sortedKeyTable, domInfoDict, SfpDomChannelThresholdMap, DomChannelThresholdUnitMap)

		thresholdData, ok := outputDom["ThresholdData"].(map[string]interface{})
		if !ok {
			thresholdData = make(map[string]interface{})
			outputDom["ThresholdData"] = thresholdData
		}

		for key, value := range outputChannelThreshold {
			thresholdData[key] = value
		}
	}
	return outputDom
}

func IsRj45Port(iface string) bool {
	queries := [][]string{
		{"STATE_DB", "TRANSCEIVER_INFO", iface},
	}
	sfpInfoDict, _ := GetMapFromQueries(queries)
	portType, _ := sfpInfoDict["type"].(string)
	return portType == "RJ45"
}

func convertInterfaceSfpInfoToCliOutputString(iface string, dumpDom bool) string {
	output := make(map[string]interface{})
	var queries [][]string

	pmr := &common.PortMappingRetriever{}
	pmr.ReadPorttabMappings()

	firstPort := common.GetFirstSubPort(pmr, iface)
	if firstPort == "" {
		fmt.Printf("Error: Unable to get first subport for %s while converting SFP info\n", iface)
		return "SFP EEPROM Not detected\n"
	}

	queries = [][]string{
		{"STATE_DB", "TRANSCEIVER_INFO", iface},
	}
	sfpInfoDict, _ := GetMapFromQueries(queries)

	queries = [][]string{
		{"STATE_DB", "TRANSCEIVER_FIRMWARE_INFO", iface},
	}
	sfpFirmwareInfoDict, _ := GetMapFromQueries(queries)

	if len(sfpInfoDict) != 0 {
		isSfpCmis := isTransceiverCmis(sfpInfoDict)
		if portType, ok := sfpInfoDict["type"].(string); ok && portType == RJ45PortType {
			return "SFP EEPROM is not applicable for RJ45 port"
		} else {
			// output = "SFP EEPROM detected\n"
			sfpInfoOutput := convertSfpInfoToOutputString(sfpInfoDict, sfpFirmwareInfoDict)
			output = sfpInfoOutput
			if dumpDom {
				queries = [][]string{
					{"STATE_DB", "TRANSCEIVER_DOM_SENSOR", firstPort},
				}
				domInfoDict, err := GetMapFromQueries(queries)
				if err != nil {
					domInfoDict = make(map[string]interface{})
				}

				queries = [][]string{
					{"STATE_DB", "TRANSCEIVER_DOM_THRESHOLD", firstPort},
				}
				domThresholdDict, err := GetMapFromQueries(queries)
				if err != nil {
					domThresholdDict = make(map[string]interface{})
				}
				if len(domThresholdDict) != 0 {
					for k, v := range domThresholdDict {
						domInfoDict[k] = v
					}
				}

				if val, ok := sfpInfoDict["type"]; ok {
					if sfpType, ok := val.(string); ok {
						domOutput := convertDomToOutputString(sfpType, isSfpCmis, domInfoDict)
						for k, v := range domOutput {
							output[k] = v
						}
					}
				}
			}
		}
	} else {
		if IsRj45Port(iface) {
			return "SFP EEPROM is not applicable for RJ45 port"
		} else {
			return "SFP EEPROM Not detected\n"
		}
	}

	b, err := json.Marshal(output)
	if err != nil {
		return "Error serializing SFP info\n"
	}
	return string(b)
}

func convertSfpStatusToOutputString(sfpStatus map[string]interface{}, statusMap map[string]string, orderedKeys []string) map[string]interface{} {
	out := make(map[string]interface{})
	for _, k := range orderedKeys {
		label, ok := statusMap[k]
		if !ok {
			continue
		}
		val, present := sfpStatus[k]
		if !present {
			continue
		}
		out[label] = val
	}
	return out
}

func convertInterfaceSfpStatusToCliOutputString(iface string) string {
	pmr := &common.PortMappingRetriever{}
	pmr.ReadPorttabMappings()

	firstSubport := common.GetFirstSubPort(pmr, iface)
	if firstSubport == "" {
		fmt.Printf("Error: Unable to get first subport for %s while converting SFP status\n", iface)
		return fmt.Sprintf("%s\n", TransceiverStatusNotApplicable)
	}

	sfpStatusDict, _ := GetMapFromQueries([][]string{{StateDb, "TRANSCEIVER_STATUS", firstSubport}})
	if len(sfpStatusDict) == 0 {
		log.V(5).Infof("No sfp status for iface=%s firstSubport=%s", iface, firstSubport)
		return fmt.Sprintf("%s\n", TransceiverStatusNotApplicable)
	}

	mergedSfpStatusDict := make(map[string]interface{})
	for k, v := range sfpStatusDict {
		mergedSfpStatusDict[k] = v
	}

	// Additional handling to ensure that the CLI output remains the same after restructuring the diagnostic data in the state DB
	mergeList := []struct {
		table string
		key   string
	}{
		{"TRANSCEIVER_STATUS_SW", iface},
		{"TRANSCEIVER_STATUS_FLAG", firstSubport},
		{"TRANSCEIVER_DOM_FLAG", firstSubport},
	}

	for _, q := range mergeList {
		if statusInDB, _ := GetMapFromQueries([][]string{{StateDb, q.table, q.key}}); len(statusInDB) != 0 {
			for k, v := range statusInDB {
				mergedSfpStatusDict[k] = v
			}
		}
	}

	if len(mergedSfpStatusDict) <= 2 {
		return fmt.Sprintf("%s\n", TransceiverStatusNotApplicable)
	}

	output := make(map[string]interface{})

	// Common section
	qsfpMap := convertSfpStatusToOutputString(mergedSfpStatusDict, QsfpStatusMap, qsfpStatusOrder)
	for k, v := range qsfpMap {
		output[k] = v
	}

	// CMIS specific section
	if _, has := mergedSfpStatusDict["module_state"]; has {
		normalizeCmisFlagKeys(mergedSfpStatusDict)
		convertVdmFieldsToLegacyFields(firstSubport, mergedSfpStatusDict, CmisVdmToLegacyStatusMap, "FLAG")
		cmisMap := convertSfpStatusToOutputString(mergedSfpStatusDict, CmisStatusMap, cmisStatusOrder)
		for k, v := range cmisMap {
			output[k] = v
		}
	}

	// C-CMIS specific section
	if _, has := mergedSfpStatusDict["tuning_in_progress"]; has {
		convertVdmFieldsToLegacyFields(firstSubport, mergedSfpStatusDict, CCmisVdmToLegacyStatusMap, "FLAG")
		ccmisMap := convertSfpStatusToOutputString(mergedSfpStatusDict, CCmisStatusMap, ccmisStatusOrder)
		for k, v := range ccmisMap {
			output[k] = v
		}
	}

	if len(output) <= 0 {
		return fmt.Sprintf("%s\n", TransceiverStatusNotApplicable)
	}

	b, err := json.Marshal(output)
	if err != nil {
		return "Error serializing SFP status\n"
	}
	return string(b)
}

// Converts VDM fields from the database into legacy field names and updates the provided dictionary with the converted values.
// This function ensures backward compatibility by mapping VDM fields to their corresponding
// legacy field names based on the specified field type ('FLAG' or 'THRESHOLD').
// Args:
// 		interfaceName (str): The name of the interface for which VDM data is being retrieved.
//      dictToBeUpdated (dict): The dictionary to be updated with the converted legacy field names and values.
//      vdmToLegacyFieldMap (dict): A mapping of VDM field names to their corresponding legacy field name prefixes.
//      vdmFieldType (str): Specifies the type of VDM fields to process. It can be either 'FLAG' or 'THRESHOLD'.

func convertVdmFieldsToLegacyFields(interfaceName string,
	dictToBeUpdated map[string]interface{},
	vdmToLegacyFieldMap map[string]string,
	vdmFieldType string) {

	if dictToBeUpdated == nil {
		return
	}
	if vdmFieldType != "FLAG" && vdmFieldType != "THRESHOLD" {
		return
	}

	// Retrieve VDM data from the database
	getVdmFromDB := func(prefix string) map[string]interface{} {
		queries := [][]string{
			{"STATE_DB", fmt.Sprintf("%s_%s", prefix, vdmFieldType), interfaceName},
		}
		m, err := GetMapFromQueries(queries)
		if err != nil || m == nil {
			return nil
		}
		return m
	}

	halarm := getVdmFromDB("TRANSCEIVER_VDM_HALARM")
	lalarm := getVdmFromDB("TRANSCEIVER_VDM_LALARM")
	hwarn := getVdmFromDB("TRANSCEIVER_VDM_HWARN")
	lwarn := getVdmFromDB("TRANSCEIVER_VDM_LWARN")

	vdmThresholdTypes := map[string]map[string]interface{}{
		"highalarm":   halarm,
		"lowalarm":    lalarm,
		"highwarning": hwarn,
		"lowwarning":  lwarn,
	}

	for vdmField, legacyPrefix := range vdmToLegacyFieldMap {
		for threshType, vdmDict := range vdmThresholdTypes {
			if vdmDict == nil {
				continue
			}
			if val, ok := vdmDict[vdmField]; ok {
				var legacyName string
				if vdmFieldType == "FLAG" {
					legacyName = fmt.Sprintf("%s%s_flag", legacyPrefix, threshType)
				} else { // THRESHOLD
					legacyName = fmt.Sprintf("%s%s", legacyPrefix, threshType)
				}
				dictToBeUpdated[legacyName] = val
			}
		}
	}
}

func normalizeCmisFlagKeys(m map[string]interface{}) {
	rules := [][2]string{
		{"temphighalarm_flag", "tempHAlarm"},
		{"temphighwarning_flag", "tempHWarn"},
		{"templowwarning_flag", "tempLWarn"},
		{"templowalarm_flag", "tempLAlarm"},
		{"vcchighalarm_flag", "vccHAlarm"},
		{"vcchighwarning_flag", "vccHWarn"},
		{"vcclowwarning_flag", "vccLWarn"},
		{"vcclowalarm_flag", "vccLAlarm"},
	}
	for _, r := range rules {
		if v, ok := m[r[0]]; ok {
			if _, exists := m[r[1]]; !exists {
				m[r[1]] = v
			}
		}
	}
}
