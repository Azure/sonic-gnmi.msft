package show_client

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// pfcCountersRxResponse represents the RX PFC counters for a single port.
type pfcCountersRxResponse struct {
	PFC0 string `json:"PFC0"`
	PFC1 string `json:"PFC1"`
	PFC2 string `json:"PFC2"`
	PFC3 string `json:"PFC3"`
	PFC4 string `json:"PFC4"`
	PFC5 string `json:"PFC5"`
	PFC6 string `json:"PFC6"`
	PFC7 string `json:"PFC7"`
}

// pfcCountersTxResponse represents the TX PFC counters for a single port.
type pfcCountersTxResponse struct {
	PFC0 string `json:"PFC0"`
	PFC1 string `json:"PFC1"`
	PFC2 string `json:"PFC2"`
	PFC3 string `json:"PFC3"`
	PFC4 string `json:"PFC4"`
	PFC5 string `json:"PFC5"`
	PFC6 string `json:"PFC6"`
	PFC7 string `json:"PFC7"`
}

// pfcCountersFullResponse wraps both RX and TX counters.
type pfcCountersFullResponse struct {
	Rx map[string]pfcCountersRxResponse `json:"rx"`
	Tx map[string]pfcCountersTxResponse `json:"tx"`
}

// pfcAsymmetricResponse represents the asymmetric PFC status for a single port.
type pfcAsymmetricResponse struct {
	Asymmetric string `json:"Asymmetric"`
}

// pfcPriorityResponse represents the lossless priorities for a single port.
type pfcPriorityResponse struct {
	LosslessPriorities string `json:"Lossless priorities"`
}

// getPfcCounters fetches PFC RX and TX counters for all ports from COUNTERS_DB.
// Uses COUNTERS_PORT_NAME_MAP to resolve port names to OIDs, then fetches
// COUNTERS:<oid> for each port. Corresponds to "show pfc counters".
func getPfcCounters(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	// Step 1: Fetch COUNTERS_PORT_NAME_MAP to get port name -> OID mapping
	portNameMap, err := common.GetMapFromQueries([][]string{{"COUNTERS_DB", "COUNTERS_PORT_NAME_MAP"}})
	if err != nil {
		log.Errorf("Unable to pull COUNTERS_PORT_NAME_MAP from COUNTERS_DB, err: %v", err)
		return nil, err
	}

	// Step 2: For each port, fetch its counters by OID
	portCounters := make(map[string]interface{})
	for port, oidVal := range portNameMap {
		oid := fmt.Sprint(oidVal)
		tableKey := "COUNTERS:" + oid
		counters, err := common.GetMapFromQueries([][]string{{"COUNTERS_DB", tableKey}})
		if err != nil {
			log.Errorf("Unable to pull counters for %s (oid %s), err: %v", port, oid, err)
			continue
		}
		if len(counters) > 0 {
			portCounters[port] = interface{}(counters)
		}
	}

	rxCounters := make(map[string]pfcCountersRxResponse)
	txCounters := make(map[string]pfcCountersTxResponse)

	for port := range portCounters {
		rxCounters[port] = pfcCountersRxResponse{
			PFC0: common.GetFieldValueString(portCounters, port, common.DefaultMissingCounterValue, "SAI_PORT_STAT_PFC_0_RX_PKTS"),
			PFC1: common.GetFieldValueString(portCounters, port, common.DefaultMissingCounterValue, "SAI_PORT_STAT_PFC_1_RX_PKTS"),
			PFC2: common.GetFieldValueString(portCounters, port, common.DefaultMissingCounterValue, "SAI_PORT_STAT_PFC_2_RX_PKTS"),
			PFC3: common.GetFieldValueString(portCounters, port, common.DefaultMissingCounterValue, "SAI_PORT_STAT_PFC_3_RX_PKTS"),
			PFC4: common.GetFieldValueString(portCounters, port, common.DefaultMissingCounterValue, "SAI_PORT_STAT_PFC_4_RX_PKTS"),
			PFC5: common.GetFieldValueString(portCounters, port, common.DefaultMissingCounterValue, "SAI_PORT_STAT_PFC_5_RX_PKTS"),
			PFC6: common.GetFieldValueString(portCounters, port, common.DefaultMissingCounterValue, "SAI_PORT_STAT_PFC_6_RX_PKTS"),
			PFC7: common.GetFieldValueString(portCounters, port, common.DefaultMissingCounterValue, "SAI_PORT_STAT_PFC_7_RX_PKTS"),
		}

		txCounters[port] = pfcCountersTxResponse{
			PFC0: common.GetFieldValueString(portCounters, port, common.DefaultMissingCounterValue, "SAI_PORT_STAT_PFC_0_TX_PKTS"),
			PFC1: common.GetFieldValueString(portCounters, port, common.DefaultMissingCounterValue, "SAI_PORT_STAT_PFC_1_TX_PKTS"),
			PFC2: common.GetFieldValueString(portCounters, port, common.DefaultMissingCounterValue, "SAI_PORT_STAT_PFC_2_TX_PKTS"),
			PFC3: common.GetFieldValueString(portCounters, port, common.DefaultMissingCounterValue, "SAI_PORT_STAT_PFC_3_TX_PKTS"),
			PFC4: common.GetFieldValueString(portCounters, port, common.DefaultMissingCounterValue, "SAI_PORT_STAT_PFC_4_TX_PKTS"),
			PFC5: common.GetFieldValueString(portCounters, port, common.DefaultMissingCounterValue, "SAI_PORT_STAT_PFC_5_TX_PKTS"),
			PFC6: common.GetFieldValueString(portCounters, port, common.DefaultMissingCounterValue, "SAI_PORT_STAT_PFC_6_TX_PKTS"),
			PFC7: common.GetFieldValueString(portCounters, port, common.DefaultMissingCounterValue, "SAI_PORT_STAT_PFC_7_TX_PKTS"),
		}
	}

	response := pfcCountersFullResponse{
		Rx: rxCounters,
		Tx: txCounters,
	}

	return json.Marshal(response)
}

// getPfcAsymmetric fetches pfc_asym field from CONFIG_DB PORT table for each Ethernet port.
// Corresponds to "show pfc asymmetric".
func getPfcAsymmetric(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	interfaceName := args.At(0)

	queries := [][]string{
		{"CONFIG_DB", "PORT"},
	}

	portData, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to pull PORT data from CONFIG_DB, err: %v", err)
		return nil, err
	}

	response := make(map[string]pfcAsymmetricResponse)

	for port, entry := range portData {
		if interfaceName != "" && port != interfaceName {
			continue
		}

		if !strings.HasPrefix(port, "Ethernet") {
			continue
		}

		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}

		pfcAsym := common.GetValueOrDefault(entryMap, "pfc_asym", "N/A")
		response[port] = pfcAsymmetricResponse{
			Asymmetric: pfcAsym,
		}
	}

	return json.Marshal(response)
}

// getPfcPriority fetches pfc_enable field from CONFIG_DB PORT_QOS_MAP table for each interface.
// Corresponds to "show pfc priority".
func getPfcPriority(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	interfaceName := args.At(0)

	queries := [][]string{
		{"CONFIG_DB", "PORT_QOS_MAP"},
	}

	portQosData, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to pull PORT_QOS_MAP data from CONFIG_DB, err: %v", err)
		return nil, err
	}

	if interfaceName != "" {
		if _, exists := portQosData[interfaceName]; !exists {
			return nil, fmt.Errorf("Cannot find interface %s", interfaceName)
		}
	}

	response := make(map[string]pfcPriorityResponse)

	for intf, entry := range portQosData {
		if interfaceName != "" && intf != interfaceName {
			continue
		}

		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}

		pfcEnable := common.GetValueOrDefault(entryMap, "pfc_enable", "N/A")
		response[intf] = pfcPriorityResponse{
			LosslessPriorities: pfcEnable,
		}
	}

	return json.Marshal(response)
}

