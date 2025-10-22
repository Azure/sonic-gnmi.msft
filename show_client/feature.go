package show_client

import (
	"encoding/json"
	"fmt"
	"strconv"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

func buildFeatureConfigEntry(name string, data map[string]interface{}) map[string]interface{} {
	// inverted fallback value
	fallbackValue := ""
	if val, exists := data["no_fallback_to_local"]; exists {
		if strVal, ok := val.(string); ok {
			if boolVal, err := strconv.ParseBool(strVal); err == nil {
				fallbackValue = strconv.FormatBool(!boolVal)
			}
		}
	}

	return map[string]interface{}{
		"name": name,
		"data": map[string]interface{}{
			"state":               common.GetValueOrDefault(data, "state", ""),
			"auto_restart":        common.GetValueOrDefault(data, "auto_restart", ""),
			"owner":           common.GetValueOrDefault(data, "set_owner", "local"),
			"fallback": fallbackValue,
		},
	}
}

func buildFeatureStatusEntry(name string, data map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"name": name,
		"data": map[string]interface{}{
			"state":               common.GetValueOrDefault(data, "state", ""),
			"auto_restart":        common.GetValueOrDefault(data, "auto_restart", ""),
			"update_time":           common.GetValueOrDefault(data, "update_time", ""),
			"container_id": common.GetValueOrDefault(data, "container_id", ""),
			"container_version": common.GetValueOrDefault(data, "container_version", ""),
			"set_owner":        common.GetValueOrDefault(data, "set_owner", ""),
			"current_owner": common.GetValueOrDefault(data, "current_owner", ""),
			"remote_state": common.GetValueOrDefault(data, "remote_state", ""),
		},
	}
}

func buildFeatureAutoRestartEntry(name string, data map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"name": name,
		"auto_restart": common.GetValueOrDefault(data, "auto_restart", "unknown"),
	}
}

func getFeatureConfig(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	queries := [][]string{{"CONFIG_DB", "FEATURE"}}

	rawData, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to get feature config data from queries %v, got err: %v", queries, err)
		return nil, err
	}

	// feature name if given
	var featureName string
	if len(args) > 0 {
		featureName = args[0]
	}

	featureTable := rawData

	if len(featureTable) == 0 {
		log.Errorf("Failed to fetch data from FEATURE table. Raw data: %+v", rawData)
		return nil, fmt.Errorf("failed to retrieve feature configuration data")
	}

	var features []map[string]interface{}

	if featureName != "" {
		// specific feature
		if featureData, exists := featureTable[featureName]; exists {
			if featureDataMap, ok := featureData.(map[string]interface{}); ok {
				features = append(features, buildFeatureConfigEntry(featureName, featureDataMap))
			}
		} else {
			log.Errorf("Feature '%s' not found in FEATURE table. Available features: %v", featureName, common.GetSortedKeys(featureTable))
			return nil, fmt.Errorf("feature '%s' not found", featureName)
		}
	} else {
		// all features, sorted
		for _, name := range common.GetSortedKeys(featureTable) {
			if featureData, ok := featureTable[name].(map[string]interface{}); ok {
				features = append(features, buildFeatureConfigEntry(name, featureData))
			}
		}
	}

	response := map[string]interface{}{
		"features": features,
	}

	return json.Marshal(response)
}


func getFeatureStatus(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	queries := [][]string{{"CONFIG_DB", "FEATURE"}}

	rawData, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to get feature status data from queries %v, got err: %v", queries, err)
		return nil, err
	}

	// feature name if given
	var featureName string
	if len(args) > 0 {
		featureName = args[0]
	}

	featureTable := rawData

	if len(featureTable) == 0 {
		log.Errorf("Failed to fetch data from FEATURE table. Raw data: %+v", rawData)
		return nil, fmt.Errorf("failed to retrieve feature status data")
	}

	var features []map[string]interface{}

	if featureName != "" {
		// specific feature
		if featureData, exists := featureTable[featureName]; exists {
			if featureDataMap, ok := featureData.(map[string]interface{}); ok {
				features = append(features, buildFeatureStatusEntry(featureName, featureDataMap))
			}
		} else {
			log.Errorf("Feature '%s' not found in FEATURE table. Available features: %v", featureName, common.GetSortedKeys(featureTable))
			return nil, fmt.Errorf("feature '%s' not found", featureName)
		}
	} else {
		// all features, sorted
		for _, name := range common.GetSortedKeys(featureTable) {
			if featureData, ok := featureTable[name].(map[string]interface{}); ok {
				features = append(features, buildFeatureStatusEntry(name, featureData))
			}
		}
	}

	response := map[string]interface{}{
		"features": features,
	}

	return json.Marshal(response)
}


func getFeatureAutoRestart(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	queries := [][]string{{"CONFIG_DB", "FEATURE"}}

	rawData, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to get feature autorestart data from queries %v, got err: %v", queries, err)
		return nil, err
	}

	// feature name if given
	var featureName string
	if len(args) > 0 {
		featureName = args[0]
	}

	featureTable := rawData

	if len(featureTable) == 0 {
		log.Errorf("Failed to fetch data from FEATURE table. Raw data: %+v", rawData)
		return nil, fmt.Errorf("failed to retrieve feature autorestart data")
	}

	var features []map[string]interface{}

	if featureName != "" {
		// specific feature
		if featureData, exists := featureTable[featureName]; exists {
			if featureDataMap, ok := featureData.(map[string]interface{}); ok {
				features = append(features, buildFeatureAutoRestartEntry(featureName, featureDataMap))
			}
		} else {
			log.Errorf("Feature '%s' not found in FEATURE table. Available features: %v", featureName, common.GetSortedKeys(featureTable))
			return nil, fmt.Errorf("feature '%s' not found", featureName)
		}
	} else {
		// all features, sorted
		for _, name := range common.GetSortedKeys(featureTable) {
			if featureData, ok := featureTable[name].(map[string]interface{}); ok {
				features = append(features, buildFeatureAutoRestartEntry(name, featureData))
			}
		}
	}

	response := map[string]interface{}{
		"features": features,
	}

	return json.Marshal(response)
}
