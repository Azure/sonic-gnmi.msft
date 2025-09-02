package show_client

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// simple process-local stat lookup cache
var statLookupCache = make(map[string]map[string]string)

// process-local reverse stat lookup cache (objectStatMap -> map[stat]counterName)
var reverseStatLookupCache = make(map[string]map[string]string)

// COUNTERS_DB Tables
const DEBUG_COUNTER_PORT_STAT_MAP = "COUNTERS_DEBUG_NAME_PORT_STAT_MAP"
const DEBUG_COUNTER_SWITCH_STAT_MAP = "COUNTERS_DEBUG_NAME_SWITCH_STAT_MAP"
const COUNTERS_PORT_NAME_MAP = "COUNTERS_PORT_NAME_MAP"
const COUNTER_TABLE_PREFIX = "COUNTERS:"
const SWITCH_LEVEL_COUNTER_PREFIX = "SWITCH_ID"

// ASIC_DB Tables
const ASIC_SWITCH_INFO_PREFIX = "ASIC_STATE:SAI_OBJECT_TYPE_SWITCH:"

// Standard Port-Level Counters
var stdPortRxCounters = []string{"SAI_PORT_STAT_IF_IN_ERRORS", "SAI_PORT_STAT_IF_IN_DISCARDS"}
var stdPortTxCounters = []string{"SAI_PORT_STAT_IF_OUT_ERRORS", "SAI_PORT_STAT_IF_OUT_DISCARDS"}

// Standard Port-Level Headers
var stdPortHeadersMap = map[string]string{
	"SAI_PORT_STAT_IF_IN_ERRORS":    "RX_ERR",
	"SAI_PORT_STAT_IF_IN_DISCARDS":  "RX_DROPS",
	"SAI_PORT_STAT_IF_OUT_ERRORS":   "TX_ERR",
	"SAI_PORT_STAT_IF_OUT_DISCARDS": "TX_DROPS",
}

// Fetch the port-level drop counters as JSON.
// Currently, only port-level drop counts are supported and switch-level drop counts are not supported
// Most VOQ and fabric platforms are multi‑ASIC. Because the current implementation doesn't perform per‑ASIC aggregation, switch‑level counters are not supported yet.
// The switch_type field is empty on FW-backend T0/T1 devices, so implementing switch-level counters isn’t necessary now.
func getDropCounters(options sdc.OptionMap) ([]byte, error) {
	var group string
	var counterType string
	if g, ok := options["group"].String(); ok {
		group = g
	}

	if t, ok := options["counter_type"].String(); ok {
		counterType = t
	}

	if os.Getenv("UTILITIES_UNIT_TESTING_DROPSTAT_CLEAN_CACHE") == "1" {
		// Temp cache needs to be cleard to avoid interference from previous test cases
		if err := os.RemoveAll(getDropstatDir()); err != nil {
			log.V(4).Infof("Failed to remove dropstat cache dir: %v", err)
		}
	}

	// Always refresh <name, stat> maps so config changes are picked up immediately
	ClearDropstatStatCaches()

	// Collect port-level drop counters as JSON-like map and return marshaled JSON
	portMap := showPortDropCounts(group, counterType)
	if portMap != nil {
		jsonBytes, err := json.Marshal(portMap)
		if err != nil {
			log.Errorf("Failed to marshal port drop counters: %v", err)
			return nil, err
		}
		return jsonBytes, nil
	}

	return nil, nil
}

// Gets the drop counts at the port level, if such counts exist.
func showPortDropCounts(group string, counterType string) map[string]map[string]string {
	// Load checkpoint file (port-stats) if present
	portDropCkpt := loadPortDropCheckpoint("port-stats")

	// Build the list of counters to gather: standard + configured
	counters := gatherCounters(append(append([]string{}, stdPortRxCounters...), stdPortTxCounters...), DEBUG_COUNTER_PORT_STAT_MAP, group, counterType)
	if len(counters) == 0 {
		return make(map[string]map[string]string)
	}

	// Build a header map (counter -> alias) and merge standard header mappings
	headmap := gatherHeadersMap(counters, DEBUG_COUNTER_PORT_STAT_MAP)
	for k, v := range stdPortHeadersMap {
		headmap[k] = v
	}

	queries := [][]string{{"APPL_DB", "PORT_TABLE"}}
	portTable, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to pull data for queries %v, got err %v", queries, err)
		return nil
	}

	countsTable, ports := getCountsTable(counters, COUNTERS_PORT_NAME_MAP)
	response := make(map[string]map[string]string, len(ports))
	for _, port := range ports {
		row := make(map[string]string, len(counters)+1)
		state := computeState(port, portTable)
		row["State"] = state

		for _, ctr := range counters {
			diff := getOrDefault(countsTable[port], ctr, int64(0)) - getOrDefault(getOrDefault(portDropCkpt, port, map[string]int64{}), ctr, int64(0))
			alias, ok := headmap[ctr]
			if !ok || alias == "" {
				alias = ctr
			}
			row[alias] = strconv.FormatInt(diff, 10)
		}

		response[port] = row
	}

	return response
}

// Gather the list of counters to be counted, filtering out those that are not in the group or not the right counter type.
// admin@sonic:~$  redis-cli -n 2 HGETALL "COUNTERS_DEBUG_NAME_PORT_STAT_MAP"
// 1) "DEBUG_2"
// 2) "SAI_PORT_STAT_IN_CONFIGURED_DROP_REASONS_0_DROPPED_PKTS"
func gatherCounters(stdCounters []string, objectStatMap string, group string, counterType string) []string {
	// configured counters returns stat names (values)
	configured := getConfiguredCounters(objectStatMap)

	// Concatenate std + configured (preserve std ordering)
	all := append(append([]string{}, stdCounters...), configured...)

	filtered := make([]string, 0, len(all))
	for _, ctr := range all {
		if inGroup(ctr, objectStatMap, group) && isType(ctr, objectStatMap, counterType) {
			filtered = append(filtered, ctr)
		}
	}
	return filtered
}

// build a mapping from counter stat name to header alias.
// Resulting mapping:
//
//	map[string]string{
//	   "SAI_PORT_STAT_IF_IN_ERRORS":    "RX_ERR",    // from stdPortHeadersMap
//	   "SAI_PORT_STAT_IF_IN_DISCARDS":  "RX_DROPS",  // from stdPortHeadersMap
//	   "SAI_PORT_STAT_IN_CONFIGURED_DROP_REASONS_0_DROPPED_PKTS": "BAD_DROPS", // from DEBUG_COUNTER alias
//	}
func gatherHeadersMap(counters []string, objectStatMap string) map[string]string {
	headers := make(map[string]string, len(counters))
	// reverse stat lookup: stat -> counterName
	counterNames := GetReverseStatLookup(objectStatMap)

	for _, counter := range counters {
		if h, ok := stdPortHeadersMap[counter]; ok {
			headers[counter] = h
			continue
		}
		alias := counter
		if counterNames != nil {
			if cn, ok := counterNames[counter]; ok {
				alias = getAlias(cn)
			}
		}
		headers[counter] = alias
	}
	return headers
}

// Get the drop counts for an individual counter.
// admin@sonic:~$  redis-cli -n 2 HGETALL "COUNTERS:oid:0x1000000000002"
//  1. "SAI_PORT_STAT_IN_DROPPED_PKTS"
//  2. "0"
//  3. "SAI_PORT_STAT_OUT_DROPPED_PKTS"
//  4. "0"
func getCounts(counters []string, oid string) map[string]int64 {
	res := make(map[string]int64, len(counters))

	tableId := COUNTER_TABLE_PREFIX + oid
	mapping, err := GetMapFromQueries([][]string{{"COUNTERS_DB", tableId}})
	if err != nil || mapping == nil || len(mapping) == 0 {
		return res
	}

	// mapping should contain field->value entries; coerce values to string then parse
	for _, c := range counters {
		if v, ok := mapping[c]; ok {
			if ival, err := strconv.ParseInt(fmt.Sprint(v), 10, 64); err == nil {
				res[c] = ival
				continue
			}
		}
		res[c] = 0
	}
	return res
}

// Returns a dictionary containing a mapping from an object (like a port) to its drop counts.
// Drop counts are contained in a dictionary that maps counter name to its counts.
func getCountsTable(counters []string, objectTable string) (map[string]map[string]int64, []string) {
	out := make(map[string]map[string]int64)
	mapping, err := GetMapFromQueries([][]string{{"COUNTERS_DB", objectTable}})
	if err != nil || mapping == nil || len(mapping) == 0 {
		log.V(6).Infof("getCountsTable GetMapFromQueries returned err=%v", err)
		return out, nil
	}

	// Build and sort object keys once
	var objs []string
	for k := range mapping {
		objs = append(objs, k)
	}
	sort.Strings(objs)

	// Populate out in any order (map is unordered); use objs when iterating later
	for _, obj := range objs {
		oid := fmt.Sprint(mapping[obj])
		out[obj] = getCounts(counters, oid)
	}
	return out, objs
}

// Retrieves the mapping from counter name -> object stat for the given object type.
// Resulting mapping:
//
//	map[string]string{
//	   "DEBUG_2": "SAI_PORT_STAT_IN_CONFIGURED_DROP_REASONS_0_DROPPED_PKTS",
//	    // ...other configured counters...
//	}
func GetStatLookup(objectStatMap string) map[string]string {
	if v, ok := statLookupCache[objectStatMap]; ok {
		return v
	}

	raw, err := GetMapFromQueries([][]string{{"COUNTERS_DB", objectStatMap}})
	if err != nil || len(raw) == 0 {
		statLookupCache[objectStatMap] = nil
		return nil
	}

	res := make(map[string]string, len(raw))
	for k, val := range raw {
		res[k] = fmt.Sprint(val)
	}

	statLookupCache[objectStatMap] = res
	return res
}

// Retrieves the mapping from object stat ->  counter name for the given object type.
// Resulting mapping:
//
//	map[string]string{
//	  "SAI_PORT_STAT_IN_CONFIGURED_DROP_REASONS_0_DROPPED_PKTS": "DEBUG_2",
//	   // ...other stat->name entries...
//	}
func GetReverseStatLookup(objectStatMap string) map[string]string {
	if v, ok := reverseStatLookupCache[objectStatMap]; ok {
		return v
	}

	statMap := GetStatLookup(objectStatMap)
	if len(statMap) == 0 {
		reverseStatLookupCache[objectStatMap] = nil
		return nil
	}

	rev := make(map[string]string, len(statMap))
	for name, stat := range statMap {
		rev[stat] = name
	}
	reverseStatLookupCache[objectStatMap] = rev
	return rev
}

// Returns the list of counters that have been configured to track packet drops.
func getConfiguredCounters(objectStatMap string) []string {
	counters := GetStatLookup(objectStatMap)

	configuredCounters := []string{}
	if len(counters) == 0 {
		log.V(6).Infof("getConfiguredCounters: no counters found for %s", objectStatMap)
		return configuredCounters
	}

	// Default: return all configured counter stat names (values of the map).
	out := make([]string, 0, len(counters))
	for _, v := range counters {
		out = append(out, v)
	}

	return out
}

// Gets the name of the counter associated with the given counter stat.
func getCounterName(objectStatMap, counterStat string) string {
	lookup := GetReverseStatLookup(objectStatMap)
	if len(lookup) == 0 {
		return ""
	}
	name := lookup[counterStat]
	return name
}

// Gets the alias for the given counter name. If the counter has no alias then the counter name is returned.
func getAlias(counterName string) string {
	aliasQuery, ok := getEntry(counterName)

	if !ok {
		return counterName
	}

	aliasVal := toString(getOrDefault(aliasQuery, "alias", interface{}(counterName)))
	return aliasVal
}

// Checks whether the given counter_stat is part of the given group.
// If no group is provided this method will return True.
// counterStat is the stat string (e.g. "SAI_PORT_STAT_IF_IN_DISCARDS").
func inGroup(counterStat string, objectStatMap string, group string) bool {
	if group == "" {
		return true
	}

	// treat standard counters as not belonging to user groups
	if ContainsString(stdPortRxCounters, counterStat) || ContainsString(stdPortTxCounters, counterStat) {
		return false
	}

	group_query, ok := getEntry(getCounterName(objectStatMap, counterStat))
	if !ok {
		return false
	}

	return group == toString(getOrDefault(group_query, "group", ""))
}

// Checks whether the type of the given counter_stat is the same as counter_type.
// If no counter_type is provided this method will return True.
func isType(counterStat string, objectStatMap string, counterType string) bool {
	if counterType == "" {
		return true
	}

	if ContainsString(stdPortRxCounters, counterStat) {
		return strings.EqualFold(counterType, "PORT_INGRESS_DROPS")
	}
	if ContainsString(stdPortTxCounters, counterStat) {
		return strings.EqualFold(counterType, "PORT_EGRESS_DROPS")
	}

	typeQuery, ok := getEntry(getCounterName(objectStatMap, counterStat))
	if !ok {
		return false
	}

	return counterType == toString(getOrDefault(typeQuery, "type", ""))
}

// getEntry returns the CONFIG_DB DEBUG_COUNTER row for a given counterName.
// admin@sonic:~$ show dropcounters configuration
// Counter    Alias      Group    Type                Reasons           Description
// ---------  ---------  -------  ------------------  ----------------  -----------------------
// DEBUG_2    BAD_DROPS  BAD      PORT_INGRESS_DROPS  ACL_ANY           More port ingress drops
//
// admin@sonic:~$ redis-cli -n 4 HGETALL "DEBUG_COUNTER|DEBUG_2"
// 1) "alias"
// 2) "BAD_DROPS"
// 3) "desc"
// 4) "More port ingress drops"
// 5) "group"
// 6) "BAD"
// 7) "type"
// 8) "PORT_INGRESS_DROPS"
func getEntry(counterName string) (map[string]interface{}, bool) {
	row, err := GetMapFromQueries([][]string{{"CONFIG_DB", "DEBUG_COUNTER", counterName}})
	if err != nil || row == nil || len(row) == 0 {
		return nil, false
	}
	return row, true
}

// getDropstatDir returns per-user cache directory for dropstat checkpoints.
func getDropstatDir() string {
	cache := NewUserCache("dropstat", "")
	return cache.GetDirectory()
}

// loadJSON reads a file and unmarshals JSON into dst. Returns true on success.
func loadJSON(path string, dst interface{}) bool {
	data, err := sdc.ImplIoutilReadFile(path)
	if err != nil {
		return false
	}
	if err := json.Unmarshal(data, dst); err != nil {
		log.Errorf("Failed to unmarshal JSON %s: %v", path, err)
		return false
	}
	return true
}

// loadPortDropCheckpoint reads a JSON checkpoint file (map[port]map[counter]value).
func loadPortDropCheckpoint(fileName string) map[string]map[string]int64 {
	path := filepath.Join(getDropstatDir(), fileName)
	out := make(map[string]map[string]int64)
	if loadJSON(path, &out) {
		return out
	}
	return make(map[string]map[string]int64)
}

// clear in-memory lookup caches.
func ClearDropstatStatCaches() {
	statLookupCache = make(map[string]map[string]string)
	reverseStatLookupCache = make(map[string]map[string]string)
}
