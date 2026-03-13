package helpers

import (
	"fmt"
	"strconv"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	"github.com/sonic-net/sonic-gnmi/show_client/helpers/platform"
)

// HealthCheckerManager manages all system health checkers and system health configuration.
type HealthCheckerManager struct {
	checkers []Checker
	Config   *Config
}

func NewHealthCheckerManager() *HealthCheckerManager {
	/* NewHealthCheckerManager creates a new HealthCheckerManager. */
	manager := &HealthCheckerManager{
		Config: NewConfig(),
	}
	manager.initialize()
	return manager
}

func (manager *HealthCheckerManager) initialize() {
	/* initialize creates service checker and hardware checker by default.
		:return:*/
	manager.checkers = append(manager.checkers, NewServiceChecker())
	manager.checkers = append(manager.checkers, NewHardwareChecker())
}

func (manager *HealthCheckerManager) Check(chassis platform.ChassisBase) map[string]interface{} {
	/* Check loads new configuration if any and performs the system health check
		for all existing checkers.
		:param chassis: A chassis object.
		:return: A dictionary that contains the status for all objects that was checked.*/
	Summary = StatusOK
	stats := make(map[string]interface{})

	manager.Config.LoadConfig()
	for _, c := range manager.checkers {
		manager.doCheck(c, stats)
	}

	for cmd := range manager.Config.UserDefinedCheckers {
		c := NewUserDefinedChecker(cmd)
		manager.doCheck(c, stats)
	}

	if chassis != nil {
		manager.setSystemLED(chassis)
	}

	return stats
}

func (manager *HealthCheckerManager) doCheck(c Checker, stats map[string]interface{}) {
	/* doCheck does check for a particular checker and collects the check statistic.
		:param c: A checker object.
		:param stats: Check statistic.
		:return:*/
	defer func() {
		if r := recover(); r != nil {
			Summary = StatusNotOK
			errMsg := fmt.Sprintf("Failed to perform health check for %s due to exception - %v", c, r)
			log.Errorf(errMsg)
			manager.addInternalError(stats, c, errMsg)
		}
	}()

	c.Check(manager.Config)	
	category := c.GetCategory()
	info := c.GetInfo()

	if _, ok := stats[category]; !ok {
		stats[category] = info
	} else {
		// Merge like Python's dict.update()
		existing := stats[category].(map[string]interface{})
		for k, v := range info {
			existing[k] = v
		}
	}
}

func (manager *HealthCheckerManager) addInternalError(stats map[string]interface{}, c Checker, msg string) {
	/* addInternalError records an internal error entry in stats
		under the "Internal" category.*/
	entry := map[string]interface{}{
		c.String(): map[string]interface{}{
			INFO_FIELD_OBJECT_STATUS: StatusNotOK,
			INFO_FIELD_OBJECT_MSG:    msg,
			INFO_FIELD_OBJECT_TYPE:   "Internal",
		},
	}

	if _, ok := stats["Internal"]; !ok {
		stats["Internal"] = entry
	} else {
		existing := stats["Internal"].(map[string]interface{})
		for k, v := range entry {
			existing[k] = v
		}
	}
}

func (manager *HealthCheckerManager) setSystemLED(chassis platform.ChassisBase) {
	/* setSystemLED sets the system status LED based on the overall health. */
	defer func() {
		if r := recover(); r != nil {
			errStr := fmt.Sprintf("%v", r)
			if strings.Contains(strings.ToLower(errStr), "not implemented") {
				log.Errorf("chassis.set_status_led is not implemented")
			} else {
				log.Errorf("Failed to set system LED due to - %v", r)
			}
		}
	}()

	chassis.SetStatusLed(manager.getLEDTargetColor())
}

func (manager *HealthCheckerManager) getLEDTargetColor() string {
	/* getLEDTargetColor gets target LED color according to health status and system uptime.
		:return: String LED color.*/
	if Summary == StatusOK {
		return manager.Config.GetLEDColor("normal")
	}

	uptimeStr := common.GetUptime([]string{"-s"})
	bootupTimeout := GetBootupTimeout()

	// Parse uptime; if we can't determine it, assume booting is done
	uptimeSeconds := parseUptimeSeconds(uptimeStr)
	if uptimeSeconds < bootupTimeout {
		return manager.Config.GetLEDColor("booting")
	}

	return manager.Config.GetLEDColor("fault")
}

func parseUptimeSeconds(uptimeStr string) int {
	/* parseUptimeSeconds converts the "uptime -s" output (boot timestamp) to
		seconds since boot. Returns a large value if parsing fails (assumes not booting).
		Note: Go-specific helper. Python uses utils.get_uptime() which returns
		elapsed seconds directly.*/
	uptimeStr = strings.TrimSpace(uptimeStr)
	if uptimeStr == "" || uptimeStr == "N/A" {
		return int(^uint(0) >> 1) // max int — assume not booting
	}

	// "uptime -s" returns the boot time, not elapsed seconds.
	// We need elapsed time. Use /proc/uptime instead for raw seconds.
	raw := common.ReadStringFromFile("/proc/uptime", "")
	if raw == "" {
		return int(^uint(0) >> 1)
	}
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return int(^uint(0) >> 1)
	}
	// /proc/uptime first field is seconds since boot (float)
	dotPos := strings.Index(fields[0], ".")
	intPart := fields[0]
	if dotPos >= 0 {
		intPart = fields[0][:dotPos]
	}
	seconds, err := strconv.Atoi(intPart)
	if err != nil {
		return int(^uint(0) >> 1)
	}
	return seconds
}
