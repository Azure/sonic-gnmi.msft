/*
	 show_client/ipv6_prefix-list_cli.go
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
	"errors"
	"io"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
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
	// Optional filter by prefix-list name (default empty means "all")
	prefixListName := args.At(0)

	// Get raw Json output from vtysh command
	rawOutput, err := common.GetDataFromHostCommand(vtyshIPv6PrefixListCommand)
	if err != nil {
		log.Errorf("Unable to execute command %q, err=%v", vtyshIPv6PrefixListCommand, err)
		return nil, err
	}

	decoder := json.NewDecoder(strings.NewReader(rawOutput))

	// Build the final result directly as we decode each JSON block.
	merged := make(prefixListData)

	// Decode JSON output into prefixListData
	for {
		var block prefixListData
		if err := decoder.Decode(&block); err != nil {
			if errors.Is(err, io.EOF) {
				break // clean end of stream
			}
			log.Errorf("Failed to decode IPv6 prefix-list JSON: %v", err)
			return nil, err
		}

		// Merge this block into the final map, with optional filtering by name.
		for proto, lists := range block {
			if prefixListName == "" {
				// No filter: copy all lists; ensure dst exists.
				protocolPrefixLists, ok := merged[proto]
				if !ok {
					protocolPrefixLists = make(map[string]prefixList)
					merged[proto] = protocolPrefixLists
				}
				for name, pl := range lists {
					protocolPrefixLists[name] = pl // last-wins if repeated
				}
			} else {
				// Filtered: only copy the requested list if present; create dst lazily.
				if pl, ok := lists[prefixListName]; ok {
					protocolPrefixLists, ok := merged[proto]
					if !ok {
						protocolPrefixLists = make(map[string]prefixList)
						merged[proto] = protocolPrefixLists
					}
					protocolPrefixLists[prefixListName] = pl // last-wins if repeated
				}
			}
		}
	}

	if prefixListName != "" && len(merged) == 0 {
		log.Infof("Prefix list %q not found in any protocol", prefixListName)
	}

	return json.Marshal(merged)
}
