package show_client

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

const (
	ALL int = iota
	UNICAST
	MULTICAST
)

var countersQueueTypeMap map[string]string = make(map[string]string)

func getQueueUserWatermarksSnapshot(ifaces []string, requestedQueueType int) (map[string]map[string]string, error) {
	var queries [][]string
	if len(ifaces) == 0 {
		// Need queue user watermarks for all interfaces
		queries = append(queries, []string{"COUNTERS_DB", "USER_WATERMARKS", "Ethernet*", "Queues"})
	} else {
		for _, iface := range ifaces {
			queries = append(queries, []string{"COUNTERS_DB", "USER_WATERMARKS", iface, "Queues"})
		}
	}

	queueUserWatermarks, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to pull data for queries %v, got err %v", queries, err)
		return nil, err
	}

	response := make(map[string]map[string]string) // port => queue (e.g., UC0 or MC10) => user watermark
	for queue, userWatermark := range queueUserWatermarks {
		userWatermarkMap, ok := userWatermark.(map[string]interface{})
		if !ok {
			log.Warningf("Ignoring invalid user watermark %v for the queue %v", userWatermark, queue)
			continue
		}
		port_qindex := strings.Split(queue, countersDBSeparator)
		if _, ok := response[port_qindex[0]]; !ok {
			response[port_qindex[0]] = make(map[string]string)
		}
		qtype, ok := countersQueueTypeMap[queue]
		if !ok {
			log.Warningf("Queue %s not found in countersQueueTypeMap.", queue)
			continue
		}
		if requestedQueueType == ALL || (requestedQueueType == UNICAST && qtype == "UC") || (requestedQueueType == MULTICAST && qtype == "MC") {
			response[port_qindex[0]][qtype+port_qindex[1]] = GetValueOrDefault(userWatermarkMap, "SAI_QUEUE_STAT_SHARED_WATERMARK_BYTES", defaultMissingCounterValue)
		}
	}
	return response, nil
}

func getQueueUserWatermarks(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	if len(countersQueueTypeMap) == 0 {
		var err error
		countersQueueTypeMap, err = sdc.GetCountersQueueTypeMap()
		if err != nil {
			log.Errorf("Failed to construct queue-type mapping. err: %v", err)
			return nil, err
		}
	}

	// TODO: Check this option
	var ifaces []string
	if interfaces, ok := options["interfaces"].Strings(); ok {
		ifaces = interfaces
	}

	var queueTypeStr string
	if queueTypeOpt, ok := options["queue-type"].String(); ok {
		queueTypeStr = queueTypeOpt
	}
	var queueType int
	if queueTypeStr == "all" {
		queueType = ALL
	} else if queueTypeStr == "unicast" {
		queueType = UNICAST
	} else if queueTypeStr == "multicast" {
		queueType = MULTICAST
	} else {
		return nil, fmt.Errorf("Invalid queue-type option '%s'. Valid values are 'all', 'unicast', and 'multicast'", queueTypeStr)
	}

	snapshot, err := getQueueUserWatermarksSnapshot(ifaces, queueType)
	if err != nil {
		log.Errorf("Unable to get queue user watermarks due to err: %v", err)
		return nil, err
	}

	return json.Marshal(snapshot)
}
