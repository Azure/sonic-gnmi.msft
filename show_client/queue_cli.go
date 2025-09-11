package show_client

import (
	"encoding/json"
	"strings"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

type queueCountersResponse struct {
	Packets               string `json:"Counter/pkts"`
	Bytes                 string `json:"Counter/bytes"`
	DroppedPackets        string `json:"Drop/pkts"`
	DroppedBytes          string `json:"Drop/bytes"`
	TrimmedPackets        string `json:"Trim/pkts"`
	TrimmedSentPackets    string `json:"TrimSent/pkts"`
	TrimmedDroppedPackets string `json:"TrimDrop/pkts"`
	WREDDroppedPackets    string `json:"WredDrp/pkts"`
	WREDDroppedBytes      string `json:"WredDrp/bytes"`
	ECNMarkedPackets      string `json:"EcnMarked/pkts"`
	ECNMarkedBytes        string `json:"EcnMarked/bytes"`
}

type queueCountersResponseNonZero struct {
	Packets               string `json:"Counter/pkts,omitempty"`
	Bytes                 string `json:"Counter/bytes,omitempty"`
	DroppedPackets        string `json:"Drop/pkts,omitempty"`
	DroppedBytes          string `json:"Drop/bytes,omitempty"`
	TrimmedPackets        string `json:"Trim/pkts,omitempty"`
	TrimmedSentPackets    string `json:"TrimSent/pkts,omitempty"`
	TrimmedDroppedPackets string `json:"TrimDrop/pkts,omitempty"`
	WREDDroppedPackets    string `json:"WredDrp/pkts,omitempty"`
	WREDDroppedBytes      string `json:"WredDrp/bytes,omitempty"`
	ECNMarkedPackets      string `json:"EcnMarked/pkts,omitempty"`
	ECNMarkedBytes        string `json:"EcnMarked/bytes,omitempty"`
}

type trimCountersResponse struct {
	TrimmedPackets        string `json:"Trim/pkts"`
	TrimmedSentPackets    string `json:"TrimSent/pkts"`
	TrimmedDroppedPackets string `json:"TrimDrop/pkts"`
}

type trimCountersResponseNonZero struct {
	TrimmedPackets        string `json:"Trim/pkts,omitempty"`
	TrimmedSentPackets    string `json:"TrimSent/pkts,omitempty"`
	TrimmedDroppedPackets string `json:"TrimDrop/pkts,omitempty"`
}

func getQueueCountersMappingNonZero(queueCounters map[string]interface{}, onlyTrim bool) map[string]interface{} {
	response := make(map[string]interface{})
	for queue, counters := range queueCounters {
		if strings.HasSuffix(queue, "periodic") {
			// Ignoring periodic queue watermarks
			continue
		}
		countersMap, ok := counters.(map[string]interface{})
		if !ok {
			log.Warningf("Ignoring invalid counters for the queue '%v': %v", queue, counters)
			continue
		}
		// Only include non-zero counters
		if onlyTrim {
			response[queue] = trimCountersResponseNonZero{
				TrimmedPackets:        GetNonZeroValueOrEmpty(countersMap, "SAI_QUEUE_STAT_TRIM_PACKETS"),
				TrimmedSentPackets:    GetNonZeroValueOrEmpty(countersMap, "SAI_QUEUE_STAT_TX_TRIM_PACKETS"),
				TrimmedDroppedPackets: GetNonZeroValueOrEmpty(countersMap, "SAI_QUEUE_STAT_DROPPED_TRIM_PACKETS"),
			}
		} else {
			response[queue] = queueCountersResponseNonZero{
				Packets:               GetNonZeroValueOrEmpty(countersMap, "SAI_QUEUE_STAT_PACKETS"),
				Bytes:                 GetNonZeroValueOrEmpty(countersMap, "SAI_QUEUE_STAT_BYTES"),
				DroppedPackets:        GetNonZeroValueOrEmpty(countersMap, "SAI_QUEUE_STAT_DROPPED_PACKETS"),
				DroppedBytes:          GetNonZeroValueOrEmpty(countersMap, "SAI_QUEUE_STAT_DROPPED_BYTES"),
				TrimmedPackets:        GetNonZeroValueOrEmpty(countersMap, "SAI_QUEUE_STAT_TRIM_PACKETS"),
				TrimmedSentPackets:    GetNonZeroValueOrEmpty(countersMap, "SAI_QUEUE_STAT_TX_TRIM_PACKETS"),
				TrimmedDroppedPackets: GetNonZeroValueOrEmpty(countersMap, "SAI_QUEUE_STAT_DROPPED_TRIM_PACKETS"),
				WREDDroppedPackets:    GetNonZeroValueOrEmpty(countersMap, "SAI_QUEUE_STAT_WRED_DROPPED_PACKETS"),
				WREDDroppedBytes:      GetNonZeroValueOrEmpty(countersMap, "SAI_QUEUE_STAT_WRED_DROPPED_BYTES"),
				ECNMarkedPackets:      GetNonZeroValueOrEmpty(countersMap, "SAI_QUEUE_STAT_WRED_ECN_MARKED_PACKETS"),
				ECNMarkedBytes:        GetNonZeroValueOrEmpty(countersMap, "SAI_QUEUE_STAT_WRED_ECN_MARKED_BYTES"),
			}
		}
	}
	return response
}

func getQueueCountersMapping(queueCounters map[string]interface{}, onlyTrim bool) map[string]interface{} {
	response := make(map[string]interface{})
	for queue, counters := range queueCounters {
		if strings.HasSuffix(queue, "periodic") {
			// Ignoring periodic queue watermarks
			continue
		}
		countersMap, ok := counters.(map[string]interface{})
		if !ok {
			log.Warningf("Ignoring invalid counters for the queue '%v': %v", queue, counters)
			continue
		}
		if onlyTrim {
			response[queue] = trimCountersResponse{
				TrimmedPackets:        GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_TRIM_PACKETS", defaultMissingCounterValue),
				TrimmedSentPackets:    GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_TX_TRIM_PACKETS", defaultMissingCounterValue),
				TrimmedDroppedPackets: GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_DROPPED_TRIM_PACKETS", defaultMissingCounterValue),
			}
		} else {
			response[queue] = queueCountersResponse{
				Packets:               GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_PACKETS", defaultMissingCounterValue),
				Bytes:                 GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_BYTES", defaultMissingCounterValue),
				DroppedPackets:        GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_DROPPED_PACKETS", defaultMissingCounterValue),
				DroppedBytes:          GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_DROPPED_BYTES", defaultMissingCounterValue),
				TrimmedPackets:        GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_TRIM_PACKETS", defaultMissingCounterValue),
				TrimmedSentPackets:    GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_TX_TRIM_PACKETS", defaultMissingCounterValue),
				TrimmedDroppedPackets: GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_DROPPED_TRIM_PACKETS", defaultMissingCounterValue),
				WREDDroppedPackets:    GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_WRED_DROPPED_PACKETS", defaultMissingCounterValue),
				WREDDroppedBytes:      GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_WRED_DROPPED_BYTES", defaultMissingCounterValue),
				ECNMarkedPackets:      GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_WRED_ECN_MARKED_PACKETS", defaultMissingCounterValue),
				ECNMarkedBytes:        GetValueOrDefault(countersMap, "SAI_QUEUE_STAT_WRED_ECN_MARKED_BYTES", defaultMissingCounterValue),
			}
		}
	}
	return response
}

func getQueueCountersSnapshot(ifaces []string, onlyNonZero bool, onlyTrim bool) (map[string]interface{}, error) {
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

	var response map[string]interface{}
	if onlyNonZero {
		response = getQueueCountersMappingNonZero(queueCounters, onlyTrim)
	} else {
		response = getQueueCountersMapping(queueCounters, onlyTrim)
	}
	return response, nil
}

func getQueueCounters(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	// TODO: cli only supports one interface provided as an argument not optio needs changes

	var ifaces []string
	if interfaces, ok := options["interfaces"].Strings(); ok {
		ifaces = interfaces
	}

	onlyNonZero := false
	if nonzeroOpt, ok := options["nonzero"].Bool(); ok {
		onlyNonZero = nonzeroOpt
	}

	onlyTrim := false
	if trimOpt, ok := options["trim"].Bool(); ok {
		onlyTrim = trimOpt
	}

	snapshot, err := getQueueCountersSnapshot(ifaces, onlyNonZero, onlyTrim)
	if err != nil {
		log.Errorf("Unable to get queue counters due to err: %v", err)
		return nil, err
	}

	return json.Marshal(snapshot)
}
