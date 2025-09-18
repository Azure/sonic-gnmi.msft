package show_client

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	if period > common.MaxShowCommandPeriod {
		return nil, status.Errorf(codes.InvalidArgument, "period value must be <= %v", common.MaxShowCommandPeriod)
	}

	rifNameMap, err := getRifNameMapping()
	if err != nil {
		return nil, fmt.Errorf("Failed to get COUNTERS_RIF_NAME_MAP: %v", err)
	}

	if interfaceName != "" {
		if _, ok := rifNameMap[interfaceName]; !ok {
			return nil, status.Errorf(codes.InvalidArgument, "Interface %s not found in COUNTERS_RIF_NAME_MAP, Make sure it exists", interfaceName)
		}
	}

	oldInterfaceRifCountersMap, err := getInterfaceCountersRifSnapshot(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("Failed to get old interface RIF counters: %v", err)
	}

	if !takeDiffSnapshot {
		return json.Marshal(oldInterfaceRifCountersMap)
	}

	if period > 0 {
		time.Sleep(time.Duration(period) * time.Second)
	}

	newInterfaceRifCountersMap, err := getInterfaceCountersRifSnapshot(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("Failed to get new interface RIF counters: %v", err)
	}

	diffInterfaceRifCountersMap := make(map[string]interfaceRifCounters, len(newInterfaceRifCountersMap))
	for interfaceName, newInterfaceRifCounters := range newInterfaceRifCountersMap {
		if _, ok := oldInterfaceRifCountersMap[interfaceName]; !ok {
			diffInterfaceRifCountersMap[interfaceName] = newInterfaceRifCounters
			continue
		}

		diffInterfaceRifCounters := interfaceRifCounters{
			RxOkPackets:  calculateDiffClampZero(oldInterfaceRifCountersMap[interfaceName].RxOkPackets, newInterfaceRifCounters.RxOkPackets),
			RxBps:        newInterfaceRifCounters.RxBps,
			RxPps:        newInterfaceRifCounters.RxPps,
			RxErrPackets: calculateDiffClampZero(oldInterfaceRifCountersMap[interfaceName].RxErrPackets, newInterfaceRifCounters.RxErrPackets),
			TxOkPackets:  calculateDiffClampZero(oldInterfaceRifCountersMap[interfaceName].TxOkPackets, newInterfaceRifCounters.TxOkPackets),
			TxBps:        newInterfaceRifCounters.TxBps,
			TxPps:        newInterfaceRifCounters.TxPps,
			TxErrPackets: calculateDiffClampZero(oldInterfaceRifCountersMap[interfaceName].TxErrPackets, newInterfaceRifCounters.TxErrPackets),
			RxErrBits:    calculateDiffClampZero(oldInterfaceRifCountersMap[interfaceName].RxErrBits, newInterfaceRifCounters.RxErrBits),
			TxErrBits:    calculateDiffClampZero(oldInterfaceRifCountersMap[interfaceName].TxErrBits, newInterfaceRifCounters.TxErrBits),
			RxOkBits:     calculateDiffClampZero(oldInterfaceRifCountersMap[interfaceName].RxOkBits, newInterfaceRifCounters.RxOkBits),
			TxOkBits:     calculateDiffClampZero(oldInterfaceRifCountersMap[interfaceName].TxOkBits, newInterfaceRifCounters.TxOkBits),
		}

		diffInterfaceRifCountersMap[interfaceName] = diffInterfaceRifCounters
	}

	return json.Marshal(diffInterfaceRifCountersMap)
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

func calculateDiffClampZero(oldValue, newValue string) string {
	if newValue == common.DefaultMissingCounterValue {
		return common.DefaultMissingCounterValue
	}

	if oldValue == common.DefaultMissingCounterValue {
		oldValue = "0"
	}

	oldCounterValue, _ := strconv.ParseInt(oldValue, common.Base10, 64)
	newCounterValue, _ := strconv.ParseInt(newValue, common.Base10, 64)

	diff := newCounterValue - oldCounterValue
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
		return nil, errors.New("No COUNTERS_RIF_NAME_MAP in DB")
	}

	return rifNameMap, nil
}
