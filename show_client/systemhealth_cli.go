package show_client

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

/* SystemHealthSummary is the top-level JSON response for "show system-health summary". */
type SystemHealthSummary struct {
	StatusLed string               `json:"system_status_led"`
	Services  ServiceHealthStatus  `json:"services"`
	Hardware  HardwareHealthStatus `json:"hardware"`
}

/* ServiceHealthStatus captures the health of system services and filesystems. */
type ServiceHealthStatus struct {
	Status        string   `json:"status"`
	NotRunning    []string `json:"not_running,omitempty"`
	NotAccessible []string `json:"not_accessible,omitempty"`
}

/* HardwareHealthStatus captures the health of hardware components. */
type HardwareHealthStatus struct {
	Status  string   `json:"status"`
	Reasons []string `json:"reasons,omitempty"`
}

/* SystemHealthDetail is the top-level JSON response for "show system-health detail". */
type SystemHealthDetail struct {
	StatusLed   string               `json:"system_status_led"`
	Services    ServiceHealthStatus  `json:"services"`
	Hardware    HardwareHealthStatus `json:"hardware"`
	MonitorList []HealthListEntry    `json:"monitor_list"`
	IgnoreList  []HealthListEntry    `json:"ignore_list"`
}

/* HealthListEntry represents a single entry in the monitor or ignore list. */
type HealthListEntry struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Type   string `json:"type"`
}

/* SystemHealthMonitorList is the top-level JSON response for "show system-health monitor-list". */
type SystemHealthMonitorList struct {
	MonitorList []HealthListEntry `json:"monitor_list"`
}

/*
	SystemHealthScriptPath is the installed location of the Python health check script.
	The script is maintained in scripts/system_health.py and installed via Makefile.
	It is invoked via runpy.run_path with the temp output file path passed as sys.argv[1].
*/
var SystemHealthScriptPath = "/usr/sbin/system_health.py"

/*
	RunSystemHealthCheck creates a temp file for the JSON result, then uses
	runpy.run_path to execute the external Python script as __main__ with the
	temp file path passed via sys.argv[1]. The script writes its result to the
	temp file, which this function reads and returns.
*/
func RunSystemHealthCheck() (string, error) {
	tmpFile, err := os.CreateTemp("", "system_health_*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	pyCode := fmt.Sprintf(`
import sys, runpy
sys.argv = ["system_health.py", "%s"]
runpy.run_path("%s", run_name="__main__")
`, tmpPath, SystemHealthScriptPath)

	err = sdc.RunPyCode(pyCode)
	if err != nil {
		return "", fmt.Errorf("Python execution failed: %v", err)
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to read health check result: %v", err)
	}
	return string(data), nil
}

func getSystemHealthSummary(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	/*
		Handler for "show system-health summary".
		Uses cgo to run an embedded Python script, parses the JSON result,
		and formats it into the SystemHealthSummary response.
	*/
	output, err := RunSystemHealthCheck()
	if err != nil {
		log.Errorf("Failed to execute system health check: %v", err)
		return nil, fmt.Errorf("failed to retrieve system health data: %v", err)
	}

	output = strings.TrimSpace(output)

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(output), &raw); err != nil {
		log.Errorf("Failed to parse system health output: %v", err)
		return nil, fmt.Errorf("failed to parse system health output: %v", err)
	}

	if errMsg, ok := raw["error"]; ok {
		if errStr, ok := errMsg.(string); ok && errStr == "config_missing" {
			return nil, fmt.Errorf("System health configuration file not found, exit")
		}
		return nil, fmt.Errorf("system health check error: %v", errMsg)
	}

	led, _ := raw["led"].(string)
	stat, _ := raw["stat"].(map[string]interface{})

	summary := buildHealthSummary(stat, led)
	return json.Marshal(summary)
}

func buildHealthSummary(stat map[string]interface{}, led string) SystemHealthSummary {
	/*
		Translates the Python health check result into the SystemHealthSummary struct.
		- Services with non-OK status and "Accessible" in the message go to NotAccessible.
		- Other non-OK services go to NotRunning.
		- Non-OK hardware entries go to Reasons.
	*/
	var servicesList []string
	var fsList []string
	var deviceList []string

	for category, elementsRaw := range stat {
		elements, ok := elementsRaw.(map[string]interface{})
		if !ok {
			continue
		}
		for element, detailRaw := range elements {
			detail, ok := detailRaw.(map[string]interface{})
			if !ok {
				continue
			}
			status, _ := detail["status"].(string)
			if status != "OK" {
				if category == "Services" {
					msg, _ := detail["message"].(string)
					if strings.Contains(msg, "Accessible") {
						fsList = append(fsList, element)
					} else {
						servicesList = append(servicesList, element)
					}
				} else {
					msg, _ := detail["message"].(string)
					deviceList = append(deviceList, msg)
				}
			}
		}
	}

	services := ServiceHealthStatus{Status: "OK"}
	if len(servicesList) > 0 || len(fsList) > 0 {
		services.Status = "Not OK"
	}
	if len(servicesList) > 0 {
		services.NotRunning = servicesList
	}
	if len(fsList) > 0 {
		services.NotAccessible = fsList
	}

	hardware := HardwareHealthStatus{Status: "OK"}
	if len(deviceList) > 0 {
		hardware.Status = "Not OK"
		reversed := make([]string, len(deviceList))
		for i, v := range deviceList {
			reversed[len(deviceList)-1-i] = v
		}
		hardware.Reasons = reversed
	}

	return SystemHealthSummary{
		StatusLed: led,
		Services:  services,
		Hardware:  hardware,
	}
}

func getSystemHealthDetail(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	/*
		Handler for "show system-health detail".
		Reuses RunSystemHealthCheck (same Python script as summary) which also
		returns ignore_services and ignore_devices from the manager config.
	*/
	output, err := RunSystemHealthCheck()
	if err != nil {
		log.Errorf("Failed to execute system health detail check: %v", err)
		return nil, fmt.Errorf("failed to retrieve system health detail data: %v", err)
	}

	output = strings.TrimSpace(output)

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(output), &raw); err != nil {
		log.Errorf("Failed to parse system health detail output: %v", err)
		return nil, fmt.Errorf("failed to parse system health detail output: %v", err)
	}

	if errMsg, ok := raw["error"]; ok {
		if errStr, ok := errMsg.(string); ok && errStr == "config_missing" {
			return nil, fmt.Errorf("System health configuration file not found, exit")
		}
		return nil, fmt.Errorf("system health check error: %v", errMsg)
	}

	led, _ := raw["led"].(string)
	stat, _ := raw["stat"].(map[string]interface{})

	summary := buildHealthSummary(stat, led)
	monitorList := buildMonitorList(stat)

	ignoreServices, _ := raw["ignore_services"].([]interface{})
	ignoreDevices, _ := raw["ignore_devices"].([]interface{})
	ignoreList := buildIgnoreList(ignoreServices, ignoreDevices)

	detail := SystemHealthDetail{
		StatusLed:   summary.StatusLed,
		Services:    summary.Services,
		Hardware:    summary.Hardware,
		MonitorList: monitorList,
		IgnoreList:  ignoreList,
	}

	return json.Marshal(detail)
}

func getSystemHealthMonitorList(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	/*
		Handler for "show system-health monitor-list".
		Shows only the monitored services and devices name list.
		Matches Python's standalone monitor_list command.
	*/
	output, err := RunSystemHealthCheck()
	if err != nil {
		log.Errorf("Failed to execute system health monitor list check: %v", err)
		return nil, fmt.Errorf("failed to retrieve system health monitor list data: %v", err)
	}

	output = strings.TrimSpace(output)

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(output), &raw); err != nil {
		log.Errorf("Failed to parse system health monitor list output: %v", err)
		return nil, fmt.Errorf("failed to parse system health monitor list output: %v", err)
	}

	if errMsg, ok := raw["error"]; ok {
		if errStr, ok := errMsg.(string); ok && errStr == "config_missing" {
			return nil, fmt.Errorf("System health configuration file not found, exit")
		}
		return nil, fmt.Errorf("system health check error: %v", errMsg)
	}

	stat, _ := raw["stat"].(map[string]interface{})
	monitorList := buildMonitorList(stat)

	result := SystemHealthMonitorList{
		MonitorList: monitorList,
	}

	return json.Marshal(result)
}

func buildMonitorList(stat map[string]interface{}) []HealthListEntry {
	/*
		Builds the monitor list from the health check stat data.
		Iterates all categories and elements, collecting Name, Status, Type.
		Sorts by Status then Name for deterministic output.
		Matches Python's display_monitor_list which sorts by status.
	*/
	entries := make([]HealthListEntry, 0)
	for _, elementsRaw := range stat {
		elements, ok := elementsRaw.(map[string]interface{})
		if !ok {
			continue
		}
		for name, detailRaw := range elements {
			detail, ok := detailRaw.(map[string]interface{})
			if !ok {
				continue
			}
			status, _ := detail["status"].(string)
			typ, _ := detail["type"].(string)
			entries = append(entries, HealthListEntry{Name: name, Status: status, Type: typ})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Status != entries[j].Status {
			return entries[i].Status < entries[j].Status
		}
		return entries[i].Name < entries[j].Name
	})
	return entries
}

func buildIgnoreList(ignoreServices, ignoreDevices []interface{}) []HealthListEntry {
	/*
		Builds the ignore list from the manager's ignore_services and ignore_devices.
		Services get Type="Service", devices get Type="Device", both get Status="Ignored".
		Matches Python's display_ignore_list output.
	*/
	entries := make([]HealthListEntry, 0)
	for _, svc := range ignoreServices {
		if name, ok := svc.(string); ok {
			entries = append(entries, HealthListEntry{Name: name, Status: "Ignored", Type: "Service"})
		}
	}
	for _, dev := range ignoreDevices {
		if name, ok := dev.(string); ok {
			entries = append(entries, HealthListEntry{Name: name, Status: "Ignored", Type: "Device"})
		}
	}
	return entries
}

// ---------------------------------------------------------------------------
// sysready-status types and constants
// ---------------------------------------------------------------------------

const (
	sysreadyTable     = "SYSTEM_READY"
	sysreadyKey       = "SYSTEM_STATE"
	sysreadyStatusKey = "Status"
	sysreadyStatusUp  = "UP"

	serviceStatusTable  = "ALL_SERVICE_STATUS"
	serviceStatusField  = "service_status"
	appReadyStatusField = "app_ready_status"
	failReasonField     = "fail_reason"
)

/* SysreadyStatus is the JSON response for "show system-health sysready-status". */
type SysreadyStatus struct {
	SystemReady string              `json:"system_status"`
	ServiceList []ServiceReadyEntry `json:"service_list"`
}

/* ServiceReadyEntry is a service entry in the sysready-status table. */
type ServiceReadyEntry struct {
	ServiceName    string `json:"service_name"`
	ServiceStatus  string `json:"service_status"`
	AppReadyStatus string `json:"app_ready_status"`
	FailReason     string `json:"fail_reason"`
}

/*
	getSysreadyState reads the SYSTEM_READY|SYSTEM_STATE table from STATE_DB
	and returns the system ready status string ("System is ready" or
	"System is not ready - one or more services are not up").
*/
func getSysreadyState() (string, error) {
	queries := [][]string{
		{common.StateDb, sysreadyTable, sysreadyKey},
	}
	data, err := common.GetMapFromQueries(queries)
	if err != nil {
		return "", fmt.Errorf("failed to read system ready state: %v", err)
	}

	if len(data) == 0 {
		return "", fmt.Errorf("no system ready status data available - system-health service might be down")
	}

	statusMap, ok := data[sysreadyKey].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected system ready state format")
	}

	status := common.GetValueOrDefault(statusMap, sysreadyStatusKey, "")
	if status == sysreadyStatusUp {
		return "System is ready", nil
	}
	return "System is not ready - one or more services are not up", nil
}

/*
	getServiceStatusData reads ALL_SERVICE_STATUS from STATE_DB and returns
	a sorted list of service names and a map of service data keyed by name.
*/
func getServiceStatusData() ([]string, map[string]map[string]interface{}, error) {
	queries := [][]string{
		{common.StateDb, serviceStatusTable},
	}
	data, err := common.GetMapFromQueries(queries)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read service status data: %v", err)
	}

	services := make(map[string]map[string]interface{})
	var names []string

	for key, val := range data {
		if entry, ok := val.(map[string]interface{}); ok {
			services[key] = entry
			names = append(names, key)
		}
	}
	sort.Strings(names)
	return names, services, nil
}

func getSysreadyStatus(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	/*
		Handler for "show system-health sysready-status".
		Returns system ready status plus a service table with columns:
		Service-Name, Service-Status, App-Ready-Status, Down-Reason.
	*/
	sysState, err := getSysreadyState()
	if err != nil {
		log.Errorf("Failed to get system ready state: %v", err)
		return nil, fmt.Errorf("failed to get system ready state: %v", err)
	}

	names, services, err := getServiceStatusData()
	if err != nil {
		log.Errorf("Failed to get service status data: %v", err)
		return nil, fmt.Errorf("failed to get service status data: %v", err)
	}

	serviceList := make([]ServiceReadyEntry, 0, len(names))
	for _, name := range names {
		entry := services[name]
		serviceList = append(serviceList, ServiceReadyEntry{
			ServiceName:    name,
			ServiceStatus:  common.GetValueOrDefault(entry, serviceStatusField, ""),
			AppReadyStatus: common.GetValueOrDefault(entry, appReadyStatusField, ""),
			FailReason:     common.GetValueOrDefault(entry, failReasonField, ""),
		})
	}

	result := SysreadyStatus{
		SystemReady: sysState,
		ServiceList: serviceList,
	}
	return json.Marshal(result)
}
