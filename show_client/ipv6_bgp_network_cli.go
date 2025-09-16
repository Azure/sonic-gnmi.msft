package show_client

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// getIPv6BGPNetwork implements: show ipv6 bgp network [ip|prefix] [bestpath|json|longer-prefixes|multipath]
// Behavior mirrors Python CLI:
// - Without ip/prefix, it runs "show bgp ipv6" (text).
// - With ip/prefix, appends argument; if info_type provided, append it.
// - Reject "longer-prefixes" when ip is a host address (no '/').
// - If info_type == json, the FRR output is JSON; pass through as JSON bytes.
// - Otherwise, wrap plain text output as {"output":"..."} to keep JSON transport.
func getIPv6BGPNetwork(args sdc.CmdArgs, _ sdc.OptionMap) ([]byte, error) {
	ipArg := args.At(0)
	infoType := args.At(1)

	// Validate infoType choices similar to Click Choice
	if infoType != "" {
		switch infoType {
		case "bestpath", "json", "longer-prefixes", "multipath":
			// ok
		default:
			return nil, errors.New(fmt.Sprintf("invalid info_type %s; must be one of: bestpath|json|longer-prefixes|multipath", infoType))
		}
	}

	// Enforce longer-prefixes only with a prefix containing '/'
	if ipArg != "" && !strings.Contains(ipArg, "/") && infoType == "longer-prefixes" {
		return nil, errors.New(fmt.Sprintf("The parameter option: \"%s\" only available if passing a network prefix", infoType))
	}

	// Build vtysh command
	cmd := "rvtysh -c \"show bgp ipv6"
	if ipArg != "" {
		cmd += " " + ipArg
		if infoType != "" {
			cmd += " " + infoType
		}
	}
	cmd += "\""

	rawOutput, err := GetDataFromHostCommand(cmd)
	if err != nil {
		log.Errorf("Unable to execute command %q, output=%q err=%v", cmd, rawOutput, err)
		return nil, err
	}

	// If infoType == json, FRR prints JSON; return as-is
	if infoType == "json" {
		// Validate it's JSON; if not, still return raw but log warning
		var js json.RawMessage
		if err := json.Unmarshal([]byte(rawOutput), &js); err == nil {
			return []byte(rawOutput), nil
		}
		log.Warningf("Expected JSON from FRR but failed to parse; returning raw text wrapped")
	}

	// Wrap plain text output
	resp := map[string]string{"output": strings.TrimRight(rawOutput, "\n")}
	return json.Marshal(resp)
}
