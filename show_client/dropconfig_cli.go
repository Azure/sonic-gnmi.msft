package show_client

import (
	"encoding/json"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// Get the device capabilities from STATE_DB
// admin@sonic: redis-cli -n 6 HGETALL 'DEBUG_COUNTER_CAPABILITIES|PORT_INGRESS_DROPS'
// 1) "reasons"
// 2) "[MPLS_MISS,FDB_AND_BLACKHOLE_DISCARDS,IP_HEADER_ERROR,L3_EGRESS_LINK_DOWN,EXCEEDS_L3_MTU,DIP_LINK_LOCAL,SIP_LINK_LOCAL,ACL_ANY,SMAC_EQUALS_DMAC]"
// 3) "count"
// 4) "10"
func getDropcountersCapabilities(options sdc.OptionMap) ([]byte, error) {
	queries := [][]string{
		{"STATE_DB", "DEBUG_COUNTER_CAPABILITIES", "*"},
	}
	data, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to get debug counter capabilities data from queries %v, got err: %v", queries, err)
		return nil, err
	}

	return json.Marshal(data)
}
