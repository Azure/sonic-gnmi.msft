package show_client

import (
	"bufio"
	"encoding/json"
	"regexp"
	"strings"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// vtysh command used by the legacy Python CLI: sudo vtysh -c "show ipv6 protocol"
// We run it in the host namespace (PID 1) via nsenter using existing helper.
var (
	vtyshIPv6ProtocolCommand = "vtysh -c \"show ipv6 protocol\""
)

// Output target structure:
// [
//   {"VRF":"default","Protocols":[{"Protocol":"system","route-map":"none"}, ...]},
//   {"VRF":"vrf1",   "Protocols":[...]} , ...
// ]

type ipv6ProtoEntry struct {
	Protocol string `json:"Protocol"`
	RouteMap string `json:"route-map"`
}

type ipv6ProtoVRF struct {
	VRF       string           `json:"VRF"`
	Protocols []ipv6ProtoEntry `json:"Protocols"`
}

var (
	reVRFLine    = regexp.MustCompile(`^VRF:\s*(\S+)`) // VRF: default
	reProtHeader = regexp.MustCompile(`^Protocol\s*:`) // Protocol                  : route-map
	reSepLine    = regexp.MustCompile(`^-{5,}`)        // ----
	reEntryLine  = regexp.MustCompile(`^([A-Za-z0-9_-]+)\s*: *(.+)$`)
)

func getIPv6Protocol(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	rawOutput, err := GetDataFromHostCommand(vtyshIPv6ProtocolCommand)
	if err != nil {
		log.Errorf("Unable to execute command %q, err=%v", vtyshIPv6ProtocolCommand, err)
		return nil, err
	}
	vrfs := parseIPv6ProtocolVRFs(rawOutput)
	b, err := json.Marshal(vrfs)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// parseIPv6ProtocolVRFs parses one or multiple VRF sections of the current FRR
// 'show ipv6 protocol' output.
func parseIPv6ProtocolVRFs(raw string) []ipv6ProtoVRF {
	scanner := bufio.NewScanner(strings.NewReader(raw))
	results := make([]ipv6ProtoVRF, 0)
	var current *ipv6ProtoVRF
	inTable := false
	sawVRF := false

	flush := func() {
		if current != nil {
			results = append(results, *current)
		}
		current = nil
		inTable = false
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if m := reVRFLine.FindStringSubmatch(trimmed); m != nil {
			sawVRF = true
			// Starting a new VRF block
			flush()
			current = &ipv6ProtoVRF{VRF: m[1]}
			continue
		}
		if current == nil {
			// Skip lines until a VRF appears
			continue
		}
		if reProtHeader.MatchString(line) { // header line indicating following lines are entries
			inTable = true
			continue
		}
		if reSepLine.MatchString(trimmed) { // separator
			continue
		}
		if !inTable { // Lines before table body
			continue
		}
		if m := reEntryLine.FindStringSubmatch(line); m != nil {
			protocolName := strings.TrimSpace(m[1])
			routeMap := strings.TrimSpace(m[2])
			current.Protocols = append(current.Protocols, ipv6ProtoEntry{Protocol: protocolName, RouteMap: routeMap})
		}
	}
	flush()

	// If no VRF discovered in non-empty output, treat as empty list (no error path).
	if !sawVRF {
		return []ipv6ProtoVRF{}
	}

	return results
}
