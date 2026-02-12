package show_client

import (
	"encoding/json"
	"fmt"
	"sort"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// PlatformSummary represents the output structure for show platform summary
type PlatformSummary struct {
	Platform         string `json:"platform"`
	Hwsku            string `json:"hwsku"`
	AsicType         string `json:"asic_type,omitempty"`
	AsicCount        string `json:"asic_count"`
	SerialNumber     string `json:"serial_number"`
	ModelNumber      string `json:"model_number"`
	HardwareRevision string `json:"hardware_revision"`
}

// SyseepromInfo represents the EEPROM TLV information structure
type SyseepromInfo struct {
	TlvHeader struct {
		ID          string `json:"id"`
		Version     string `json:"version"`
		TotalLength string `json:"length"`
	} `json:"tlvHeader"`
	TlvList        []SyseepromTlv `json:"tlv_list"`
	ChecksumValid  bool           `json:"checksum_valid"`
}

// SyseepromTlv represents a single TLV entry
type SyseepromTlv struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Length string `json:"length"`
	Value  string `json:"value"`
}

// getPlatformSummary implements the "show platform summary" command
func getPlatformSummary(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	// Get version info to extract ASIC type
	versionInfo, err := common.ReadYamlToMap(SonicVersionYamlPath)
	if err != nil {
		log.Errorf("Failed to read version info from %s: %v", SonicVersionYamlPath, err)
		versionInfo = nil
	}

	// Get platform information (platform, hwsku, asic_type, asic_count)
	platformInfo, err := common.GetPlatformInfo(versionInfo)
	if err != nil {
		log.Errorf("Failed to get platform info: %v", err)
		return nil, err
	}

	// Get chassis information (serial, model, revision)
	chassisInfo, err := common.GetChassisInfo()
	if err != nil {
		log.Errorf("Failed to get chassis info: %v", err)
		return nil, err
	}

	// Build the response structure
	summary := PlatformSummary{
		Platform:         common.GetValueOrDefault(platformInfo, "platform", "N/A"),
		Hwsku:            common.GetValueOrDefault(platformInfo, "hwsku", "N/A"),
		AsicCount:        common.GetValueOrDefault(platformInfo, "asic_count", "N/A"),
		SerialNumber:     chassisInfo["serial"],
		ModelNumber:      chassisInfo["model"],
		HardwareRevision: chassisInfo["revision"],
	}

	// Add ASIC type if available
	if asicType, ok := platformInfo["asic_type"]; ok {
		if asicTypeStr, ok := asicType.(string); ok && asicTypeStr != "" {
			summary.AsicType = asicTypeStr
		}
	}

	// Marshal to JSON
	return json.Marshal(summary)
}

// getPlatformSyseeprom implements the "show platform syseeprom" command
func getPlatformSyseeprom(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	queries := [][]string{
		{"STATE_DB", "EEPROM_INFO"},
	}

	eepromData, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get EEPROM info from STATE_DB: %v", err)
		return nil, err
	}

	// Check if EEPROM is initialized
	if stateInfo, ok := eepromData["State"].(map[string]interface{}); ok {
		if initialized := common.GetValueOrDefault(stateInfo, "Initialized", "0"); initialized != "1" {
			log.Errorf("EEPROM info not initialized in STATE_DB")
			return nil, fmt.Errorf("EEPROM info not initialized")
		}
	} else {
		log.Errorf("EEPROM State info not found in STATE_DB")
		return nil, fmt.Errorf("EEPROM State info not found")
	}

	// Detect format: TLV (Mellanox/ONIE) vs simple key-value (Broadcom)
	_, hasTlvHeader := eepromData["TlvHeader"].(map[string]interface{})

	if hasTlvHeader {
		return marshalTlvFormat(eepromData)
	} else {
		return marshalSimpleFormat(eepromData)
	}
}

// marshalTlvFormat handles TLV-based EEPROM format (Mellanox/ONIE)
func marshalTlvFormat(eepromData map[string]interface{}) ([]byte, error) {
	var syseepromInfo SyseepromInfo

	// Parse TLV Header
	if tlvHeader, ok := eepromData["TlvHeader"].(map[string]interface{}); ok {
		syseepromInfo.TlvHeader.ID = common.GetValueOrDefault(tlvHeader, "Id String", "N/A")
		syseepromInfo.TlvHeader.Version = common.GetValueOrDefault(tlvHeader, "Version", "N/A")
		syseepromInfo.TlvHeader.TotalLength = common.GetValueOrDefault(tlvHeader, "Total Length", "N/A")
	}

	// Parse TLV entries dynamically from database
	syseepromInfo.TlvList = make([]SyseepromTlv, 0)

	for key, value := range eepromData {
		// Skip metadata entries
		if key == "State" || key == "TlvHeader" || key == "Checksum" {
			continue
		}

		tlvData, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		tlvCode := key // Key is the TLV code (e.g., "0x21", "0xfd")

		// Handle Vendor Extension (0xfd) which may have multiple entries
		if tlvCode == "0xfd" {
			numVendorExt := 0
			if numVendorExtStr := common.GetValueOrDefault(tlvData, "Num_vendor_ext", "0"); numVendorExtStr != "" {
				fmt.Sscanf(numVendorExtStr, "%d", &numVendorExt)
			}

			for i := 0; i < numVendorExt; i++ {
				tlv := SyseepromTlv{
					Code:   tlvCode,
					Name:   common.GetValueOrDefault(tlvData, fmt.Sprintf("Name_%d", i), "N/A"),
					Length: common.GetValueOrDefault(tlvData, fmt.Sprintf("Len_%d", i), "N/A"),
					Value:  common.GetValueOrDefault(tlvData, fmt.Sprintf("Value_%d", i), "N/A"),
				}
				syseepromInfo.TlvList = append(syseepromInfo.TlvList, tlv)
			}
		} else {
			// Regular TLV entry
			tlv := SyseepromTlv{
				Code:   tlvCode,
				Name:   common.GetValueOrDefault(tlvData, "Name", "N/A"),
				Length: common.GetValueOrDefault(tlvData, "Len", "N/A"),
				Value:  common.GetValueOrDefault(tlvData, "Value", "N/A"),
			}
			syseepromInfo.TlvList = append(syseepromInfo.TlvList, tlv)
		}
	}

	// Parse checksum validity
	if checksumData, ok := eepromData["Checksum"].(map[string]interface{}); ok {
		checksumValid := common.GetValueOrDefault(checksumData, "Valid", "0")
		syseepromInfo.ChecksumValid = (checksumValid == "1")
	}

	return json.Marshal(syseepromInfo)
}

// marshalSimpleFormat handles simple key-value EEPROM format (Broadcom)
func marshalSimpleFormat(eepromData map[string]interface{}) ([]byte, error) {
	simpleFormat := make(map[string]string)

	for key, value := range eepromData {
		// Skip State entry
		if key == "State" {
			continue
		}

		if valueMap, ok := value.(map[string]interface{}); ok {
			simpleFormat[key] = common.GetValueOrDefault(valueMap, "Value", "N/A")
		} else {
			simpleFormat[key] = fmt.Sprintf("%v", value)
		}
	}

	return json.Marshal(simpleFormat)
}

// PsuStatus represents the status information for a single PSU
// Matches the Python psushow output format exactly
type PsuStatus struct {
	Index    string `json:"index"`
	Name     string `json:"name"`
	Presence string `json:"presence"`
	Model    string `json:"model"`
	Serial   string `json:"serial"`
	Revision string `json:"revision"`
	Voltage  string `json:"voltage"`
	Current  string `json:"current"`
	Power    string `json:"power"`
	Status   string `json:"status"`
	LED      string `json:"led_status"`
}

// getPlatformPsustatus implements the "show platform psustatus" command
// This replicates the logic from sonic-utilities/scripts/psushow
// Supports filtering by PSU index via options (index=INTEGER)
func getPlatformPsustatus(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	// Get optional PSU index filter
	psuIndex := 0
	if indexValue, ok := options[common.OptionKeyPsuIndex]; ok {
		if idx, ok := indexValue.(int); ok {
			psuIndex = idx
		}
	}

	queries := [][]string{
		{"STATE_DB", "PSU_INFO"},
		{"STATE_DB", "CHASSIS_INFO"},
	}

	allData, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get PSU info from STATE_DB: %v", err)
		return nil, err
	}

	// Check if PSUs exist by checking chassis info
	chassisKey := "chassis 1"
	numPsusStr := ""
	if chassisInfo, ok := allData[chassisKey].(map[string]interface{}); ok {
		numPsusStr = common.GetValueOrDefault(chassisInfo, "psu_num", "")
	}

	if numPsusStr == "" {
		log.Errorf("No PSUs found in CHASSIS_INFO")
		return nil, fmt.Errorf("no PSUs detected")
	}

	// Collect all PSU keys and sort them naturally
	psuKeys := make([]string, 0)
	for key := range allData {
		if key != chassisKey {
			psuKeys = append(psuKeys, key)
		}
	}
	sort.Strings(psuKeys)

	// Collect all PSU status information
	psuStatusList := make([]PsuStatus, 0)

	// Iterate through sorted PSU keys with 1-based index
	for psuIdx, psuName := range psuKeys {
		value := allData[psuName]
		psuData, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		presence := common.GetValueOrDefault(psuData, "presence", "false")

		// Determine PSU status
		var status string
		if presence == "true" {
			operStatus := common.GetValueOrDefault(psuData, "status", "")
			if operStatus == "true" {
				powerOverload := common.GetValueOrDefault(psuData, "power_overload", "")
				// Python checks: 'WARNING' if power_overload == 'True' else 'OK'
				if powerOverload == "True" {
					status = "WARNING"
				} else {
					status = "OK"
				}
			} else {
				status = "NOT OK"
			}
		} else {
			status = "NOT PRESENT"
		}

		// Build PSU status entry matching Python format exactly
		psuStatus := PsuStatus{
			Index:    fmt.Sprintf("%d", psuIdx+1), // 1-based index as string
			Name:     psuName,
			Presence: presence,
			Status:   status,
		}

		// LED status (can be empty/None in Python)
		psuStatus.LED = common.GetValueOrDefault(psuData, "led_status", "")

		// Python: model = db.get(..., 'model') if presence else 'N/A'
		if presence == "true" {
			psuStatus.Model = common.GetValueOrDefault(psuData, "model", "N/A")
			psuStatus.Serial = common.GetValueOrDefault(psuData, "serial", "N/A")
			psuStatus.Revision = common.GetValueOrDefault(psuData, "revision", "N/A")
			psuStatus.Voltage = common.GetValueOrDefault(psuData, "voltage", "N/A")
			psuStatus.Current = common.GetValueOrDefault(psuData, "current", "N/A")
			psuStatus.Power = common.GetValueOrDefault(psuData, "power", "N/A")
		} else {
			psuStatus.Model = "N/A"
			psuStatus.Serial = "N/A"
			psuStatus.Revision = "N/A"
			psuStatus.Voltage = "N/A"
			psuStatus.Current = "N/A"
			psuStatus.Power = "N/A"
		}

		psuStatusList = append(psuStatusList, psuStatus)
	}

	// Filter by index if specified (1-based index like Python)
	if psuIndex > 0 {
		if psuIndex > len(psuStatusList) {
			log.Errorf("PSU %d is not available. Number of supported PSUs: %d", psuIndex, len(psuStatusList))
			return nil, fmt.Errorf("PSU %d is not available. Number of supported PSUs: %d", psuIndex, len(psuStatusList))
		}
		// Return only the requested PSU (convert to 0-based index)
		psuStatusList = []PsuStatus{psuStatusList[psuIndex-1]}
	}

	// Marshal to JSON
	return json.Marshal(psuStatusList)
}
