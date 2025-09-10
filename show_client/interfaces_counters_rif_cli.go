package show_client

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

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

func getInterfaceRifCounters(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	period := 0
	interfaceName := args.At(0)
	takeDiffSnapshot := false
	if periodValue, ok := options["period"].Int(); ok {
		takeDiffSnapshot = true
		period = periodValue
	}

	rifNameMap, err := getRifNameMapping()
	if err != nil {
		log.Errorf("Failed to get COUNTERS_RIF_NAME_MAP: %v", err)
		return nil, err
	}

	if interfaceName != "" {
		if _, ok := rifNameMap[interfaceName]; !ok {
			return nil, errors.New(fmt.Sprintf("Interface %s not found in COUNTERS_RIF_NAME_MAP, Make sure it exists", interfaceName))
		}
	}

	oldInterfaceRifCountersMap, err := getInterfaceCountersRifSnapshot(interfaceName)
	if err != nil {
		log.Errorf("Failed to get old interface RIF counters: %v", err)
		return nil, err
	}

	if !takeDiffSnapshot {
		return json.Marshal(oldInterfaceRifCountersMap)
	}

	if period > 0 {
		if period > maxShowCommandPeriod {
			return nil, fmt.Errorf("period value must be <= %v", maxShowCommandPeriod)
		}
		time.Sleep(time.Duration(period) * time.Second)
	}

	newInterfaceRifCountersMap, err := getInterfaceCountersRifSnapshot(interfaceName)
	if err != nil {
		log.Errorf("Failed to get new interface RIF counters: %v", err)
		return nil, err
	}

	diffInterfaceRifCountersMap := make(map[string]interfaceRifCounters, len(newInterfaceRifCountersMap))
	for interfaceName, newInterfaceRifCounters := range newInterfaceRifCountersMap {
		if _, ok := oldInterfaceRifCountersMap[interfaceName]; !ok {
			diffInterfaceRifCountersMap[interfaceName] = newInterfaceRifCounters
			continue
		}

		diffInterfaceRifCounters := interfaceRifCounters{
			RxOkPackets:  calculateDiff(oldInterfaceRifCountersMap[interfaceName].RxOkPackets, newInterfaceRifCounters.RxOkPackets),
			RxBps:        newInterfaceRifCounters.RxBps,
			RxPps:        newInterfaceRifCounters.RxPps,
			RxErrPackets: calculateDiff(oldInterfaceRifCountersMap[interfaceName].RxErrPackets, newInterfaceRifCounters.RxErrPackets),
			TxOkPackets:  calculateDiff(oldInterfaceRifCountersMap[interfaceName].TxOkPackets, newInterfaceRifCounters.TxOkPackets),
			TxBps:        newInterfaceRifCounters.TxBps,
			TxPps:        newInterfaceRifCounters.TxPps,
			TxErrPackets: calculateDiff(oldInterfaceRifCountersMap[interfaceName].TxErrPackets, newInterfaceRifCounters.TxErrPackets),
			RxErrBits:    calculateDiff(oldInterfaceRifCountersMap[interfaceName].RxErrBits, newInterfaceRifCounters.RxErrBits),
			TxErrBits:    calculateDiff(oldInterfaceRifCountersMap[interfaceName].TxErrBits, newInterfaceRifCounters.TxErrBits),
			RxOkBits:     calculateDiff(oldInterfaceRifCountersMap[interfaceName].RxOkBits, newInterfaceRifCounters.RxOkBits),
			TxOkBits:     calculateDiff(oldInterfaceRifCountersMap[interfaceName].TxOkBits, newInterfaceRifCounters.TxOkBits),
		}

		diffInterfaceRifCountersMap[interfaceName] = diffInterfaceRifCounters
	}

	return json.Marshal(diffInterfaceRifCountersMap)
}

func getInterfaceCountersRifSnapshot(interfaceName string) (map[string]interfaceRifCounters, error) {
	rifNameMap, err := getRifNameMapping()
	if err != nil {
		log.Errorf("Failed to get COUNTERS_RIF_NAME_MAP: %v", err)
		return nil, err
	}

	queries := [][]string{
		{CountersDb, "COUNTERS"},
	}

	rifCountersMap, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to pull data for queries %v, got err %v", queries, err)
		return nil, err
	}

	queries = [][]string{
		{CountersDb, "RATES:*"},
	}

	rifRatesMap, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to pull data for queries %v, got err %v", queries, err)
		return nil, err
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

		interfaceRifCounter := interfaceRifCounters{
			RxOkPackets:  GetFieldValueString(rifCountersMap, oidStr, defaultMissingCounterValue, "SAI_ROUTER_INTERFACE_STAT_IN_PACKETS"),
			RxBps:        GetFieldValueString(rifRatesMap, oidStr, defaultMissingCounterValue, "RX_BPS"),
			RxPps:        GetFieldValueString(rifRatesMap, oidStr, defaultMissingCounterValue, "RX_PPS"),
			RxErrPackets: GetFieldValueString(rifCountersMap, oidStr, defaultMissingCounterValue, "SAI_ROUTER_INTERFACE_STAT_IN_ERROR_PACKETS"),
			TxOkPackets:  GetFieldValueString(rifCountersMap, oidStr, defaultMissingCounterValue, "SAI_ROUTER_INTERFACE_STAT_OUT_PACKETS"),
			TxBps:        GetFieldValueString(rifRatesMap, oidStr, defaultMissingCounterValue, "TX_BPS"),
			TxPps:        GetFieldValueString(rifRatesMap, oidStr, defaultMissingCounterValue, "TX_PPS"),
			TxErrPackets: GetFieldValueString(rifCountersMap, oidStr, defaultMissingCounterValue, "SAI_ROUTER_INTERFACE_STAT_OUT_ERROR_PACKETS"),
			RxErrBits:    GetFieldValueString(rifCountersMap, oidStr, defaultMissingCounterValue, "SAI_ROUTER_INTERFACE_STAT_IN_ERROR_OCTETS"),
			TxErrBits:    GetFieldValueString(rifCountersMap, oidStr, defaultMissingCounterValue, "SAI_ROUTER_INTERFACE_STAT_OUT_ERROR_OCTETS"),
			RxOkBits:     GetFieldValueString(rifCountersMap, oidStr, defaultMissingCounterValue, "SAI_ROUTER_INTERFACE_STAT_IN_OCTETS"),
			TxOkBits:     GetFieldValueString(rifCountersMap, oidStr, defaultMissingCounterValue, "SAI_ROUTER_INTERFACE_STAT_OUT_OCTETS"),
		}

		interfaceRifCountersMap[rifName] = interfaceRifCounter
	}

	return interfaceRifCountersMap, nil
}

func calculateDiff(oldValue, newValue string) string {
	if newValue == defaultMissingCounterValue {
		return defaultMissingCounterValue
	}

	if oldValue == defaultMissingCounterValue {
		oldValue = "0"
	}

	oldCounterValue, err := strconv.ParseInt(oldValue, base10, 64)
	if err != nil {
		log.Warningf("Invalid old counter value %s: %v", oldValue, err)
		return defaultMissingCounterValue
	}

	newCounterValue, err := strconv.ParseInt(newValue, base10, 64)
	if err != nil {
		log.Warningf("Invalid new counter value %s: %v", newValue, err)
		return defaultMissingCounterValue
	}

	diff := newCounterValue - oldCounterValue
	if diff < 0 {
		diff = 0
	}

	return strconv.FormatInt(diff, base10)
}

func getRifNameMapping() (map[string]interface{}, error) {
	queries := [][]string{
		{CountersDb, "COUNTERS_RIF_NAME_MAP"},
	}

	rifNameMap, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get COUNTERS_RIF_NAME_MAP from %s: %v", CountersDb, err)
		return nil, err
	}

	if len(rifNameMap) == 0 {
		return nil, errors.New("No COUNTERS_RIF_NAME_MAP in DB")
	}

	return rifNameMap, nil
}
