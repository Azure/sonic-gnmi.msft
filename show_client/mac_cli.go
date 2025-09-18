package show_client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

/*
admin@sonic:~$ show mac aging-time
Aging time for switch is 600 seconds
admin@sonic:~$ redis-cli -n 0 hget "SWITCH_TABLE:switch" "fdb_aging_time"
"600"
*/

func getMacAgingTime(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	queries := [][]string{
		{"APPL_DB", "SWITCH_TABLE", "switch"},
	}
	data, err := common.GetMapFromQueries(queries)
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
	Vlan       int    `json:"vlan"`
	MacAddress string `json:"macAddress"`
	Port       string `json:"port"`
	Type       string `json:"type"`
}

// getMacTable queries STATE_DB FDB_TABLE entries and returns either the list or count per options
func getMacTable(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	// Parse filters
	vlanFilter := -1
	if v, ok := options["vlan"].Int(); ok {
		vlanFilter = v
	}
	portFilter, _ := options["port"].String()
	addrFilter, _ := options["address"].String()
	typeFilter, _ := options["type"].String()
	wantCount, _ := options["count"].Bool()

	// Check vlanFilter is valid
	if vlanFilter != -1 {
		if vlanFilter < 1 || vlanFilter > 4095 {
			return nil, errors.New("Error: Invalid vlan " + fmt.Sprint(vlanFilter))
		}
	}

	// Check if typeFilter is valid
	if typeFilter != "" && (strings.ToLower(typeFilter) != "static" && strings.ToLower(typeFilter) != "dynamic") {
		return nil, errors.New("Error: Invalid type " + typeFilter)
	}

	// Check mac address format is valid
	if addrFilter != "" {
		_, err := net.ParseMAC(addrFilter)
		if err != nil {
			return nil, errors.New("Error: Invalid mac address " + addrFilter)
		}
	}

	stateData, err := common.GetMapFromQueries([][]string{{common.StateDb, common.FDBTable}})
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
		allPorts, err := common.GetMapFromQueries([][]string{{common.ConfigDb, common.ConfigDBPortTable}})
		if err != nil {
			log.Errorf("Unable to get CONFIG_DB port, err: %v", err)
			return nil, err
		}

		for port, _ := range allPorts {
			if strings.EqualFold(port, portFilter) {
				portIsValid = true
				break
			}
		}
	}

	if !portIsValid {
		return nil, errors.New("Error: Invalid port " + portFilter)
	}

	addIfMatch := func(vlan int, macAddress, port, mtype string) {
		// Filters
		if vlanFilter >= 0 && vlan != vlanFilter {
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
		key := fmt.Sprint(vlan, "|", strings.ToLower(macAddress))
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

	processFDBData(stateData, addIfMatch)

	if wantCount {
		resp := map[string]int{"total": len(entries)}
		return json.Marshal(resp)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Vlan == entries[j].Vlan {
			return strings.ToLower(entries[i].MacAddress) < strings.ToLower(entries[j].MacAddress)
		}
		return entries[i].Vlan < entries[j].Vlan
	})

	resp := map[string]interface{}{"entries": entries, "total": len(entries)}
	return json.Marshal(resp)
}

func processFDBData(data map[string]interface{}, addIfMatch func(int, string, string, string)) {
	for k, v := range data {
		fv, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		vlan, mac, ok := parseFDBTableKey(k)
		if !ok {
			continue
		}
		addIfMatch(vlan, mac, fmt.Sprint(fv["port"]), fmt.Sprint(fv["type"]))
	}
}

func parseFDBTableKey(k string) (vlan int, mac string, ok bool) {
	idx := strings.Index(k, ":")
	if idx <= 0 || idx >= len(k)-1 {
		return -1, "", false
	}
	vlanStr := strings.TrimPrefix(k[:idx], "Vlan")
	vlan, err := strconv.Atoi(vlanStr)
	if err != nil {
		return -1, "", false
	}
	mac = k[idx+1:]
	return vlan, mac, true
}
