package show_client

import (
	"net"
	"sort"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
)

const fibRouteTable = "ROUTE_TABLE"

// It's usable for IPv4 / IPv6
// Python source code: https://github.com/Azure/sonic-utilities.msft/blob/3fb3258806c25b8d60a255ce0508dcd20018bdc6/scripts/fibshow
type fibEntry struct {
	Index   int    `json:"index"`
	Vrf     string `json:"vrf,omitempty"`
	Route   string `json:"route"`
	NextHop string `json:"nexthop,omitempty"`
	IfName  string `json:"ifname,omitempty"`
}

type fibResult struct {
	Total   int        `json:"total"`
	Entries []fibEntry `json:"entries"`
}

func getFibEntries(filter string, wantIPv6 bool) ([]fibEntry, error) {
	queries := [][]string{{common.ApplDb, fibRouteTable}}
	msi, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("[show fib] failed to query %s: %v", fibRouteTable, err)
		return nil, err
	}

	out := make([]fibEntry, 0, len(msi))
	for rawKey, rowAny := range msi {
		row, ok := rowAny.(map[string]interface{})
		if !ok {
			continue
		}
		vrf, prefix := parseFibVrf(rawKey)
		if !matchIPFamily(prefix, wantIPv6) {
			continue
		}
		if filter != "" && filter != prefix && filter != rawKey {
			continue
		}
		nh := common.GetValueOrDefault(row, "nexthop", "")
		ifn := common.GetValueOrDefault(row, "ifname", "")
		out = append(out, fibEntry{
			Vrf:     vrf,
			Route:   prefix,
			NextHop: nh,
			IfName:  ifn,
		})
	}

	// Same as python https://github.com/Azure/sonic-utilities.msft/blob/3fb3258806c25b8d60a255ce0508dcd20018bdc6/scripts/fibshow#L88C8-L88C53
	// sort by route and update the Index of fibEntry
	sort.Slice(out, func(i, j int) bool { return out[i].Route < out[j].Route })
	for i := range out {
		out[i].Index = i + 1
	}
	return out, nil
}

// parseFibVrf supports forms: VRF-<Name>:<prefix> or <prefix>
// https://github.com/Azure/sonic-utilities.msft/blob/3fb3258806c25b8d60a255ce0508dcd20018bdc6/scripts/fibshow#L100C13-L104C25
func parseFibVrf(key string) (string, string) {
	if strings.HasPrefix(key, "VRF-") {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) == 2 {
			return strings.TrimPrefix(parts[0], "VRF-"), parts[1]
		}
	}
	return "", key
}

// matchIPFamily checks if prefix belongs to desired family
func matchIPFamily(prefix string, wantIPv6 bool) bool {
	host := prefix
	if i := strings.IndexByte(prefix, '/'); i >= 0 {
		host = prefix[:i]
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	isV6 := ip.To4() == nil
	return isV6 == wantIPv6
}
