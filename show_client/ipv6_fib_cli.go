package show_client

import (
	"encoding/json"
	"sort"
	"strings"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

const (
	applRouteTable = "ROUTE_TABLE"
)

// ipv6FibEntry represents one IPv6 FIB row.
type ipv6FibEntry struct {
	Index   int    `json:"index"`
	Vrf     string `json:"vrf,omitempty"`
	Route   string `json:"route"`
	NextHop string `json:"nexthop,omitempty"`
	IfName  string `json:"ifname,omitempty"`
}

type ipv6FibResult struct {
	Total   int            `json:"total"`
	Entries []ipv6FibEntry `json:"entries"`
}

// https://github.com/Azure/sonic-utilities.msft/blob/master/scripts/fibshow
// For command 'show ipv6 fib'  otption: ipv6address
// :~$ show ipv6 fib
//
//	No.  Vrf    Route           Nexthop    Ifname
//
// -----  -----  --------------  ---------  ---------
//
//	1         fc00:1::/64     ::         Loopback0
//	2         fc00:1::32      ::         Loopback0
//	3         fc02:1000::/64  ::         Vlan1000
//
// Total number of entries 3
func getIPv6Fib(options sdc.OptionMap) ([]byte, error) {

	var filter string
	if ov, ok := options[OptionKeyIpAddress]; ok {
		if v, ok2 := ov.String(); ok2 {
			filter = strings.TrimSpace(v)
		}
	}

	// Query ROUTE_TABLE
	queries := [][]string{{ApplDb, applRouteTable}}
	msi, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("[show ipv6 fib]|Failed to read %s: %v", applRouteTable, err)
		return nil, err
	}

	entries := make([]ipv6FibEntry, 0, len(msi))
	idx := 1
	for rawKey, rowAny := range msi {
		row, ok := rowAny.(map[string]interface{})
		if !ok {
			log.Warningf("[show ipv6 fib]|Skip unexpected row type key=%s val=%T", rawKey, rowAny)
			continue
		}

		vrf, prefix := parseFibVrf(rawKey)

		// Keep only IPv6
		if !strings.Contains(prefix, ":") {
			continue
		}

		// Exact filter (match prefix or original key)
		if filter != "" && filter != prefix && filter != rawKey {
			continue
		}

		nhStr := GetValueOrDefault(row, "nexthop", "")
		ifStr := GetValueOrDefault(row, "ifname", "")

		entries = append(entries, ipv6FibEntry{
			Index:   idx,
			Vrf:     vrf,
			Route:   prefix,
			NextHop: nhStr,
			IfName:  ifStr,
		})
		idx++
	}

	// Same as python https://github.com/Azure/sonic-utilities.msft/blob/3fb3258806c25b8d60a255ce0508dcd20018bdc6/scripts/fibshow#L88C8-L88C53
	// sort by route and update the Index of ipv6FibEntry
	sort.Slice(entries, func(i, j int) bool { return entries[i].Route < entries[j].Route })
	for i := range entries {
		entries[i].Index = i + 1
	}

	log.Infof("[show ipv6 fib]|Found %d entries", len(entries))
	res := ipv6FibResult{
		Total:   len(entries),
		Entries: entries,
	}

	return json.Marshal(res)
}

// https://github.com/Azure/sonic-utilities.msft/blob/3fb3258806c25b8d60a255ce0508dcd20018bdc6/scripts/fibshow#L100C13-L104C25
// parseFibVrf supports key forms： 1.VRF-<Name>:<prefix> 2.<prefix>
func parseFibVrf(key string) (string, string) {
	if strings.HasPrefix(key, "VRF-") {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) == 2 {
			return strings.TrimPrefix(parts[0], "VRF-"), parts[1]
		}
	}
	return "", key
}
