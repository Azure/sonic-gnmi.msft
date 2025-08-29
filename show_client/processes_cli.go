package show_client

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// processEntry represents a single process row from STATE_DB PROCESS_STATS
// Output JSON keys are required to be PID, PPID, CMD, %MEM, %CPU
type processEntry struct {
	Pid  string `json:"PID"`
	Ppid string `json:"PPID"`
	Cmd  string `json:"CMD"`
	Mem  string `json:"%MEM"`
	Cpu  string `json:"%CPU"`
	Stime string `json:"STIME,omitempty"`
	Time  string `json:"TIME,omitempty"`
	Tt    string `json:"TT,omitempty"`
	Uid   string `json:"UID,omitempty"`
}

// Root help handler: SHOW processes
func getProcessesRoot(options sdc.OptionMap) ([]byte, error) {
	help := map[string]interface{}{
		"subcommands": map[string]string{
			"summary":	"show/processes/summary",
			"cpu":      "show/processes/cpu",
			"mem":      "show/processes/mem",
		},
	}
	return json.Marshal(help)
}

// SHOW processes summary (PID asc)
func getProcessesSummary(options sdc.OptionMap) ([]byte, error) {
	return getProcessesSorted("pid")
}

// SHOW processes cpu (CPU desc)
func getProcessesCpu(options sdc.OptionMap) ([]byte, error) {
	return getProcessesSorted("cpu")
}

// SHOW processes mem (MEM desc)
func getProcessesMem(options sdc.OptionMap) ([]byte, error) {
	return getProcessesSorted("mem")
}

// Shared fetch + sort
func getProcessesSorted(sortKey string) ([]byte, error) {
	queries := [][]string{{"STATE_DB", "PROCESS_STATS"}}
	processesSummary, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to query PROCESS_STATS from queries %v, got err: %v", queries, err)
		return nil, err
	}
	if len(processesSummary) == 0 {
		return []byte("[]"), nil
	}
	entries := buildProcessEntries(processesSummary, sortKey)
	if len(entries) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(entries)
}

func buildProcessEntries(processesSummary map[string]interface{}, sortKey string) []processEntry {
	entries := make([]processEntry, 0, len(processesSummary))
	for key, raw := range processesSummary {
		rec, _ := raw.(map[string]interface{})
		if rec == nil {
			continue
		}

		// Derive PID key: support both "123" and "PROCESS_STATS|123"
		pid := key
		if idx := lastIndexByte(key, '|'); idx >= 0 && idx+1 < len(key) {
			pid = key[idx+1:]
		}
		// Skip non-numeric PIDs (e.g. metadata keys like LastUpdateTime)
		if _, err := strconv.Atoi(pid); err != nil {
			continue
		}

		// Some schemas wrap actual values under "value" object
		if vRaw, ok := rec["value"]; ok {
			if inner, ok2 := vRaw.(map[string]interface{}); ok2 {
				rec = inner
			}
		}

		// Helper accessor with defaults to avoid "<nil>" strings
		get := func(name, def string) string {
			if v, ok := rec[name]; ok && v != nil {
				s := fmt.Sprint(v)
				if s != "<nil>" { // defensive
					return s
				}
			}
			return def
		}

		entries = append(entries, processEntry{
			Pid:  pid,
			Ppid: get("PPID", ""),
			Cmd:  get("CMD", ""),
			Mem:  get("%MEM", "0.0"),
			Cpu:  get("%CPU", "0.0"),
			Stime: get("STIME", ""),
			Time:  get("TIME", ""),
			Tt:    get("TT", ""),
			Uid:   get("UID", ""),
		})
	}
	switch sortKey {
	case "cpu":
		sort.Slice(entries, func(i, j int) bool {
			fi, _ := strconv.ParseFloat(entries[i].Cpu, 64)
			fj, _ := strconv.ParseFloat(entries[j].Cpu, 64)
			if fi == fj {
				pi, _ := strconv.Atoi(entries[i].Pid)
				pj, _ := strconv.Atoi(entries[j].Pid)
				return pi < pj
			}
			return fi > fj
		})
	case "mem":
		sort.Slice(entries, func(i, j int) bool {
			fi, _ := strconv.ParseFloat(entries[i].Mem, 64)
			fj, _ := strconv.ParseFloat(entries[j].Mem, 64)
			if fi == fj {
				pi, _ := strconv.Atoi(entries[i].Pid)
				pj, _ := strconv.Atoi(entries[j].Pid)
				return pi < pj
			}
			return fi > fj
		})
	case "pid":
		sort.Slice(entries, func(i, j int) bool {
			pi, _ := strconv.Atoi(entries[i].Pid)
			pj, _ := strconv.Atoi(entries[j].Pid)
			return pi < pj
		})
	default:
	}
	return entries
}

// lastIndexByte is a tiny helper to avoid importing strings for a single use.
func lastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}
