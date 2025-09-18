type InterfaceCountersSnapshot struct {
	// Port Status
	State  string
	// Port Counters
	RxOk   string
	RxErr  string
	RxDrp  string
	RxOvr  string
	TxOk   string
	TxErr  string
	TxDrp  string
	TxOvr  string
	// Port Rates
	RxBps  string
	RxPps  string
	RxUtil string
	TxBps  string
	TxPps  string
	TxUtil string
	// FEC counters
	FecCorr      string
	FecUncorr    string
	FecSymbolErr string
	FecPreBer    string
	FecPostBer   string
	// Trim Counters
	TrimPkts string
	TrimSent string
	TrimDrp  string
	// Detailed Counters for Octets
	Rx64         string
	Rx65_127     string
	Rx128_255    string
	Rx256_511    string
	Rx512_1023   string
	Rx1024_1518  string
	Rx1519_2047  string
	Rx2048_4095  string
	Rx4096_9126  string
	Rx9127_16383 string
	Tx64         string
	Tx65_127     string
	Tx128_255    string
	Tx256_511    string
	Tx512_1023   string
	Tx1024_1518  string
	Tx1519_2047  string
	Tx2048_4095  string
	Tx4096_9126  string
	Tx9127_16383 string
	// Detailed Counters
	RxAll       string
	RxUnicast   string
	RxMulticast string
	RxBroadcast string
	TxAll       string
	TxUnicast   string
	TxMulticast string
	TxBroadcast string
	RxJabbers   string
	RxFragments string
	RxUndersize string
	RxOverruns  string
	// FEC Codewords per symbol error index
	FecErrCWs []FecErrCW
}

type FecErrCW struct {
	BinIndex  string
	Codewords  string
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
}

type InterfaceCountersErrorsResponse struct {
	State  string
	RxErr  string
	RxDrp  string
	RxOvr  string
	TxErr  string
	TxDrp  string
	TxOvr  string
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
	Rx4096_9126  string
	Rx9127_16383 string
	Tx64         string
	Tx65_127     string
	Tx128_255    string
	Tx256_511    string
	Tx512_1023   string
	Tx1024_1518  string
	Tx1519_2047  string
	Tx2048_4095  string
	Tx4096_9126  string
	Tx9127_16383 string
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
}

func getInterfaceCountersSnapshot(ifaces []string) (map[string]InterfaceCountersSnapshot, error) {
	queries := [][]string{
		{"COUNTERS_DB", "COUNTERS", "Ethernet*"},
	}

	aliasCountersOutput, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to pull data for queries %v, got err %v", queries, err)
		return nil, err
	}

	portCounters := RemapAliasToPortName(aliasCountersOutput)

	queries = [][]string{
		{"COUNTERS_DB", "RATES", "Ethernet*"},
	}

	aliasRatesOutput, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to pull data for queries %v, got err %v", queries, err)
		return nil, err
	}

	portRates := RemapAliasToPortName(aliasRatesOutput)

	queries = [][]string{
		{"APPL_DB", "PORT_TABLE"},
	}

	portTable, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to pull data for queries %v, got err %v", queries, err)
		return nil, err
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

	response := make(map[string]InterfaceCountersSnapshot, len(ifaces))

	for _, iface := range validatedIfaces {
		state := computeState(iface, portTable)
		portSpeed := GetFieldValueString(portTable, iface, defaultMissingCounterValue, "speed")
		rxBps := GetFieldValueString(portRates, iface, defaultMissingCounterValue, "RX_BPS")
		txBps := GetFieldValueString(portRates, iface, defaultMissingCounterValue, "TX_BPS")
		rxPps := GetFieldValueString(portRates, iface, defaultMissingCounterValue, "RX_PPS")
		txPps := GetFieldValueString(portRates, iface, defaultMissingCounterValue, "TX_PPS")
		preBer := GetFieldValueString(portRates, iface, defaultMissingCounterValue, "FEC_PRE_BER")
		postBer := GetFieldValueString(portRates, iface, defaultMissingCounterValue, "FEC_POST_BER")

		response[iface] = InterfaceCountersSnapshot{
			State:  state,
			RxOk:   GetSumFields(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_UCAST_PKTS", "SAI_PORT_STAT_IF_IN_NON_UCAST_PKTS"),
			RxBps:  calculateByteRate(rxBps),
			RxPps:  calculatePacketRate(rxPps),
			RxUtil: calculateUtil(rxBps, portSpeed),
			RxErr:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_ERRORS"),
			RxDrp:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_DISCARDS"),
			RxOvr:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_RX_OVERSIZE_PKTS"),
			TxOk:   GetSumFields(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_OUT_UCAST_PKTS", "SAI_PORT_STAT_IF_OUT_NON_UCAST_PKTS"),
			TxBps:  calculateByteRate(txBps),
			TxPps:  calculatePacketRate(txPps),
			TxUtil: calculateUtil(txBps, portSpeed),
			TxErr:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_OUT_ERRORS"),
			TxDrp:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_OUT_DISCARDS"),
			TxOvr:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_TX_OVERSIZE_PKTS"),
			FecCorr: GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_FEC_CORRECTABLE_FRAMES"),
			FecUncorr: GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_FEC_NOT_CORRECTABLE_FRAMES"),
			FecSymbolErr: GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_FEC_SYMBOL_ERRORS"),
			FecPreBer: calculateBerRate(preBer),
			FecPostBer: calculateBerRate(postBer),
			TrimPkts: GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "TRIM_PKTS"),
			TrimSent: GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "TRIM_SENT_PKTS"),
			TrimDrp: GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "TRIM_DROP_PKTS"),
			Rx64: GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_IN_PKTS_64_OCTETS"),
			Rx65_127: GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_IN_PKTS_65_TO_127_OCTETS"),
			Rx128_255:    GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_IN_PKTS_128_TO_255_OCTETS"),
			Rx256_511:    GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_IN_PKTS_256_TO_511_OCTETS"),
			Rx512_1023:   GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_IN_PKTS_512_TO_1023_OCTETS"),
			Rx1024_1518:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_IN_PKTS_1024_TO_1518_OCTETS"),
			Rx1519_2047:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_IN_PKTS_1519_TO_2047_OCTETS"),
			Rx2048_4095:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_IN_PKTS_2048_TO_4095_OCTETS"),
			Rx4096_9126:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_IN_PKTS_4096_TO_9216_OCTETS"),
			Rx9127_16383: GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_IN_PKTS_9217_TO_16383_OCTETS"),
			Tx64:         GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_OUT_PKTS_64_OCTETS"),
			Tx65_127:     GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_OUT_PKTS_65_TO_127_OCTETS"),
			Tx128_255:    GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_OUT_PKTS_128_TO_255_OCTETS"),
			Tx256_511:    GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_OUT_PKTS_256_TO_511_OCTETS"),
			Tx512_1023:   GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_OUT_PKTS_512_TO_1023_OCTETS"),
			Tx1024_1518:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_OUT_PKTS_1024_TO_1518_OCTETS"),
			Tx1519_2047:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_OUT_PKTS_1519_TO_2047_OCTETS"),
			Tx2048_4095:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_OUT_PKTS_2048_TO_4095_OCTETS"),
			Tx4096_9126:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_OUT_PKTS_4096_TO_9216_OCTETS"),
			Tx9127_16383: GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_OUT_PKTS_9217_TO_16383_OCTETS"),
			RxAll:        GetSumFields(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_UCAST_PKTS", "SAI_PORT_STAT_IF_IN_MULTICAST_PKTS", "SAI_PORT_STAT_IF_IN_BROADCAST_PKTS"),
			RxUnicast:    GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_UCAST_PKTS"),
			RxMulticast:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_MULTICAST_PKTS"),
			RxBroadcast:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_BROADCAST_PKTS"),
			TxAll:        GetSumFields(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_OUT_UCAST_PKTS", "SAI_PORT_STAT_IF_OUT_MULTICAST_PKTS", "SAI_PORT_STAT_IF_OUT_BROADCAST_PKTS"),
			TxUnicast:    GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_OUT_UCAST_PKTS"),
			TxMulticast:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_OUT_MULTICAST_PKTS"),
			TxBroadcast:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_OUT_BROADCAST_PKTS"),
			RxJabbers:    GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_STATS_JABBERS"),
			RxFragments:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_STATS_FRAGMENTS"),
			RxUndersize:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_STATS_UNDERSIZE_PKTS"),
			RxOverruns:   GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IP_IN_RECEIVES"),
		}

		var fecErrCWs []fecErrCW
		for i := 0; i < len(fecBinCount); i++ {
			binIndex := fmt.Sprintf("BIN%d", i)
			fecCodewordsKey := fmt.Sprintf("SAI_PORT_STAT_IF_IN_FEC_CODEWORD_ERRORS_S%d", i)
			fecCodewordsValue := GetFieldValueString(portCounters, iface, "0", fecCodewordsKey)
			entry := FecErrCW{
				BinIndex:  binIndex,
				Codewords: fecCodewordsValue,
			}
			fecErrCWs = append(fecErrCWs, entry)
		}
		response[iface].FecErrCWs = fecErrCWs
	}
	return response, nil
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
			State:        newResp.State,
			RxOk:         calculateDiffReturnDefault(oldResp.RxOk, newResp.RxOk, defaultMissingCounterValue),
			RxErr:        calculateDiffReturnDefault(oldResp.RxErr, newResp.RxErr, defaultMissingCounterValue),
			RxDrp:        calculateDiffReturnDefault(oldResp.RxDrp, newResp.RxDrp, defaultMissingCounterValue),
			RxOvr:        calculateDiffReturnDefault(oldResp.RxOvr, newResp.RxOvr, defaultMissingCounterValue),
			TxOk:         calculateDiffReturnDefault(oldResp.TxOk, newResp.TxOk, defaultMissingCounterValue),
			TxErr:        calculateDiffReturnDefault(oldResp.TxErr, newResp.TxErr, defaultMissingCounterValue),
			TxDrp:        calculateDiffReturnDefault(oldResp.TxDrp, newResp.TxDrp, defaultMissingCounterValue),
			TxOvr:        calculateDiffReturnDefault(oldResp.TxOvr, newResp.TxOvr, defaultMissingCounterValue),
			RxBps:        newResp.RxBps,
			RxPps:        newResp.RxPps,
			RxUtil:       newResp.RxUtil,
			TxBps:        newResp.TxBps,
			TxPps:        newResp.TxPps,
			TxUtil:       newResp.TxUtil,
			FecCorr:      calculateDiffReturnDefault(oldResp.FecCorr, newResp.FecCorr, defaultMissingCounterValue),
			FecUncorr:    calculateDiffReturnDefault(oldResp.FecUncorr, newResp.FecUncorr, defaultMissingCounterValue),
			FecSymbolErr: calculateDiffReturnDefault(oldResp.FecSymbolErr, newResp.FecSymbolErr, defaultMissingCounterValue),
			FecPreBer:    newResp.FecPreBer,
			FecPostBer:   newResp.FecPostBer,
			TrimPkts:     calculateDiffReturnDefault(oldResp.TrimPkts, newResp.TrimPkts, defaultMissingCounterValue),
			TrimSent:     calculateDiffReturnDefault(oldResp.TrimSent, newResp.TrimSent, defaultMissingCounterValue),
			TrimDrp:      calculateDiffReturnDefault(oldResp.TrimDrp, newResp.TrimDrp, defaultMissingCounterValue),
			Rx64:         calculateDiffReturnDefault(oldResp.Rx64, newResp.Rx64, defaultMissingCounterValue),
			Rx65_127:     calculateDiffReturnDefault(oldResp.Rx65_127, newResp.Rx65_127, defaultMissingCounterValue),
			Rx128_255:    calculateDiffReturnDefault(oldResp.Rx128_255, newResp.Rx128_255, defaultMissingCounterValue),
			Rx256_511:    calculateDiffReturnDefault(oldResp.Rx256_511, newResp.Rx256_511, defaultMissingCounterValue),
			Rx512_1023:   calculateDiffReturnDefault(oldResp.Rx512_1023, newResp.Rx512_1023, defaultMissingCounterValue),
			Rx1024_1518:  calculateDiffReturnDefault(oldResp.Rx1024_1518, newResp.Rx1024_1518, defaultMissingCounterValue),
			Rx1519_2047:  calculateDiffReturnDefault(oldResp.Rx1519_2047, newResp.Rx1519_2047, defaultMissingCounterValue),
			Rx2048_4095:  calculateDiffReturnDefault(oldResp.Rx2048_4095, newResp.Rx2048_4095, defaultMissingCounterValue),
			Rx4096_9126:  calculateDiffReturnDefault(oldResp.Rx4096_9126, newResp.Rx4096_9126, defaultMissingCounterValue),
			Rx9127_16383: calculateDiffReturnDefault(oldResp.Rx9127_16383, newResp.Rx9127_16383, defaultMissingCounterValue),
			Tx64:         calculateDiffReturnDefault(oldResp.Tx64, newResp.Tx64, defaultMissingCounterValue),
			Tx65_127:     calculateDiffReturnDefault(oldResp.Tx65_127, newResp.Tx65_127, defaultMissingCounterValue),
			Tx128_255:    calculateDiffReturnDefault(oldResp.Tx128_255, newResp.Tx128_255, defaultMissingCounterValue),
			Tx256_511:    calculateDiffReturnDefault(oldResp.Tx256_511, newResp.Tx256_511, defaultMissingCounterValue),
			Tx512_1023:   calculateDiffReturnDefault(oldResp.Tx512_1023, newResp.Tx512_1023, defaultMissingCounterValue),
			Tx1024_1518:  calculateDiffReturnDefault(oldResp.Tx1024_1518, newResp.Tx1024_1518, defaultMissingCounterValue),
			Tx1519_2047:  calculateDiffReturnDefault(oldResp.Tx1519_2047, newResp.Tx1519_2047, defaultMissingCounterValue),
			Tx2048_4095:  calculateDiffReturnDefault(oldResp.Tx2048_4095, newResp.Tx2048_4095, defaultMissingCounterValue),
			Tx4096_9126:  calculateDiffReturnDefault(oldResp.Tx4096_9126, newResp.Tx4096_9126, defaultMissingCounterValue),
			Tx9127_16383: calculateDiffReturnDefault(oldResp.Tx9127_16383, newResp.Tx9127_16383, defaultMissingCounterValue),
			RxAll:        calculateDiffReturnDefault(oldResp.RxAll, newResp.RxAll, defaultMissingCounterValue),
			RxUnicast:    calculateDiffReturnDefault(oldResp.RxUnicast, newResp.RxUnicast, defaultMissingCounterValue),
			RxMulticast:  calculateDiffReturnDefault(oldResp.RxMulticast, newResp.RxMulticast, defaultMissingCounterValue),
			RxBroadcast:  calculateDiffReturnDefault(oldResp.RxBroadcast, newResp.RxBroadcast, defaultMissingCounterValue),
			TxAll:        calculateDiffReturnDefault(oldResp.TxAll, newResp.TxAll, defaultMissingCounterValue),
			TxUnicast:    calculateDiffReturnDefault(oldResp.TxUnicast, newResp.TxUnicast, defaultMissingCounterValue),
			TxMulticast:  calculateDiffReturnDefault(oldResp.TxMulticast, newResp.TxMulticast, defaultMissingCounterValue),
			TxBroadcast:  calculateDiffReturnDefault(oldResp.TxBroadcast, newResp.TxBroadcast, defaultMissingCounterValue),
			RxJabbers:    calculateDiffReturnDefault(oldResp.RxJabbers, newResp.RxJabbers, defaultMissingCounterValue),
			RxFragments:  calculateDiffReturnDefault(oldResp.RxFragments, newResp.RxFragments, defaultMissingCounterValue),
			RxUndersize:  calculateDiffReturnDefault(oldResp.RxUndersize, newResp.RxUndersize, defaultMissingCounterValue),
			RxOverruns:   calculateDiffReturnDefault(oldResp.RxOverruns, newResp.RxOverruns, defaultMissingCounterValue),
			FecErrCWs:    newResp.FecErrCWs,
		}
	}
	return diffResponse
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
	for intf, s := range snapshot {
		output[intf] = InterfaceCountersAllResponse{
			State:  value.State,
			RxOk:   value.RxOk,
			RxBps:  value.RxBps,
			RxPps:  value.RxPps,
			RxUtil: value.RxUtil,
			RxErr:  value.RxErr,
			RxDrp:  value.RxDrp,
			RxOvr:  value.RxOvr,
			TxOk:   value.TxOk,
			TxBps:  value.TxBps,
			TxPps:  value.TxPps,
			TxUtil: value.TxUtil,
			TxErr:  value.TxErr,
			TxDrp:  value.TxDrp,
			TxOvr:  value.TxOvr,
			Trim:   value.TrimPkts,
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
			TrimPkts:     value.TrimPkts,
			TrimSent:     value.Trim,
			TrimDrp:      value.TrimDrp,
			Rx64:         value.Rx64,
			Rx65_127:     value.Rx65_127,
			Rx128_255:    value.Rx128_255,
			Rx256_511:    value.Rx256_511,
			Rx512_1023:   value.Rx512_1023,
			Rx1024_1518:  value.Rx1024_1518,
			Rx1519_2047:  value.Rx1519_2047,
			Rx2048_4095:  value.Rx2048_4095,
			Rx4096_9126:  value.Rx4096_9126,
			Rx9127_16383: value.Rx9127_16383,
			Tx64:         value.Tx64,
			Tx65_127:     value.Tx65_127,
			Tx128_255:    value.Tx128_255,
			Tx256_511:    value.Tx256_511,
			Tx512_1023:   value.Tx512_1023,
			Tx1024_1518:  value.Tx1024_1518,
			Tx1519_2047:  value.Tx1519_2047,
			Tx2048_4095:  value.Tx2048_4095,
			Tx4096_9126:  value.Tx4096_9126,
			Tx9127_16383: value.Tx9127_16383,
			RxAll:        value.RxAll,
			RxUnicast:    value.RxUnicast,
			RxMulticast:  value.RxMulticast,
			RxBroadcast:  value.RxBroadcast,
			TxAll:        value.TxAll,
			TxUnicast:    value.TxUnicast,
			TxMulticast:  value.TxMulticast,
			TxBroadcast:  value.TxBroadcast,
			RxJabbers:    value.RxJabbers,
			RxFragments:  value.RxFragments,
			RxUndersize:  value.RxUndersize,
			RxOverruns:   value.RxOverruns,
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

func calculateDiffReturnDefault(oldCounter, newCounter, defaultValue string) string {
	if oldCounter == defaultValue || newCounter == defaultValue {
		return defaultValue
	}
	oldV, err := strconv.ParseInt(oldCounter, base10, 64)
	if err != nil {
		return defaultValue
	}
	newV, err := strconv.ParseInt(newCounter, base10, 64)
	if err != nil || newV < oldV { // guard reset/rollover
		return defaultValue
	}
	return strconv.FormatInt(newV-oldV, base10)
}

func calculateByteRate(rate string) string {
	if rate == defaultMissingCounterValue {
		return defaultMissingCounterValue
	}
	rateFloatValue, err := strconv.ParseFloat(rate, 64)
	if err != nil {
		return defaultMissingCounterValue
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
	if rate == defaultMissingCounterValue || portSpeed == defaultMissingCounterValue {
		return defaultMissingCounterValue
	}
	byteRate, err := strconv.ParseFloat(rate, 64)
	if err != nil {
		return defaultMissingCounterValue
	}
	portRate, err := strconv.ParseFloat(portSpeed, 64)
	if err != nil {
		return defaultMissingCounterValue
	}
	util := byteRate / (portRate * 1e6 / 8.0) * 100.0
	return fmt.Sprintf("%.2f%%", util)
}

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
	if rate == defaultMissingCounterValue {
		return defaultMissingCounterValue
	}
	rateFloatValue, err := strconv.ParseFloat(rate, 64)
	if err != nil {
		return defaultMissingCounterValue
	}
	return fmt.Sprintf("%.2f/s", rateFloatValue)
}

func calculateBerRate(rate string) string {
	if rate == defaultMissingCounterValue {
		return defaultMissingCounterValue
	}
	rateFloatValue, err := strconv.ParseFloat(rate, 64)
	if err != nil {
		return defaultMissingCounterValue
	}
	return fmt.Sprintf("%.2e", rateFloatValue)
}
