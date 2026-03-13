package show_client

import (
	"encoding/json"
	"fmt"
	"strings"

	helpers "github.com/sonic-net/sonic-gnmi/show_client/helpers/health_check_manager"
	"github.com/sonic-net/sonic-gnmi/show_client/helpers/platform"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// SystemHealthSummary represents the output structure for show system-health summary.
type SystemHealthSummary struct {
	Summary       string              `json:"summary"`
	StatusLed     string              `json:"system_status_led"`
	Services      ServiceHealthStatus `json:"services"`
	Hardware      HardwareHealthStatus `json:"hardware"`
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

func getSystemHealthSummary(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	/* getSystemHealthSummary implements "show system-health summary".
		Shows system-health summary information.
	*/
	_, chassis, stat, err := getSystemHealthStatus()
	if err != nil {
		return nil, err
	}

	led := ""
	if chassis != nil {
		led = chassis.GetStatusLed()
	}

	summary := displaySystemHealthSummary(stat, led)
	return json.Marshal(summary)
}

func getSystemHealthStatus() (*helpers.HealthCheckerManager, platform.ChassisBase, map[string]interface{}, error) {
	/* getSystemHealthStatus creates a HealthCheckerManager, verifies config exists,
		and performs the system health check.
		:return: manager, chassis, stat (check results), error.
	*/
	manager := helpers.NewHealthCheckerManager()

	if !manager.Config.ConfigFileExists() {
		return nil, nil, nil, fmt.Errorf("System health configuration file not found")
	}

	// Create platform-specific chassis — Python: from sonic_platform.chassis import Chassis; chassis = Chassis()
	chassis := platform.NewChassis()

	// Run health checks and set LED
	stat := manager.Check(chassis)

	return manager, chassis, stat, nil
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
		// Python reverses device_list, then shows first as main reason
		// and rest as additional reasons
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