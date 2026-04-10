package health_checker

import (
	"fmt"
	"strconv"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
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

func (manager *HealthCheckerManager) Check() map[string]interface{} {
	/* Check loads new configuration if any and performs the system health check
	for all existing checkers.
	:return: A dictionary that contains the status for all objects that was checked.*/
	Summary = StatusOK
	stats := make(map[string]interface{})

	manager.Config.LoadConfig()
	for _, checker := range manager.checkers {
		manager.doCheck(checker, stats)
	}

	for cmd := range manager.Config.UserDefinedCheckers {
		checker := NewUserDefinedChecker(cmd)
		manager.doCheck(checker, stats)
	}

	manager.setSystemLED()

	return stats
}

func (manager *HealthCheckerManager) doCheck(checker Checker, stats map[string]interface{}) {
	/* do Check for a particular checker and collects the check statistic.
	:param checker: A checker object.
	:param stats: Check statistic.
	:return:*/
	defer func() {
		if r := recover(); r != nil {
			Summary = StatusNotOK
			errMsg := fmt.Sprintf("Failed to perform health check for %s due to exception - %v", checker.Str(), r)
			log.Errorf(errMsg)
			manager.addInternalError(stats, checker, errMsg)
		}
	}()

	checker.Check(manager.Config)
	category := checker.GetCategory()
	info := checker.GetInfo()

	if _, ok := stats[category]; !ok {
		stats[category] = info
	} else {
		existing := stats[category].(map[string]interface{})
		for objectName, objectInfo := range info {
			existing[objectName] = objectInfo
		}
	}
}

func (manager *HealthCheckerManager) addInternalError(stats map[string]interface{}, checker Checker, msg string) {
	/* addInternalError records an internal error entry in stats
	under the "Internal" category.*/
	entry := map[string]interface{}{
		checker.Str(): map[string]interface{}{
			INFO_FIELD_OBJECT_STATUS: StatusNotOK,
			INFO_FIELD_OBJECT_MSG:    msg,
			INFO_FIELD_OBJECT_TYPE:   "Internal",
		},
	}

	if _, ok := stats["Internal"]; !ok {
		stats["Internal"] = entry
	} else {
		existing := stats["Internal"].(map[string]interface{})
		for checkerName, errorInfo := range entry {
			existing[checkerName] = errorInfo
		}
	}
}

// setSystemLED sets the system status LED by calling the platform-specific
// sonic_platform.chassis module via nsenter into the host namespace.
func (manager *HealthCheckerManager) setSystemLED() {
	color := manager.getLEDTargetColor()

	pyCmd := fmt.Sprintf(
		`python3 -c "from sonic_platform.chassis import Chassis; chassis = Chassis(); chassis.set_status_led('%s')"`,
		color,
	)

	_, err := common.GetDataFromHostCommand(pyCmd)
	if err != nil {
		errStr := fmt.Sprintf("%v", err)
		if strings.Contains(strings.ToLower(errStr), "not implemented") {
			log.Errorf("chassis.set_status_led is not implemented")
		} else {
			log.Errorf("Failed to set system LED: %v", err)
		}
	}
}

// getLEDTargetColor gets target LED color according to health status and system uptime.
func (manager *HealthCheckerManager) getLEDTargetColor() string {
	if Summary == StatusOK {
		return manager.Config.GetLEDColor("normal")
	}

	uptime := getUptime()
	if uptime < float64(manager.Config.GetBootupTimeout()) {
		return manager.Config.GetLEDColor("booting")
	}

	return manager.Config.GetLEDColor("fault")
}

// getUptime reads system uptime in seconds from /proc/uptime.
func getUptime() float64 {
	raw := common.ReadStringFromFile("/proc/uptime", "")
	if raw == "" {
		return 0
	}
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return 0
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	return seconds
}
