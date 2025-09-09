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
	"bufio"
	"encoding/json"
	"strconv"
	"strings"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// vtysh command used by the legacy Python CLI: sudo vtysh -c "show ipv6 prefix-list"
// We run it in the host namespace (PID 1) via nsenter using existing helper.
var (
	vtyshIPv6PrefixListCommand = "vtysh -c \"show ipv6 prefix-list\""
)

type prefixListEntry struct {
	Seq    int    `json:"seq"`
	Action string `json:"action"`
	Prefix string `json:"prefix,omitempty"`
}

type prefixList struct {
	Name    string            `json:"name"`
	Entries []prefixListEntry `json:"entries"`
}

type sourcePrefixLists struct {
	Source      string       `json:"source"`
	PrefixLists []prefixList `json:"prefix_lists"`
}

func parsePrefixLists(raw string) []sourcePrefixLists {
	scanner := bufio.NewScanner(strings.NewReader(raw))
	results := make([]sourcePrefixLists, 0)
	var currentSource string
	var currentList *prefixList
	sourceMap := make(map[string][]prefixList)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		// Detect source and list name
		if strings.Contains(line, ": ipv6 prefix-list") {
			parts := strings.Split(line, ": ipv6 prefix-list ")
			if len(parts) == 2 {
				currentSource = parts[0]
				listParts := strings.Split(parts[1], ":")
				listName := strings.TrimSpace(listParts[0])
				currentList = &prefixList{Name: listName}
				sourceMap[currentSource] = append(sourceMap[currentSource], *currentList)
			} else {
				log.Errorf("Unexpected format in line: %q", line)
			}
		} else if strings.HasPrefix(line, "seq") {
			// Parse entry line
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				seq, err := strconv.Atoi(fields[1])
				if err != nil {
					log.Errorf("Failed to convert string:%s to int, error: %v", fields[1], err)
					continue
				}

				action := fields[2]
				prefix := ""
				if len(fields) > 3 {
					prefix = fields[3]
				}
				entry := prefixListEntry{Seq: seq, Action: action, Prefix: prefix}
				lastIndex := len(sourceMap[currentSource]) - 1
				sourceMap[currentSource][lastIndex].Entries = append(sourceMap[currentSource][lastIndex].Entries, entry)
			} else {
				log.Errorf("Unexpected format in line: %q", line)
			}
		}
	}

	for source, lists := range sourceMap {
		results = append(results, sourcePrefixLists{
			Source:      source,
			PrefixLists: lists,
		})
	}

	return results
}

func getIPv6PrefixList(options sdc.OptionMap) ([]byte, error) {
	// Filter by prefix-list-name if provided
	prefixListName := ""
	if option, ok := options["prefix_list_name"].String(); ok {
		prefixListName = option
	}

	rawOutput, err := GetDataFromHostCommand(vtyshIPv6PrefixListCommand)
	if err != nil {
		log.Errorf("Unable to execute command %q, err=%v", vtyshIPv6PrefixListCommand, err)
		return nil, err
	}
	prefixLists := parsePrefixLists(rawOutput)
	// If a prefix-list-name filter is provided, apply it to each source's prefix lists
	if prefixListName != "" {
		filtered := make([]sourcePrefixLists, 0)
		for _, srcList := range prefixLists {
			filteredLists := make([]prefixList, 0)
			for _, pl := range srcList.PrefixLists {
				if pl.Name == prefixListName {
					filteredLists = append(filteredLists, pl)
				}
			}
			if len(filteredLists) > 0 {
				filtered = append(filtered, sourcePrefixLists{
					Source:      srcList.Source,
					PrefixLists: filteredLists,
				})
			}
		}
		prefixLists = filtered
	}

	return json.Marshal(prefixLists)
}