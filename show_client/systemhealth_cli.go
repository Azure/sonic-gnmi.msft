package show_client

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	helpers "github.com/sonic-net/sonic-gnmi/show_client/helpers/health_checker"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// SystemHealthSummary represents the output structure for show system-health summary.
type SystemHealthSummary struct {
	Summary   string               `json:"summary"`
	StatusLed string               `json:"system_status_led"`
	Services  ServiceHealthStatus  `json:"services"`
	Hardware  HardwareHealthStatus `json:"hardware"`
}

// ServiceHealthStatus represents the services portion of the health summary.
type ServiceHealthStatus struct {
	Status        string   `json:"status"`
	NotRunning    []string `json:"not_running,omitempty"`
	NotAccessible []string `json:"not_accessible,omitempty"`
}

// HardwareHealthStatus represents the hardware portion of the health summary.
type HardwareHealthStatus struct {
	Status  string   `json:"status"`
	Reasons []string `json:"reasons,omitempty"`
}

// SystemHealthDetail is the top-level JSON response for "show system-health detail".
type SystemHealthDetail struct {
	Summary     string               `json:"summary"`
	StatusLed   string               `json:"system_status_led"`
	Services    ServiceHealthStatus  `json:"services"`
	Hardware    HardwareHealthStatus `json:"hardware"`
	MonitorList []HealthListEntry    `json:"monitor_list"`
	IgnoreList  []HealthListEntry    `json:"ignore_list"`
}

// HealthListEntry represents a single entry in the monitor or ignore list.
type HealthListEntry struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Type   string `json:"type"`
}

// SystemHealthMonitorList is the top-level JSON response for "show system-health monitor-list".
type SystemHealthMonitorList struct {
	MonitorList []HealthListEntry `json:"monitor_list"`
}

func getSystemHealthSummary(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	/* getSystemHealthSummary implements "show system-health summary".
	Shows system-health summary information.
	*/
	manager, stat, err := getSystemHealthStatus()
	if err != nil {
		return nil, err
	}

	led := getStatusLed()

	summary := displaySystemHealthSummary(stat, led)
	return json.Marshal(summary)
}

// getStatusLed retrieves the current system status LED color by calling
// the platform-specific sonic_platform.chassis module via nsenter.
func getStatusLed() string {
	pyCmd := `python3 -c "from sonic_platform.chassis import Chassis; c = Chassis(); c.initizalize_system_led(); print(c.get_status_led())"`

	output, err := common.GetDataFromHostCommand(pyCmd)
	if err != nil {
		log.Errorf("Failed to get system LED status: %v", err)
		return ""
	}
	return strings.TrimSpace(output)
}

func getSystemHealthStatus() (*helpers.HealthCheckerManager, map[string]interface{}, error) {
	/* getSystemHealthStatus creates a HealthCheckerManager, verifies config exists,
	and performs the system health check.
	:return: manager, stat (check results), error.
	*/
	manager := helpers.NewHealthCheckerManager()

	if !manager.Config.ConfigFileExists() {
		return nil, nil, fmt.Errorf("System health configuration file not found")
	}

	// Run health checks — LED is set internally via nsenter into host namespace
	// calling: sonic_platform.chassis.Chassis().set_status_led(color)
	stat := manager.Check()

	return manager, stat, nil
}

func displaySystemHealthSummary(stat map[string]interface{}, led string) SystemHealthSummary {
	/* displaySystemHealthSummary builds the system health summary output.
	Categorizes failures into services_list (not running), fs_list (not accessible),
	and device_list (hardware issues).
	:param stat: Check results from HealthCheckerManager.Check().
	:param led: System status LED color string from chassis.get_status_led().
	:return: A SystemHealthSummary containing summary, services, hardware status, and LED color.
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
			status, _ := detail[helpers.INFO_FIELD_OBJECT_STATUS].(string)
			if status != helpers.StatusOK {
				if category == "Services" {
					msg, _ := detail[helpers.INFO_FIELD_OBJECT_MSG].(string)
					if strings.Contains(msg, "Accessible") {
						fsList = append(fsList, element)
					} else {
						servicesList = append(servicesList, element)
					}
				} else {
					msg, _ := detail[helpers.INFO_FIELD_OBJECT_MSG].(string)
					deviceList = append(deviceList, msg)
				}
			}
		}
	}

	// Build services status
	services := ServiceHealthStatus{
		Status: helpers.StatusOK,
	}
	if len(servicesList) > 0 || len(fsList) > 0 {
		services.Status = "Not OK"
	}
	if len(servicesList) > 0 {
		services.NotRunning = servicesList
	}
	if len(fsList) > 0 {
		services.NotAccessible = fsList
	}

	// Build hardware status
	hardware := HardwareHealthStatus{
		Status: helpers.StatusOK,
	}
	if len(deviceList) > 0 {
		hardware.Status = "Not OK"
		reversed := make([]string, len(deviceList))
		for i, v := range deviceList {
			reversed[len(deviceList)-1-i] = v
		}
		hardware.Reasons = reversed
	}

	return SystemHealthSummary{
		Summary:   helpers.Summary,
		StatusLed: led,
		Services:  services,
		Hardware:  hardware,
	}
}

func getSystemHealthDetail(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	/* getSystemHealthDetail implements "show system-health detail".
	Shows system-health summary, monitor list, and ignore list.
	*/
	manager, stat, err := getSystemHealthStatus()
	if err != nil {
		return nil, err
	}

	led := getStatusLed()

	summary := displaySystemHealthSummary(stat, led)
	monitorList := displayMonitorList(stat)
	ignoreList := displayIgnoreList(manager)

	detail := SystemHealthDetail{
		Summary:     summary.Summary,
		StatusLed:   summary.StatusLed,
		Services:    summary.Services,
		Hardware:    summary.Hardware,
		MonitorList: monitorList,
		IgnoreList:  ignoreList,
	}

	return json.Marshal(detail)
}

func getSystemHealthMonitorList(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	/* getSystemHealthMonitorList implements "show system-health monitor-list".
	Shows system-health monitored services and devices name list.
	*/
	_, stat, err := getSystemHealthStatus()
	if err != nil {
		return nil, err
	}

	monitorList := displayMonitorList(stat)

	result := SystemHealthMonitorList{
		MonitorList: monitorList,
	}

	return json.Marshal(result)
}

func displayMonitorList(stat map[string]interface{}) []HealthListEntry {
	/* displayMonitorList builds the monitor list from the health check stat data.
	Iterates all categories and elements, collecting Name, Status, Type.
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
	return entries
}

func displayIgnoreList(manager *helpers.HealthCheckerManager) []HealthListEntry {
	/* displayIgnoreList builds the ignore list from the manager's config.
	Services get Type="Service", devices get Type="Device", both get Status="Ignored".
	*/
	entries := make([]HealthListEntry, 0)
	for svc := range manager.Config.IgnoreServices {
		entries = append(entries, HealthListEntry{Name: svc, Status: "Ignored", Type: "Service"})
	}
	for dev := range manager.Config.IgnoreDevices {
		entries = append(entries, HealthListEntry{Name: dev, Status: "Ignored", Type: "Device"})
	}
	return entries
}
