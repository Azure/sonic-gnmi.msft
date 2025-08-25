package show_client

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

/*
admin@sonic:~$ show mac aging-time
Aging time for switch is 600 seconds
admin@sonic:~$ redis-cli -n 0 hget "SWITCH_TABLE:switch" "fdb_aging_time"
"600"
*/

func getMacAgingTime(options sdc.OptionMap) ([]byte, error) {
	queries := [][]string{
		{"APPL_DB", "SWITCH_TABLE", "switch"},
	}
	data, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to get mac aging time data from queries %v, got err: %v", queries, err)
		return nil, err
	}
	log.V(6).Infof("GetMapFromQueries result: %+v", data)

	// Default value if not found
	agingTime := "N/A"

	if val, ok := data["fdb_aging_time"]; ok && val != nil {
		strVal := fmt.Sprintf("%v", val)
		if strVal != "" {
			agingTime = strVal + "s"
		} else {
			log.Warningf("Key 'fdb_aging_time' found but empty in data")
		}
	} else {
		log.Warningf("Key 'fdb_aging_time' not found or empty in data")
	}

	// Build response, append "s" for seconds
	result := map[string]string{
		"fdb_aging_time": agingTime,
	}
	return json.Marshal(result)
}

// macEntry represents a single FDB entry
type macEntry struct {
	Vlan       string `json:"vlan"`
	MacAddress string `json:"macAddress"`
	Port       string `json:"port"`
	Type       string `json:"type"`
}

// getMacTable queries APPL_DB and STATE_DB FDB_TABLE entries and returns either the list or count per options
func getMacTable(options sdc.OptionMap) ([]byte, error) {
	// Parse filters
	vlanFilter := -1
	if v, ok := options["vlan"].Int(); ok {
		vlanFilter = v
	}
	portFilter, _ := options["port"].String()
	addrFilter, _ := options["address"].String()
	typeFilter, _ := options["type"].String()
	wantCount, _ := options["count"].Bool()

	stateData, err := GetMapFromQueries([][]string{{StateDb, FDBTable}})
	if err != nil {
		log.Errorf("Unable to get STATE_DB FDB_TABLE, err: %v", err)
		return nil, err
	}

	// Prefer APPL_DB entries on duplicates; track seen keys "vlan|mac"
	seen := make(map[string]struct{})
	entries := make([]macEntry, 0, len(stateData))

	// Check if portFilter is valid
	portIsValid := false
	if portFilter == "" {
		portIsValid = true
	} else {
		for _, v := range stateData {
			fv, ok := v.(map[string]interface{})
			if !ok {
				continue
			}
			port := toString(fv["port"])
			if strings.EqualFold(port, portFilter) {
				portIsValid = true
				break
			}
		}
	}

	if !portIsValid {
		return nil, errors.New("Invalid port " + portFilter)
	}

	// Check if typeFilter is valid
	if typeFilter != "" && (strings.ToLower(typeFilter) != "static" && strings.ToLower(typeFilter) != "dynamic") {
		return nil, errors.New("Invalid type " + typeFilter)
	}

	addIfMatch := func(vlan, macAddress, port, mtype string) {
		// Filters
		if vlanFilter >= 0 && vlan != fmt.Sprint(vlanFilter) {
			return
		}
		if portFilter != "" && !strings.EqualFold(port, portFilter) {
			return
		}
		if addrFilter != "" && !strings.EqualFold(strings.ToLower(addrFilter), strings.ToLower(macAddress)) {
			return
		}
		if typeFilter != "" && strings.ToLower(typeFilter) != strings.ToLower(mtype) {
			return
		}
		key := vlan + "|" + strings.ToLower(macAddress)
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		entries = append(entries, macEntry{
			Vlan:       vlan,
			MacAddress: macAddress,
			Port:       port,
			Type:       strings.ToLower(mtype),
		})
	}

	processFDBData(stateData, StateDb, addIfMatch)

	if !portIsValid {
		return nil, errors.New("Invalid port " + portFilter)
	}

	if wantCount {
		resp := map[string]int{"count": len(entries)}
		return json.Marshal(resp)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Vlan == entries[j].Vlan {
			return strings.ToLower(entries[i].MacAddress) < strings.ToLower(entries[j].MacAddress)
		}
		return entries[i].Vlan < entries[j].Vlan
	})
	return json.Marshal(entries)
}

func processFDBData(data map[string]interface{}, source string, addIfMatch func(string, string, string, string)) {
	for k, v := range data {
		fv, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		vlan, mac, ok := parseKey(k)
		if !ok {
			continue
		}
		addIfMatch(vlan, mac, toString(fv["port"]), toString(fv["type"]))
	}
}

func parseKey(k string) (vlan string, mac string, ok bool) {
	idx := strings.Index(k, ":")
	if idx <= 0 || idx >= len(k)-1 {
		return "", "", false
	}
	vlan = strings.TrimPrefix(k[:idx], "Vlan")
	mac = k[idx+1:]
	return vlan, mac, true
}
