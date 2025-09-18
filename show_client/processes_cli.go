package show_client

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// processEntry for STATE_DB PROCESS_STATS
type processEntry struct {
	Pid   string `json:"PID"`
	Ppid  string `json:"PPID"`
	Cmd   string `json:"CMD"`
	Mem   string `json:"%MEM"`
	Cpu   string `json:"%CPU"`
	Stime string `json:"STIME,omitempty"`
	Time  string `json:"TIME,omitempty"`
	Tt    string `json:"TT,omitempty"`
	Uid   string `json:"UID,omitempty"`
	// pidNum caches the numeric PID for sorting
	pidNum int `json:"-"`
}

func getProcessesRoot(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	help := map[string]interface{}{
		"subcommands": map[string]string{
			"summary": "show/processes/summary: Show processses info",
			"cpu":     "show/processes/cpu: Show processes CPU info",
			"memory":  "show/processes/memory: Show processes information sorted by memory usage",
		},
	}
	return json.Marshal(help)
}

func getProcessesSummary(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	queries := [][]string{{"STATE_DB", "PROCESS_STATS"}}
	processesSummary, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to query PROCESS_STATS from queries %v, got err: %v", queries, err)
		return nil, err
	}
	entries := buildProcessEntries(processesSummary)
	return json.Marshal(entries)
}

func buildProcessEntries(processesSummary map[string]interface{}) []processEntry {
	entries := make([]processEntry, 0, len(processesSummary))
	for key, raw := range processesSummary {
		rec, ok := raw.(map[string]interface{})
		if !ok || rec == nil {
			continue
		}

		pid := key
		if idx := strings.LastIndexByte(key, '|'); idx >= 0 && idx+1 < len(key) {
			pid = key[idx+1:]
		}

		pn, err := strconv.Atoi(pid)
		if err != nil {
			continue
		}

		if vRaw, ok := rec["value"]; ok {
			if inner, ok2 := vRaw.(map[string]interface{}); ok2 && inner != nil {
				rec = inner
			}
		}

		// Helper accessor: return string value if present & non-empty, else default.
		get := func(name, def string) string {
			if v, ok := rec[name]; ok && v != nil {
				s := fmt.Sprint(v)
				if s != "" {
					return s
				}
			}
			return def
		}

		entries = append(entries, processEntry{
			Pid:    pid,
			Ppid:   get("PPID", ""),
			Cmd:    get("CMD", ""),
			Mem:    get("%MEM", "0.0"),
			Cpu:    get("%CPU", "0.0"),
			Stime:  get("STIME", ""),
			Time:   get("TIME", ""),
			Tt:     get("TT", ""),
			Uid:    get("UID", ""),
			pidNum: pn,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].pidNum < entries[j].pidNum
	})
	return entries
}
