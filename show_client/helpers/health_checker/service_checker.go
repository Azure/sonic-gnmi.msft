package health_checker

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
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

	// Command to read critical processes file from inside a container.
	// No shell wrapper (sh -c) or redirection (2>/dev/null) — those
	// get mangled by shlex.Split + exec.Command + nsenter + docker exec.
	// If the file doesn't exist, cat exits non-zero and the error
	// message (from CombinedOutput) is harmlessly ignored by
	// parseCriticalProcesses since it won't match "program:xxx".
	CriticalProcessesCatCmd = `docker exec %s cat /etc/supervisor/critical_processes`

	// Command to get supervisorctl status inside a container.
	// Uses docker exec for the same reason as CriticalProcessesCatCmd.
	SupervisorctlStatusCmd = `docker exec %s bash -c "supervisorctl status"`

	// Cache file to save container_critical_processes.
	CriticalProcessCache = "/tmp/critical_process_cache"

	// Expect status for all system service categories.
	// Monit 5.34.3+ (Debian 13) uses 'OK' for all service types.
	ExpectedStatus = "OK"
)

// ContainerK8SWhitelist is the whitelist of containers managed by KubeSonic
// to bypass health checking entirely. These containers will be excluded from
// both expected and running container sets.
var ContainerK8SWhitelist = map[string]struct{}{
	"telemetry": {},
	"acms":      {},
	"restapi":   {},
}

// ServiceChecker checks critical system service status via monit service.
type ServiceChecker struct {
	HealthChecker
	containerCriticalProcesses map[string][]string
	badContainers              map[string]struct{}
	containerFeatureDict       map[string]string
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

func (sc *ServiceChecker) Str() string {
	// Str returns the checker name for error messages.
	return reflect.TypeOf(sc).Elem().Name()
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
		if status != ExpectedStatus {
			sc.SetObjectNotOK(serviceType, serviceName,
				fmt.Sprintf("%s status is %s, expected %s", serviceName, status, ExpectedStatus))
		} else {
			sc.SetObjectOK(serviceType, serviceName)
		}
	}
}

func (sc *ServiceChecker) checkServices(config *Config) {
	/* checkServices checks status of critical services and critical processes.
	:param config: Health checker configuration.*/
	queries := [][]string{{common.ConfigDb, "FEATURE"}}
	featureData, err := common.GetMapFromQueries(queries)
	if err != nil {
		return
	}

	expectedRunningContainers, containerFeatureDict, err := sc.getExpectedRunningContainers(featureData)
	if err != nil {
		log.Errorf("Failed to get expected running containers: %v", err)
		return
	}
	sc.containerFeatureDict = containerFeatureDict

	sc.loadCriticalProcessCache()
	currentRunningContainers, err := sc.getCurrentRunningContainers()
	if err != nil {
		log.Errorf("%v", err)
		return
	}

	log.V(1).Infof("checkServices: expectedRunning=%d, containerFeatureDict=%d, containerCriticalProcesses=%d",
		len(expectedRunningContainers), len(containerFeatureDict), len(sc.containerCriticalProcesses))

	// Remove newly disabled containers from critical process tracking
	for containerName := range sc.containerCriticalProcesses {
		if _, expected := expectedRunningContainers[containerName]; !expected {
			log.V(1).Infof("Removing non-expected container from critical process tracking: %s", containerName)
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

	log.V(1).Infof("checkServices: after pruning, containerCriticalProcesses has %d entries", len(sc.containerCriticalProcesses))
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

func (sc *ServiceChecker) getExpectedRunningContainers(featureTable map[string]interface{}) (map[string]struct{}, map[string]string, error) {
	/* getExpectedRunningContainers gets a set of containers that are expected to be running on SONiC.
	:param featureTable: FEATURE table in CONFIG_DB.
	:return: expected_running_containers: A set of container names that are expected running.
	         container_feature_dict: A dictionary {<container_name>:<feature_name>}.*/
	expectedRunningContainers := make(map[string]struct{})
	containerFeatureDict := make(map[string]string)

	// Fetch docker image list once to avoid repeated heavy calls
	allImagesData := common.GetDockerInfo()
	checkDockerImage := func(imageName string) bool {
		return strings.Contains(allImagesData, imageName)
	}

	containerList := []string{}
	for containerName := range featureTable {
		// Skip containers in the KubeSonic whitelist
		if _, ok := ContainerK8SWhitelist[containerName]; ok {
			log.V(1).Infof("Skipping whitelisted kubesonic managed container '%s' from expected running check", containerName)
			continue
		}
		// skip frr_bmp since it's not a container, just a bmp option used by bgpd
		if containerName == "frr_bmp" {
			continue
		}
		// slim image does not have telemetry container and corresponding docker image
		if containerName == "telemetry" {
			if !checkDockerImage("docker-sonic-telemetry") {
				// If telemetry container image is not present, check gnmi container image
				// If gnmi container image is not present, ignore telemetry container check
				// if gnmi container image is present, check gnmi container instead of telemetry
				if !checkDockerImage("docker-sonic-gnmi") {
					log.Warningf("Ignoring telemetry container check on image which has no corresponding docker image")
				} else {
					containerList = append(containerList, "gnmi")
				}
				continue
			}
		}
		// Some platforms may not include the OTEL container; skip when image absent
		if containerName == "otel" {
			if !checkDockerImage("docker-sonic-otel") {
				log.V(1).Infof("Ignoring otel container check on image which has no corresponding docker image")
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
				return nil, nil, fmt.Errorf("multi-ASIC is not supported")
			}
			expectedRunningContainers[containerName] = struct{}{}
			containerFeatureDict[containerName] = containerName
		}
	}

	if common.IsSupervisor() || common.IsDisaggregatedChassis() {
		expectedRunningContainers["database-chassis"] = struct{}{}
		containerFeatureDict["database-chassis"] = "database"
	}

	return expectedRunningContainers, containerFeatureDict, nil
}

func CheckDockerImageExist(imageName string) bool {
	/* CheckDockerImageExist checks if docker image exists.
	:return: True if the image exists, otherwise False.*/
	allImagesData := common.GetDockerInfo()
	return strings.Contains(allImagesData, imageName)
}

func (sc *ServiceChecker) getCurrentRunningContainers() (map[string]struct{}, error) {
	/* getCurrentRunningContainers gets current running containers, if the running container is not
	in containerCriticalProcesses, tries to get the critical process list.
	:return: running_containers: A set of running container names.*/
	runningContainers := make(map[string]struct{})

	cmdOutput, err := common.GetDataFromHostCommand(
		`bash -o pipefail -c 'docker ps --format "{{.Names}}\t{{.Label \"io.kubernetes.pod.namespace\"}}"'`)
	if err != nil {
		return runningContainers, fmt.Errorf("Failed to retrieve the running container list. Error: '%v'", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(cmdOutput), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		name := strings.TrimSpace(parts[0])
		if name == "" {
			continue
		}
		// Check if this is a Kubernetes-managed container
		if len(parts) > 1 && strings.TrimSpace(parts[1]) == "sonic" {
			continue
		}
		// Skip kubesonic managed containers in the whitelist
		if _, ok := ContainerK8SWhitelist[name]; ok {
			log.V(1).Infof("Skipping whitelisted container from running set: %s", name)
			continue
		}
		runningContainers[name] = struct{}{}
		if _, exists := sc.containerCriticalProcesses[name]; !exists {
			sc.fillCriticalProcessByContainer(name)
		}
	}
	log.V(1).Infof("getCurrentRunningContainers found %d containers, containerCriticalProcesses has %d entries", len(runningContainers), len(sc.containerCriticalProcesses))
	return runningContainers, nil
}

func (sc *ServiceChecker) fillCriticalProcessByContainer(container string) {
	/* fillCriticalProcessByContainer gets critical process for a given container.
	Uses docker exec  to read critical processes file from inside a container
	An alternative is to run docker inspect on the host to obtain the
	overlay filesystem path, then cat that path. However:
	- This code runs inside the gnmi container where host overlay
	paths are not accessible.
	- docker inspect + cat requires 2 nsenter calls per container
	vs 1 with docker exec.
	docker exec avoids all two issues: it is atomic, needs only a single call.
	:param container: Container name.*/
	cmd := fmt.Sprintf(CriticalProcessesCatCmd, container)
	output, err := common.GetDataFromHostCommand(cmd)
	log.V(1).Infof("CriticalProcessesCatCmd for %s: err=%v, output=%q", container, err, output)
	if strings.TrimSpace(output) == "" {
		// Critical process file does not exist or container is not accessible.
		if err != nil {
			log.V(1).Infof("Failed to get critical process file for %s: %v", container, err)
		}
		sc.containerCriticalProcesses[container] = []string{}
		return
	}

	// Parse critical process list from file content
	criticalProcessList := sc.parseCriticalProcesses(container, output)
	sc.containerCriticalProcesses[container] = criticalProcessList
	log.V(1).Infof("Stored %d critical processes for %s: %v", len(criticalProcessList), container, criticalProcessList)
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

	for containerName, criticalProcesses := range cached {
		sc.containerCriticalProcesses[containerName] = criticalProcesses
	}
}

func (sc *ServiceChecker) parseCriticalProcesses(container string, data string) []string {
	/* parseCriticalProcesses parses critical process names from critical_processes file content.
	:param container: Container name.
	:param data: Content of the critical_processes file.
	:return: critical_process_list: A list of critical process names.*/
	criticalProcessList := []string{}
	re := regexp.MustCompile(`^\s*(?:(.+):(.*))*\s*$`)
	lines := strings.Split(data, "\n")
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
		log.V(1).Infof("checkProcessExistence: container %s not in containerFeatureDict, skipping", containerName)
		return
	}

	featureEntry, ok := featureData[featureName].(map[string]interface{})
	if !ok {
		log.V(1).Infof("checkProcessExistence: feature %s not in featureData, skipping container %s", featureName, containerName)
		return
	}

	// We look into the 'FEATURE' table to verify whether the container is disabled or not.
	// If the container is disabled, we exit.
	state, _ := featureEntry["state"].(string)
	if state == "disabled" || state == "always_disabled" {
		return
	}

	// We are using supervisorctl status to check the critical process status. We cannot leverage psutil here because
	// it not always possible to get process cmdline in supervisor.conf. E.g, cmdline of orchagent is "/usr/bin/orchagent",
	// however, in supervisor.conf it is "/usr/bin/orchagent.sh"
	cmd := fmt.Sprintf(SupervisorctlStatusCmd, containerName)
	output, err := common.GetDataFromHostCommand(cmd)
	log.V(1).Infof("SupervisorctlStatusCmd for %s: err=%v, output=%q", containerName, err, output)
	if err != nil && strings.TrimSpace(output) == "" {
		// Only treat as fatal if there is no output at all.
		// supervisorctl status exits 3 when any process is not RUNNING
		// but still produces valid output that we need to parse.
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
	log.V(1).Infof("checkProcessExistence for %s: %d status entries, criticalProcesses=%v", containerName, len(allProcessStatus), criticalProcesses)

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
