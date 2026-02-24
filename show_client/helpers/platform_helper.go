package helpers

import (
	"encoding/json"
	"fmt"
	"sort"

	log "github.com/golang/glog"
	natural "github.com/maruel/natural"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
)

// GetSensorData is a common helper that retrieves and processes sensor data
// Returns a slice of maps containing all sensor fields
func GetSensorData(tableName, valueFieldName string) ([]map[string]string, error) {
	queries := [][]string{
		{common.StateDb, tableName},
	}

	sensorData, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get %s from STATE_DB: %v", tableName, err)
		return nil, err
	}

	if len(sensorData) == 0 {
		log.Info("Sensor not detected")
		return []map[string]string{}, nil
	}

	// Collect all sensor keys and sort them naturally
	sensorKeys := make([]string, 0)
	for key := range sensorData {
		sensorKeys = append(sensorKeys, key)
	}
	sort.Sort(natural.StringSlice(sensorKeys))

	// Collect all sensor information
	result := make([]map[string]string, 0)

	for _, sensorName := range sensorKeys {
		value := sensorData[sensorName]
		sensorInfo, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		// Format value with unit (e.g., "12.0 V" or "5.5 A")
		valueRaw := common.GetValueOrDefault(sensorInfo, valueFieldName, "N/A")
		unit := common.GetValueOrDefault(sensorInfo, "unit", "")
		formattedValue := valueRaw
		if valueRaw != "N/A" && unit != "" {
			formattedValue = fmt.Sprintf("%s %s", valueRaw, unit)
		}

		sensorMap := map[string]string{
			"sensor":       sensorName,
			"value":        formattedValue,
			"high_th":      common.GetValueOrDefault(sensorInfo, "high_threshold", "N/A"),
			"low_th":       common.GetValueOrDefault(sensorInfo, "low_threshold", "N/A"),
			"crit_high_th": common.GetValueOrDefault(sensorInfo, "critical_high_threshold", "N/A"),
			"crit_low_th":  common.GetValueOrDefault(sensorInfo, "critical_low_threshold", "N/A"),
			"warning":      common.GetValueOrDefault(sensorInfo, "warning_status", "N/A"),
			"timestamp":    common.GetValueOrDefault(sensorInfo, "timestamp", "N/A"),
		}

		result = append(result, sensorMap)
	}

	return result, nil
}

// GetSensorJSON is a generic helper that retrieves sensor data and marshals it to JSON
// Takes a table name, value field name, and a mapper function to convert data to the desired struct type
func GetSensorJSON[T any](tableName, valueFieldName string, mapper func(map[string]string) T) ([]byte, error) {
	sensorData, err := GetSensorData(tableName, valueFieldName)
	if err != nil {
		return nil, err
	}

	if len(sensorData) == 0 {
		return json.Marshal("Sensor not detected")
	}

	result := make([]T, 0, len(sensorData))
	for _, data := range sensorData {
		result = append(result, mapper(data))
	}

	return json.Marshal(result)
}
