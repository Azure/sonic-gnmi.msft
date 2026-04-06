package helpers

import (
	"fmt"
	"strings"

	hc "github.com/sonic-net/sonic-gnmi/show_client/helpers/health_checker"
)

// SystemHealthSummary represents the output structure for show system-health summary.
type SystemHealthSummary struct {
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

// HealthListEntry represents a single entry in the monitor or ignore list.
type HealthListEntry struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Type   string `json:"type"`
}

func GetSystemHealthStatus() (*hc.HealthCheckerManager, map[string]interface{}, error) {
	/* GetSystemHealthStatus creates a HealthCheckerManager, verifies config exists,
	and performs the system health check.
	:return: manager, stat (check results), error.
	*/
	manager := hc.NewHealthCheckerManager()

	if !manager.Config.ConfigFileExists() {
		return nil, nil, fmt.Errorf("System health configuration file not found")
	}

	// Run health checks — LED is set internally via nsenter into host namespace
	// calling: sonic_platform.chassis.Chassis().set_status_led(color)
	stat := manager.Check()

	return manager, stat, nil
}

func DisplaySystemHealthSummary(stat map[string]interface{}, led string) SystemHealthSummary {
	/* DisplaySystemHealthSummary builds the system health summary output.
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
			status, _ := detail[hc.INFO_FIELD_OBJECT_STATUS].(string)
			if status != hc.StatusOK {
				if category == "Services" {
					msg, _ := detail[hc.INFO_FIELD_OBJECT_MSG].(string)
					if strings.Contains(msg, "Accessible") {
						fsList = append(fsList, element)
					} else {
						servicesList = append(servicesList, element)
					}
				} else {
					msg, _ := detail[hc.INFO_FIELD_OBJECT_MSG].(string)
					deviceList = append(deviceList, msg)
				}
			}
		}
	}

	// Build services status
	services := ServiceHealthStatus{
		Status: hc.StatusOK,
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
		Status: hc.StatusOK,
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
		StatusLed: led,
		Services:  services,
		Hardware:  hardware,
	}
}

func DisplayMonitorList(stat map[string]interface{}) []HealthListEntry {
	/* DisplayMonitorList builds the monitor list from the health check stat data.
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

func DisplayIgnoreList(manager *hc.HealthCheckerManager) []HealthListEntry {
	/* DisplayIgnoreList builds the ignore list from the manager's config.
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
