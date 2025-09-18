package show_client

import (
	"encoding/json"
	"strings"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// https://github.com/Azure/sonic-utilities.msft/blob/master/scripts/fibshow
// For command 'show ipv6 fib'  arg: ipv6address
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
func getIPv6Fib(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {

	filter := strings.TrimSpace(args.At(0))
	entries, err := getFibEntries(filter, true) // true -> IPv6
	if err != nil {
		return nil, err
	}

	log.Infof("[show ipv6 fib]|Found %d entries", len(entries))
	res := fibResult{
		Total:   len(entries),
		Entries: entries,
	}
	return json.Marshal(res)
}
