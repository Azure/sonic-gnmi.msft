package show_client

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// macEntry represents one FDB entry
type macEntry struct {
	Vlan    string `json:"vlan"`
	Port    string `json:"port"`
	MacAddress string `json:"macAddress"`
	Type    string `json:"type"`
}

// getMacTable implements: show mac [-v vlan] [-p port] [-a address] [-t type] [-c]
func getMacTable(options sdc.OptionMap) ([]byte, error) {
	// Build fdbshow command like python show uses
	args := []string{"fdbshow"}
	if v, ok := options["vlan"].Int(); ok {
		args = append(args, "-v", fmt.Sprintf("%d", v))
	}
	if p, ok := options["port"].String(); ok && p != "" {
		args = append(args, "-p", p)
	}
	if a, ok := options["address"].String(); ok && a != "" {
		args = append(args, "-a", a)
	}
	if t, ok := options["type"].String(); ok && t != "" {
		args = append(args, "-t", t)
	}
	countOnly, _ := options["count"].Bool()
	if countOnly {
		args = append(args, "-c")
	}

	cmd := strings.Join(args, " ")
	out, err := GetDataFromHostCommand(cmd)
	if err != nil {
		log.Errorf("fdbshow failed: %v", err)
		return nil, err
	}

	if countOnly {
		n, perr := ParseFdbshowCount(out)
		if perr != nil {
			return nil, perr
		}
		return json.Marshal(map[string]int{"count": n})
	}

	entries, perr := ParseFdbshowTable(out)
	if perr != nil {
		return nil, perr
	}
	return json.Marshal(entries)
}

// Helpers
var fdbshowRowRe = regexp.MustCompile(`^\s*(\d+)\s+(\d+)\s+([0-9A-Fa-f:]{17})\s+(\S+)\s+(Dynamic|Static)\s*$`)
var fdbshowTotalRe = regexp.MustCompile(`(?i)Total\s+number\s+of\s+entries\s+(\d+)`)

// ParseFdbshowCount parses the total count line from fdbshow output.
func ParseFdbshowCount(out string) (int, error) {
	m := fdbshowTotalRe.FindStringSubmatch(out)
	if len(m) < 2 {
		return 0, fmt.Errorf("failed to parse fdbshow count from output")
	}
	var n int
	_, err := fmt.Sscanf(m[1], "%d", &n)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// ParseFdbshowTable parses the table rows from fdbshow output into entries.
func ParseFdbshowTable(out string) ([]macEntry, error) {
	lines := strings.Split(out, "\n")
	entries := make([]macEntry, 0, 64)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip header or separators
		if strings.HasPrefix(line, "No.") || strings.HasPrefix(line, "----") || strings.HasPrefix(strings.ToLower(line), "total number of entries") {
			continue
		}
		m := fdbshowRowRe.FindStringSubmatch(line)
		if len(m) == 6 {
			// m[1]=index, m[2]=vlan, m[3]=mac, m[4]=port, m[5]=type
			entries = append(entries, macEntry{
				Vlan:    m[2],
				MacAddress: strings.ToUpper(m[3]),
				Port:    m[4],
				Type:    m[5],
			})
		}
	}
	return entries, nil
}

