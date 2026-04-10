package health_checker

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	natural "github.com/maruel/natural"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
)

const (
	TEMPERATURE_TABLE_NAME    = "TEMPERATURE_INFO"
	FAN_TABLE_NAME            = "FAN_INFO"
	PSU_TABLE_NAME            = "PSU_INFO"
	LIQUID_COOLING_TABLE_NAME = "LIQUID_COOLING_INFO"
)

// HardwareChecker checks system hardware status. For now, it checks ASIC, PSU, fan,
// and liquid cooling status.
type HardwareChecker struct {
	HealthChecker
	leakingSensors []string
}

func NewHardwareChecker() *HardwareChecker {
	/* NewHardwareChecker creates a new HardwareChecker. */
	return &HardwareChecker{
		HealthChecker: NewHealthChecker(),
	}
}

func (hwc *HardwareChecker) GetCategory() string {
	/* GetCategory returns the category for hardware checks. */
	return "Hardware"
}

func (hwc *HardwareChecker) Str() string {
	/* Str returns the checker name for error messages. */
	return reflect.TypeOf(hwc).Elem().Name()
}

func (hwc *HardwareChecker) Check(config *Config) {
	/* Check performs all hardware health checks.*/
	hwc.Reset()
	hwc.checkAsicStatus(config)
	hwc.checkFanStatus(config)
	hwc.checkPsuStatus(config)
	hwc.checkLiquidCoolingStatus(config)
}

func (hwc *HardwareChecker) checkAsicStatus(config *Config) {
	/* checkAsicStatus checks if ASIC temperature is in valid range.
	:param config: Health checker configuration.
	:return:*/
	if _, ok := config.IgnoreDevices["asic"]; ok {
		return
	}

	queries := [][]string{
		{common.StateDb, TEMPERATURE_TABLE_NAME},
	}
	asicTempData, err := common.GetMapFromQueries(queries)
	if err != nil {
		return
	}

	// Filter for ASIC keys only
	for key, val := range asicTempData {
		if !strings.HasPrefix(key, "ASIC") {
			continue
		}
		asicName := key

		dataDict, ok := val.(map[string]interface{})
		if !ok {
			continue
		}

		temperatureStr, _ := dataDict["temperature"].(string)
		thresholdStr, _ := dataDict["high_threshold"].(string)

		if temperatureStr == "" {
			hwc.SetObjectNotOK("ASIC", asicName,
				fmt.Sprintf("Failed to get %s temperature", asicName))
		} else if thresholdStr == "" {
			hwc.SetObjectNotOK("ASIC", asicName,
				fmt.Sprintf("Failed to get %s temperature threshold", asicName))
		} else {
			temperature, errT := strconv.ParseFloat(temperatureStr, 64)
			threshold, errTh := strconv.ParseFloat(thresholdStr, 64)
			if errT != nil || errTh != nil {
				hwc.SetObjectNotOK("ASIC", asicName,
					fmt.Sprintf("Invalid %s temperature data, temperature=%s, threshold=%s",
						asicName, temperatureStr, thresholdStr))
			} else if temperature > threshold {
				hwc.SetObjectNotOK("ASIC", asicName,
					fmt.Sprintf("%s temperature is too hot, temperature=%v, threshold=%v",
						asicName, temperature, threshold))
			} else {
				hwc.SetObjectOK("ASIC", asicName)
			}
		}
	}
}

func (hwc *HardwareChecker) checkFanStatus(config *Config) {
	/* checkFanStatus checks fan status including:
	1. Check all fans are present.
	2. Check all fans are in good state.
	3. Check fan speed is in valid range.
	4. Check all fans direction are the same.
	:param config: Health checker configuration.
	:return:*/
	if _, ok := config.IgnoreDevices["fan"]; ok {
		return
	}

	queries := [][]string{
		{common.StateDb, FAN_TABLE_NAME},
	}
	fanInfoData, err := common.GetMapFromQueries(queries)
	if err != nil {
		return
	}
	if len(fanInfoData) == 0 {
		hwc.SetObjectNotOK("Fan", "Fan", "Failed to get fan information")
		return
	}

	keys := make([]string, 0, len(fanInfoData))
	for fanName := range fanInfoData {
		keys = append(keys, fanName)
	}
	sort.Sort(natural.StringSlice(keys))

	expectFanDirection := ""
	expectFanName := ""

	for _, name := range keys {
		if _, ok := config.IgnoreDevices[name]; ok {
			continue
		}

		dataDict, ok := fanInfoData[name].(map[string]interface{})
		if !ok {
			hwc.SetObjectNotOK("Fan", name, fmt.Sprintf("Invalid data for FAN_INFO: %s", name))
			continue
		}

		presence, _ := dataDict["presence"].(string)
		if strings.ToLower(presence) != "true" {
			hwc.SetObjectNotOK("Fan", name, fmt.Sprintf("%s is missing", name))
			continue
		}

		if !ignoreCheck(config, "fan", name, "speed") {
			speedStr, _ := dataDict["speed"].(string)
			speedTarget, _ := dataDict["speed_target"].(string)
			isUnderSpeed, _ := dataDict["is_under_speed"].(string)
			isOverSpeed, _ := dataDict["is_over_speed"].(string)

			if speedStr == "" {
				hwc.SetObjectNotOK("Fan", name, fmt.Sprintf("Failed to get actual speed data for %s", name))
				continue
			} else if speedTarget == "" {
				hwc.SetObjectNotOK("Fan", name, fmt.Sprintf("Failed to get target speed data for %s", name))
				continue
			} else if isUnderSpeed == "" {
				hwc.SetObjectNotOK("Fan", name, fmt.Sprintf("Failed to get under speed threshold check for %s", name))
				continue
			} else if isOverSpeed == "" {
				hwc.SetObjectNotOK("Fan", name, fmt.Sprintf("Failed to get over speed threshold check for %s", name))
				continue
			} else {
				_, errSpeed := strconv.ParseFloat(speedStr, 64)
				_, errTarget := strconv.ParseFloat(speedTarget, 64)
				if errSpeed != nil || errTarget != nil {
					hwc.SetObjectNotOK("Fan", name,
						fmt.Sprintf("Invalid fan speed data for %s, speed=%s, target=%s, is_under_speed=%s, is_over_speed=%s",
							name, speedStr, speedTarget, isUnderSpeed, isOverSpeed))
					continue
				}
				if strings.ToLower(isUnderSpeed) == "true" || strings.ToLower(isOverSpeed) == "true" {
					hwc.SetObjectNotOK("Fan", name,
						fmt.Sprintf("%s speed is out of range, speed=%s, target=%s", name, speedStr, speedTarget))
					continue
				}
			}
		}

		if !ignoreCheck(config, "fan", name, "direction") {
			direction, ok := dataDict["direction"].(string)
			if !ok {
				direction = "N/A"
			}
			if direction != "N/A" {
				if expectFanDirection == "" {
					expectFanDirection = direction
					expectFanName = name
				} else if direction != expectFanDirection {
					hwc.SetObjectNotOK("Fan", name,
						fmt.Sprintf("%s direction %s is not aligned with %s direction %s",
							name, direction, expectFanName, expectFanDirection))
					continue
				}
			}
		}

		status, _ := dataDict["status"].(string)
		if strings.ToLower(status) != "true" {
			hwc.SetObjectNotOK("Fan", name, fmt.Sprintf("%s is broken", name))
			continue
		}

		hwc.SetObjectOK("Fan", name)
	}
}

func (hwc *HardwareChecker) checkPsuStatus(config *Config) {
	/* checkPsuStatus checks PSU status including:
	1. Check all PSUs are present.
	2. Check all PSUs are power on.
	3. Check PSU temperature is in valid range.
	4. Check PSU voltage is in valid range.
	:param config: Health checker configuration.
	:return:*/
	if _, ok := config.IgnoreDevices["psu"]; ok {
		return
	}

	queries := [][]string{
		{common.StateDb, PSU_TABLE_NAME},
	}
	psuInfoData, err := common.GetMapFromQueries(queries)
	if err != nil {
		return
	}
	if len(psuInfoData) == 0 {
		hwc.SetObjectNotOK("PSU", "PSU", "Failed to get PSU information")
		return
	}

	keys := make([]string, 0, len(psuInfoData))
	for psuName := range psuInfoData {
		keys = append(keys, psuName)
	}
	sort.Sort(natural.StringSlice(keys))

	for _, name := range keys {
		if _, ok := config.IgnoreDevices[name]; ok {
			continue
		}

		dataDict, ok := psuInfoData[name].(map[string]interface{})
		if !ok {
			hwc.SetObjectNotOK("PSU", name, fmt.Sprintf("Invalid data for PSU_INFO: %s", name))
			continue
		}

		presence, _ := dataDict["presence"].(string)
		if strings.ToLower(presence) != "true" {
			hwc.SetObjectNotOK("PSU", name, fmt.Sprintf("%s is missing or not available", name))
			continue
		}

		status, _ := dataDict["status"].(string)
		if strings.ToLower(status) != "true" {
			hwc.SetObjectNotOK("PSU", name, fmt.Sprintf("%s is out of power", name))
			continue
		}

		// Check temperature
		if !ignoreCheck(config, "psu", name, "temperature") {
			tempStr, _ := dataDict["temp"].(string)
			tempThStr, _ := dataDict["temp_threshold"].(string)
			if tempStr == "" {
				hwc.SetObjectNotOK("PSU", name, fmt.Sprintf("Failed to get temperature data for %s", name))
				continue
			} else if tempThStr == "" {
				hwc.SetObjectNotOK("PSU", name, fmt.Sprintf("Failed to get temperature threshold data for %s", name))
				continue
			} else {
				temp, errT := strconv.ParseFloat(tempStr, 64)
				tempTh, errTh := strconv.ParseFloat(tempThStr, 64)
				if errT != nil || errTh != nil {
					hwc.SetObjectNotOK("PSU", name,
						fmt.Sprintf("Invalid temperature data for %s, temperature=%s, threshold=%s", name, tempStr, tempThStr))
					continue
				}
				if temp > tempTh {
					hwc.SetObjectNotOK("PSU", name,
						fmt.Sprintf("%s temperature is too hot, temperature=%v, threshold=%v", name, temp, tempTh))
					continue
				}
			}
		}

		// Check voltage
		if !ignoreCheck(config, "psu", name, "voltage") {
			voltage, _ := dataDict["voltage"].(string)
			voltMinTh, _ := dataDict["voltage_min_threshold"].(string)
			voltMaxTh, _ := dataDict["voltage_max_threshold"].(string)
			if voltage == "" {
				hwc.SetObjectNotOK("PSU", name, fmt.Sprintf("Failed to get voltage data for %s", name))
				continue
			} else if voltMinTh == "" {
				hwc.SetObjectNotOK("PSU", name, fmt.Sprintf("Failed to get voltage minimum threshold data for %s", name))
				continue
			} else if voltMaxTh == "" {
				hwc.SetObjectNotOK("PSU", name, fmt.Sprintf("Failed to get voltage maximum threshold data for %s", name))
				continue
			} else {
				volt, errV := strconv.ParseFloat(voltage, 64)
				voltMinTh, errMin := strconv.ParseFloat(voltMinTh, 64)
				voltMaxTh, errMax := strconv.ParseFloat(voltMaxTh, 64)
				if errV != nil || errMin != nil || errMax != nil {
					hwc.SetObjectNotOK("PSU", name,
						fmt.Sprintf("Invalid voltage data for %s, voltage=%s, range=[%s,%s]", name, voltage, voltMinTh, voltMaxTh))
					continue
				}
				if volt < voltMinTh || volt > voltMaxTh {
					hwc.SetObjectNotOK("PSU", name,
						fmt.Sprintf("%s voltage is out of range, voltage=%s, range=[%s,%s]", name, voltage, voltMinTh, voltMaxTh))
					continue
				}
			}
		}

		// Check power threshold
		if !ignoreCheck(config, "psu", name, "power_threshold") {
			powerOverload, _ := dataDict["power_overload"].(string)
			if powerOverload == "True" {
				powerCriticalVal, criticalExists := dataDict["power_critical_threshold"]
				_, powerExists := dataDict["power"]
				if powerExists && criticalExists {
					hwc.SetObjectNotOK("PSU", name,
						fmt.Sprintf("System power exceeds threshold (%vw)", powerCriticalVal))
				} else {
					hwc.SetObjectNotOK("PSU", name,
						"System power exceeds threshold but power_critical_threshold is invalid")
				}
				continue
			}
		}

		hwc.SetObjectOK("PSU", name)
	}
}

func ignoreCheck(config *Config, category, objectName, checkPoint string) bool {
	/* ignoreCheck checks if a specific check point should be ignored based on the
	devices_to_ignore config list. Supports patterns: "category.checkPoint"
	and "objectName.checkPoint".*/
	if config == nil {
		return false
	}
	if _, ok := config.IgnoreDevices[fmt.Sprintf("%s.%s", category, checkPoint)]; ok {
		return true
	}
	if _, ok := config.IgnoreDevices[fmt.Sprintf("%s.%s", objectName, checkPoint)]; ok {
		return true
	}
	return false
}

func (hwc *HardwareChecker) checkLiquidCoolingStatus(config *Config) {
	/* Check liquid cooling status including:
	   1. Check all leakage sensors are in good state
	   :param config: Health checker configuration*/
	if len(config.IncludeDevices) == 0 {
		return
	}
	if _, ok := config.IncludeDevices["liquid_cooling"]; !ok {
		return
	}

	queries := [][]string{
		{common.StateDb, LIQUID_COOLING_TABLE_NAME},
	}
	liquidCoolingData, err := common.GetMapFromQueries(queries)
	if err != nil {
		return
	}
	if len(liquidCoolingData) == 0 {
		hwc.SetObjectNotOK("Liquid Cooling", "Liquid Cooling", "Failed to get liquid cooling information")
		return
	}

	keys := make([]string, 0, len(liquidCoolingData))
	for sensorName := range liquidCoolingData {
		keys = append(keys, sensorName)
	}
	sort.Sort(natural.StringSlice(keys))

	newLeakingSensors := []string{}

	for _, name := range keys {
		if _, ok := config.IgnoreDevices[name]; ok {
			continue
		}

		dataDict, ok := liquidCoolingData[name].(map[string]interface{})
		if !ok {
			hwc.SetObjectNotOK("Liquid Cooling", name, fmt.Sprintf("Invalid key for LIQUID_COOLING_INFO: %s", name))
			continue
		}

		leakStatus, _ := dataDict["leak_status"].(string)
		if leakStatus == "" || leakStatus == "N/A" {
			hwc.SetObjectNotOK("Liquid Cooling", name, fmt.Sprintf("Failed to get leakage sensor status for %s", name))
			continue
		}

		if strings.ToLower(leakStatus) == "yes" && !containsString(hwc.leakingSensors, name) {
			hwc.leakingSensors = append(hwc.leakingSensors, name)
			newLeakingSensors = append(newLeakingSensors, name)
			hwc.SetObjectNotOK("Liquid Cooling", name, fmt.Sprintf("Leakage sensor %s is leaking", name))
			continue
		}

		if strings.ToLower(leakStatus) == "no" {
			hwc.SetObjectOK("Liquid Cooling", name)
			if containsString(hwc.leakingSensors, name) {
				// publish_events not implemented
				hwc.leakingSensors = removeString(hwc.leakingSensors, name)
			}
		}
	}
}

// containsString checks if a string slice contains a given value.
func containsString(s []string, v string) bool {
	for _, item := range s {
		if item == v {
			return true
		}
	}
	return false
}

// removeString returns a new slice with all occurrences of v removed.
func removeString(s []string, v string) []string {
	result := make([]string, 0, len(s))
	for _, item := range s {
		if item != v {
			result = append(result, item)
		}
	}
	return result
}
