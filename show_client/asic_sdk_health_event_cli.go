package show_client

import (
	"encoding/json"
	"fmt"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

const (
	switchCapabilityTable          = "SWITCH_CAPABILITY"
	switchCapabilityKey            = "switch"
	asicSdkHealthEventField        = "ASIC_SDK_HEALTH_EVENT"
	suppressAsicSdkHealthEventTable = "SUPPRESS_ASIC_SDK_HEALTH_EVENT"
	asicSdkHealthEventTable        = "ASIC_SDK_HEALTH_EVENT_TABLE"
)

// checkAsicSdkHealthEventSupported checks if the ASIC SDK health event feature
// is supported by reading SWITCH_CAPABILITY|switch from STATE_DB.
func checkAsicSdkHealthEventSupported() (bool, error) {
	queries := [][]string{
		{common.StateDb, switchCapabilityTable, switchCapabilityKey},
	}
	data, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.V(2).Infof("Unable to query switch capability: %v", err)
		return false, nil
	}
	if val, ok := data[asicSdkHealthEventField]; ok {
		if strVal, isStr := val.(string); isStr && strVal == "true" {
			return true, nil
		}
	}
	return false, nil
}

// getAsicSdkHealthEventSuppressConfig handles "show asic-sdk-health-event suppress-configuration".
// It reads the SUPPRESS_ASIC_SDK_HEALTH_EVENT table from CONFIG_DB and returns
// JSON with severity entries containing suppressed categories and max_events.
func getAsicSdkHealthEventSuppressConfig(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	supported, err := checkAsicSdkHealthEventSupported()
	if err != nil {
		return nil, err
	}
	if !supported {
		return nil, fmt.Errorf("ASIC/SDK health event is not supported on the platform")
	}

	queries := [][]string{
		{common.ConfigDb, suppressAsicSdkHealthEventTable},
	}
	data, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to get suppress config data from queries %v, err: %v", queries, err)
		return nil, err
	}

	var entries []map[string]interface{}
	for _, severity := range common.NatsortInterfaces(common.GetSortedKeys(data)) {
		entryData, ok := data[severity].(map[string]interface{})
		if !ok {
			continue
		}

		categories := "none"
		if catVal, exists := entryData["categories"]; exists {
			if catStr, ok := catVal.(string); ok && catStr != "" {
				categories = catStr
			}
		}

		maxEvents := "unlimited"
		if meVal, exists := entryData["max_events"]; exists {
			if meStr, ok := meVal.(string); ok && meStr != "" {
				maxEvents = meStr
			}
		}

		entries = append(entries, map[string]interface{}{
			"severity":   severity,
			"categories": categories,
			"max_events": maxEvents,
		})
	}

	if entries == nil {
		entries = []map[string]interface{}{}
	}

	response := map[string]interface{}{
		"suppress_configuration": entries,
	}
	return json.Marshal(response)
}

// getAsicSdkHealthEventReceived handles "show asic-sdk-health-event received".
// It reads the ASIC_SDK_HEALTH_EVENT_TABLE from STATE_DB and returns
// JSON with event entries containing date, severity, category, and description.
func getAsicSdkHealthEventReceived(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	supported, err := checkAsicSdkHealthEventSupported()
	if err != nil {
		return nil, err
	}
	if !supported {
		return nil, fmt.Errorf("ASIC/SDK health event is not supported on the platform")
	}

	queries := [][]string{
		{common.StateDb, asicSdkHealthEventTable, "*"},
	}
	data, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to get ASIC SDK health event data from queries %v, err: %v", queries, err)
		return nil, err
	}

	var events []map[string]interface{}
	for _, key := range common.NatsortInterfaces(common.GetSortedKeys(data)) {
		eventData, ok := data[key].(map[string]interface{})
		if !ok {
			continue
		}

		// The key is the timestamp portion (e.g. "2023-11-22 09:18:12")
		date := key

		events = append(events, map[string]interface{}{
			"date":        date,
			"severity":    common.GetValueOrDefault(eventData, "severity", ""),
			"category":    common.GetValueOrDefault(eventData, "category", ""),
			"description": common.GetValueOrDefault(eventData, "description", ""),
		})
	}

	if events == nil {
		events = []map[string]interface{}{}
	}

	response := map[string]interface{}{
		"events": events,
	}
	return json.Marshal(response)
}


