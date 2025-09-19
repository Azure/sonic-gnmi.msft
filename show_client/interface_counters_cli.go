package show_client

import (
	"encoding/json"
	"fmt"
	"time"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func getInterfaceCounters(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	var ifaces []string
	period := 0
	takeDiffSnapshot := false
	fetchAllCounters := false

	if interfaces, ok := options["interfaces"].Strings(); ok {
		ifaces = interfaces
	}

	if periodValue, ok := options["period"].Int(); ok {
		takeDiffSnapshot = true
		period = periodValue
	}

	if getAllCounters, ok := options["printall"].Bool(); ok {
		fetchAllCounters = getAllCounters
	}

	if period > maxShowCommandPeriod || period < 0 {
		return nil, fmt.Errorf("period value must be <= %v and non negative", maxShowCommandPeriod)
	}

	oldSnapshot, err := getInterfaceCountersSnapshot(ifaces)
	if err != nil {
		log.Errorf("Unable to get interfaces counter snapshot due to err: %v", err)
		return nil, err
	}
	finalSnapshot := oldSnapshot

	if takeDiffSnapshot && period > 0 {
		time.Sleep(time.Duration(period) * time.Second)

		newSnapshot, err := getInterfaceCountersSnapshot(ifaces)
		if err != nil {
			log.Errorf("Unable to get new interface counters snapshot due to err %v", err)
			return nil, err
		}

		// Compare diff between snapshot
		finalSnapshot = calculateDiffSnapshot(oldSnapshot, newSnapshot)
	}

	if fetchAllCounters {
		return json.Marshal(projectAllCounters(finalSnapshot))
	}

	return json.Marshal(projectCounters(finalSnapshot))
}

func getInterfaceCountersErrors(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	period := 0
	takeDiffSnapshot := false

	if periodValue, ok := options["period"].Int(); ok {
		takeDiffSnapshot = true
		period = periodValue
	}

	if period > maxShowCommandPeriod || period < 0 {
		return nil, fmt.Errorf("period value must be <= %v and non negative", maxShowCommandPeriod)
	}

	oldSnapshot, err := getInterfaceCountersSnapshot(nil)
	if err != nil {
		log.Errorf("Unable to get interfaces counter snapshot due to err: %v", err)
		return nil, err
	}
	finalSnapshot := oldSnapshot

	if takeDiffSnapshot && period > 0 {
		time.Sleep(time.Duration(period) * time.Second)

		newSnapshot, err := getInterfaceCountersSnapshot(nil)
		if err != nil {
			log.Errorf("Unable to get new interface counters snapshot due to err %v", err)
			return nil, err
		}

		// Compare diff between snapshot
		finalSnapshot = calculateDiffSnapshot(oldSnapshot, newSnapshot)
	}

	return json.Marshal(projectErrorCounters(finalSnapshot))
}

func getInterfaceCountersTrim(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	var ifaces []string
	period := 0
	takeDiffSnapshot := false
	intf := args.At(0)

	if intf != "" {
		ifaces = []string{intf}
	}

	if periodValue, ok := options["period"].Int(); ok {
		takeDiffSnapshot = true
		period = periodValue
	}

	if period > maxShowCommandPeriod || period < 0 {
		return nil, fmt.Errorf("period value must be <= %v and non negative", maxShowCommandPeriod)
	}

	oldSnapshot, err := getInterfaceCountersSnapshot(ifaces)
	if err != nil {
		log.Errorf("Unable to get interfaces counter snapshot due to err: %v", err)
		return nil, err
	}
	finalSnapshot := oldSnapshot

	if takeDiffSnapshot && period > 0 {
		time.Sleep(time.Duration(period) * time.Second)

		newSnapshot, err := getInterfaceCountersSnapshot(ifaces)
		if err != nil {
			log.Errorf("Unable to get new interface counters snapshot due to err %v", err)
			return nil, err
		}

		// Compare diff between snapshot
		finalSnapshot = calculateDiffSnapshot(oldSnapshot, newSnapshot)
	}

	return json.Marshal(projectTrimCounters(finalSnapshot))
}

func getInterfaceCountersRates(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	period := 0
	takeDiffSnapshot := false

	if periodValue, ok := options["period"].Int(); ok {
		takeDiffSnapshot = true
		period = periodValue
	}

	if period > maxShowCommandPeriod || period < 0 {
		return nil, fmt.Errorf("period value must be <= %v and non negative", maxShowCommandPeriod)
	}

	oldSnapshot, err := getInterfaceCountersSnapshot(nil)
	if err != nil {
		log.Errorf("Unable to get interfaces counter snapshot due to err: %v", err)
		return nil, err
	}
	finalSnapshot := oldSnapshot

	if takeDiffSnapshot && period > 0 {
		time.Sleep(time.Duration(period) * time.Second)

		newSnapshot, err := getInterfaceCountersSnapshot(nil)
		if err != nil {
			log.Errorf("Unable to get new interface counters snapshot due to err %v", err)
			return nil, err
		}

		// Compare diff between snapshot
		finalSnapshot = calculateDiffSnapshot(oldSnapshot, newSnapshot)
	}

	return json.Marshal(projectRateCounters(finalSnapshot))
}

func getInterfaceCountersFecStats(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	period := 0
	takeDiffSnapshot := false

	if periodValue, ok := options["period"].Int(); ok {
		takeDiffSnapshot = true
		period = periodValue
	}

	if period > maxShowCommandPeriod || period < 0 {
		return nil, fmt.Errorf("period value must be <= %v and non negative", maxShowCommandPeriod)
	}

	oldSnapshot, err := getInterfaceCountersSnapshot(nil)
	if err != nil {
		log.Errorf("Unable to get interfaces counter snapshot due to err: %v", err)
		return nil, err
	}
	finalSnapshot := oldSnapshot

	if takeDiffSnapshot && period > 0 {
		time.Sleep(time.Duration(period) * time.Second)

		newSnapshot, err := getInterfaceCountersSnapshot(nil)
		if err != nil {
			log.Errorf("Unable to get new interface counters snapshot due to err %v", err)
			return nil, err
		}

		// Compare diff between snapshot
		finalSnapshot = calculateDiffSnapshot(oldSnapshot, newSnapshot)
	}

	return json.Marshal(projectFecStatCounters(finalSnapshot))
}

func getInterfaceCountersFecHistogram(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	intf := args.At(0)
	if intf == "" {
		return nil, status.Errorf(codes.InvalidArgument, "No interface name passed")
	}

	finalSnapshot, err := getInterfaceCountersSnapshot([]string{intf})
	if err != nil {
		log.Errorf("Unable to get interfaces counter snapshot due to err: %v", err)
		return nil, err
	}

	return json.Marshal(projectFecHistogramCounters(finalSnapshot))
}

func getInterfaceCountersDetailed(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	var ifaces []string
	period := 0
	takeDiffSnapshot := false
	intf := args.At(0)

	if intf == "" {
		return nil, status.Errorf(codes.InvalidArgument, "No interface name passed")
	}

	ifaces = []string{intf}

	if periodValue, ok := options["period"].Int(); ok {
		takeDiffSnapshot = true
		period = periodValue
	}

	if period > maxShowCommandPeriod || period < 0 {
		return nil, fmt.Errorf("period value must be <= %v and non negative", maxShowCommandPeriod)
	}

	oldSnapshot, err := getInterfaceCountersSnapshot(ifaces)
	if err != nil {
		log.Errorf("Unable to get interfaces counter snapshot due to err: %v", err)
		return nil, err
	}
	finalSnapshot := oldSnapshot

	if takeDiffSnapshot && period > 0 {
		time.Sleep(time.Duration(period) * time.Second)

		newSnapshot, err := getInterfaceCountersSnapshot(ifaces)
		if err != nil {
			log.Errorf("Unable to get new interface counters snapshot due to err %v", err)
			return nil, err
		}

		// Compare diff between snapshot
		finalSnapshot = calculateDiffSnapshot(oldSnapshot, newSnapshot)
	}

	return json.Marshal(projectDetailedCounters(finalSnapshot))
}

func getInterfaceRifCounters(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	period := 0
	interfaceName := args.At(0)
	takeDiffSnapshot := false
	if periodValue, ok := options["period"].Int(); ok {
		takeDiffSnapshot = true
		period = periodValue
	}

	if period > maxShowCommandPeriod {
		return nil, status.Errorf(codes.InvalidArgument, "period value must be <= %v", maxShowCommandPeriod)
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
