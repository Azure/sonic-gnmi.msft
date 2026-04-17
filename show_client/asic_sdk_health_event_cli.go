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

type suppressConfigEntry struct {
	Severity   string `json:"severity"`
	Categories string `json:"categories"`
	MaxEvents  string `json:"max_events"`
}

type healthEventEntry struct {
	Date        string `json:"date"`
	Severity    string `json:"severity"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

// getAsicSdkHealthEventSuppressConfig handles "show asic-sdk-health-event suppress-configuration".
// It reads the SUPPRESS_ASIC_SDK_HEALTH_EVENT table from CONFIG_DB and returns
// JSON with severity entries containing suppressed categories and max_events.
func getAsicSdkHealthEventSuppressConfig(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	supported, err := common.CheckFeatureSupported(common.StateDb, switchCapabilityTable, switchCapabilityKey, asicSdkHealthEventField)
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

	entries := make([]suppressConfigEntry, 0)
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

		entries = append(entries, suppressConfigEntry{
			Severity:   severity,
			Categories: categories,
			MaxEvents:  maxEvents,
		})
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
	supported, err := common.CheckFeatureSupported(common.StateDb, switchCapabilityTable, switchCapabilityKey, asicSdkHealthEventField)
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

	events := make([]healthEventEntry, 0)
	for _, key := range common.NatsortInterfaces(common.GetSortedKeys(data)) {
		eventData, ok := data[key].(map[string]interface{})
		if !ok {
			continue
		}

		// The key is the timestamp portion (e.g. "2023-11-22 09:18:12")
		date := key

		events = append(events, healthEventEntry{
			Date:        date,
			Severity:    common.GetValueOrDefault(eventData, "severity", ""),
			Category:    common.GetValueOrDefault(eventData, "category", ""),
			Description: common.GetValueOrDefault(eventData, "description", ""),
		})
	}

	response := map[string]interface{}{
		"events": events,
	}
	return json.Marshal(response)
}


