package show_client

import (
	"encoding/json"
	"strings"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

type queueCountersResponse struct {
	Packets            string `json:"Counter/pkts"`
	Bytes              string `json:"Counter/bytes"`
	DroppedPackets     string `json:"Drop/pkts"`
	DroppedBytes       string `json:"Drop/bytes"`
	TrimmedPackets     string `json:"Trim/pkts"`
	WREDDroppedPackets string `json:"WredDrp/pkts"`
	WREDDroppedBytes   string `json:"WredDrp/bytes"`
	ECNMarkedPackets   string `json:"EcnMarked/pkts"`
	ECNMarkedBytes     string `json:"EcnMarked/bytes"`
}

func getQueueCountersSnapshot(ifaces []string) (map[string]queueCountersResponse, error) {
	var queries [][]string
	if len(ifaces) == 0 {
		// Need queue counters for all interfaces
		queries = append(queries, []string{"COUNTERS_DB", "COUNTERS", "Ethernet*", "Queues"})
	} else {
		for _, iface := range ifaces {
			queries = append(queries, []string{"COUNTERS_DB", "COUNTERS", iface, "Queues"})
		}
	}

	queryMap, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to pull data for queries %v, got err %v", queries, err)
		return nil, err
	}

	queueCounters := RemapAliasToPortNameForQueues(queryMap)

	response := make(map[string]queueCountersResponse)
	for queue, counters := range queueCounters {
		if strings.HasSuffix(queue, ":periodic") {
			// Ignoring periodic queue watermarks
			continue
		}
		countersMap, ok := counters.(map[string]interface{})
		if !ok {
			log.Warningf("Ignoring invalid counters for the queue '%v': %v", queue, counters)
			continue
		}
		response[queue] = queueCountersResponse{
			Packets:            GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_PACKETS", defaultMissingCounterValue),
			Bytes:              GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_BYTES", defaultMissingCounterValue),
			DroppedPackets:     GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_DROPPED_PACKETS", defaultMissingCounterValue),
			DroppedBytes:       GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_DROPPED_BYTES", defaultMissingCounterValue),
			TrimmedPackets:     GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_TRIM_PACKETS", defaultMissingCounterValue),
			WREDDroppedPackets: GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_WRED_DROPPED_PACKETS", defaultMissingCounterValue),
			WREDDroppedBytes:   GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_WRED_DROPPED_BYTES", defaultMissingCounterValue),
			ECNMarkedPackets:   GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_WRED_ECN_MARKED_PACKETS", defaultMissingCounterValue),
			ECNMarkedBytes:     GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_WRED_ECN_MARKED_BYTES", defaultMissingCounterValue),
		}
	}
	return response, nil
}

func getQueueCounters(options sdc.OptionMap) ([]byte, error) {
	var ifaces []string

	if interfaces, ok := options["interfaces"].Strings(); ok {
		ifaces = interfaces
	}

	snapshot, err := getQueueCountersSnapshot(ifaces)
	if err != nil {
		log.Errorf("Unable to get queue counters due to err: %v", err)
		return nil, err
	}

	return json.Marshal(snapshot)
}
