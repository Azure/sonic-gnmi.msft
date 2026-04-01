package helpers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
)

const (
	// Command to query the status of monit service.
	checkMonitServiceCmd = "systemctl is-active monit.service"
	// Command to get summary of critical system service.
	checkMonitCmd = "monit summary -B"
	// Minimum number of lines expected from monit summary output.
	minCheckCmdLines = 3
)

const (
	// Command to get merged directory of a container.
	GetContainerFolderCmd = `docker inspect %s --format "{{.GraphDriver.Data.MergedDir}}"`

	// Path to critical processes file inside a container.
	CriticalProcessesPath = "etc/supervisor/critical_processes"

	// Command to get supervisorctl status inside a container.
	SupervisorctlStatusCmd = `docker exec %s bash -c "supervisorctl status"`

	// Cache file to save container_critical_processes.
	CriticalProcessCache = "/tmp/critical_process_cache"
)

// ExpectStatusDict defines the expected status for different system service categories.
var ExpectStatusDict = map[string]string{
	"System":     "Running",
	"Process":    "Running",
	"Filesystem": "Accessible",
	"Program":    "Status ok",
}

// ServiceChecker checks critical system service status via monit service.
type ServiceChecker struct {
	HealthChecker
	containerCriticalProcesses map[string][]string
	badContainers              map[string]struct{}
	containerFeatureDict       map[string]string
	needSaveCache              bool
}

func NewServiceChecker() *ServiceChecker {
	// NewServiceChecker creates a new ServiceChecker.
	return &ServiceChecker{
		HealthChecker:              NewHealthChecker(),
		containerCriticalProcesses: make(map[string][]string),
		badContainers:              make(map[string]struct{}),
		containerFeatureDict:       make(map[string]string),
	}
}

func (sc *ServiceChecker) GetCategory() string {
	// GetCategory returns the category for service checks.
	return "Services"
}

func (sc *ServiceChecker) String() string {
	//String returns the checker name for error messages.
	return "ServiceChecker"
}

func (sc *ServiceChecker) Check(config *Config) {
	/* Check checks critical system service status.
	:param config: Health checker configuration.*/
	sc.Reset()
	sc.checkByMonit(config)
	sc.checkServices(config)
}

func (sc *ServiceChecker) checkByMonit(config *Config) {
	/* checkByMonit gets and analyzes the output of CHECK_CMD, collects status for
	file system or customize checker if any.
	:param config: Health checker configuration.
	:return:*/
	output, err := common.GetDataFromHostCommand(checkMonitServiceCmd)
	if err != nil {
		log.Errorf("Unable to execute monit service check command: %v", err)
		return
	}

	if strings.TrimSpace(output) != "active" {
		sc.SetObjectNotOK("Service", "monit", "monit service is not running")
		return
	}

	output, err = common.GetDataFromHostCommand(checkMonitCmd)
	if err != nil {
		log.Errorf("Unable to execute monit summary command: %v", err)
		return
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) < minCheckCmdLines {
		sc.SetObjectNotOK("Service", "monit", "monit service is not ready")
		return
	}

	statusBegin := strings.Index(lines[1], "Status")
	typeBegin := strings.Index(lines[1], "Type")
	if statusBegin < 0 || typeBegin < 0 {
		sc.SetObjectNotOK("Service", "monit",
			`output of "monit summary -B" is invalid or incompatible`)
		return
	}

	for _, line := range lines[2:] {
		if len(line) < typeBegin {
			continue
		}
		serviceName := strings.TrimSpace(line[:statusBegin])
		if _, ok := config.IgnoreServices[serviceName]; ok {
			continue
		}
		status := strings.TrimSpace(line[statusBegin:typeBegin])
		serviceType := strings.TrimSpace(line[typeBegin:])
		expectStatus, ok := ExpectStatusDict[serviceType]
		if !ok {
			continue
		}
		if expectStatus != status {
			sc.SetObjectNotOK(serviceType, serviceName,
				fmt.Sprintf("%s is not %s", serviceName, expectStatus))
		} else {
			sc.SetObjectOK(serviceType, serviceName)
		}
	}
}

func (sc *ServiceChecker) checkServices(config *Config) {
	/* checkServices checks status of critical services and critical processes.
	:param config: Health checker configuration.*/
	queries := [][]string{{"CONFIG_DB", "FEATURE"}}
	featureData, err := common.GetMapFromQueries(queries)
	if err != nil {
		return
	}

	expectedRunningContainers, err := sc.getExpectedRunningContainers(featureData)
	if err != nil {
		log.Errorf("Failed to get expected running containers: %v", err)
		return
	}

	sc.loadCriticalProcessCache()
	currentRunningContainers := sc.getDockerRunningContainers()

	// Remove newly disabled containers from critical process tracking
	for containerName := range sc.containerCriticalProcesses {
		if _, expected := expectedRunningContainers[containerName]; !expected {
			delete(sc.containerCriticalProcesses, containerName)
		}
	}

	// Check for containers that should be running but aren't
	for expectedContainer := range expectedRunningContainers {
		if _, exists := currentRunningContainers[expectedContainer]; !exists {
			sc.SetObjectNotOK("Service", expectedContainer,
				fmt.Sprintf("Container '%s' is not running", expectedContainer))
		}
	}

	if len(sc.containerCriticalProcesses) == 0 {
		sc.SetObjectNotOK("Service", "system", "no critical process found")
		return
	}

	// Check critical processes in each container
	for container, criticalProcesses := range sc.containerCriticalProcesses {
		sc.checkProcessExistence(container, criticalProcesses, config.IgnoreServices, featureData)
	}

	// Report containers with bad critical_processes files
	for badContainer := range sc.badContainers {
		sc.SetObjectNotOK("Service", badContainer,
			"Syntax of critical_processes file is incorrect")
	}
}

func (sc *ServiceChecker) getExpectedRunningContainers(featureTable map[string]interface{}) (map[string]struct{}, error) {
	/* getExpectedRunningContainers gets a set of containers that are expected to be running on SONiC.
	:param featureTable: FEATURE table in CONFIG_DB.
	:return: expected_running_containers: A set of container names that are expected running.
	         container_feature_dict: Populated via sc.containerFeatureDict.*/
	expectedRunningContainers := make(map[string]struct{})
	sc.containerFeatureDict = make(map[string]string)

	containerList := []string{}
	for containerName := range featureTable {
		// slim image does not have telemetry container and corresponding docker image
		if containerName == "telemetry" {
			if !CheckDockerImageExist("docker-sonic-telemetry") {
				// If telemetry container image is not present, check gnmi container image
				// If gnmi container image is not present, ignore telemetry container check
				// if gnmi container image is present, check gnmi container instead of telemetry
				if !CheckDockerImageExist("docker-sonic-gnmi") {
					log.Warningf("Ignoring telemetry container check on image which has no corresponding docker image")
				} else {
					containerList = append(containerList, "gnmi")
				}
				continue
			}
		}
		containerList = append(containerList, containerName)
	}

	for _, containerName := range containerList {
		featureEntry, ok := featureTable[containerName].(map[string]interface{})
		if !ok {
			continue
		}
		state, _ := featureEntry["state"].(string)
		if state != "disabled" && state != "always_disabled" {
			if common.IsMultiAsic() {
				return nil, fmt.Errorf("multi-ASIC is not supported")
			}
			expectedRunningContainers[containerName] = struct{}{}
			sc.containerFeatureDict[containerName] = containerName
		}
	}

	if common.IsSupervisor() {
		expectedRunningContainers["database-chassis"] = struct{}{}
		sc.containerFeatureDict["database-chassis"] = "database"
	}

	return expectedRunningContainers, nil
}

func CheckDockerImageExist(imageName string) bool {
	/* CheckDockerImageExist checks if docker image exists.
	:return: True if the image exists, otherwise False.*/
	allImagesData := common.GetDockerInfo()
	return strings.Contains(allImagesData, imageName)
}

func (sc *ServiceChecker) getDockerRunningContainers() map[string]struct{} {
	/* getDockerRunningContainers gets current running containers, if the running container is not
	in containerCriticalProcesses, tries to get the critical process list.
	:return: running_containers: A set of running container names.*/
	cmdOutput, err := common.GetDataFromHostCommand(`bash -o pipefail -c 'docker ps --format "{{.Names}}"'`)
	if err != nil {
		log.Errorf("Failed to retrieve the running container list. Error: '%v'", err)
		return nil
	}

	runningContainers := make(map[string]struct{})
	for _, name := range strings.Split(strings.TrimSpace(cmdOutput), "\n") {
		name = strings.TrimSpace(name)
		if name != "" {
			runningContainers[name] = struct{}{}
			if _, exists := sc.containerCriticalProcesses[name]; !exists {
				sc.fillCriticalProcessByContainer(name)
			}
		}
	}
	return runningContainers
}

func GetContainerFolder(containerName string) string {
	/* GetContainerFolder returns the merged directory of a container. */
	cmd := fmt.Sprintf(GetContainerFolderCmd, containerName)
	output, err := common.GetDataFromHostCommand(cmd)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(output)
}

func (sc *ServiceChecker) fillCriticalProcessByContainer(container string) {
	/* fillCriticalProcessByContainer gets critical process for a given container.
	:param container: Container name.*/
	// Get container volume folder
	containerFolder := GetContainerFolder(container)
	if containerFolder == "" {
		log.Warningf("Could not find MergedDir of container %s, was container stopped?", container)
		return
	}

	if !common.DirExists(containerFolder) {
		log.Warningf("MergedDir %s of container %s not found in filesystem, was container stopped?", containerFolder, container)
		return
	}

	// Get critical_processes file path
	criticalProcessesFile := filepath.Join(containerFolder, CriticalProcessesPath)
	if !common.FileExists(criticalProcessesFile) {
		// Critical process file does not exist, the container has no critical processes.
		log.V(1).Infof("Failed to get critical process file for %s, %s does not exist", container, criticalProcessesFile)
		sc.containerCriticalProcesses[container] = []string{}
		sc.needSaveCache = true
		return
	}

	// Get critical process list from critical_processes
	criticalProcessList := sc.getCriticalProcessListFromFile(container, criticalProcessesFile)
	sc.containerCriticalProcesses[container] = criticalProcessList
	sc.needSaveCache = true
}

func (sc *ServiceChecker) loadCriticalProcessCache() {
	/* loadCriticalProcessCache loads containerCriticalProcesses from a cache file.
	Note: Go uses JSON deserialization instead of Python's pickle.*/
	if !common.FileExists(CriticalProcessCache) {
		// cache file does not exist
		return
	}

	data, err := os.ReadFile(CriticalProcessCache)
	if err != nil {
		log.Errorf("Failed to read critical process cache: %v", err)
		return
	}

	var cached map[string][]string
	if err := json.Unmarshal(data, &cached); err != nil {
		log.Errorf("Failed to unmarshal critical process cache: %v", err)
		return
	}

	for k, v := range cached {
		sc.containerCriticalProcesses[k] = v
	}
}

func (sc *ServiceChecker) getCriticalProcessListFromFile(container string, criticalProcessesFile string) []string {
	/* getCriticalProcessListFromFile reads critical process name list from critical processes file.
	:param container: Container name.
	:param criticalProcessesFile: Critical processes file path.
	:return: critical_process_list: A list of critical process names.*/
	data, err := common.GetDataFromFile(criticalProcessesFile)
	if err != nil {
		return []string{}
	}

	criticalProcessList := []string{}
	re := regexp.MustCompile(`^\s*(?:(.+):(.*))*\s*$`)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		match := re.FindStringSubmatch(line)
		if match == nil {
			if strings.TrimSpace(line) != "" {
				if _, alreadyBad := sc.badContainers[container]; !alreadyBad {
					sc.badContainers[container] = struct{}{}
					log.Errorf("Invalid syntax in critical_processes file of %s", container)
				}
			}
			continue
		}
		if len(match) > 2 && match[1] != "" {
			identifierKey := strings.TrimSpace(match[1])
			identifierValue := strings.TrimSpace(match[2])
			if identifierKey == "program" && identifierValue != "" {
				criticalProcessList = append(criticalProcessList, identifierValue)
			}
		}
	}
	return criticalProcessList
}

func ParseSupervisorctlStatus(processStatus []string) map[string]string {
	/* ParseSupervisorctlStatus parses supervisorctl status output into a process→status map.
	Expected input:
		arp_update                       RUNNING   pid 67, uptime 1:03:56
		buffermgrd                       RUNNING   pid 81, uptime 1:03:56
	:param processStatus: List of process status.
	:return: A map of process name to status string.*/
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

func (sc *ServiceChecker) checkProcessExistence(containerName string, criticalProcesses []string, ignoreServices map[string]struct{}, featureData map[string]interface{}) {
	/* checkProcessExistence checks whether the process in the specified container is running or not.
	:param containerName: Container name.
	:param criticalProcesses: Critical processes.
	:param ignoreServices: Services to ignore from config.
	:param featureData: Feature table.*/
	featureName, ok := sc.containerFeatureDict[containerName]
	if !ok {
		return
	}

	featureEntry, ok := featureData[featureName].(map[string]interface{})
	if !ok {
		return
	}

	state, _ := featureEntry["state"].(string)
	if state == "disabled" || state == "always_disabled" {
		return
	}

	cmd := fmt.Sprintf(SupervisorctlStatusCmd, containerName)
	output, err := common.GetDataFromHostCommand(cmd)
	if err != nil {
		log.Errorf("Command execution failed for container %s: %v", containerName, err)
		return
	}

	if strings.TrimSpace(output) == "" {
		for _, processName := range criticalProcesses {
			sc.SetObjectNotOK("Process", containerName+":"+processName,
				fmt.Sprintf("Process '%s' in container '%s' is not running", processName, containerName))
		}
		return
	}

	allStatus := strings.Split(strings.TrimSpace(output), "\n")
	allProcessStatus := ParseSupervisorctlStatus(allStatus)

	for _, processName := range criticalProcesses {
		if _, ignored := ignoreServices[processName]; ignored {
			continue
		}

		if status, ok := allProcessStatus[processName]; ok {
			if status != "RUNNING" {
				sc.SetObjectNotOK("Process", containerName+":"+processName,
					fmt.Sprintf("Process '%s' in container '%s' is not running", processName, containerName))
			} else {
				sc.SetObjectOK("Process", containerName+":"+processName)
			}
		}
	}
}
