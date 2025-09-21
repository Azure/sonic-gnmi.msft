package common

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var HostDevicePath string = "/usr/share/sonic/device"
var MachineConfPath string = "/host/machine.conf"

const (
	asicConfFilename      = "asic.conf"
	containerPlatformPath = "/usr/share/sonic/platform"
	platformEnvConfFile   = "platform_env.conf"
	serial                = "serial"
	model                 = "model"
	revision              = "revision"
	platform              = "platform"
	hwsku                 = "hwsku"
	platformEnvVar        = "PLATFORM"
	chassisInfoKey        = "chassis 1"
	space                 = " "
)

var hwInfoDict map[string]interface{}
var hwInfoOnce sync.Once

func GetChassisInfo() (map[string]string, error) {
	chassisDict := make(map[string]string)
	queries := [][]string{
		{"STATE_DB", "CHASSIS_INFO"},
	}

	chassisInfo, err := GetMapFromQueries(queries)
	if err != nil {
		return nil, err
	}
	chassisDict[serial] = "N/A"
	chassisDict[model] = "N/A"
	chassisDict[revision] = "N/A"

	if chassisMetadata, ok := chassisInfo[chassisInfoKey].(map[string]interface{}); ok {
		chassisDict[serial] = GetValueOrDefault(chassisMetadata, serial, "N/A")
		chassisDict[model] = GetValueOrDefault(chassisMetadata, model, "N/A")
		chassisDict[revision] = GetValueOrDefault(chassisMetadata, revision, "N/A")
	}

	return chassisDict, nil
}

func GetUptime(params []string) string {
	uptimeCommand := "uptime"

	if params != nil && len(params) > 0 {
		for _, param := range params {
			uptimeCommand += (space + param)
		}
	}
	uptime, err := GetDataFromHostCommand(uptimeCommand)
	if err != nil {
		return "N/A"
	}

	return strings.TrimSpace(uptime)
}

func GetDockerInfo() string {
	cmdOutput, err := GetDataFromHostCommand(`bash -o pipefail -c 'docker images --no-trunc --format '\''{"Repository":"{{.Repository}}","Tag":"{{.Tag}}","ID":"{{.ID}}","Size":"{{.Size}}"}'\'' | jq -s .'`)

	if err != nil {
		return "N/A"
	}

	return cmdOutput
}

func GetPlatformInfo(versionInfo map[string]interface{}) (map[string]interface{}, error) {
	hwInfoOnce.Do(func() {
		hwInfoDict = make(map[string]interface{})
		hwInfoDict[platform] = GetPlatform()
		hwInfoDict[hwsku] = GetHwsku()
		if versionInfo != nil {
			if asicType, ok := versionInfo["asic_type"]; ok {
				hwInfoDict["asic_type"] = asicType
			}
		}
		hwInfoDict["asic_count"] = "N/A"
		asicCount, err := GetAsicCount()
		if err == nil {
			hwInfoDict["asic_count"] = asicCount
		}
		switchType := GetLocalhostInfo("switch_type")
		hwInfoDict["switch_type"] = switchType
	})
	return hwInfoDict, nil
}

// Platform and hardware info functions
func GetPlatform() string {
	platformEnv := os.Getenv(platformEnvVar)
	if platformEnv != "" {
		return platformEnv
	}
	machineInfo := GetMachineInfo()
	if machineInfo != nil {
		if val, ok := machineInfo["onie_platform"]; ok {
			return val
		} else if val, ok := machineInfo["aboot_platform"]; ok {
			return val
		}
	}
	return GetLocalhostInfo("platform")
}

func GetMachineInfo() map[string]string {
	data, err := ReadConfToMap(MachineConfPath)
	if err != nil {
		return nil
	}
	result := make(map[string]string)
	for k, v := range data {
		if strVal, ok := v.(string); ok {
			result[k] = strVal
		}
	}
	return result
}

func GetHwsku() string {
	return GetLocalhostInfo(hwsku)
}

func GetAsicCount() (int, error) {
	val := GetAsicPresenceList()
	if val == nil {
		return 0, fmt.Errorf("no ASIC presence list found")
	}
	if len(val) == 0 {
		return 0, fmt.Errorf("ASIC presence list is empty")
	}
	return len(val), nil
}

// ASIC and multi-ASIC functions
func IsMultiAsic() bool {
	configuredAsicCount := ReadAsicConfValue()
	return configuredAsicCount > 1
}

func ReadAsicConfValue() int {
	asicConfFilePath := GetAsicConfFilePath()
	if asicConfFilePath == "" {
		return 1
	}
	file, err := os.Open(asicConfFilePath)
	if err != nil {
		return 1
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.SplitN(line, "=", 2)
		if len(tokens) < 2 {
			continue
		}
		if strings.ToLower(tokens[0]) == "num_asic" {
			numAsics, err := strconv.Atoi(strings.TrimSpace(tokens[1]))
			if err == nil {
				return numAsics
			}
		}
	}
	return 1
}

// ConfigDB and info helpers
func GetLocalhostInfo(field string) string {
	queries := [][]string{
		{"CONFIG_DB", "DEVICE_METADATA"},
	}
	metadata, err := GetMapFromQueries(queries)
	if err != nil {
		return ""
	}
	if localhost, ok := metadata["localhost"].(map[string]interface{}); ok {
		if val, ok := localhost[field].(string); ok {
			return val
		}
	}
	return ""
}

// GetAsicConfFilePath retrieves the path to the ASIC configuration file on the device.
// Returns the path as a string if found, or an empty string if not found.
func GetAsicConfFilePath() string {
	// 1. Check container platform path
	candidate := filepath.Join(containerPlatformPath, asicConfFilename)
	if FileExists(candidate) {
		return candidate
	}

	// 2. Check host device path with platform
	platform := GetPlatform()
	if platform != "" {
		candidate = filepath.Join(HostDevicePath, platform, asicConfFilename)
		if FileExists(candidate) {
			return candidate
		}
	}

	// Not found
	return ""
}

func GetAsicPresenceList() []int {
	var asicsList []int
	if IsMultiAsic() {
		//Currently MultiAsic is not configured. One can refer PR change history to refer the removed code(MultiAsic support).
		asicsList = append(asicsList, 0)
	} else {
		numAsics := ReadAsicConfValue()
		for i := 0; i < numAsics; i++ {
			asicsList = append(asicsList, i)
		}
	}
	return asicsList
}
