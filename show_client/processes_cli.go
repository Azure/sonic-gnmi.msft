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
	for pid, raw := range processesSummary {
		rec, _ := raw.(map[string]interface{})
		if rec == nil {
			continue
		}
		get := func(name string) string { return fmt.Sprint(rec[name]) }
		entries = append(entries, processEntry{
			Pid:  pid,
			Ppid: get("PPID"),
			Cmd:  get("CMD"),
			Mem:  get("%MEM"),
			Cpu:  get("%CPU"),
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
		fallthrough
	default:
		sort.Slice(entries, func(i, j int) bool {
			pi, _ := strconv.Atoi(entries[i].Pid)
			pj, _ := strconv.Atoi(entries[j].Pid)
			return pi < pj
		})
	}
	return entries
}
