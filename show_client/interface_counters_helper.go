package show_client

import (
	"encoding/json"
	"fmt"
	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
	"strconv"
	"time"
)

const (
	fecBinCount       = 16 // BIN index from 0 - 15
	defaultTimestamp  = "None"
	portStatCachePath = "/tmp/cache/portstat/1000/portstat"
)

type InterfaceCountersSnapshot struct { // json fields defined from portstat cache
	// Port Status
	State string `json:"-"`
	// Port Counters
	RxOk  string `json:"rx_ok"`
	RxErr string `json:"rx_err"`
	RxDrp string `json:"rx_drop"`
	RxOvr string `json:"rx_ovr"`
	TxOk  string `json:"tx_ok"`
	TxErr string `json:"tx_err"`
	TxDrp string `json:"tx_drop"`
	TxOvr string `json:"tx_ovr"`
	// Port Rates
	RxBps  string `json:"-"`
	RxPps  string `json:"-"`
	RxUtil string `json:"-"`
	TxBps  string `json:"-"`
	TxPps  string `json:"-"`
	TxUtil string `json:"-"`
	// FEC counters
	FecCorr      string `json:"fec_corr"`
	FecUncorr    string `json:"fec_uncorr"`
	FecSymbolErr string `json:"fec_symbol_err"`
	FecPreBer    string `json:"-"`
	FecPostBer   string `json:"-"`
	// Trim Counters
	TrimPkts string `json:"trim"`
	TrimSent string `json:"-"`
	TrimDrp  string `json:"-"`
	// Detailed Counters for Octets
	Rx64         string `json:"rx_64"`
	Rx65_127     string `json:"rx_65_127"`
	Rx128_255    string `json:"rx_128_255"`
	Rx256_511    string `json:"rx_256_511"`
	Rx512_1023   string `json:"rx_512_1023"`
	Rx1024_1518  string `json:"rx_1024_1518"`
	Rx1519_2047  string `json:"rx_1519_2047"`
	Rx2048_4095  string `json:"rx_2048_4095"`
	Rx4096_9216  string `json:"rx_4096_9216"`
	Rx9217_16383 string `json:"rx_9217_16383"`
	Tx64         string `json:"tx_64"`
	Tx65_127     string `json:"tx_65_127"`
	Tx128_255    string `json:"tx_128_255"`
	Tx256_511    string `json:"tx_256_511"`
	Tx512_1023   string `json:"tx_512_1023"`
	Tx1024_1518  string `json:"tx_1024_1518"`
	Tx1519_2047  string `json:"tx_1519_2047"`
	Tx2048_4095  string `json:"tx_2048_4095"`
	Tx4096_9216  string `json:"tx_4096_9216"`
	Tx9217_16383 string `json:"tx_9217_16383"`
	// Detailed Counters
	RxAll       string `json:"rx_all"`
	RxUnicast   string `json:"rx_uca"`
	RxMulticast string `json:"rx_mca"`
	RxBroadcast string `json:"rx_bca"`
	TxAll       string `json:"tx_all"`
	TxUnicast   string `json:"tx_uca"`
	TxMulticast string `json:"tx_mca"`
	TxBroadcast string `json:"tx_bca"`
	RxJabbers   string `json:"rx_jbr"`
	RxFragments string `json:"rx_frag"`
	RxUndersize string `json:"rx_usize"`
	RxOverruns  string `json:"rx_ovrrun"`
	// FEC Codewords per symbol error index (not in cache JSON)
	FecErrCWs []FecErrCW `json:"-"`
	// Timestamp Cleared Counters
	TimestampClearedCounters string `json:"-"`
}

type FecErrCW struct {
	BinIndex  string
	Codewords string
}

type InterfaceCountersResponse struct {
	State  string
	RxOk   string
	RxBps  string
	RxUtil string
	RxErr  string
	RxDrp  string
	RxOvr  string
	TxOk   string
	TxBps  string
	TxUtil string
	TxErr  string
	TxDrp  string
	TxOvr  string
}

type InterfaceCountersAllResponse struct {
	State    string
	RxOk     string
	RxBps    string
	RxPps    string
	RxUtil   string
	RxErr    string
	RxDrp    string
	RxOvr    string
	TxOk     string
	TxBps    string
	TxPps    string
	TxUtil   string
	TxErr    string
	TxDrp    string
	TxOvr    string
	TrimPkts string
	TrimSent string
	TrimDrp  string
}

type InterfaceCountersErrorsResponse struct {
	State string
	RxErr string
	RxDrp string
	RxOvr string
	TxErr string
	TxDrp string
	TxOvr string
}

type InterfaceCountersRatesResponse struct {
	State  string
	RxOk   string
	RxBps  string
	RxPps  string
	RxUtil string
	TxOk   string
	TxBps  string
	TxPps  string
	TxUtil string
}

type InterfaceCountersTrimResponse struct {
	State    string
	TrimPkts string
	TrimSent string
	TrimDrp  string
}

type InterfaceCountersFecStatsResponse struct {
	State        string
	FecCorr      string
	FecUncorr    string
	FecSymbolErr string
	FecPreBer    string
	FecPostBer   string
}

type InterfaceCountersDetailedResponse struct {
	TrimPkts     string
	TrimSent     string
	TrimDrp      string
	Rx64         string
	Rx65_127     string
	Rx128_255    string
	Rx256_511    string
	Rx512_1023   string
	Rx1024_1518  string
	Rx1519_2047  string
	Rx2048_4095  string
	Rx4096_9216  string
	Rx9217_16383 string
	Tx64         string
	Tx65_127     string
	Tx128_255    string
	Tx256_511    string
	Tx512_1023   string
	Tx1024_1518  string
	Tx1519_2047  string
	Tx2048_4095  string
	Tx4096_9216  string
	Tx9217_16383 string
	RxAll        string
	RxUnicast    string
	RxMulticast  string
	RxBroadcast  string
	TxAll        string
	TxUnicast    string
	TxMulticast  string
	TxBroadcast  string
	RxJabbers    string
	RxFragments  string
	RxUndersize  string
	RxOverruns   string
	// Field has to be fetched from host as not in DB
	TimestampClearedCounters string
}

type interfaceRifCounters struct {
	RxOkPackets  string `json:"RxOkPackets"`
	RxBps        string `json:"RxBps"`
	RxPps        string `json:"RxPps"`
	RxErrPackets string `json:"RxErrPackets"`
	TxOkPackets  string `json:"TxOkPackets"`
	TxBps        string `json:"TxBps"`
	TxPps        string `json:"TxPps"`
	TxErrPackets string `json:"TxErrPackets"`
	RxErrBits    string `json:"RxErrBits"`
	TxErrBits    string `json:"TxErrBits"`
	RxOkBits     string `json:"RxOkBits"`
	TxOkBits     string `json:"TxOkBits"`
}

func validatePeriod(options sdc.OptionMap) (period int, takeDiffSnapshot bool, err error) {
	if periodValue, ok := options["period"].Int(); ok {
		takeDiffSnapshot = true
		period = periodValue
	}
	if period > common.MaxShowCommandPeriod || period < 0 {
		err = fmt.Errorf("period value must be <= %v and non negative", common.MaxShowCommandPeriod)
		return
	}
	return
}

func snapshotWithOptionalDiff(ifaces []string, period int, takeDiffSnapshot bool) (map[string]InterfaceCountersSnapshot, error) {
	oldSnapshot, err := getInterfaceCountersSnapshot(ifaces)
	if err != nil {
		log.Errorf("Unable to get interfaces counter snapshot due to err: %v", err)
		return nil, err
	}

	if takeDiffSnapshot && period > 0 {
		time.Sleep(time.Duration(period) * time.Second)
		newSnapshot, err := getInterfaceCountersSnapshot(ifaces)
		if err != nil {
			log.Errorf("Unable to get new interface counters snapshot due to err %v", err)
			return nil, err
		}
		return calculateDiffSnapshot(oldSnapshot, newSnapshot), nil
	}
	return oldSnapshot, nil
}

func getInterfaceCountersSnapshot(ifaces []string) (map[string]InterfaceCountersSnapshot, error) {
	queries := [][]string{
		{"COUNTERS_DB", "COUNTERS", "Ethernet*"},
	}

	aliasCountersOutput, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to pull data for queries %v, got err %v", queries, err)
		return nil, err
	}

	portCounters := common.RemapAliasToPortName(aliasCountersOutput)

	queries = [][]string{
		{"COUNTERS_DB", "RATES", "Ethernet*"},
	}

	aliasRatesOutput, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to pull data for queries %v, got err %v", queries, err)
		return nil, err
	}

	portRates := common.RemapAliasToPortName(aliasRatesOutput)

	queries = [][]string{
		{"APPL_DB", "PORT_TABLE"},
	}

	portTable, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to pull data for queries %v, got err %v", queries, err)
		return nil, err
	}

	queries = [][]string{
		{"STATE_DB", "PORT_TABLE"},
	}

	statePortTable, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to pull data from queries %v, got err %v", queries, err)
		statePortTable = map[string]interface{}{} // used for port speed, will default to APPL_DB if failure
	}

	validatedIfaces := []string{}

	if len(ifaces) == 0 {
		for port, _ := range portCounters {
			validatedIfaces = append(validatedIfaces, port)
		}
	} else { // Validate
		for _, iface := range ifaces {
			_, found := portCounters[iface]
			if found { // Drop none valid interfaces
				validatedIfaces = append(validatedIfaces, iface)
			}
		}
	}

	response := make(map[string]InterfaceCountersSnapshot, len(validatedIfaces))

	for _, iface := range validatedIfaces {
		state := computeState(iface, portTable)
		portSpeed := computeSpeed(iface, statePortTable, portTable)
		rxBps := common.GetFieldValueString(portRates, iface, common.DefaultMissingCounterValue, "RX_BPS")
		txBps := common.GetFieldValueString(portRates, iface, common.DefaultMissingCounterValue, "TX_BPS")
		rxPps := common.GetFieldValueString(portRates, iface, common.DefaultMissingCounterValue, "RX_PPS")
		txPps := common.GetFieldValueString(portRates, iface, common.DefaultMissingCounterValue, "TX_PPS")
		rxUtil := common.GetFieldValueString(portRates, iface, common.DefaultMissingCounterValue, "RX_UTIL")
		txUtil := common.GetFieldValueString(portRates, iface, common.DefaultMissingCounterValue, "TX_UTIL")
		preBer := common.GetFieldValueString(portRates, iface, common.DefaultMissingCounterValue, "FEC_PRE_BER")
		postBer := common.GetFieldValueString(portRates, iface, common.DefaultMissingCounterValue, "FEC_POST_BER")

		snapshot := InterfaceCountersSnapshot{
			State:        state,
			RxOk:         common.GetSumFields(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_UCAST_PKTS", "SAI_PORT_STAT_IF_IN_NON_UCAST_PKTS"),
			RxBps:        calculateByteRate(rxBps),
			RxPps:        calculatePacketRate(rxPps),
			RxUtil:       computeUtil(rxUtil, rxBps, portSpeed),
			RxErr:        common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_ERRORS"),
			RxDrp:        common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_DISCARDS"),
			RxOvr:        common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_RX_OVERSIZE_PKTS"),
			TxOk:         common.GetSumFields(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_IF_OUT_UCAST_PKTS", "SAI_PORT_STAT_IF_OUT_NON_UCAST_PKTS"),
			TxBps:        calculateByteRate(txBps),
			TxPps:        calculatePacketRate(txPps),
			TxUtil:       computeUtil(txUtil, txBps, portSpeed),
			TxErr:        common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_IF_OUT_ERRORS"),
			TxDrp:        common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_IF_OUT_DISCARDS"),
			TxOvr:        common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_TX_OVERSIZE_PKTS"),
			FecCorr:      common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_FEC_CORRECTABLE_FRAMES"),
			FecUncorr:    common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_FEC_NOT_CORRECTABLE_FRAMES"),
			FecSymbolErr: common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_FEC_SYMBOL_ERRORS"),
			FecPreBer:    calculateBerRate(preBer),
			FecPostBer:   calculateBerRate(postBer),
			TrimPkts:     common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_TRIM_PKTS"),
			TrimSent:     common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_TX_TRIM_SENT_PKTS"),
			TrimDrp:      common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_DROPPED_TRIM_PKTS"),
			Rx64:         common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_IN_PKTS_64_OCTETS"),
			Rx65_127:     common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_IN_PKTS_65_TO_127_OCTETS"),
			Rx128_255:    common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_IN_PKTS_128_TO_255_OCTETS"),
			Rx256_511:    common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_IN_PKTS_256_TO_511_OCTETS"),
			Rx512_1023:   common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_IN_PKTS_512_TO_1023_OCTETS"),
			Rx1024_1518:  common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_IN_PKTS_1024_TO_1518_OCTETS"),
			Rx1519_2047:  common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_IN_PKTS_1519_TO_2047_OCTETS"),
			Rx2048_4095:  common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_IN_PKTS_2048_TO_4095_OCTETS"),
			Rx4096_9216:  common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_IN_PKTS_4096_TO_9216_OCTETS"),
			Rx9217_16383: common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_IN_PKTS_9217_TO_16383_OCTETS"),
			Tx64:         common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_OUT_PKTS_64_OCTETS"),
			Tx65_127:     common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_OUT_PKTS_65_TO_127_OCTETS"),
			Tx128_255:    common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_OUT_PKTS_128_TO_255_OCTETS"),
			Tx256_511:    common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_OUT_PKTS_256_TO_511_OCTETS"),
			Tx512_1023:   common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_OUT_PKTS_512_TO_1023_OCTETS"),
			Tx1024_1518:  common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_OUT_PKTS_1024_TO_1518_OCTETS"),
			Tx1519_2047:  common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_OUT_PKTS_1519_TO_2047_OCTETS"),
			Tx2048_4095:  common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_OUT_PKTS_2048_TO_4095_OCTETS"),
			Tx4096_9216:  common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_OUT_PKTS_4096_TO_9216_OCTETS"),
			Tx9217_16383: common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_OUT_PKTS_9217_TO_16383_OCTETS"),
			RxAll:        common.GetSumFields(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_UCAST_PKTS", "SAI_PORT_STAT_IF_IN_MULTICAST_PKTS", "SAI_PORT_STAT_IF_IN_BROADCAST_PKTS"),
			RxUnicast:    common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_UCAST_PKTS"),
			RxMulticast:  common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_MULTICAST_PKTS"),
			RxBroadcast:  common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_BROADCAST_PKTS"),
			TxAll:        common.GetSumFields(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_IF_OUT_UCAST_PKTS", "SAI_PORT_STAT_IF_OUT_MULTICAST_PKTS", "SAI_PORT_STAT_IF_OUT_BROADCAST_PKTS"),
			TxUnicast:    common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_IF_OUT_UCAST_PKTS"),
			TxMulticast:  common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_IF_OUT_MULTICAST_PKTS"),
			TxBroadcast:  common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_IF_OUT_BROADCAST_PKTS"),
			RxJabbers:    common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_STATS_JABBERS"),
			RxFragments:  common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_STATS_FRAGMENTS"),
			RxUndersize:  common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_ETHER_STATS_UNDERSIZE_PKTS"),
			RxOverruns:   common.GetFieldValueString(portCounters, iface, common.DefaultMissingCounterValue, "SAI_PORT_STAT_IP_IN_RECEIVES"),
		}

		fecErrCWs := make([]FecErrCW, 0, fecBinCount)
		for i := 0; i < fecBinCount; i++ {
			binIndex := fmt.Sprintf("BIN%d", i)
			fecCodewordsKey := fmt.Sprintf("SAI_PORT_STAT_IF_IN_FEC_CODEWORD_ERRORS_S%d", i)
			fecCodewordsValue := common.GetFieldValueString(portCounters, iface, "0", fecCodewordsKey)
			entry := FecErrCW{
				BinIndex:  binIndex,
				Codewords: fecCodewordsValue,
			}
			fecErrCWs = append(fecErrCWs, entry)
		}
		snapshot.FecErrCWs = fecErrCWs
		snapshot.TimestampClearedCounters = defaultTimestamp
		response[iface] = snapshot
	}
	if cacheSnapshot, ok := getPortStatCacheSnapshot(); ok { // if cache exists then we provide current counters as a diff
		return calculateDiffSnapshot(cacheSnapshot, response), nil
	}
	return response, nil
}

func getInterfaceCountersRifSnapshot(interfaceName string) (map[string]interfaceRifCounters, error) {
	rifNameMap, err := getRifNameMapping()
	if err != nil {
		return nil, fmt.Errorf("Failed to get COUNTERS_RIF_NAME_MAP: %v", err)
	}

	queries := [][]string{
		{common.CountersDb, "COUNTERS"},
	}

	rifCountersMap, err := common.GetMapFromQueries(queries)
	if err != nil {
		return nil, fmt.Errorf("Unable to pull data for queries %v, got err %v", queries, err)
	}

	queries = [][]string{
		{common.CountersDb, "RATES:*"},
	}

	rifRatesMap, err := common.GetMapFromQueries(queries)
	if err != nil {
		return nil, fmt.Errorf("Unable to pull data for queries %v, got err %v", queries, err)
	}

	interfaceRifCountersMap := make(map[string]interfaceRifCounters, len(rifNameMap))
	for rifName, oid := range rifNameMap {
		if interfaceName != "" && rifName != interfaceName {
			continue
		}

		oidStr, ok := oid.(string)
		if !ok {
			log.Warningf("Invalid OID for RIF %s: %v", rifName, oid)
			continue
		}

		if oidStr == "" {
			log.Warningf("Empty OID for RIF %s", rifName)
			continue
		}

		interfaceRifCounter := interfaceRifCounters{
			RxOkPackets:  validateAndGetIntValue(common.GetFieldValueString(rifCountersMap, oidStr, common.DefaultMissingCounterValue, "SAI_ROUTER_INTERFACE_STAT_IN_PACKETS")),
			RxBps:        common.GetFieldValueString(rifRatesMap, oidStr, common.DefaultMissingCounterValue, "RX_BPS"),
			RxPps:        common.GetFieldValueString(rifRatesMap, oidStr, common.DefaultMissingCounterValue, "RX_PPS"),
			RxErrPackets: validateAndGetIntValue(common.GetFieldValueString(rifCountersMap, oidStr, common.DefaultMissingCounterValue, "SAI_ROUTER_INTERFACE_STAT_IN_ERROR_PACKETS")),
			TxOkPackets:  validateAndGetIntValue(common.GetFieldValueString(rifCountersMap, oidStr, common.DefaultMissingCounterValue, "SAI_ROUTER_INTERFACE_STAT_OUT_PACKETS")),
			TxBps:        common.GetFieldValueString(rifRatesMap, oidStr, common.DefaultMissingCounterValue, "TX_BPS"),
			TxPps:        common.GetFieldValueString(rifRatesMap, oidStr, common.DefaultMissingCounterValue, "TX_PPS"),
			TxErrPackets: validateAndGetIntValue(common.GetFieldValueString(rifCountersMap, oidStr, common.DefaultMissingCounterValue, "SAI_ROUTER_INTERFACE_STAT_OUT_ERROR_PACKETS")),
			RxErrBits:    validateAndGetIntValue(common.GetFieldValueString(rifCountersMap, oidStr, common.DefaultMissingCounterValue, "SAI_ROUTER_INTERFACE_STAT_IN_ERROR_OCTETS")),
			TxErrBits:    validateAndGetIntValue(common.GetFieldValueString(rifCountersMap, oidStr, common.DefaultMissingCounterValue, "SAI_ROUTER_INTERFACE_STAT_OUT_ERROR_OCTETS")),
			RxOkBits:     validateAndGetIntValue(common.GetFieldValueString(rifCountersMap, oidStr, common.DefaultMissingCounterValue, "SAI_ROUTER_INTERFACE_STAT_IN_OCTETS")),
			TxOkBits:     validateAndGetIntValue(common.GetFieldValueString(rifCountersMap, oidStr, common.DefaultMissingCounterValue, "SAI_ROUTER_INTERFACE_STAT_OUT_OCTETS")),
		}

		interfaceRifCountersMap[rifName] = interfaceRifCounter
	}

	return interfaceRifCountersMap, nil
}

func calculateDiffSnapshot(oldSnapshot map[string]InterfaceCountersSnapshot, newSnapshot map[string]InterfaceCountersSnapshot) map[string]InterfaceCountersSnapshot {
	diffResponse := make(map[string]InterfaceCountersSnapshot, len(newSnapshot))

	for iface, newResp := range newSnapshot {
		oldResp, found := oldSnapshot[iface]
		if !found {
			log.Errorf("Previous snapshot not found for intf %v when diffing interface counters snapshot", iface)
			diffResponse[iface] = newResp
			continue
		}
		diffResponse[iface] = InterfaceCountersSnapshot{
			State:                    newResp.State,
			RxOk:                     calculateDiff(oldResp.RxOk, newResp.RxOk, false),
			RxErr:                    calculateDiff(oldResp.RxErr, newResp.RxErr, false),
			RxDrp:                    calculateDiff(oldResp.RxDrp, newResp.RxDrp, false),
			RxOvr:                    calculateDiff(oldResp.RxOvr, newResp.RxOvr, false),
			TxOk:                     calculateDiff(oldResp.TxOk, newResp.TxOk, false),
			TxErr:                    calculateDiff(oldResp.TxErr, newResp.TxErr, false),
			TxDrp:                    calculateDiff(oldResp.TxDrp, newResp.TxDrp, false),
			TxOvr:                    calculateDiff(oldResp.TxOvr, newResp.TxOvr, false),
			RxBps:                    newResp.RxBps,
			RxPps:                    newResp.RxPps,
			RxUtil:                   newResp.RxUtil,
			TxBps:                    newResp.TxBps,
			TxPps:                    newResp.TxPps,
			TxUtil:                   newResp.TxUtil,
			FecCorr:                  calculateDiff(oldResp.FecCorr, newResp.FecCorr, false),
			FecUncorr:                calculateDiff(oldResp.FecUncorr, newResp.FecUncorr, false),
			FecSymbolErr:             calculateDiff(oldResp.FecSymbolErr, newResp.FecSymbolErr, false),
			FecPreBer:                newResp.FecPreBer,
			FecPostBer:               newResp.FecPostBer,
			TrimPkts:                 calculateDiff(oldResp.TrimPkts, newResp.TrimPkts, false),
			TrimSent:                 calculateDiff(oldResp.TrimSent, newResp.TrimSent, false),
			TrimDrp:                  calculateDiff(oldResp.TrimDrp, newResp.TrimDrp, true),
			Rx64:                     calculateDiff(oldResp.Rx64, newResp.Rx64, false),
			Rx65_127:                 calculateDiff(oldResp.Rx65_127, newResp.Rx65_127, false),
			Rx128_255:                calculateDiff(oldResp.Rx128_255, newResp.Rx128_255, false),
			Rx256_511:                calculateDiff(oldResp.Rx256_511, newResp.Rx256_511, false),
			Rx512_1023:               calculateDiff(oldResp.Rx512_1023, newResp.Rx512_1023, false),
			Rx1024_1518:              calculateDiff(oldResp.Rx1024_1518, newResp.Rx1024_1518, false),
			Rx1519_2047:              calculateDiff(oldResp.Rx1519_2047, newResp.Rx1519_2047, false),
			Rx2048_4095:              calculateDiff(oldResp.Rx2048_4095, newResp.Rx2048_4095, false),
			Rx4096_9216:              calculateDiff(oldResp.Rx4096_9216, newResp.Rx4096_9216, false),
			Rx9217_16383:             calculateDiff(oldResp.Rx9217_16383, newResp.Rx9217_16383, false),
			Tx64:                     calculateDiff(oldResp.Tx64, newResp.Tx64, false),
			Tx65_127:                 calculateDiff(oldResp.Tx65_127, newResp.Tx65_127, false),
			Tx128_255:                calculateDiff(oldResp.Tx128_255, newResp.Tx128_255, false),
			Tx256_511:                calculateDiff(oldResp.Tx256_511, newResp.Tx256_511, false),
			Tx512_1023:               calculateDiff(oldResp.Tx512_1023, newResp.Tx512_1023, false),
			Tx1024_1518:              calculateDiff(oldResp.Tx1024_1518, newResp.Tx1024_1518, false),
			Tx1519_2047:              calculateDiff(oldResp.Tx1519_2047, newResp.Tx1519_2047, false),
			Tx2048_4095:              calculateDiff(oldResp.Tx2048_4095, newResp.Tx2048_4095, false),
			Tx4096_9216:              calculateDiff(oldResp.Tx4096_9216, newResp.Tx4096_9216, false),
			Tx9217_16383:             calculateDiff(oldResp.Tx9217_16383, newResp.Tx9217_16383, false),
			RxAll:                    calculateDiff(oldResp.RxAll, newResp.RxAll, false),
			RxUnicast:                calculateDiff(oldResp.RxUnicast, newResp.RxUnicast, false),
			RxMulticast:              calculateDiff(oldResp.RxMulticast, newResp.RxMulticast, false),
			RxBroadcast:              calculateDiff(oldResp.RxBroadcast, newResp.RxBroadcast, false),
			TxAll:                    calculateDiff(oldResp.TxAll, newResp.TxAll, false),
			TxUnicast:                calculateDiff(oldResp.TxUnicast, newResp.TxUnicast, false),
			TxMulticast:              calculateDiff(oldResp.TxMulticast, newResp.TxMulticast, false),
			TxBroadcast:              calculateDiff(oldResp.TxBroadcast, newResp.TxBroadcast, false),
			RxJabbers:                calculateDiff(oldResp.RxJabbers, newResp.RxJabbers, false),
			RxFragments:              calculateDiff(oldResp.RxFragments, newResp.RxFragments, false),
			RxUndersize:              calculateDiff(oldResp.RxUndersize, newResp.RxUndersize, false),
			RxOverruns:               calculateDiff(oldResp.RxOverruns, newResp.RxOverruns, false),
			FecErrCWs:                newResp.FecErrCWs,
			TimestampClearedCounters: getTimestampClearedCounters(oldResp.TimestampClearedCounters, newResp.TimestampClearedCounters),
		}
	}
	return diffResponse
}

func getTimestampClearedCounters(oldTS, newTS string) string {
	if oldTS == defaultTimestamp && newTS == defaultTimestamp { // cache was not available for either
		return defaultTimestamp
	} else if newTS != defaultTimestamp { // prioritize new TS
		return newTS
	}
	return oldTS
}

func getPortStatCacheSnapshot() (map[string]InterfaceCountersSnapshot, bool) {
	portStatCacheStr, err := common.GetDataFromFile(portStatCachePath)
	if err != nil || len(portStatCacheStr) == 0 {
		return nil, false
	}
	var portStatCacheMap map[string]json.RawMessage
	if err := json.Unmarshal(portStatCacheStr, &portStatCacheMap); err != nil {
		return nil, false
	}
	timestampClearedCounters := defaultTimestamp

	if timestamp, ok := portStatCacheMap["time"]; ok {
		var clearedCountersTS string
		if err := json.Unmarshal(timestamp, &clearedCountersTS); err == nil && clearedCountersTS != "" {
			timestampClearedCounters = clearedCountersTS
		}
	}

	delete(portStatCacheMap, "time") // portstat cache json contains "time" as the top most element
	output := make(map[string]InterfaceCountersSnapshot, len(portStatCacheMap))
	for ifname, value := range portStatCacheMap {
		var snapshot InterfaceCountersSnapshot
		if err := json.Unmarshal(value, &snapshot); err != nil {
			continue
		}
		snapshot.TimestampClearedCounters = timestampClearedCounters
		output[ifname] = snapshot
	}
	if len(output) == 0 { // no interface had proper data
		return nil, false
	}
	return output, true
}

func projectCounters(snapshot map[string]InterfaceCountersSnapshot) map[string]InterfaceCountersResponse {
	output := make(map[string]InterfaceCountersResponse, len(snapshot))
	for intf, value := range snapshot {
		output[intf] = InterfaceCountersResponse{
			State:  value.State,
			RxOk:   value.RxOk,
			RxBps:  value.RxBps,
			RxUtil: value.RxUtil,
			RxErr:  value.RxErr,
			RxDrp:  value.RxDrp,
			RxOvr:  value.RxOvr,
			TxOk:   value.TxOk,
			TxBps:  value.TxBps,
			TxUtil: value.TxUtil,
			TxErr:  value.TxErr,
			TxDrp:  value.TxDrp,
			TxOvr:  value.TxOvr,
		}
	}
	return output
}

func projectAllCounters(snapshot map[string]InterfaceCountersSnapshot) map[string]InterfaceCountersAllResponse {
	output := make(map[string]InterfaceCountersAllResponse, len(snapshot))
	for intf, value := range snapshot {
		output[intf] = InterfaceCountersAllResponse{
			State:    value.State,
			RxOk:     value.RxOk,
			RxBps:    value.RxBps,
			RxPps:    value.RxPps,
			RxUtil:   value.RxUtil,
			RxErr:    value.RxErr,
			RxDrp:    value.RxDrp,
			RxOvr:    value.RxOvr,
			TxOk:     value.TxOk,
			TxBps:    value.TxBps,
			TxPps:    value.TxPps,
			TxUtil:   value.TxUtil,
			TxErr:    value.TxErr,
			TxDrp:    value.TxDrp,
			TxOvr:    value.TxOvr,
			TrimPkts: value.TrimPkts,
			TrimSent: value.TrimSent,
			TrimDrp:  value.TrimDrp,
		}
	}
	return output
}

func projectTrimCounters(snapshot map[string]InterfaceCountersSnapshot) map[string]InterfaceCountersTrimResponse {
	output := make(map[string]InterfaceCountersTrimResponse, len(snapshot))
	for intf, value := range snapshot {
		output[intf] = InterfaceCountersTrimResponse{
			State:    value.State,
			TrimPkts: value.TrimPkts,
			TrimSent: value.TrimSent,
			TrimDrp:  value.TrimDrp,
		}
	}
	return output
}

func projectRateCounters(snapshot map[string]InterfaceCountersSnapshot) map[string]InterfaceCountersRatesResponse {
	output := make(map[string]InterfaceCountersRatesResponse, len(snapshot))
	for intf, value := range snapshot {
		output[intf] = InterfaceCountersRatesResponse{
			State:  value.State,
			RxOk:   value.RxOk,
			RxBps:  value.RxBps,
			RxPps:  value.RxPps,
			RxUtil: value.RxUtil,
			TxOk:   value.TxOk,
			TxBps:  value.TxBps,
			TxPps:  value.TxPps,
			TxUtil: value.TxUtil,
		}
	}
	return output
}

func projectErrorCounters(snapshot map[string]InterfaceCountersSnapshot) map[string]InterfaceCountersErrorsResponse {
	output := make(map[string]InterfaceCountersErrorsResponse, len(snapshot))
	for intf, value := range snapshot {
		output[intf] = InterfaceCountersErrorsResponse{
			State: value.State,
			RxErr: value.RxErr,
			RxDrp: value.RxDrp,
			RxOvr: value.RxOvr,
			TxErr: value.TxErr,
			TxDrp: value.TxDrp,
			TxOvr: value.TxOvr,
		}
	}
	return output
}

func projectDetailedCounters(snapshot map[string]InterfaceCountersSnapshot) map[string]InterfaceCountersDetailedResponse {
	output := make(map[string]InterfaceCountersDetailedResponse, len(snapshot))
	for intf, value := range snapshot {
		output[intf] = InterfaceCountersDetailedResponse{
			TrimPkts:                 value.TrimPkts,
			TrimSent:                 value.TrimSent,
			TrimDrp:                  value.TrimDrp,
			Rx64:                     value.Rx64,
			Rx65_127:                 value.Rx65_127,
			Rx128_255:                value.Rx128_255,
			Rx256_511:                value.Rx256_511,
			Rx512_1023:               value.Rx512_1023,
			Rx1024_1518:              value.Rx1024_1518,
			Rx1519_2047:              value.Rx1519_2047,
			Rx2048_4095:              value.Rx2048_4095,
			Rx4096_9216:              value.Rx4096_9216,
			Rx9217_16383:             value.Rx9217_16383,
			Tx64:                     value.Tx64,
			Tx65_127:                 value.Tx65_127,
			Tx128_255:                value.Tx128_255,
			Tx256_511:                value.Tx256_511,
			Tx512_1023:               value.Tx512_1023,
			Tx1024_1518:              value.Tx1024_1518,
			Tx1519_2047:              value.Tx1519_2047,
			Tx2048_4095:              value.Tx2048_4095,
			Tx4096_9216:              value.Tx4096_9216,
			Tx9217_16383:             value.Tx9217_16383,
			RxAll:                    value.RxAll,
			RxUnicast:                value.RxUnicast,
			RxMulticast:              value.RxMulticast,
			RxBroadcast:              value.RxBroadcast,
			TxAll:                    value.TxAll,
			TxUnicast:                value.TxUnicast,
			TxMulticast:              value.TxMulticast,
			TxBroadcast:              value.TxBroadcast,
			RxJabbers:                value.RxJabbers,
			RxFragments:              value.RxFragments,
			RxUndersize:              value.RxUndersize,
			RxOverruns:               value.RxOverruns,
			TimestampClearedCounters: value.TimestampClearedCounters,
		}
	}
	return output
}

func projectFecStatCounters(snapshot map[string]InterfaceCountersSnapshot) map[string]InterfaceCountersFecStatsResponse {
	output := make(map[string]InterfaceCountersFecStatsResponse, len(snapshot))
	for intf, value := range snapshot {
		output[intf] = InterfaceCountersFecStatsResponse{
			State:        value.State,
			FecCorr:      value.FecCorr,
			FecUncorr:    value.FecUncorr,
			FecSymbolErr: value.FecSymbolErr,
			FecPreBer:    value.FecPreBer,
			FecPostBer:   value.FecPostBer,
		}
	}
	return output
}

func projectFecHistogramCounters(snapshot map[string]InterfaceCountersSnapshot) []FecErrCW {
	for _, value := range snapshot {
		if len(value.FecErrCWs) != 0 {
			return value.FecErrCWs
		}
		break
	}
	return nil
}

func calculateDiff(oldValue, newValue string, raw bool) string {
	if newValue == common.DefaultMissingCounterValue {
		return common.DefaultMissingCounterValue
	}

	if oldValue == common.DefaultMissingCounterValue {
		oldValue = "0"
	}

	oldCounterValue, _ := strconv.ParseInt(oldValue, common.Base10, 64)
	newCounterValue, _ := strconv.ParseInt(newValue, common.Base10, 64)

	diff := newCounterValue - oldCounterValue
	if raw { // Don't check for negative
		return strconv.FormatInt(diff, common.Base10)
	}
	if diff < 0 {
		diff = 0
	}
	return strconv.FormatInt(diff, common.Base10)
}

// Validate counter value is an integer, return common.DefaultMissingCounterValue if not
func validateAndGetIntValue(value string) string {
	_, valueParseErr := strconv.ParseInt(value, common.Base10, 64)
	if valueParseErr != nil {
		log.Warningf("Invalid counter value %s: %v", value, valueParseErr)
		return common.DefaultMissingCounterValue
	}

	return value
}

func getRifNameMapping() (map[string]interface{}, error) {
	queries := [][]string{
		{common.CountersDb, "COUNTERS_RIF_NAME_MAP"},
	}

	rifNameMap, err := common.GetMapFromQueries(queries)
	if err != nil {
		return nil, fmt.Errorf("Failed to get COUNTERS_RIF_NAME_MAP from %s: %v", common.CountersDb, err)
	}

	if len(rifNameMap) == 0 {
		return nil, fmt.Errorf("No COUNTERS_RIF_NAME_MAP in DB")
	}

	return rifNameMap, nil
}

func calculateByteRate(rate string) string {
	if rate == common.DefaultMissingCounterValue {
		return common.DefaultMissingCounterValue
	}
	rateFloatValue, err := strconv.ParseFloat(rate, 64)
	if err != nil {
		return common.DefaultMissingCounterValue
	}
	var formatted string
	switch {
	case rateFloatValue > 10*1e6:
		formatted = fmt.Sprintf("%.2f MB", rateFloatValue/1e6)
	case rateFloatValue > 10*1e3:
		formatted = fmt.Sprintf("%.2f KB", rateFloatValue/1e3)
	default:
		formatted = fmt.Sprintf("%.2f B", rateFloatValue)
	}

	return formatted + "/s"
}

func calculateUtil(rate string, portSpeed string) string {
	if rate == common.DefaultMissingCounterValue || portSpeed == common.DefaultMissingCounterValue {
		return common.DefaultMissingCounterValue
	}
	byteRate, err := strconv.ParseFloat(rate, 64)
	if err != nil {
		return common.DefaultMissingCounterValue
	}
	portRate, err := strconv.ParseFloat(portSpeed, 64)
	if err != nil {
		return common.DefaultMissingCounterValue
	}
	util := byteRate / (portRate * 1e6 / 8.0) * 100.0
	return fmt.Sprintf("%.2f%%", util)
}

func computeSpeed(iface string, statePortTable, appPortTable map[string]interface{}) string {
	speedFromState := common.GetFieldValueString(statePortTable, iface, common.DefaultMissingCounterValue, "speed")
	operStatus := common.GetFieldValueString(appPortTable, iface, common.DefaultMissingCounterValue, "oper_status")
	if speedFromState == common.DefaultMissingCounterValue || operStatus != "up" {
		return common.GetFieldValueString(appPortTable, iface, common.DefaultMissingCounterValue, "speed")
	}
	return speedFromState
}

func computeState(iface string, portTable map[string]interface{}) string {
	entry, ok := portTable[iface].(map[string]interface{})
	if !ok {
		return "X"
	}
	adminStatus := fmt.Sprint(entry["admin_status"])
	operStatus := fmt.Sprint(entry["oper_status"])
	switch {
	case adminStatus == "down":
		return "X"
	case adminStatus == "up" && operStatus == "up":
		return "U"
	case adminStatus == "up" && operStatus == "down":
		return "D"
	default:
		return "X"
	}
}

func calculatePacketRate(rate string) string {
	if rate == common.DefaultMissingCounterValue {
		return common.DefaultMissingCounterValue
	}
	rateFloatValue, err := strconv.ParseFloat(rate, 64)
	if err != nil {
		return common.DefaultMissingCounterValue
	}
	return fmt.Sprintf("%.2f/s", rateFloatValue)
}

func calculateBerRate(rate string) string {
	if rate == common.DefaultMissingCounterValue {
		return common.DefaultMissingCounterValue
	}
	rateFloatValue, err := strconv.ParseFloat(rate, 64)
	if err != nil {
		return common.DefaultMissingCounterValue
	}
	return fmt.Sprintf("%.2e", rateFloatValue)
}

func formatUtil(rate string) string {
	if rate == common.DefaultMissingCounterValue {
		return common.DefaultMissingCounterValue
	}
	utilRate, err := strconv.ParseFloat(rate, 64)
	if err != nil {
		return common.DefaultMissingCounterValue
	}
	return fmt.Sprintf("%.2f%%", utilRate)
}

func computeUtil(utilRate string, byteRate string, portSpeed string) string {
	if utilRate == common.DefaultMissingCounterValue {
		return calculateUtil(byteRate, portSpeed)
	} else {
		return formatUtil(utilRate)
	}
}
