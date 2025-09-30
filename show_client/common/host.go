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
    DockerInspectCmd = "docker inspect "
    DockerInspectDirCmd = " --format \"{{{{.GraphDriver.Data.MergedDir}}}}\""
)

var hwInfoDict map[string]interface{}
var hwInfoOnce sync.Once

func GetPlatformConfigFilePath() {
	candidate := filepath.Join(containerPlatformPath, platformEnvConfFile)
	if FileExists(candidate) {
		return candidate
	}

	// 2. Check host device path with platform
	platform := GetPlatform()
	if platform != "" {
		candidate = filepath.Join(HostDevicePath, platform, platformEnvConfFile)
		if FileExists(candidate) {
			return candidate
		}
	}

	// Not found
	return ""
}

func GetPlatformEnvConfig(varName string) (string, bool) {
    platformConfigFilePath := GetPlatformConfigFilePath()
	if platformConfigFilePath == "" {
		return "", false 
	}
	file, err := os.Open(platformConfigFilePath)
	if err != nil {
		return "", false
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.SplitN(line, "=", 2)
		if len(tokens) < 2 {
			continue
		}
		if strings.ToLower(tokens[0]) == varName {
            return tokens[1], true
		}
	}
	return "", false 

}

func IsExpectedValue(val string, expectedVal string) {
    if strings.TrimSpace(val) == expectedVal {
        return true
    }
    
    return false

}

func IsSupervisor() bool {
    val, found := GetPlatformEnvConfig("supervisor")
    if !found {
        return found
    }
    return IsExpectedValue(val, "1")
}

func IsDisaggregatedChassis() bool {
    val, found := GetPlatformEnvConfig("disaggregated_chassis")
    if !found {
        return found
    }
    return IsExpectedValue(val, "1")
}

func GetExpectedRunningContainers(featureTable map[string]interface) (map[string]struct{}, map[string]string) {
	expectedRunningContainers := make(map[string]struct{})
	containerFeatureDict := make(map[string]string)

	runAllInstanceList := map[string]struct{}{
		"database": {},
		"bgp":      {},
	}

	containerList := []string{}
	for containerName := range featureTable {
		if containerName == "frr_bmp" {
			continue
		}
		// slim image does not have telemetry container and corresponding docker image
		if containerName == "telemetry" {
			if !common.CheckDockerImageExist("docker-sonic-telemetry") {
				if !common.CheckDockerImageExist("docker-sonic-gnmi") {
					log.Errorf("Ignoring telemetry container check on image which has no corresponding docker image")
				} else {
					containerList = append(containerList, "gnmi")
				}
				continue
			}
		}
		containerList = append(containerList, containerName)
	}

	for _, containerName := range containerList {
		featureEntry := featureTable[containerName].(map[string]interface)
		state := featureEntry["state"].(string)
		if state != "disabled" && state != "always_disabled" {
			if common.IsMultiAsic() {
                log.Errorf("Currently multi ASIC not supported.")
			} else {
				expectedRunningContainers[containerName] = struct{}{}
				containerFeatureDict[containerName] = containerName
			}
		}
	}

	if IsSupervisor() || IsDisaggregatedChassis() {
		expectedRunningContainers["database-chassis"] = struct{}{}
		containerFeatureDict["database-chassis"] = "database"
	}

	return expectedRunningContainers, containerFeatureDict
}

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

func CheckDockerImageExist(imageName string) bool {
    allImagesData := GetDockerInfo()
    return strings.Contains(allImagesData, imageName)
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

func GetDockerRunningContainers() map[string]struct{} {
	cmdOutput, err := GetDataFromHostCommand(`bash -o pipefail -c 'docker ps --format "{{.Names}}",`)

	if err != nil {
		return []string{} 
	}

	runningContainerSlice := strings.Splice(cmdOutput,",")
    runningContainer := map[string]struct{}

    for _, containerName := range runningContainerSlice {
        runningContainer[containerName] = struct{}{}
    }
    return runningContainer
}

func GetContainerFolder(containerName string) string {
    cmd = DockerInspectCmd + containerName + DockerInspectDirCmd
    output, err := GetDataFromHostCommand(cmd)

    if err != nil {
        return ""
    }
    return strings.TrimSpace(output)
}

func GetContainerCriticalProcesses(runningContainer []string) (map[string]interface, map[string]struct{}) {
    criticalProcesses := make(map[string]interface)
    badProcesses := make(map[string]struct{}

    for _, container := range runningContainer {
        containerFolder = GetContainerFolder(container)
        if containerFolder != "" {
            criticalProcessesFile := filepath.Join(containerFolder, "etc/supervisor/critical_processes")
            if !FileExists(criticalProcessesFile) {
                criticalProcesses[container] = []string{}
                continue
            }

            data, err := GetDataFromFile(criticalProcessesFile)
            if err != nil {
                criticalProcesses[container] = []string{}
                continue
            }

            processList := []string{}

            re := regexp.MustCompile(`^\s*(?:(.+):(.*))*\s*$`)
            content := string(data)
	        lines := strings.Split(content, "\n")
            for _, line := range lines {
                match := re.FindStringSubmatch(line)
                if len(match) > 1 && match[1] != "" {
			        identifierKey := strings.TrimSpace(match[2])
			        identifierValue := strings.TrimSpace(match[3]) //

			        if identifierKey == "program" && identifierValue != "" {
                        processList.append(processList, identifierValue)
                    }
                } else {
                   badProcesses[container] = struct{}{} 
                }
            }
        }
    }
    return criticalProcesses, badProcesses
}
