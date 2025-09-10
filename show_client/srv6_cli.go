package show_client

import (
	"encoding/json"
	"fmt"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

func getSRv6Stats(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	// Get SRv6 statistics per MY_SID entry
	// TODO
	sid := ""
	if option, ok := options["sid"].String(); ok {
		sid = option
	}

	// First, query SID -> Counter OID mapping
	queries := [][]string{
		{"COUNTERS_DB", "COUNTERS_SRV6_NAME_MAP"},
	}
	sidCounterMap, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to pull sid->counter_oid map for queries %v, got err %v", queries, err)
		return nil, err
	}

	if sid != "" {
		if _, ok := sidCounterMap[sid]; !ok {
			log.Errorf("No such sid %s in COUNTERS_SRV6_NAME_MAP", sid)
			return nil, fmt.Errorf("sid %s not found in COUNTERS_SRV6_NAME_MAP", sid)
		}
		sidCounterMap = map[string]interface{}{
			sid: sidCounterMap[sid],
		}
	}

	// Create a slice to hold the keys
	sids := make([]string, 0, len(sidCounterMap))
	// Iterate over the map and collect the keys
	for k, _ := range sidCounterMap {
		sids = append(sids, fmt.Sprintf("%s", k))
	}
	// Natsort the slice
	sids = natsortInterfaces(sids)

	sidCounters := make([]map[string]string, 0, len(sids))
	for _, sid := range sids {
		counterOid := fmt.Sprint(sidCounterMap[sid])
		// Pull statistics for each sid and counterOid pair
		log.V(2).Infof("Processing SID: %s with Counter OID: %v", sid, counterOid)
		queries := [][]string{
			{"COUNTERS_DB", "COUNTERS", counterOid},
		}
		sidStats, err := GetMapFromQueries(queries)
		if err != nil {
			log.Errorf("Unable to pull counters data for queries %v, got err %v", queries, err)
			return nil, err
		}
		if _, ok := sidStats["SAI_COUNTER_STAT_PACKETS"]; !ok {
			sidStats["SAI_COUNTER_STAT_PACKETS"] = "N/A"
		}
		if _, ok := sidStats["SAI_COUNTER_STAT_BYTES"]; !ok {
			sidStats["SAI_COUNTER_STAT_BYTES"] = "N/A"
		}

		sidCounters = append(sidCounters, map[string]string{
			"MySID":   sid,
			"Packets": fmt.Sprintf("%v", sidStats["SAI_COUNTER_STAT_PACKETS"]),
			"Bytes":   fmt.Sprintf("%v", sidStats["SAI_COUNTER_STAT_BYTES"]),
		})
	}

	return json.Marshal(sidCounters)
}
