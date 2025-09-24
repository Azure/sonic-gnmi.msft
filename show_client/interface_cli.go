package show_client

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"sort"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	oper_field  = "oper_status"
	admin_field = "admin_status"
	alias_field = "alias"
	desc_field  = "description"
)

type interfaceDescriptionDetails struct {
	Admin       string `json:"Admin"`
	Alias       string `json:"Alias"`
	Description string `json:"Description"`
	Oper        string `json:"Oper"`
}

type interfaceDescription map[string]interfaceDescriptionDetails

type namingModeResponse struct {
	NamingMode string `json:"naming_mode"`
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

func loadDescriptionFromInterfaceDetails(interfaceConfig map[string]interface{}, interfaceDetails map[string]interface{}, interfaceName string) interfaceDescription {
	description := make(interfaceDescription)
	var parsedKey string

	for key, details := range interfaceDetails {
		splitKeys := strings.SplitN(key, ":", 2)
		if len(splitKeys) > 0 {
			parsedKey = strings.TrimSpace(splitKeys[0])
		} else {
			continue
		}

		if interfaceName != "" && parsedKey != interfaceName {
			continue
		}

		if _, ok := interfaceConfig[parsedKey]; ok {
			if detailsMap, retValue := details.(map[string]interface{}); retValue {
				description[parsedKey] = interfaceDescriptionDetails{
					Oper:        detailsMap[oper_field].(string),
					Admin:       detailsMap[admin_field].(string),
					Alias:       detailsMap[alias_field].(string),
					Description: detailsMap[desc_field].(string),
				}
			}
		}
	}

	return description
}

func getInterfacesDescription(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	intf, ok := options["interface"].String()
	var interfaceName string

	if ok {
		interfaceName = intf
	}

	queries := [][]string{{"CONFIG_DB", "PORT"}}
	interfaceConfig, err := common.GetMapFromQueries(queries)
	if err != nil {
		return []byte(""), err
	}

	queries = [][]string{{"APPL_DB", "PORT_TABLE"}}

	interfaceDetails, err := common.GetMapFromQueries(queries)
	if err != nil {
		return []byte(""), err
	}

	interfaceDesc := loadDescriptionFromInterfaceDetails(interfaceConfig, interfaceDetails, interfaceName)

	return json.Marshal(interfaceDesc)
}

func getInterfaceErrors(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	intf := args.At(0)
	if intf == "" {
		return nil, status.Errorf(codes.InvalidArgument, "No interface name passed in as option")
	}
	// Query Port Operational Errors Table from STATE_DB
	queries := [][]string{
		{"STATE_DB", "PORT_OPERR_TABLE", intf},
	}
	portErrorsTbl, _ := common.GetMapFromQueries(queries)
	portErrorsTbl = common.RemapAliasToPortName(portErrorsTbl)

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
		{"CONFIG_DB", common.ConfigDBPortTable},
	}
	portTable, err := common.GetMapFromQueries(queries)
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

func getInterfaceFecStatus(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	intf := args.At(0)

	ports, err := getIntfsFromConfigDB(intf)
	if err != nil {
		log.Errorf("Failed to get front panel ports: %v", err)
		return nil, err
	}
	ports = common.NatsortInterfaces(ports)

	portFecStatus := make([]map[string]string, 0, len(ports)+1)
	for i := range ports {
		port := ports[i]
		adminFecStatus := ""
		operStatus := ""
		operFecStatus := ""

		// Query port admin FEC status and operation status from APPL_DB
		queries := [][]string{
			{"APPL_DB", common.AppDBPortTable, port},
		}
		data, err := common.GetMapFromQueries(queries)
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
			{"STATE_DB", common.StateDBPortTable, port},
		}
		data, err = common.GetMapFromQueries(queries)
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
		{"CONFIG_DB", common.ConfigDBPortChannelTable},
	}
	portTable, err := common.GetMapFromQueries(queries)
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
	data, err := common.GetMapFromQueries(queries)
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
		{"STATE_DB", common.StateDBPortTable, intf},
	}
	stateData, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get state for port %s from STATE_DB: %v", intf, err)
		return "N/A"
	}

	queries = [][]string{
		{"APPL_DB", common.AppDBPortTable, intf},
	}
	appData, err := common.GetMapFromQueries(queries)
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
	vlanMemberTable, err := common.GetMapFromQueries(queries)
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
	portChannelMemberTable, err := common.GetMapFromQueries(queries)
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
		portData, err := common.GetMapFromQueries(queries)
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
	vlanMemberTable, err := common.GetMapFromQueries(queries)
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
		portData, err := common.GetMapFromQueries(queries)
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
	portChannelMemberTable, err := common.GetMapFromQueries(queries)
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
	portTable, err := common.GetMapFromQueries(queries)
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
	ports = common.NatsortInterfaces(ports)

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

func getInterfaceStatus(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	isSubIntf := false
	intf := args.At(0)
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
	ports = common.NatsortInterfaces(ports)
	portchannels = common.NatsortInterfaces(portchannels)
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
			{"APPL_DB", common.AppDBPortTable, port},
		}
		data, err := common.GetMapFromQueries(queries)
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
			{"APPL_DB", common.AppDBPortChannelTable, port},
		}
		data, err := common.GetMapFromQueries(queries)
		if err != nil {
			log.Errorf("Failed to get status for portchannel %s: %v", port, err)
			return nil, err
		}

		// Query portchannel config from CONFIG_DB
		queries = [][]string{
			{"CONFIG_DB", common.ConfigDBPortChannelTable, port},
		}
		config, err := common.GetMapFromQueries(queries)
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

func getInterfaceAlias(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	intf := args.At(0)
	namingMode, _ := options[SonicCliIfaceMode].String()

	// Read CONFIG_DB.PORT
	queries := [][]string{{"CONFIG_DB", "PORT"}}
	portEntries, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get ports from CONFIG_DB: %v", err)
		return nil, err
	}

	nameToAlias := make(map[string]string, len(portEntries))
	for name := range portEntries {
		alias := common.GetFieldValueString(portEntries, name, "", "alias")
		if alias == "" {
			// fallback to itself if alias field is missing
			alias = name
		}
		nameToAlias[name] = alias
	}

	// If a specific interface was requested, accept port name
	if intf != "" {
		intf, err := common.TryConvertInterfaceNameFromAlias(intf, namingMode)
		if err != nil {
			log.Errorf("Error: %v", err)
			return nil, err
		}

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

func getInterfaceSwitchportConfig(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	namingMode, _ := options[SonicCliIfaceMode].String()

	// Read CONFIG_DB tables
	portTbl, err := common.GetMapFromQueries([][]string{{"CONFIG_DB", "PORT"}})
	if err != nil {
		log.Errorf("Failed to get PORT: %v", err)
		return nil, err
	}
	portChannelTbl, err := common.GetMapFromQueries([][]string{{"CONFIG_DB", "PORTCHANNEL"}})
	if err != nil {
		log.Errorf("Failed to get PORTCHANNEL: %v", err)
		return nil, err
	}
	portChannelMemberTbl, err := common.GetMapFromQueries([][]string{{"CONFIG_DB", "PORTCHANNEL_MEMBER"}})
	if err != nil {
		log.Errorf("Failed to get PORTCHANNEL_MEMBER: %v", err)
		return nil, err
	}
	vlanMemberTbl, err := common.GetMapFromQueries([][]string{{"CONFIG_DB", "VLAN_MEMBER"}})
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
	keys = common.NatsortInterfaces(keys)

	// Build VLAN membership maps
	untaggedMap := make(map[string][]string)
	taggedMap := make(map[string][]string)
	for k := range vlanMemberTbl {
		vlan, ifname, ok := common.SplitCompositeKey(k)
		if !ok {
			continue
		}
		tagMode := common.GetFieldValueString(vlanMemberTbl, k, "", "tagging_mode")
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
			"Interface": common.GetInterfaceNameForDisplay(k, namingMode),
			"Mode":      mode,
			"Untagged":  strings.Join(untagged, ","),
			"Tagged":    strings.Join(tagged, ","),
		})
	}

	return json.Marshal(switchportConfig)
}

func getInterfaceSwitchportStatus(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	namingMode, _ := options[SonicCliIfaceMode].String()

	// Read CONFIG_DB tables
	portTbl, err := common.GetMapFromQueries([][]string{{"CONFIG_DB", "PORT"}})
	if err != nil {
		log.Errorf("Failed to get PORT: %v", err)
		return nil, err
	}
	portChannelTbl, err := common.GetMapFromQueries([][]string{{"CONFIG_DB", "PORTCHANNEL"}})
	if err != nil {
		log.Errorf("Failed to get PORTCHANNEL: %v", err)
		return nil, err
	}
	portChannelMemberTbl, err := common.GetMapFromQueries([][]string{{"CONFIG_DB", "PORTCHANNEL_MEMBER"}})
	if err != nil {
		log.Errorf("Failed to get PORTCHANNEL_MEMBER: %v", err)
		return nil, err
	}
	vlanMemberTbl, err := common.GetMapFromQueries([][]string{{"CONFIG_DB", "VLAN_MEMBER"}})
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
	keys = common.NatsortInterfaces(keys)

	// Emit switchportStatus
	switchportStatus := make([]map[string]string, 0, len(keys))
	for _, k := range keys {
		mode := GetInterfaceSwitchportMode(portTbl, portChannelTbl, vlanMemberTbl, k)

		switchportStatus = append(switchportStatus, map[string]string{
			"Interface": common.GetInterfaceNameForDisplay(k, namingMode),
			"Mode":      mode,
		})
	}

	return json.Marshal(switchportStatus)
}

// GetInterfaceSwitchportMode returns the switchport mode.
func GetInterfaceSwitchportMode(
	portTbl, portChannelTbl, vlanMemberTbl map[string]interface{},
	name string,
) string {
	if m := common.GetFieldValueString(portTbl, name, "", "mode"); m != "" {
		return m
	}
	if m := common.GetFieldValueString(portChannelTbl, name, "", "mode"); m != "" {
		return m
	}
	for k := range vlanMemberTbl {
		_, member, ok := common.SplitCompositeKey(k)
		if ok && member == name {
			return "trunk"
		}
	}
	return "routed"
}

// IsInterfaceInPortchannel reports whether interfaceName is a member of any portchannel.
func IsInterfaceInPortchannel(portchannelMemberTable map[string]interface{}, interfaceName string) bool {
	if portchannelMemberTable == nil || interfaceName == "" {
		return false
	}
	for k := range portchannelMemberTable {
		_, member, ok := common.SplitCompositeKey(k)
		if ok && member == interfaceName {
			return true
		}
	}
	return false
}

func getInterfaceFlap(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	intf := args.At(0)
	namingMode, _ := options[SonicCliIfaceMode].String()

	// Query APPL_DB PORT_TABLE
	queries := [][]string{
		{common.ApplDb, common.AppDBPortTable},
	}
	portTable, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get PORT_TABLE: %v", err)
		return nil, err
	}

	// Collect ports (optionally filter by interface)
	var ports []string
	if intf != "" {
		intf, err := common.TryConvertInterfaceNameFromAlias(intf, namingMode)
		if err != nil {
			log.Errorf("Error: %v", err)
			return nil, err
		}
		if _, ok := portTable[intf]; !ok {
			return nil, fmt.Errorf("Invalid interface name %s", intf)
		}
		ports = []string{intf}
	} else {
		for p := range portTable {
			ports = append(ports, p)
		}
	}
	ports = common.NatsortInterfaces(ports)

	// Build rows
	rows := make([]map[string]string, 0, len(ports))
	for _, p := range ports {
		flapCount := common.GetFieldValueString(portTable, p, "Never", "flap_count")
		adminStatus := common.GetFieldValueString(portTable, p, "Unknown", "admin_status")
		operStatus := common.GetFieldValueString(portTable, p, "Unknown", "oper_status")
		// Capitalize like Python's .capitalize()
		if adminStatus != "" {
			adminStatus = common.Capitalize(adminStatus)
		}
		if operStatus != "" {
			operStatus = common.Capitalize(operStatus)
		}
		lastDown := common.GetFieldValueString(portTable, p, "Never", "last_down_time")
		lastUp := common.GetFieldValueString(portTable, p, "Never", "last_up_time")

		rows = append(rows, map[string]string{
			"Interface":                p,
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
func getInterfaceNeighborExpected(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	intf := args.At(0)
	namingMode, _ := options[SonicCliIfaceMode].String()

	neighborTbl, err := common.GetMapFromQueries([][]string{{"CONFIG_DB", "DEVICE_NEIGHBOR"}})
	if err != nil {
		log.Errorf("Failed to get DEVICE_NEIGHBOR: %v", err)
		return nil, err
	}
	metaTbl, err := common.GetMapFromQueries([][]string{{"CONFIG_DB", "DEVICE_NEIGHBOR_METADATA"}})
	if err != nil {
		log.Errorf("Failed to get DEVICE_NEIGHBOR_METADATA: %v", err)
		return nil, err
	}

	buildEntry := func(canonIf string) (map[string]string, bool) {
		device := common.GetFieldValueString(neighborTbl, canonIf, "", "name")
		if device == "" {
			return nil, false
		}
		// Require metadata key to exist (python try/except KeyError: pass)
		if _, ok := metaTbl[device]; !ok {
			return nil, false
		}

		remotePort := common.GetFieldValueString(neighborTbl, canonIf, "None", "port")
		if remotePort == "" {
			remotePort = "None"
		}
		loopback := common.GetFieldValueString(metaTbl, device, "None", "lo_addr")
		if loopback == "" {
			loopback = "None"
		}
		mgmt := common.GetFieldValueString(metaTbl, device, "None", "mgmt_addr")
		if mgmt == "" {
			mgmt = "None"
		}
		ntype := common.GetFieldValueString(metaTbl, device, "None", "type")
		if ntype == "" {
			ntype = "None"
		}

		return map[string]string{
			"Neighbor":         device,
			"NeighborPort":     remotePort,
			"NeighborLoopback": loopback,
			"NeighborMgmt":     mgmt,
			"NeighborType":     ntype,
		}, true
	}

	canonicalKeys := make([]string, 0, len(neighborTbl))
	for k := range neighborTbl {
		canonicalKeys = append(canonicalKeys, k)
	}
	canonicalKeys = common.NatsortInterfaces(canonicalKeys)

	finalMap := make(map[string]map[string]string, len(canonicalKeys))
	for _, c := range canonicalKeys {
		if entry, ok := buildEntry(c); ok {
			key := c
			if namingMode == "alias" {
				key = common.GetInterfaceNameForDisplay(c, namingMode)
			}
			finalMap[key] = entry
		}
	}

	if intf != "" {
		entry, ok := finalMap[intf]
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid interface name %s", intf)
		}
		return json.Marshal(map[string]map[string]string{intf: entry})
	}

	return json.Marshal(finalMap)
}

func getInterfaceNamingMode(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	namingMode, _ := options[SonicCliIfaceMode].String()

	namingMode = common.GetInterfaceNamingMode(namingMode)
	namingModeResp := namingModeResponse{NamingMode: namingMode}
	return json.Marshal(namingModeResp)
}
