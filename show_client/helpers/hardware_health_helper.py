package helpers

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Mocked DB connector interface, replace with actual implementation
type DBConnector interface {
	Keys(table string, pattern string) []string
	Get(table string, key string, field string) string
	GetAll(table string, key string) map[string]string
	Connect(table string)
}

type HealthChecker struct {
	info map[string]map[string]string
}

func (hc *HealthChecker) SetObjectOk(category string, name string) {
	if hc.info == nil {
		hc.info = make(map[string]map[string]string)
	}
	if _, ok := hc.info[category]; !ok {
		hc.info[category] = make(map[string]string)
	}
	hc.info[category][name] = "OK"
}

func (hc *HealthChecker) SetObjectNotOk(category string, name string, message string) {
	if hc.info == nil {
		hc.info = make(map[string]map[string]string)
	}
	if _, ok := hc.info[category]; !ok {
		hc.info[category] = make(map[string]string)
	}
	hc.info[category][name] = "NOT OK: " + message
}

func (hc *HealthChecker) Reset() {
	hc.info = make(map[string]map[string]string)
}

// Config struct to pass configuration, e.g. ignored devices
type Config struct {
	IgnoreDevices map[string]bool
}

// HardwareChecker implements the hardware health checks
type HardwareChecker struct {
	HealthChecker
	db DBConnector
}

func NewHardwareChecker(db DBConnector) *HardwareChecker {
	hc := &HardwareChecker{
		db: db,
	}
	hc.db.Connect("STATE_DB")
	return hc
}

func GetCategory() string {
	return "Hardware"
}

func HardwareHealthCheck(config map[string]interface{}) map[string]interface{} {
	Reset()
	CheckAsicStatus(config)
	CheckFanStatus(config)
	CheckPsuStatus(config)
}

func CheckAsicStatus(config map[string]interface{}) {
	const asicTempKey = "TEMPERATURE_INFO|ASIC"
	if config != nil && config.IgnoreDevices["asic"] {
		return
	}
	asicKeys := hc.db.Keys("STATE_DB", asicTempKey+"*")
	for _, asicKey := range asicKeys {
		temperatureStr := hc.db.Get("STATE_DB", asicKey, "temperature")
		thresholdStr := hc.db.Get("STATE_DB", asicKey, "high_threshold")
		parts := strings.Split(asicKey, "|")
		asicName := ""
		if len(parts) > 1 {
			asicName = parts[1]
		} else {
			asicName = asicKey
		}
		if temperatureStr == "" {
			hc.SetObjectNotOk("ASIC", asicName, fmt.Sprintf("Failed to get %s temperature", asicName))
			continue
		}
		if thresholdStr == "" {
			hc.SetObjectNotOk("ASIC", asicName, fmt.Sprintf("Failed to get %s temperature threshold", asicName))
			continue
		}
		temperature, errT := strconv.ParseFloat(temperatureStr, 64)
		threshold, errTh := strconv.ParseFloat(thresholdStr, 64)
		if errT != nil || errTh != nil {
			hc.SetObjectNotOk("ASIC", asicName, fmt.Sprintf("Invalid %s temperature data, temperature=%s, threshold=%s", asicName, temperatureStr, thresholdStr))
			continue
		}
		if temperature > threshold {
			hc.SetObjectNotOk("ASIC", asicName, fmt.Sprintf("%s temperature is too hot, temperature=%f, threshold=%f", asicName, temperature, threshold))
		} else {
			hc.SetObjectOk("ASIC", asicName)
		}
	}
}

func CheckFanStatus(config map[string]interface{}) {
	const fanTable = "FAN_INFO"
	if config != nil && config.IgnoreDevices["fan"] {
		return
	}
	keys := hc.db.Keys("STATE_DB", fanTable+"*")
	if len(keys) == 0 {
		hc.SetObjectNotOk("Fan", "Fan", "Failed to get fan information")
		return
	}
	sort.Strings(keys) // natsorted equivalent for typical keys
	expectFanDirection := ""
	expectFanName := ""
	for _, key := range keys {
		parts := strings.Split(key, "|")
		if len(parts) != 2 {
			hc.SetObjectNotOk("Fan", key, "Invalid key for FAN_INFO: "+key)
			continue
		}
		name := parts[1]
		if config != nil && config.IgnoreDevices[name] {
			continue
		}
		data := hc.db.GetAll("STATE_DB", key)
		presence := strings.ToLower(data["presence"])
		if presence != "true" {
			hc.SetObjectNotOk("Fan", name, fmt.Sprintf("%s is missing", name))
			continue
		}
		if !ignoreCheck(config, "fan", name, "speed") {
			speedStr := data["speed"]
			speedTargetStr := data["speed_target"]
			isUnder := data["is_under_speed"]
			isOver := data["is_over_speed"]
			if speedStr == "" {
				hc.SetObjectNotOk("Fan", name, fmt.Sprintf("Failed to get actual speed data for %s", name))
				continue
			}
			if speedTargetStr == "" {
				hc.SetObjectNotOk("Fan", name, fmt.Sprintf("Failed to get target speed data for %s", name))
				continue
			}
			if isUnder == "" {
				hc.SetObjectNotOk("Fan", name, fmt.Sprintf("Failed to get under speed threshold check for %s", name))
				continue
			}
			if isOver == "" {
				hc.SetObjectNotOk("Fan", name, fmt.Sprintf("Failed to get over speed threshold check for %s", name))
				continue
			}
			speed, errSpeed := strconv.ParseFloat(speedStr, 64)
			speedTarget, errTarget := strconv.ParseFloat(speedTargetStr, 64)
			if errSpeed != nil || errTarget != nil || strings.ToLower(isUnder) == "true" || strings.ToLower(isOver) == "true" {
				hc.SetObjectNotOk("Fan", name, fmt.Sprintf("%s speed is out of range, speed=%s, target=%s", name, speedStr, speedTargetStr))
				continue
			}
		}
		if !ignoreCheck(config, "fan", name, "direction") {
			direction := data["direction"]
			if direction != "N/A" && direction != "" {
				if expectFanDirection == "" {
					expectFanDirection = direction
					expectFanName = name
				} else if direction != expectFanDirection {
					hc.SetObjectNotOk("Fan", name, fmt.Sprintf("%s direction %s is not aligned with %s direction %s", name, direction, expectFanName, expectFanDirection))
					continue
				}
			}
		}
		status := strings.ToLower(data["status"])
		if status != "true" {
			hc.SetObjectNotOk("Fan", name, fmt.Sprintf("%s is broken", name))
			continue
		}
		hc.SetObjectOk("Fan", name)
	}
}

func CheckPsuStatus(config map[string]interface{}) {
	const psuTable = "PSU_INFO"
	if config != nil && config.IgnoreDevices["psu"] {
		return
	}
	keys := hc.db.Keys("STATE_DB", psuTable+"*")
	if len(keys) == 0 {
		hc.SetObjectNotOk("PSU", "PSU", "Failed to get PSU information")
		return
	}
	sort.Strings(keys)
	for _, key := range keys {
		parts := strings.Split(key, "|")
		if len(parts) != 2 {
			hc.SetObjectNotOk("PSU", key, "Invalid key for PSU_INFO: "+key)
			continue
		}
		name := parts[1]
		if config != nil && config.IgnoreDevices[name] {
			continue
		}
		data := hc.db.GetAll("STATE_DB", key)
		presence := strings.ToLower(data["presence"])
		if presence != "true" {
			hc.SetObjectNotOk("PSU", name, fmt.Sprintf("%s is missing or not available", name))
			continue
		}
		status := strings.ToLower(data["status"])
		if status != "true" {
			hc.SetObjectNotOk("PSU", name, fmt.Sprintf("%s is out of power", name))
			continue
		}
		if !ignoreCheck(config, "psu", name, "temperature") {
			tempStr := data["temp"]
			tempThStr := data["temp_threshold"]
			if tempStr == "" || tempThStr == "" {
				hc.SetObjectNotOk("PSU", name, fmt.Sprintf("Failed to get temperature data for %s", name))
				continue
			}
			temp, errT := strconv.ParseFloat(tempStr, 64)
			tempTh, errTh := strconv.ParseFloat(tempThStr, 64)
			if errT != nil || errTh != nil {
				hc.SetObjectNotOk("PSU", name, fmt.Sprintf("Invalid temperature data for %s, temperature=%s, threshold=%s", name, tempStr, tempThStr))
				continue
			}
			if temp > tempTh {
				hc.SetObjectNotOk("PSU", name, fmt.Sprintf("%s temperature is too hot, temperature=%f, threshold=%f", name, temp, tempTh))
				continue
			}
		}
		if !ignoreCheck(config, "psu", name, "voltage") {
			voltStr := data["voltage"]
			voltMinStr := data["voltage_min_threshold"]
			voltMaxStr := data["voltage_max_threshold"]
			if voltStr == "" || voltMinStr == "" || voltMaxStr == "" {
				hc.SetObjectNotOk("PSU", name, fmt.Sprintf("Failed to get voltage data for %s", name))
				continue
			}
			volt, errV := strconv.ParseFloat(voltStr, 64)
			voltMin, errMin := strconv.ParseFloat(voltMinStr, 64)
			voltMax, errMax := strconv.ParseFloat(voltMaxStr, 64)
			if errV != nil || errMin != nil || errMax != nil || volt < voltMin || volt > voltMax {
				hc.SetObjectNotOk("PSU", name, fmt.Sprintf("%s voltage is out of range, voltage=%s, range=[%s,%s]", name, voltStr, voltMinStr, voltMaxStr))
				continue
			}
		}
		if !ignoreCheck(config, "psu", name, "power_threshold") {
			if data["power_overload"] == "True" {
				powerCritical := data["power_critical_threshold"]
				if powerCritical != "" {
					hc.SetObjectNotOk("PSU", name, fmt.Sprintf("System power exceeds threshold (%sw)", powerCritical))
				} else {
					hc.SetObjectNotOk("PSU", name, "System power exceeds threshold but power_critical_threshold is invalid")
				}
				continue
			}
		}
		hc.SetObjectOk("PSU", name)
	}
}

func ignoreCheck(config map[string]interface{}, category, objectName, checkPoint string) bool {
	if config == nil || config.IgnoreDevices == nil {
		return false
	}
	if config.IgnoreDevices[fmt.Sprintf("%s.%s", category, checkPoint)] {
		return true
	}
	if config.IgnoreDevices[fmt.Sprintf("%s.%s", objectName, checkPoint)] {
		return true
	}
	return false
}

