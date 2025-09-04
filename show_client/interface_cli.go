package show_client

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"sort"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

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

type namingModeResponse struct {
	NamingMode string `json:"naming_mode"`
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

func getInterfaceCounters(options sdc.OptionMap) ([]byte, error) {
	var ifaces []string
	period := 0
	takeDiffSnapshot := false

	if interfaces, ok := options["interfaces"].Strings(); ok {
		ifaces = interfaces
	}

	if periodValue, ok := options["period"].Int(); ok {
		takeDiffSnapshot = true
		period = periodValue
	}

	if period > maxShowCommandPeriod {
		return nil, fmt.Errorf("period value must be <= %v", maxShowCommandPeriod)
	}

	oldSnapshot, err := getInterfaceCountersSnapshot(ifaces)
	if err != nil {
		log.Errorf("Unable to get interfaces counter snapshot due to err: %v", err)
		return nil, err
	}

	if !takeDiffSnapshot {
		return json.Marshal(oldSnapshot)
	}

	time.Sleep(time.Duration(period) * time.Second)

	newSnapshot, err := getInterfaceCountersSnapshot(ifaces)
	if err != nil {
		log.Errorf("Unable to get new interface counters snapshot due to err %v", err)
		return nil, err
	}

	// Compare diff between snapshot
	diffSnapshot := calculateDiffSnapshot(oldSnapshot, newSnapshot)

	return json.Marshal(diffSnapshot)
}

func getInterfaceCountersSnapshot(ifaces []string) (map[string]InterfaceCountersResponse, error) {
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

	response := make(map[string]InterfaceCountersResponse, len(ifaces))

	for _, iface := range validatedIfaces {
		state := computeState(iface, portTable)
		portSpeed := GetFieldValueString(portTable, iface, defaultMissingCounterValue, "speed")
		rxBps := GetFieldValueString(portRates, iface, defaultMissingCounterValue, "RX_BPS")
		txBps := GetFieldValueString(portRates, iface, defaultMissingCounterValue, "TX_BPS")

		response[iface] = InterfaceCountersResponse{
			State:  state,
			RxOk:   GetSumFields(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_UCAST_PKTS", "SAI_PORT_STAT_IF_IN_NON_UCAST_PKTS"),
			RxBps:  calculateByteRate(rxBps),
			RxUtil: calculateUtil(rxBps, portSpeed),
			RxErr:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_ERRORS"),
			RxDrp:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_IN_DISCARDS"),
			RxOvr:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_RX_OVERSIZE_PKTS"),
			TxOk:   GetSumFields(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_OUT_UCAST_PKTS", "SAI_PORT_STAT_IF_OUT_NON_UCAST_PKTS"),
			TxBps:  calculateByteRate(txBps),
			TxUtil: calculateUtil(txBps, portSpeed),
			TxErr:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_OUT_ERRORS"),
			TxDrp:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_IF_OUT_DISCARDS"),
			TxOvr:  GetFieldValueString(portCounters, iface, defaultMissingCounterValue, "SAI_PORT_STAT_ETHER_TX_OVERSIZE_PKTS"),
		}
	}
	return response, nil
}

func calculateDiffSnapshot(oldSnapshot map[string]InterfaceCountersResponse, newSnapshot map[string]InterfaceCountersResponse) map[string]InterfaceCountersResponse {
	diffResponse := make(map[string]InterfaceCountersResponse, len(newSnapshot))

	for iface, newResp := range newSnapshot {
		oldResp, found := oldSnapshot[iface]
		if !found {
			oldResp = InterfaceCountersResponse{
				RxOk:  "0",
				RxErr: "0",
				RxDrp: "0",
				TxOk:  "0",
				TxErr: "0",
				TxDrp: "0",
				TxOvr: "0",
			}
		}
		diffResponse[iface] = InterfaceCountersResponse{
			State:  newResp.State,
			RxOk:   calculateDiffCounters(oldResp.RxOk, newResp.RxOk, defaultMissingCounterValue),
			RxBps:  newResp.RxBps,
			RxUtil: newResp.RxUtil,
			RxErr:  calculateDiffCounters(oldResp.RxErr, newResp.RxErr, defaultMissingCounterValue),
			RxDrp:  calculateDiffCounters(oldResp.RxDrp, newResp.RxDrp, defaultMissingCounterValue),
			RxOvr:  calculateDiffCounters(oldResp.RxOvr, newResp.RxOvr, defaultMissingCounterValue),
			TxOk:   calculateDiffCounters(oldResp.TxOk, newResp.TxOk, defaultMissingCounterValue),
			TxBps:  newResp.TxBps,
			TxUtil: newResp.TxUtil,
			TxErr:  calculateDiffCounters(oldResp.TxErr, newResp.TxErr, defaultMissingCounterValue),
			TxDrp:  calculateDiffCounters(oldResp.TxDrp, newResp.TxDrp, defaultMissingCounterValue),
			TxOvr:  calculateDiffCounters(oldResp.TxOvr, newResp.TxOvr, defaultMissingCounterValue),
		}
	}
	return diffResponse
}

var allPortErrors = [][]string{
	{"oper_error_status", "oper_error_status_time"},
	{"mac_local_fault_count", "mac_local_fault_time"},
	{"mac_remote_fault_count", "mac_remote_fault_time"},
	{"fec_sync_loss_count", "fec_sync_loss_time"},
	{"fec_alignment_loss_count", "fec_alignment_loss_time"},
	{"high_ser_error_count", "high_ser_error_time"},
	{"high_ber_error_count", "high_ber_error_time"},
	{"data_unit_crc_error_count", "data_unit_crc_error_time"},
	{"data_unit_misalignment_error_count", "data_unit_misalignment_error_time"},
	{"signal_local_error_count", "signal_local_error_time"},
	{"crc_rate_count", "crc_rate_time"},
	{"data_unit_size_count", "data_unit_size_time"},
	{"code_group_error_count", "code_group_error_time"},
	{"no_rx_reachability_count", "no_rx_reachability_time"},
}

func getInterfaceErrors(options sdc.OptionMap) ([]byte, error) {
	intf, ok := options["interface"].String()
	if !ok {
		return nil, fmt.Errorf("No interface name passed in as option")
	}

	// Query Port Operational Errors Table from STATE_DB
	queries := [][]string{
		{"STATE_DB", "PORT_OPERR_TABLE", intf},
	}
	portErrorsTbl, _ := GetMapFromQueries(queries)
	portErrorsTbl = RemapAliasToPortName(portErrorsTbl)

	// Format the port errors data
	portErrors := make([]map[string]string, 0, len(allPortErrors)+1)
	// Iterate through all port errors types and create the result
	for _, portError := range allPortErrors {
		count := "0"
		timestamp := "Never"
		if portErrorsTbl != nil {
			if val, ok := portErrorsTbl[portError[0]]; ok {
				count = fmt.Sprintf("%v", val)
			}
			if val, ok := portErrorsTbl[portError[1]]; ok {
				timestamp = fmt.Sprintf("%v", val)
			}
		}

		portErrors = append(portErrors, map[string]string{
			"Port Errors":         strings.Replace(strings.Replace(portError[0], "_", " ", -1), " count", "", -1),
			"Count":               count,
			"Last timestamp(UTC)": timestamp},
		)
	}

	// Convert [][]string to []byte using JSON serialization
	return json.Marshal(portErrors)
}

func getIntfsFromConfigDB(intf string) ([]string, error) {
	// Get the list of ports from the SONiC CONFIG_DB
	queries := [][]string{
		{"CONFIG_DB", ConfigDBPortTable},
	}
	portTable, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get interface list from CONFIG_DB: %v", err)
		return nil, err
	}

	// If intf is specified, return only that interface if exists
	if intf != "" {
		if _, ok := portTable[intf]; !ok {
			return []string{}, nil
		}
		return []string{intf}, nil
	}

	// If no specific interface is requested, return all interfaces
	ports := make([]string, 0, len(portTable))
	for key := range portTable {
		ports = append(ports, key)
	}
	return ports, nil
}

func getInterfaceFecStatus(options sdc.OptionMap) ([]byte, error) {
	intf, _ := options["interface"].String()

	ports, err := getIntfsFromConfigDB(intf)
	if err != nil {
		log.Errorf("Failed to get front panel ports: %v", err)
		return nil, err
	}
	ports = natsortInterfaces(ports)

	portFecStatus := make([]map[string]string, 0, len(ports)+1)
	for i := range ports {
		port := ports[i]
		adminFecStatus := ""
		operStatus := ""
		operFecStatus := ""

		// Query port admin FEC status and operation status from APPL_DB
		queries := [][]string{
			{"APPL_DB", AppDBPortTable, port},
		}
		data, err := GetMapFromQueries(queries)
		if err != nil {
			log.Errorf("Failed to get admin FEC status for port %s: %v", port, err)
			return nil, err
		}
		if _, ok := data["fec"]; !ok {
			adminFecStatus = "N/A"
		} else {
			adminFecStatus = fmt.Sprint(data["fec"])
		}
		if _, ok := data["oper_status"]; !ok {
			operStatus = "N/A"
		} else {
			operStatus = fmt.Sprint(data["oper_status"])
		}

		// Query port's oper FEC status from STATE_DB
		queries = [][]string{
			{"STATE_DB", StateDBPortTable, port},
		}
		data, err = GetMapFromQueries(queries)
		if err != nil {
			log.Errorf("Failed to get oper FEC status for port %s: %v", port, err)
			return nil, err
		}
		if _, ok := data["fec"]; !ok {
			operFecStatus = "N/A"
		} else {
			operFecStatus = fmt.Sprint(data["fec"])
		}

		if operStatus != "up" {
			// If port is down or oper FEC status is not available, set it to "N/A"
			operFecStatus = "N/A"
		}
		portFecStatus = append(portFecStatus, map[string]string{"Interface": port, "FEC Oper": operFecStatus, "FEC Admin": adminFecStatus})
	}

	return json.Marshal(portFecStatus)
}

func getPortchannelIntfsFromConfigDB(intf string) ([]string, error) {
	// Get the list of portchannel interfaces from the SONiC CONFIG_DB
	queries := [][]string{
		{"CONFIG_DB", ConfigDBPortChannelTable},
	}
	portTable, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get portchannel interface list from CONFIG_DB: %v", err)
		return nil, err
	}

	// If intf is specified, return only that interface if exists
	if intf != "" {
		if _, ok := portTable[intf]; !ok {
			return []string{}, nil
		}
		return []string{intf}, nil
	}

	// If no specific interface is requested, return all interfaces
	ports := make([]string, 0, len(portTable))
	for key := range portTable {
		ports = append(ports, key)
	}
	return ports, nil
}

func getPortOptics(intf string) string {
	// Query port optics type from STATE_DB
	queries := [][]string{
		{"STATE_DB", "TRANSCEIVER_INFO", intf},
	}
	data, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get optics type for port %s: %v", intf, err)
		return "N/A"
	}

	if _, ok := data["type"]; !ok {
		return "N/A"
	}
	return fmt.Sprint(data["type"])
}

func portSpeedFmt(inSpeed, opticsType string) string {
	// fetched speed is in megabits per second
	speed, err := strconv.Atoi(inSpeed)
	if err != nil {
		// If parse fails, return "N/A"
		return "N/A"
	}

	if opticsType == "RJ45" && speed <= 1000 {
		return fmt.Sprintf("%dM", speed)
	} else if speed < 1000 {
		return fmt.Sprintf("%dM", speed)
	} else if speed%1000 >= 100 {
		return fmt.Sprintf("%.1fG", float64(speed)/1000.0)
	}
	return fmt.Sprintf("%.0fG", float64(speed)/1000.0)
}

// portSpeedParse converts a human-readable port speed string to an integer Mbps value.
// Examples:
//
//	"100M"   -> 100
//	"1G"     -> 1000
//	"2.5G"   -> 2500
//	"N/A" or parse errors -> 0
func portSpeedParse(speedStr string) int {
	s := strings.TrimSpace(strings.ToUpper(speedStr))
	if s == "" || s == "N/A" {
		return 0
	}

	if strings.HasSuffix(s, "G") {
		v := strings.TrimSuffix(s, "G")
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0
		}
		return int(math.Round(f * 1000.0))
	}

	if strings.HasSuffix(s, "M") {
		v := strings.TrimSuffix(s, "M")
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0
		}
		return int(math.Round(f))
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return int(math.Round(f))
}

func getPortOperSpeed(intf string) string {
	// Query port optics type from STATE_DB
	queries := [][]string{
		{"STATE_DB", StateDBPortTable, intf},
	}
	stateData, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get state for port %s from STATE_DB: %v", intf, err)
		return "N/A"
	}

	queries = [][]string{
		{"APPL_DB", AppDBPortTable, intf},
	}
	appData, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get state for port %s from APPL_DB: %v", intf, err)
		return "N/A"
	}

	opticsType := getPortOptics(intf)
	if status, ok := appData["oper_status"]; !ok || fmt.Sprint(status) != "up" {
		return portSpeedFmt(fmt.Sprint(appData["speed"]), opticsType)
	}
	if _, ok := stateData["speed"]; !ok {
		return portSpeedFmt(fmt.Sprint(appData["speed"]), opticsType)
	} else {
		return portSpeedFmt(fmt.Sprint(stateData["speed"]), opticsType)
	}
}

func getIntfModeMap(ports []string) map[string]string {
	queries := [][]string{
		{"CONFIG_DB", "VLAN_MEMBER"},
	}
	vlanMemberTable, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get VLAN_MEMBER table from CONFIG_DB: %v", err)
		return nil
	}

	// Get the map of interfaces to VLANs
	vlanMembers := map[string]string{}
	for key := range vlanMemberTable {
		content := strings.Split(key, "|")
		if len(content) < 2 {
			// Invalid Key, ignoring it
			continue
		}
		vlanMemberKey := content[1]

		vlanMembers[vlanMemberKey] = content[0]
	}

	queries = [][]string{
		{"CONFIG_DB", "PORTCHANNEL_MEMBER"},
	}
	portChannelMemberTable, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get PORTCHANNEL_MEMBER table from CONFIG_DB: %v", err)
		return nil
	}

	// Get the map of interfaces to Portchannels
	portChannelMembers := map[string]string{}
	for key := range portChannelMemberTable {
		content := strings.Split(key, "|")
		if len(content) < 2 {
			// Invalid Key, ignoring it
			continue
		}
		portChannelMemberKey := content[1]

		portChannelMembers[portChannelMemberKey] = content[0]
	}

	// Create a map to hold the interface mode
	intfModeMap := make(map[string]string)
	for i := range ports {
		port := ports[i]
		queries = [][]string{
			{"CONFIG_DB", "PORT", port},
		}
		portData, err := GetMapFromQueries(queries)
		if err != nil {
			log.Errorf("Failed to get port data for %s: %v", port, err)
			continue
		}

		if mode, ok := portData["mode"]; ok {
			intfModeMap[port] = fmt.Sprint(mode)
		} else if _, ok := portChannelMembers[port]; ok {
			intfModeMap[port] = portChannelMembers[port]
		} else if _, ok := vlanMembers[port]; ok {
			intfModeMap[port] = "trunk"
		} else {
			intfModeMap[port] = "routed"
		}
	}
	return intfModeMap
}

func getPortchannelModeMap(portchannels []string) map[string]string {
	queries := [][]string{
		{"CONFIG_DB", "VLAN_MEMBER"},
	}
	vlanMemberTable, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get VLAN_MEMBER table from CONFIG_DB: %v", err)
		return nil
	}

	// Get the map of interfaces to VLANs
	vlanMembers := map[string]string{}
	for key := range vlanMemberTable {
		content := strings.Split(key, "|")
		if len(content) < 2 {
			// Invalid Key, ignoring it
			continue
		}
		vlanMemberKey := content[1]

		vlanMembers[vlanMemberKey] = content[0]
	}

	poModeMap := make(map[string]string)
	for i := range portchannels {
		port := portchannels[i]
		queries = [][]string{
			{"CONFIG_DB", "PORTCHANNEL", port},
		}
		portData, err := GetMapFromQueries(queries)
		if err != nil {
			log.Errorf("Failed to get port data for %s: %v", port, err)
			continue
		}

		if mode, ok := portData["mode"]; ok {
			poModeMap[port] = fmt.Sprint(mode)
		} else if _, ok := vlanMembers[port]; ok {
			poModeMap[port] = "trunk"
		} else {
			poModeMap[port] = "routed"
		}
	}
	return poModeMap
}

func getPortchannelSpeedMap(portchannels []string) map[string]string {
	queries := [][]string{
		{"CONFIG_DB", "PORTCHANNEL_MEMBER"},
	}
	portChannelMemberTable, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get PORTCHANNEL_MEMBER table from CONFIG_DB: %v", err)
		return nil
	}

	// Get the map of Portchannels to Interfaces
	portChannelMembership := map[string][]string{}
	for key := range portChannelMemberTable {
		content := strings.Split(key, "|")
		if len(content) < 2 {
			// Invalid Key, ignoring it
			continue
		}
		portChannel := content[0]

		portChannelMembership[portChannel] = append(portChannelMembership[portChannel], content[1])
	}

	// Calculate the speed for each portchannel by summing the speeds of its member interfaces
	poSpeedMap := make(map[string]string)
	for portchannel := range portChannelMembership {
		speedList := []string{}
		for i := range portChannelMembership[portchannel] {
			speed := getPortOperSpeed(portChannelMembership[portchannel][i])
			speedList = append(speedList, speed)
		}

		aggSpeed := 0
		for _, speed := range speedList {
			aggSpeed += portSpeedParse(speed)
		}
		poSpeedMap[portchannel] = portSpeedFmt(fmt.Sprint(aggSpeed), "N/A")
	}

	return poSpeedMap
}

func getSubIntfsFromAppDB(intf string) ([]string, error) {
	// get the list of sub-interfaces from APPL_DB
	queries := [][]string{
		{"APPL_DB", "INTF_TABLE"},
	}
	portTable, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get sub-interface list from APPL_DB: %v", err)
		return nil, err
	}

	// If intf is specified, return only that interface
	if intf != "" {
		if _, ok := portTable[intf]; !ok {
			return nil, fmt.Errorf("Sub-interface %s not found in APPL_DB", intf)
		}
		return []string{intf}, nil
	}

	// If no specific interface is requested, return all interfaces
	ports := make([]string, 0, len(portTable))
	for key := range portTable {
		ports = append(ports, key)
	}
	return ports, nil
}

func getSubInterfaceStatus(intf string) ([]byte, error) {
	// Get the status of sub-interfaces
	ports, err := getSubIntfsFromAppDB(intf)
	if err != nil {
		log.Errorf("Failed to get sub-interfaces from APPL_DB: %v", err)
		return nil, err
	}
	ports = natsortInterfaces(ports)

	interfaceStatus := make([]map[string]string, 0, len(ports))
	for i := range ports {
		interfaceStatus = append(interfaceStatus, map[string]string{
			"Interface": ports[i],
			"Speed":     "N/A",
			"MTU":       "N/A",
			"Vlan":      "N/A",
			"Oper":      "N/A",
			"Admin":     "N/A",
			"Type":      "N/A",
		})
	}
	return json.Marshal(interfaceStatus)
}

func getInterfaceStatus(options sdc.OptionMap) ([]byte, error) {
	isSubIntf := false
	intf, _ := options["interface"].String()
	if intf != "" {
		if intf == "subport" {
			isSubIntf = true
			intf = ""
		} else if strings.ContainsRune(intf, '.') {
			isSubIntf = true
		}
	}

	if isSubIntf {
		return getSubInterfaceStatus(intf)
	}

	ports, err := getIntfsFromConfigDB(intf)
	if err != nil {
		log.Errorf("Failed to get front panel port list from CONFIG_DB: %v", err)
		return nil, err
	}
	portchannels, err := getPortchannelIntfsFromConfigDB(intf)
	if err != nil {
		log.Errorf("Failed to get portchannel list from CONFIG_DB: %v", err)
		return nil, err
	}
	ports = natsortInterfaces(ports)
	portchannels = natsortInterfaces(portchannels)
	intfModeMap := getIntfModeMap(ports)
	poModeMap := getPortchannelModeMap(portchannels)
	poSpeedMap := getPortchannelSpeedMap(portchannels)
	interfaceStatus := make([]map[string]string, 0, len(ports)+len(portchannels))

	// Get status of front panel interfaces
	for i := range ports {
		port := ports[i]
		portLanesStatus := ""
		portOperSpeed := getPortOperSpeed(port)
		portMtuStatus := ""
		portFecStatus := ""
		portAlias := ""
		portMode := intfModeMap[port]
		operStatus := ""
		adminStatus := ""
		portOpticsType := getPortOptics(port)
		portPfcAsymStatus := ""

		// Query port status from APPL_DB
		queries := [][]string{
			{"APPL_DB", AppDBPortTable, port},
		}
		data, err := GetMapFromQueries(queries)
		if err != nil {
			log.Errorf("Failed to get status for port %s: %v", port, err)
			return nil, err
		}

		// parse all fields from APP_DB status data
		if _, ok := data["lanes"]; !ok {
			portLanesStatus = "N/A"
		} else {
			portLanesStatus = fmt.Sprint(data["lanes"])
		}
		if _, ok := data["mtu"]; !ok {
			portMtuStatus = "N/A"
		} else {
			portMtuStatus = fmt.Sprint(data["mtu"])
		}
		if _, ok := data["fec"]; !ok {
			portFecStatus = "N/A"
		} else {
			portFecStatus = fmt.Sprint(data["fec"])
		}
		if _, ok := data["alias"]; !ok {
			portAlias = "N/A"
		} else {
			portAlias = fmt.Sprint(data["alias"])
		}
		if _, ok := data["oper_status"]; !ok {
			operStatus = "N/A"
		} else {
			operStatus = fmt.Sprint(data["oper_status"])
		}
		if _, ok := data["admin_status"]; !ok {
			adminStatus = "N/A"
		} else {
			adminStatus = fmt.Sprint(data["admin_status"])
		}
		if _, ok := data["pfc_asym"]; !ok {
			portPfcAsymStatus = "N/A"
		} else {
			portPfcAsymStatus = fmt.Sprint(data["pfc_asym"])
		}

		interfaceStatus = append(interfaceStatus, map[string]string{
			"Interface": port,
			"Lanes":     portLanesStatus,
			"Speed":     portOperSpeed,
			"MTU":       portMtuStatus,
			"FEC":       portFecStatus,
			"Alias":     portAlias,
			"Vlan":      portMode,
			"Oper":      operStatus,
			"Admin":     adminStatus,
			"Type":      portOpticsType,
			"Asym":      portPfcAsymStatus,
		})
	}

	// Get status of portchannel interfaces
	for i := range portchannels {
		port := portchannels[i]
		portLanesStatus := ""
		portOperSpeed := poSpeedMap[port]
		portMtuStatus := ""
		portFecStatus := ""
		portAlias := ""
		portMode := poModeMap[port]
		operStatus := ""
		adminStatus := ""
		portOpticsType := getPortOptics(port)
		portPfcAsymStatus := ""

		// Query portchannel status from APPL_DB
		queries := [][]string{
			{"APPL_DB", AppDBPortChannelTable, port},
		}
		data, err := GetMapFromQueries(queries)
		if err != nil {
			log.Errorf("Failed to get status for portchannel %s: %v", port, err)
			return nil, err
		}

		// Query portchannel config from CONFIG_DB
		queries = [][]string{
			{"CONFIG_DB", ConfigDBPortChannelTable, port},
		}
		config, err := GetMapFromQueries(queries)
		if err != nil {
			log.Errorf("Failed to get status for portchannel %s: %v", port, err)
			return nil, err
		}

		// parse all fields from APP_DB status data
		if _, ok := data["lanes"]; !ok {
			portLanesStatus = "N/A"
		} else {
			portLanesStatus = fmt.Sprint(data["lanes"])
		}
		if _, ok := config["mtu"]; !ok {
			portMtuStatus = "N/A"
		} else {
			portMtuStatus = fmt.Sprint(config["mtu"])
		}
		if _, ok := data["fec"]; !ok {
			portFecStatus = "N/A"
		} else {
			portFecStatus = fmt.Sprint(data["fec"])
		}
		if _, ok := data["alias"]; !ok {
			portAlias = "N/A"
		} else {
			portAlias = fmt.Sprint(data["alias"])
		}
		if _, ok := data["oper_status"]; !ok {
			operStatus = "N/A"
		} else {
			operStatus = fmt.Sprint(data["oper_status"])
		}
		if _, ok := data["admin_status"]; !ok {
			adminStatus = "N/A"
		} else {
			adminStatus = fmt.Sprint(data["admin_status"])
		}
		if _, ok := data["pfc_asym"]; !ok {
			portPfcAsymStatus = "N/A"
		} else {
			portPfcAsymStatus = fmt.Sprint(data["pfc_asym"])
		}

		interfaceStatus = append(interfaceStatus, map[string]string{
			"Interface": port,
			"Lanes":     portLanesStatus,
			"Speed":     portOperSpeed,
			"MTU":       portMtuStatus,
			"FEC":       portFecStatus,
			"Alias":     portAlias,
			"Vlan":      portMode,
			"Oper":      operStatus,
			"Admin":     adminStatus,
			"Type":      portOpticsType,
			"Asym":      portPfcAsymStatus,
		})
	}

	return json.Marshal(interfaceStatus)
}

func getInterfaceAlias(options sdc.OptionMap) ([]byte, error) {
	intf, _ := options["interface"].String()

	// Read CONFIG_DB.PORT
	queries := [][]string{{"CONFIG_DB", "PORT"}}
	portEntries, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get ports from CONFIG_DB: %v", err)
		return nil, err
	}

	nameToAlias := make(map[string]string, len(portEntries))
	for name := range portEntries {
		alias := GetFieldValueString(portEntries, name, "", "alias")
		if alias == "" {
			// fallback to itself if alias field is missing
			alias = name
		}
		nameToAlias[name] = alias
	}

	// If a specific interface was requested, accept port name
	if intf != "" {
		name := intf
		if _, ok := nameToAlias[name]; !ok {
			return nil, fmt.Errorf("Invalid interface name %s", name)
		}
		out := map[string]map[string]string{
			name: {"alias": nameToAlias[name]},
		}
		return json.Marshal(out)
	}

	// Build {"Ethernet0":{"alias":"etp0"}, ...} from CONFIG_DB PORT only
	out := make(map[string]map[string]string, len(nameToAlias))
	for name, alias := range nameToAlias {
		out[name] = map[string]string{"alias": alias}
	}
	return json.Marshal(out)
}

func getInterfaceSwitchportConfig(options sdc.OptionMap) ([]byte, error) {
	intf, _ := options["interface"].String()

	// Read CONFIG_DB tables
	portTbl, err := GetMapFromQueries([][]string{{"CONFIG_DB", "PORT"}})
	if err != nil {
		log.Errorf("Failed to get PORT: %v", err)
		return nil, err
	}
	portChannelTbl, err := GetMapFromQueries([][]string{{"CONFIG_DB", "PORTCHANNEL"}})
	if err != nil {
		log.Errorf("Failed to get PORTCHANNEL: %v", err)
		return nil, err
	}
	portChannelMemberTbl, err := GetMapFromQueries([][]string{{"CONFIG_DB", "PORTCHANNEL_MEMBER"}})
	if err != nil {
		log.Errorf("Failed to get PORTCHANNEL_MEMBER: %v", err)
		return nil, err
	}
	vlanMemberTbl, err := GetMapFromQueries([][]string{{"CONFIG_DB", "VLAN_MEMBER"}})
	if err != nil {
		log.Errorf("Failed to get VLAN_MEMBER: %v", err)
		return nil, err
	}

	// Exclude LAG members from standalone ports
	var ports []string
	for port := range portTbl {
		if !IsInterfaceInPortchannel(portChannelMemberTbl, port) {
			ports = append(ports, port)
		}
	}
	var portchannels []string
	for pc := range portChannelTbl {
		portchannels = append(portchannels, pc)
	}
	keys := append(ports, portchannels...)
	keys = natsortInterfaces(keys)

	// Optionally filter by interface
	if intf != "" {
		found := false
		for _, k := range keys {
			if k == intf {
				found = true
				keys = []string{intf}
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("Got unexpected extra argument %s", intf)
		}
	}

	// Build VLAN membership maps
	untaggedMap := make(map[string][]string)
	taggedMap := make(map[string][]string)
	for k := range vlanMemberTbl {
		vlan, ifname, ok := SplitCompositeKey(k)
		if !ok {
			continue
		}
		tagMode := GetFieldValueString(vlanMemberTbl, k, "", "tagging_mode")
		vlanID := strings.TrimPrefix(vlan, "Vlan")
		if tagMode == "untagged" {
			untaggedMap[ifname] = append(untaggedMap[ifname], vlanID)
		} else if tagMode == "tagged" {
			taggedMap[ifname] = append(taggedMap[ifname], vlanID)
		}
	}
	for k := range untaggedMap {
		sort.Strings(untaggedMap[k])
	}
	for k := range taggedMap {
		sort.Strings(taggedMap[k])
	}

	// Emit switchportConfig
	switchportConfig := make([]map[string]string, 0, len(keys))
	for _, k := range keys {
		untagged := untaggedMap[k]
		tagged := taggedMap[k]

		mode := GetInterfaceSwitchportMode(portTbl, portChannelTbl, vlanMemberTbl, k)

		switchportConfig = append(switchportConfig, map[string]string{
			"Interface": GetInterfaceNameForDisplay(k),
			"Mode":      mode,
			"Untagged":  strings.Join(untagged, ","),
			"Tagged":    strings.Join(tagged, ","),
		})
	}

	return json.Marshal(switchportConfig)
}

func getInterfaceSwitchportStatus(options sdc.OptionMap) ([]byte, error) {
	intf, _ := options["interface"].String()

	// Read CONFIG_DB tables
	portTbl, err := GetMapFromQueries([][]string{{"CONFIG_DB", "PORT"}})
	if err != nil {
		log.Errorf("Failed to get PORT: %v", err)
		return nil, err
	}
	portChannelTbl, err := GetMapFromQueries([][]string{{"CONFIG_DB", "PORTCHANNEL"}})
	if err != nil {
		log.Errorf("Failed to get PORTCHANNEL: %v", err)
		return nil, err
	}
	portChannelMemberTbl, err := GetMapFromQueries([][]string{{"CONFIG_DB", "PORTCHANNEL_MEMBER"}})
	if err != nil {
		log.Errorf("Failed to get PORTCHANNEL_MEMBER: %v", err)
		return nil, err
	}
	vlanMemberTbl, err := GetMapFromQueries([][]string{{"CONFIG_DB", "VLAN_MEMBER"}})
	if err != nil {
		log.Errorf("Failed to get VLAN_MEMBER: %v", err)
		return nil, err
	}

	// Exclude LAG members from standalone ports
	var ports []string
	for port := range portTbl {
		if !IsInterfaceInPortchannel(portChannelMemberTbl, port) {
			ports = append(ports, port)
		}
	}
	var portchannels []string
	for pc := range portChannelTbl {
		portchannels = append(portchannels, pc)
	}
	keys := append(ports, portchannels...)
	keys = natsortInterfaces(keys)

	// Optionally filter by interface
	if intf != "" {
		found := false
		for _, k := range keys {
			if k == intf {
				found = true
				keys = []string{intf}
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("Got unexpected extra argument %s", intf)
		}
	}

	// Emit switchportStatus
	switchportStatus := make([]map[string]string, 0, len(keys))
	for _, k := range keys {
		mode := GetInterfaceSwitchportMode(portTbl, portChannelTbl, vlanMemberTbl, k)

		switchportStatus = append(switchportStatus, map[string]string{
			"Interface": GetInterfaceNameForDisplay(k),
			"Mode":      mode,
		})
	}

	return json.Marshal(switchportStatus)
}

// IsInterfaceInPortchannel reports whether interfaceName is a member of any portchannel.
func IsInterfaceInPortchannel(portchannelMemberTable map[string]interface{}, interfaceName string) bool {
	if portchannelMemberTable == nil || interfaceName == "" {
		return false
	}
	for k := range portchannelMemberTable {
		_, member, ok := SplitCompositeKey(k)
		if ok && member == interfaceName {
			return true
		}
	}
	return false
}

func getInterfaceFlap(options sdc.OptionMap) ([]byte, error) {
	intf, _ := options["interface"].String()

	// Query APPL_DB PORT_TABLE
	queries := [][]string{
		{ApplDb, AppDBPortTable},
	}
	portTable, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get PORT_TABLE: %v", err)
		return nil, err
	}

	// Collect ports (optionally filter by interface)
	var ports []string
	if intf != "" {
		if _, ok := portTable[intf]; !ok {
			return nil, fmt.Errorf("Invalid interface name %s", intf)
		}
		ports = []string{intf}
	} else {
		for p := range portTable {
			ports = append(ports, p)
		}
	}
	ports = natsortInterfaces(ports)

	// Build rows
	rows := make([]map[string]string, 0, len(ports))
	for _, p := range ports {
		flapCount := GetFieldValueString(portTable, p, "Never", "flap_count")
		adminStatus := GetFieldValueString(portTable, p, "Unknown", "admin_status")
		operStatus := GetFieldValueString(portTable, p, "Unknown", "oper_status")
		// Capitalize like Python's .capitalize()
		if adminStatus != "" {
			adminStatus = Capitalize(adminStatus)
		}
		if operStatus != "" {
			operStatus = Capitalize(operStatus)
		}
		lastDown := GetFieldValueString(portTable, p, "Never", "last_down_time")
		lastUp := GetFieldValueString(portTable, p, "Never", "last_up_time")

		rows = append(rows, map[string]string{
			"Interface":                GetInterfaceNameForDisplay(p),
			"Flap Count":               flapCount,
			"Admin":                    adminStatus,
			"Oper":                     operStatus,
			"Link Down TimeStamp(UTC)": fmt.Sprint(lastDown),
			"Link Up TimeStamp(UTC)":   fmt.Sprint(lastUp),
		})
	}

	return json.Marshal(rows)
}

// 'expected' subcommand ("show interface neighbor expected")
// admin@sonic: redis-cli -n 4 HGETALL 'DEVICE_NEIGHBOR|Ethernet2'
// 1) "name"
// 2) "DEVICE01T1"
// 3) "port"
// 4) "Ethernet1"
//
// admin@sonic: redis-cli -n 4 HGETALL "DEVICE_NEIGHBOR_METADATA|DEVICE01T1"
// 1) "hwsku"
// 2) "Arista-VM"
// 3) "mgmt_addr"
// 4) "0.0.0.0"
// 5) "type"
// 6) "BackEndLeafRouter"
func getInterfaceNeighborExpected(options sdc.OptionMap) ([]byte, error) {
	// Fetch DEVICE_NEIGHBOR
	neighborTbl, err := GetMapFromQueries([][]string{{"CONFIG_DB", "DEVICE_NEIGHBOR"}})
	if err != nil {
		log.Errorf("Failed to get DEVICE_NEIGHBOR: %v", err)
		return nil, err
	}

	// Fetch DEVICE_NEIGHBOR_METADATA
	metaTbl, err := GetMapFromQueries([][]string{{"CONFIG_DB", "DEVICE_NEIGHBOR_METADATA"}})
	if err != nil {
		log.Errorf("Failed to get DEVICE_NEIGHBOR_METADATA: %v", err)
		return nil, err
	}

	out := make(map[string]map[string]string)
	for localIf := range neighborTbl {
		device := GetFieldValueString(neighborTbl, localIf, "", "name")
		if device == "" {
			continue
		}
		remotePort := GetFieldValueString(neighborTbl, localIf, "None", "port")
		if remotePort == "" {
			remotePort = "None"
		}

		loopback := GetFieldValueString(metaTbl, device, "None", "lo_addr")
		if loopback == "" {
			loopback = "None"
		}
		mgmt := GetFieldValueString(metaTbl, device, "None", "mgmt_addr")
		if mgmt == "" {
			mgmt = "None"
		}
		ntype := GetFieldValueString(metaTbl, device, "None", "type")
		if ntype == "" {
			ntype = "None"
		}

		displayIf := GetInterfaceNameForDisplay(localIf)

		out[displayIf] = map[string]string{
			"Neighbor":         device,
			"NeighborPort":     remotePort,
			"NeighborLoopback": loopback,
			"NeighborMgmt":     mgmt,
			"NeighborType":     ntype,
		}
	}

	return json.Marshal(out)
}

func getInterfaceNamingMode(options sdc.OptionMap) ([]byte, error) {
	mode := GetInterfaceNamingMode()
	namingModeResp := namingModeResponse{NamingMode: mode}
	return json.Marshal(namingModeResp)
}
