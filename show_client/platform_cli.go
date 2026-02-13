package show_client

import (
	"encoding/json"
	"fmt"
	"sort"

	log "github.com/golang/glog"
	natural "github.com/maruel/natural"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// PlatformSummary represents the output structure for show platform summary
type PlatformSummary struct {
	Platform         string `json:"platform"`
	Hwsku            string `json:"hwsku"`
	AsicType         string `json:"asic_type"`
	AsicCount        string `json:"asic_count"`
	SerialNumber     string `json:"serial"`
	ModelNumber      string `json:"model"`
	HardwareRevision string `json:"revision"`
	SwitchType       string `json:"switch_type,omitempty"`
}

// PsuStatus represents the status information for a single PSU
type PsuStatus struct {
	Index    string `json:"index"`
	Name     string `json:"name"`
	Presence string `json:"presence"`
	Status   string `json:"status"`
	LED      string `json:"led_status"`
	Model    string `json:"model"`
	Serial   string `json:"serial"`
	Revision string `json:"revision"`
	Voltage  string `json:"voltage"`
	Current  string `json:"current"`
	Power    string `json:"power"`
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
		AsicType:         common.GetValueOrDefault(platformInfo, "asic_type", "N/A"),
		AsicCount:        common.GetValueOrDefault(platformInfo, "asic_count", "N/A"),
		SerialNumber:     chassisInfo["serial"],
		ModelNumber:      chassisInfo["model"],
		HardwareRevision: chassisInfo["revision"],
	}

	// Add switch_type if available (omitempty)
	if switchType, ok := platformInfo["switch_type"]; ok {
		if switchTypeStr, ok := switchType.(string); ok && switchTypeStr != "" {
			summary.SwitchType = switchTypeStr
		}
	}

	// Marshal to JSON
	return json.Marshal(summary)
}

// getPlatformPsustatus implements the "show platform psustatus" command
// Supports filtering by PSU index via options (index=INTEGER)
func getPlatformPsustatus(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	// Get optional PSU index filter
	psuIndex := 0
	if idx, ok := options[OptionKeyPsuIndex].Int(); ok {
		psuIndex = idx
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
	sort.Sort(natural.StringSlice(psuKeys))

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

		// Build PSU status entry
		psuStatus := PsuStatus{
			Index:    fmt.Sprintf("%d", psuIdx+1), // 1-based index as string
			Name:     psuName,
			Presence: presence,
			Status:   status,
		}

		// LED status
		psuStatus.LED = common.GetValueOrDefault(psuData, "led_status", "")

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

	// Filter by index if specified
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
