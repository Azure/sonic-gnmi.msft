package show_client

import (
	"encoding/json"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
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

	queueUserWatermarks, err := common.GetMapFromQueries(queries)
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
			response[port_qindex[0]][qtype+port_qindex[1]] = common.GetValueOrDefault(userWatermarkMap, "SAI_QUEUE_STAT_SHARED_WATERMARK_BYTES", common.DefaultMissingCounterValue)
		}
	}
	return response, nil
}

func getQueueUserWatermarksCommon(options sdc.OptionMap, requestedQueueType int) ([]byte, error) {
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

	snapshot, err := getQueueUserWatermarksSnapshot(ifaces, requestedQueueType)
	if err != nil {
		log.Errorf("Unable to get queue user watermarks due to err: %v", err)
		return nil, err
	}

	return json.Marshal(snapshot)
}

func getQueueUserWatermarks(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	help := map[string]interface{}{
		"subcommands": map[string]string{
			"all":       "show/queue/watermark/all",
			"unicast":   "show/queue/watermark/unicast",
			"multicast": "show/queue/watermark/multicast",
		},
	}
	return json.Marshal(help)
}

func getQueueUserWatermarksAll(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	return getQueueUserWatermarksCommon(options, ALL)
}

func getQueueUserWatermarksUnicast(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	return getQueueUserWatermarksCommon(options, UNICAST)
}

func getQueueUserWatermarksMulticast(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	return getQueueUserWatermarksCommon(options, MULTICAST)
}
