/* show_client/ipv6_prefix-list_cli.go
 * This file contains the implementation of the 'show ipv6 prefix-list' command for the Sonic gNMI client.

    Example output of 'show ipv6 prefix-list' command:

        admin@sonic:~$ show ipv6 prefix-list
		ZEBRA: ipv6 prefix-list DEFAULT_IPV6: 2 entries
		seq 5 permit any
		seq 10 permit any
		BGP: ipv6 prefix-list DEFAULT_IPV6: 2 entries
		seq 5 permit any
		seq 10 permit any
		BGP: ipv6 prefix-list LOCAL_VLAN_IPV6_PREFIX: 1 entries
		seq 5 permit <IPv6-address>/64
		BGP: ipv6 prefix-list PL_LoopbackV6: 1 entries
		seq 5 permit <IPv6-address>/64
*/
package show_client

import (
	"encoding/json"
	"strings"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// vtysh command used by the legacy Python CLI: sudo vtysh -c "show ipv6 prefix-list"
// We run it in the host namespace (PID 1) via nsenter using existing helper.
var (
	vtyshIPv6PrefixListCommand = "vtysh -c \"show ipv6 prefix-list json\""
)

type prefixListEntry struct {
    SequenceNumber      int    `json:"sequenceNumber"`
    Type                string `json:"type"`
    Prefix              string `json:"prefix"`
    MaximumPrefixLength int    `json:"maximumPrefixLength,omitempty"`
}

type prefixList struct {
    AddressFamily string            `json:"addressFamily"`
    Entries       []prefixListEntry `json:"entries"`
}

// Top-level structure: map of protocol -> map of list name -> prefixList
type prefixListData map[string]map[string]prefixList

func getIPv6PrefixList(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	// Get prefix-list_namefrom args, if provided, default to ""
	prefixListName := args.At(0)

	// get raw Json output from vtysh command
	rawOutput, err := GetDataFromHostCommand(vtyshIPv6PrefixListCommand)
	if err != nil {
		log.Errorf("Unable to execute command %q, err=%v", vtyshIPv6PrefixListCommand, err)
		return nil, err
	}

	decoder := json.NewDecoder(strings.NewReader(rawOutput))

	// Decode JSON output into prefixListData
	var blocks []prefixListData
	for {
		var pl prefixListData
		if err := decoder.Decode(&pl); err != nil {
			break // End of input or error
		}
		blocks = append(blocks, pl)
	}

	// merge parsed blocks
	merged := make(prefixListData)
	for _, block := range blocks {
		for proto, lists := range block {
			if _, exists := merged[proto]; !exists {
				merged[proto] = make(map[string]prefixList)
			}
			for name, pl := range lists {
				merged[proto][name] = pl
			}
		}
	}

	// If a specific prefix-list name is requested, filter the results
	if prefixListName != "" {
		filtered := make(prefixListData)
		for proto, lists := range merged {
			for name, pl := range lists {
				if name == prefixListName {
					if _, exists := filtered[proto]; !exists {
						filtered[proto] = make(map[string]prefixList)
					}
					filtered[proto][name] = pl
				}
			}
		}
		merged = filtered
	}

	// If no data found after filtering, return empty JSON array
	if len(merged) == 0 {
		empty, err := json.Marshal([]interface{}{})
		if err != nil {
			log.Errorf("Failed to marshal empty result: %v", err)
			return nil, err
		}
		return empty, nil
	}

	// marshal merged data back to JSON for downstream parsing
	mergedJSON, err := json.Marshal(merged)
	if err != nil {
		log.Errorf("Failed to marshal merged IPv6 prefix-list data: %v", err)
		return nil, err
	}

	return mergedJSON, nil
}