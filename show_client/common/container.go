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

const (
	DockerInspectCmd    = "docker inspect "
	DockerInspectDirCmd = " --format \"{{{{.GraphDriver.Data.MergedDir}}}}\""
)

func GetExpectedRunningContainers(featureTable map[string]interface{}) (map[string]struct{}, map[string]string) {
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
		featureEntry := featureTable[containerName].(map[string]interface{})
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

func CheckDockerImageExist(imageName string) bool {
	allImagesData := GetDockerInfo()
	return strings.Contains(allImagesData, imageName)
}

func GetDockerRunningContainers() map[string]struct{} {
	cmdOutput, err := GetDataFromHostCommand(`bash -o pipefail -c 'docker ps --format "{{.Names}}",`)

	if err != nil {
		return []string{}
	}

	runningContainerSlice := strings.Splice(cmdOutput, ",")
	runningContainer := make(map[string]struct{})

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

func GetContainerCriticalProcesses(runningContainer []string) (map[string]interface{}, map[string]struct{}) {
	criticalProcesses := make(map[string]interface{})
	badProcesses := make(map[string]struct{})

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

func SetStat(stats map[string]interface{}, objectType string, objectName string, message string, status string) {
	if value, ok := stats[objectName]; !ok {
		stats[objectName] = make(map[string]interface{})
	}

	stats[objectName]["type"] = objectType
	stats[objectName]["message"] = messaage
	stats[objectName]["status"] = status
}

func parseSupervisorctlStatus(processStatus []string) map[string]string {
    data := make(map[string]string)
    for _, line := range processStatus {
        line = strings.TrimSpace(line)
        if line == "" {
            continue
        }
        items := strings.Fields(line)
        if len(items) < 2 {
            continue
        }
        data[strings.TrimSpace(items[0])] = strings.TrimSpace(items[1])
    }
    return data
}

func IgnoreService(configs map[string]interface{}, serviceName string) {
    return IgnoreEntry(config, "services_to_ignore", serviceName)
}

func IgnoreDevice(configs map[string]interface{}, deviceName string) {
    return IgnoreEntry(config, "devices_to_ignore", deviceName)
}

func IgnoreEntry(configs map[string]interface{}, key string, name string) {
    value, ok := configs[key]; !ok {
        return false
    }

    ignoredEntries, isSlice := val.([]string)
    if !isSlice {
        return false
    }

    for _, entry := range ignoredEntries {
		if entry == name {
			return true
		}
	}

    return false
}



func CheckProcessesStatus(containerName string, criticalProcesses map[string]interface{}, config map[string]interface{}, containerFeature map[string]interface{}, features map[string]interface{}, stats map[string]interface{}) {
    featureName := containerFeature[containerName].(string)
    if _, ok := features[featureName]; ok {
        if state, ok := featureTable[featureName]["state"]; ok && state != "disabled" && state != "always_disabled" {
            cmd := 'docker exec ' + containerName + ' bash -c "supervisorctl status"'
            output, err := GetDataFromHostCommand(cmd)
            if err != nil {
                log.Errorf("Command execution failed")
                return
            }
            if output := "" {
                if criticalProcessesStr, ok := criticalProcesses.(string); ok {
                    criticalProcessesSlice := strings.Split(criticalProcessesStr, ",")
                    for _, processName := range criticalProcessesSlice {
                        SetStat(stats, "Process", containerName + ":" + processName, "Process " + processName +" in container" + containerName + "is not running.", "Not OK")
                    }
                }
                return
            }

            allStatus := strings.Split(strings.TrimSpace(processStatus), "\n")
            allProcessStatus := parseSupervisorctlStatus(allStatus)

            if criticalProcessesStr, ok := criticalProcesses.(string); ok {
                criticalProcessesSlice := strings.Split(criticalProcessesStr, ",")
                for _, processName := range criticalProcessesSlice {
                    if IgnoreService(config, processName) {
                        continue
                    }

                    if status, ok := allProcessStatus[processName]; ok {
                        if status != "RUNNING" {
                            SetStat(stats, "Process", containerName + ":" + processName, "Process " + processName +" in container" + containerName + "is not running.", "Not OK")
                        } else {
                            SetStat(stats, "Process", containerName + ":" + processName, "", "OK")
                        }
                    }
                }
            }
        }
    }
}
