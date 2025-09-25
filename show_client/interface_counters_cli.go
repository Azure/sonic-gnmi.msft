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

	if interfaces, ok := options["interface"].Strings(); ok {
		ifaces = interfaces
	}

	if getAllCounters, ok := options["printall"].Bool(); ok {
		fetchAllCounters = getAllCounters
	}

	period, takeDiffSnapshot, err := validatePeriod(options)
	if err != nil {
		return nil, err
	}

	finalSnapshot, err := snapshotWithOptionalDiff(ifaces, period, takeDiffSnapshot)
	if err != nil {
		return nil, err
	}

	if fetchAllCounters {
		return json.Marshal(projectAllCounters(finalSnapshot))
	}

	return json.Marshal(projectCounters(finalSnapshot))
}

func getInterfaceCountersErrors(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	period, takeDiffSnapshot, err := validatePeriod(options)
	if err != nil {
		return nil, err
	}

	finalSnapshot, err := snapshotWithOptionalDiff(nil, period, takeDiffSnapshot)
	if err != nil {
		return nil, err
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

	period, takeDiffSnapshot, err := validatePeriod(options)
	if err != nil {
		return nil, err
	}

	finalSnapshot, err := snapshotWithOptionalDiff(ifaces, period, takeDiffSnapshot)
	if err != nil {
		return nil, err
	}

	return json.Marshal(projectTrimCounters(finalSnapshot))
}

func getInterfaceCountersRates(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	period, takeDiffSnapshot, err := validatePeriod(options)
	if err != nil {
		return nil, err
	}

	finalSnapshot, err := snapshotWithOptionalDiff(nil, period, takeDiffSnapshot)
	if err != nil {
		return nil, err
	}

	return json.Marshal(projectRateCounters(finalSnapshot))
}

func getInterfaceCountersFecStats(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	period, takeDiffSnapshot, err := validatePeriod(options)
	if err != nil {
		return nil, err
	}

	finalSnapshot, err := snapshotWithOptionalDiff(nil, period, takeDiffSnapshot)
	if err != nil {
		return nil, err
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
	intf := args.At(0)

	if intf == "" {
		return nil, status.Errorf(codes.InvalidArgument, "No interface name passed")
	}

	ifaces := []string{intf}

	period, takeDiffSnapshot, err := validatePeriod(options)
	if err != nil {
		return nil, err
	}

	finalSnapshot, err := snapshotWithOptionalDiff(ifaces, period, takeDiffSnapshot)
	if err != nil {
		return nil, err
	}

	return json.Marshal(projectDetailedCounters(finalSnapshot))
}

func getInterfaceRifCounters(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	interfaceName := args.At(0)

	period, takeDiffSnapshot, err := validatePeriod(options)
	if err != nil {
		return nil, err
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
			RxOkPackets:  calculateDiff(oldInterfaceRifCountersMap[interfaceName].RxOkPackets, newInterfaceRifCounters.RxOkPackets, false),
			RxBps:        newInterfaceRifCounters.RxBps,
			RxPps:        newInterfaceRifCounters.RxPps,
			RxErrPackets: calculateDiff(oldInterfaceRifCountersMap[interfaceName].RxErrPackets, newInterfaceRifCounters.RxErrPackets, false),
			TxOkPackets:  calculateDiff(oldInterfaceRifCountersMap[interfaceName].TxOkPackets, newInterfaceRifCounters.TxOkPackets, false),
			TxBps:        newInterfaceRifCounters.TxBps,
			TxPps:        newInterfaceRifCounters.TxPps,
			TxErrPackets: calculateDiff(oldInterfaceRifCountersMap[interfaceName].TxErrPackets, newInterfaceRifCounters.TxErrPackets, false),
			RxErrBits:    calculateDiff(oldInterfaceRifCountersMap[interfaceName].RxErrBits, newInterfaceRifCounters.RxErrBits, false),
			TxErrBits:    calculateDiff(oldInterfaceRifCountersMap[interfaceName].TxErrBits, newInterfaceRifCounters.TxErrBits, false),
			RxOkBits:     calculateDiff(oldInterfaceRifCountersMap[interfaceName].RxOkBits, newInterfaceRifCounters.RxOkBits, false),
			TxOkBits:     calculateDiff(oldInterfaceRifCountersMap[interfaceName].TxOkBits, newInterfaceRifCounters.TxOkBits, false),
		}

		diffInterfaceRifCountersMap[interfaceName] = diffInterfaceRifCounters
	}

	return json.Marshal(diffInterfaceRifCountersMap)
}
