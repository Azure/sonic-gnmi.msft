package show_client

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	"github.com/sonic-net/sonic-gnmi/show_client/helpers"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// SystemHealthDetail is the top-level JSON response for "show system-health detail".
type SystemHealthDetail struct {
	StatusLed   string                       `json:"system_status_led"`
	Services    helpers.ServiceHealthStatus  `json:"services"`
	Hardware    helpers.HardwareHealthStatus `json:"hardware"`
	MonitorList []helpers.HealthListEntry    `json:"monitor_list"`
	IgnoreList  []helpers.HealthListEntry    `json:"ignore_list"`
}

// SystemHealthMonitorList is the top-level JSON response for "show system-health monitor-list".
type SystemHealthMonitorList struct {
	MonitorList []helpers.HealthListEntry `json:"monitor_list"`
}

func getSystemHealthSummary(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	/* getSystemHealthSummary implements "show system-health summary".
	Shows system-health summary information.
	*/
	if common.IsMultiAsic() {
		log.Errorf("Attempted to execute 'show system-health summary' on a multi-ASIC platform")
		return nil, fmt.Errorf("'show system-health summary' is not supported on multi-ASIC platforms")
	}

	_, stat, err := helpers.GetSystemHealthStatus()
	if err != nil {
		return nil, err
	}

	led := getStatusLed()

	summary := helpers.DisplaySystemHealthSummary(stat, led)
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

func getSystemHealthDetail(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	/* getSystemHealthDetail implements "show system-health detail".
	Shows system-health summary, monitor list, and ignore list.
	*/
	if common.IsMultiAsic() {
		log.Errorf("Attempted to execute 'show system-health detail' on a multi-ASIC platform")
		return nil, fmt.Errorf("'show system-health detail' is not supported on multi-ASIC platforms")
	}

	manager, stat, err := helpers.GetSystemHealthStatus()
	if err != nil {
		return nil, err
	}

	led := getStatusLed()

	summary := helpers.DisplaySystemHealthSummary(stat, led)
	monitorList := helpers.DisplayMonitorList(stat)
	ignoreList := helpers.DisplayIgnoreList(manager)

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
	/* getSystemHealthMonitorList implements "show system-health monitor-list".
	Shows system-health monitored services and devices name list.
	*/
	if common.IsMultiAsic() {
		log.Errorf("Attempted to execute 'show system-health monitor-list' on a multi-ASIC platform")
		return nil, fmt.Errorf("'show system-health monitor-list' is not supported on multi-ASIC platforms")
	}

	_, stat, err := helpers.GetSystemHealthStatus()
	if err != nil {
		return nil, err
	}

	monitorList := helpers.DisplayMonitorList(stat)

	result := SystemHealthMonitorList{
		MonitorList: monitorList,
	}

	return json.Marshal(result)
}

func getSystemHealthSysreadyStatus(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	/* getSystemHealthSysreadyStatus implements "show system-health sysready-status".
	   Shows system ready status and per-service table.
	*/
	services, err := helpers.GetSysreadyServices()
	if err != nil {
		return nil, fmt.Errorf("failed to query service status: %w", err)
	}

	if services == nil {
		return nil, fmt.Errorf("No system ready status data available - system-health service might be down")
	}

	sysStatus, err := helpers.GetSysreadyStatus()
	if err != nil {
		return nil, err
	}

	result := helpers.SysreadyStatus{
		SystemStatus: sysStatus,
		Services:     services,
	}
	return json.Marshal(result)
}
