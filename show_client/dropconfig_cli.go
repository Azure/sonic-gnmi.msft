package show_client

import (
	"encoding/json"
	"sort"
	"strings"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// Get the device capabilities from STATE_DB
// admin@sonic: redis-cli -n 6 HGETALL 'DEBUG_COUNTER_CAPABILITIES|PORT_INGRESS_DROPS'
// 1) "reasons"
// 2) "[MPLS_MISS,FDB_AND_BLACKHOLE_DISCARDS,IP_HEADER_ERROR,L3_EGRESS_LINK_DOWN,EXCEEDS_L3_MTU,DIP_LINK_LOCAL,SIP_LINK_LOCAL,ACL_ANY,SMAC_EQUALS_DMAC]"
// 3) "count"
// 4) "10"
func getDropcountersCapabilities(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	queries := [][]string{
		{"STATE_DB", "DEBUG_COUNTER_CAPABILITIES", "*"},
	}
	data, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to get debug counter capabilities data from queries %v, got err: %v", queries, err)
		return nil, err
	}

	return json.Marshal(data)
}

// Get the dropcounters reason configuration from CONFIG_DB
func getDropCountersReasons(counter_name string) []string {
	queries := [][]string{
		{"CONFIG_DB", "DEBUG_COUNTER_DROP_REASON"},
	}
	data, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to get drop counters reasons data from queries %v, got err: %v", queries, err)
		return []string{}
	}

	result := make([]string, 0)
	for key := range data {
		fields := strings.Split(key, "|")
		if len(fields) > 0 && fields[0] == counter_name {
			result = append(result, key)
		}
	}

	return result
}

// Get the dropcounters configuration from CONFIG_DB
func getDropCountersConfiguration(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	group, _ := options["group"].String()

	queries := [][]string{
		{"CONFIG_DB", "DEBUG_COUNTER"},
	}
	config_table, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to get dropcounters configuration data from queries %v, got err: %v", queries, err)
		return nil, err
	}

	// Get a sorted list of counter names
	counter_names := make([]string, 0)
	for key := range config_table {
		counter_names = append(counter_names, key)
	}
	sort.Strings(counter_names)

	// Get the configuration of each counter
	result := make([]map[string]string, 0)
	for i := range counter_names {
		counter_name := counter_names[i]
		counter_attributes := config_table[counter_name].(map[string]interface{})
		if group != "" {
			if counter_attributes["group"] != group {
				continue
			}
		}

		counter_metadata := map[string]string{
			"name":                     counter_name,
			"alias":                    GetValueOrDefault(counter_attributes, "alias", counter_name),
			"group":                    GetValueOrDefault(counter_attributes, "group", "N/A"),
			"type":                     GetValueOrDefault(counter_attributes, "type", "N/A"),
			"description":              GetValueOrDefault(counter_attributes, "desc", "N/A"),
			"drop_monitor_status":      GetValueOrDefault(counter_attributes, "drop_monitor_status", "N/A"),
			"window":                   GetValueOrDefault(counter_attributes, "window", "N/A"),
			"drop_count_threshold":     GetValueOrDefault(counter_attributes, "drop_count_threshold", "N/A"),
			"incident_count_threshold": GetValueOrDefault(counter_attributes, "incident_count_threshold", "N/A"),
		}

		// Fill in the drop reason, concat the reasons with ',' when there are more than 1.
		drop_reasons_keys := getDropCountersReasons(counter_name)
		sort.Strings(drop_reasons_keys)
		num_reasons := len(drop_reasons_keys)
		if num_reasons == 0 {
			counter_metadata["reason"] = "None"
		} else {
			fields := strings.Split(drop_reasons_keys[0], "|")
			counter_metadata["reason"] = fields[1]
			for _, key := range drop_reasons_keys[1:] {
				fields := strings.Split(key, "|")
				counter_metadata["reason"] = counter_metadata["reason"] + "," + fields[1]
			}
		}
		result = append(result, counter_metadata)
	}

	return json.Marshal(result)
}
