package show_client

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	log "github.com/golang/glog"
	natural "github.com/maruel/natural"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	"github.com/sonic-net/sonic-gnmi/show_client/helpers"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

const (
	chassisKey = "chassis 1"
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

// FanStatus represents the status information for a single fan
type FanStatus struct {
	Drawer    string `json:"drawer"`
	LED       string `json:"led"`
	FAN       string `json:"fan"`
	Speed     string `json:"speed"`
	Direction string `json:"direction"`
	Presence  string `json:"presence"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// TemperatureInfo represents temperature sensor information
type TemperatureInfo struct {
	Sensor      string `json:"sensor"`
	Temperature string `json:"temperature"`
	HighTH      string `json:"high_th"`
	LowTH       string `json:"low_th"`
	CritHighTH  string `json:"crit_high_th"`
	CritLowTH   string `json:"crit_low_th"`
	Warning     string `json:"warning"`
	Timestamp   string `json:"timestamp"`
}

// VoltageInfo represents voltage sensor information
type VoltageInfo struct {
	Sensor     string `json:"sensor"`
	Voltage    string `json:"voltage"`
	HighTH     string `json:"high_th"`
	LowTH      string `json:"low_th"`
	CritHighTH string `json:"crit_high_th"`
	CritLowTH  string `json:"crit_low_th"`
	Warning    string `json:"warning"`
	Timestamp  string `json:"timestamp"`
}

// CurrentInfo represents current sensor information
type CurrentInfo struct {
	Sensor     string `json:"sensor"`
	Current    string `json:"current"`
	HighTH     string `json:"high_th"`
	LowTH      string `json:"low_th"`
	CritHighTH string `json:"crit_high_th"`
	CritLowTH  string `json:"crit_low_th"`
	Warning    string `json:"warning"`
	Timestamp  string `json:"timestamp"`
}

// ComponentInfo represents individual component firmware information
type ComponentInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// FirmwareInfo represents the complete firmware status
type FirmwareInfo struct {
	Chassis    string          `json:"chassis"`
	Module     string          `json:"module"`
	Components []ComponentInfo `json:"components"`
}

// SsdHealthInfo represents SSD health information.
// Fields are conditionally populated based on verbose/vendor options,
// matching Python ssdutil output:
//
//	always:  disk_type, device_model, health, temperature
//	verbose: +firmware, +serial
//	vendor:  +vendor_output
type SsdHealthInfo struct {
	DiskType     string `json:"disk_type"`
	DeviceModel  string `json:"device_model"`
	Firmware     string `json:"firmware,omitempty"`
	Serial       string `json:"serial,omitempty"`
	Health       string `json:"health"`
	Temperature  string `json:"temperature"`
	VendorOutput string `json:"vendor_output,omitempty"`
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

	// Get CHASSIS_INFO
	chassisQueries := [][]string{
		{common.StateDb, "CHASSIS_INFO"},
	}
	chassisData, err := common.GetMapFromQueries(chassisQueries)
	if err != nil {
		log.Errorf("Failed to get CHASSIS_INFO from STATE_DB: %v", err)
		return nil, err
	}

	// Check if PSUs exist by checking chassis info
	numPsusStr := ""
	if chassisInfo, ok := chassisData[chassisKey].(map[string]interface{}); ok {
		numPsusStr = common.GetValueOrDefault(chassisInfo, "psu_num", "")
	}

	if numPsusStr == "" {
		log.Errorf("No PSUs found in CHASSIS_INFO")
		return nil, fmt.Errorf("no PSUs detected")
	}

	// Get PSU_INFO
	psuQueries := [][]string{
		{common.StateDb, "PSU_INFO"},
	}
	psuData, err := common.GetMapFromQueries(psuQueries)
	if err != nil {
		log.Errorf("Failed to get PSU_INFO from STATE_DB: %v", err)
		return nil, err
	}

	// Collect all PSU keys and sort them naturally
	psuKeys := make([]string, 0)
	for key := range psuData {
		psuKeys = append(psuKeys, key)
	}
	sort.Sort(natural.StringSlice(psuKeys))

	// Collect all PSU status information
	psuStatusList := make([]PsuStatus, 0)

	// Iterate through sorted PSU keys with 1-based index
	for psuIdx, psuName := range psuKeys {
		value := psuData[psuName]
		psuInfo, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		presence := common.GetValueOrDefault(psuInfo, "presence", "false")

		// Determine PSU status
		var status string
		if presence == "true" {
			operStatus := common.GetValueOrDefault(psuInfo, "status", "")
			if operStatus == "true" {
				powerOverload := common.GetValueOrDefault(psuInfo, "power_overload", "")
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
		psuStatus.LED = common.GetValueOrDefault(psuInfo, "led_status", "")

		if presence == "true" {
			psuStatus.Model = common.GetValueOrDefault(psuInfo, "model", "N/A")
			psuStatus.Serial = common.GetValueOrDefault(psuInfo, "serial", "N/A")
			psuStatus.Revision = common.GetValueOrDefault(psuInfo, "revision", "N/A")
			psuStatus.Voltage = common.GetValueOrDefault(psuInfo, "voltage", "N/A")
			psuStatus.Current = common.GetValueOrDefault(psuInfo, "current", "N/A")
			psuStatus.Power = common.GetValueOrDefault(psuInfo, "power", "N/A")
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

// getPlatformFan implements the "show platform fan" command
func getPlatformFan(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	queries := [][]string{
		{common.StateDb, "FAN_INFO"},
	}

	fanData, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get FAN_INFO from STATE_DB: %v", err)
		return nil, err
	}

	if len(fanData) == 0 {
		log.Info("Fan Not detected")
		return json.Marshal(map[string]string{"message": "Fan not detected"})
	}

	// Collect all fan keys and sort them naturally
	fanKeys := make([]string, 0)
	for key := range fanData {
		fanKeys = append(fanKeys, key)
	}
	sort.Sort(natural.StringSlice(fanKeys))

	// Collect all fan status information
	fanStatusList := make([]FanStatus, 0)

	for _, fanKey := range fanKeys {
		value := fanData[fanKey]
		fanInfo, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract fan name from key (after the pipe)
		fanName := fanKey

		// Format presence value: "true" -> "Present", "false" -> "Not Present"
		presence := common.GetValueOrDefault(fanInfo, "presence", "false")
		presenceLower := strings.ToLower(presence)
		presence = "Not Present"
		if presenceLower == "true" {
			presence = "Present"
		}

		// Format status value: "true" -> "OK", "false" -> "Not OK"
		status := common.GetValueOrDefault(fanInfo, "status", "false")
		statusLower := strings.ToLower(status)
		status = "Not OK"
		if statusLower == "true" {
			status = "OK"
		}

		// Format speed value: if > 100 -> "{speed}RPM", if <= 100 -> "{speed}%"
		speedRaw := common.GetValueOrDefault(fanInfo, "speed", "N/A")
		var speed string
		speedFloat, err := strconv.ParseFloat(speedRaw, 64)
		if err != nil {
			// Parsing failed (e.g., "N/A" or invalid value), use raw value as-is
			speed = speedRaw
		} else {
			// Successfully parsed, format based on value
			if speedFloat > 100 {
				speed = fmt.Sprintf("%dRPM", int(speedFloat))
			} else {
				speed = fmt.Sprintf("%s%%", speedRaw)
			}
		}

		fanStatus := FanStatus{
			Drawer:    common.GetValueOrDefault(fanInfo, "drawer_name", "N/A"),
			LED:       common.GetValueOrDefault(fanInfo, "led_status", "N/A"),
			FAN:       fanName,
			Speed:     speed,
			Direction: common.GetValueOrDefault(fanInfo, "direction", "N/A"),
			Presence:  presence,
			Status:    status,
			Timestamp: common.GetValueOrDefault(fanInfo, "timestamp", "N/A"),
		}

		fanStatusList = append(fanStatusList, fanStatus)
	}

	return json.Marshal(fanStatusList)
}

// getPlatformTemperature implements the "show platform temperature" command
func getPlatformTemperature(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	return helpers.GetSensorJSON("TEMPERATURE_INFO", "temperature", func(data map[string]string) TemperatureInfo {
		return TemperatureInfo{
			Sensor:      data["sensor"],
			Temperature: data["value"],
			HighTH:      data["high_th"],
			LowTH:       data["low_th"],
			CritHighTH:  data["crit_high_th"],
			CritLowTH:   data["crit_low_th"],
			Warning:     data["warning"],
			Timestamp:   data["timestamp"],
		}
	})
}

// getPlatformVoltage implements the "show platform voltage" command
func getPlatformVoltage(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	return helpers.GetSensorJSON("VOLTAGE_INFO", "voltage", func(data map[string]string) VoltageInfo {
		return VoltageInfo{
			Sensor:     data["sensor"],
			Voltage:    data["value"],
			HighTH:     data["high_th"],
			LowTH:      data["low_th"],
			CritHighTH: data["crit_high_th"],
			CritLowTH:  data["crit_low_th"],
			Warning:    data["warning"],
			Timestamp:  data["timestamp"],
		}
	})
}

// getPlatformCurrent implements the "show platform current" command
func getPlatformCurrent(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	return helpers.GetSensorJSON("CURRENT_INFO", "current", func(data map[string]string) CurrentInfo {
		return CurrentInfo{
			Sensor:     data["sensor"],
			Current:    data["value"],
			HighTH:     data["high_th"],
			LowTH:      data["low_th"],
			CritHighTH: data["crit_high_th"],
			CritLowTH:  data["crit_low_th"],
			Warning:    data["warning"],
			Timestamp:  data["timestamp"],
		}
	})
}

// getPlatformFirmware implements the "show platform firmware status" command
func getPlatformFirmware(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	// Get all firmware data using helper
	firmwareDataList, err := helpers.GetAllFirmwareData()
	if err != nil {
		log.V(1).Infof("Error getting firmware data: %v", err)
		return json.Marshal(FirmwareInfo{
			Chassis:    "N/A",
			Module:     "N/A",
			Components: []ComponentInfo{},
		})
	}

	if len(firmwareDataList) == 0 {
		return json.Marshal(FirmwareInfo{
			Chassis:    "N/A",
			Module:     "N/A",
			Components: []ComponentInfo{},
		})
	}

	// Extract chassis and module from first entry, build components list
	chassisName := "N/A"
	moduleName := "N/A"
	components := make([]ComponentInfo, 0, len(firmwareDataList))

	for i, data := range firmwareDataList {
		// Use chassis name from first non-empty entry
		if i == 0 || (chassisName == "N/A" && data.Chassis != "") {
			if data.Chassis != "" {
				chassisName = data.Chassis
			}
		}

		// For module, use first non-empty module name, or keep "N/A" if all are empty
		if i == 0 || (moduleName == "N/A" && data.Module != "") {
			if data.Module != "" {
				moduleName = data.Module
			}
		}

		// Create component info
		component := ComponentInfo{
			Name:        data.Component,
			Version:     data.Version,
			Description: data.Description,
		}

		components = append(components, component)
	}

	// Build final response structure
	firmwareInfo := FirmwareInfo{
		Chassis:    chassisName,
		Module:     moduleName,
		Components: components,
	}

	return json.Marshal(firmwareInfo)
}

// getPlatformSyseeprom implements "show platform syseeprom".
//  1. Get platform name
//  2. If platform matches kvm → return "does not support EEPROM"
//  3. If platform matches arista or kvm → no DB support, use platform API via nsenter
//  4. Otherwise → read from STATE_DB (EEPROM_INFO cached by syseepromd)
func getPlatformSyseeprom(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	platform := common.GetPlatform()
	log.V(2).Infof("getPlatformSyseeprom: detected platform=%q", platform)

	// Platforms that do not support EEPROM at all
	if helpers.MatchesPlatformPattern(platform, helpers.PlatformsWithoutEeprom) {
		return nil, fmt.Errorf("platform %s does not support EEPROM", platform)
	}

	// Platforms without EEPROM DB support — fall back to platform API via nsenter
	if helpers.MatchesPlatformPattern(platform, helpers.PlatformsWithoutEepromDb) {
		log.V(2).Infof("getPlatformSyseeprom: platform %q does not support EEPROM DB, using platform API", platform)
		return helpers.ReadEepromViaPlatformApi()
	}

	// Default path: read from STATE_DB (equivalent to decode-syseeprom -d)
	return helpers.ReadEepromFromDb()
}

// getPlatformSsdhealth implements the "show platform ssdhealth" command.
func getPlatformSsdhealth(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	var device string
	if len(args) > 0 {
		device = args[0]
	}

	ssdInfo, err := helpers.GetSsdHealthData(device)
	if err != nil {
		log.Infof("SSD not detected: %v", err)
		return json.Marshal(map[string]string{"message": "SSD not detected"})
	}

	ssdHealth := SsdHealthInfo{
		DiskType:    strings.ToUpper(ssdInfo.DiskType),
		DeviceModel: ssdInfo.Model,
		Health:      helpers.FormatHealth(ssdInfo.Health),
		Temperature: helpers.FormatTemperature(ssdInfo.Temperature),
	}

	if verbose, ok := options[OptionKeyVerbose].Bool(); ok && verbose {
		log.V(4).Infof("Verbose mode enabled, adding firmware=%s, serial=%s", ssdInfo.Firmware, ssdInfo.Serial)
		ssdHealth.Firmware = ssdInfo.Firmware
		ssdHealth.Serial = ssdInfo.Serial
	}

	if vendor, ok := options[OptionKeyVendor].Bool(); ok && vendor {
		log.V(4).Infof("Vendor mode enabled, adding vendor_output (len=%d)", len(ssdInfo.VendorOutput))
		ssdHealth.VendorOutput = ssdInfo.VendorOutput
	}

	return json.Marshal(ssdHealth)
}
