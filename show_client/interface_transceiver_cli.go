package show_client

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

func getAllPortsFromConfigDB() ([]string, error) {
	queries := [][]string{
		{"CONFIG_DB", "PORT"},
	}
	data, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to get data from CONFIG_DB queries %v, got err: %v", queries, err)
		return nil, err
	}
	log.V(6).Infof("Data from CONFIG_DB: %v", data)

	ports := make([]string, 0, len(data))
	for iface := range data {
		ports = append(ports, iface)
	}
	return ports, nil
}

func getTransceiverErrorStatus(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	intf := args.At(0)

	var queries [][]string
	if intf == "" {
		queries = [][]string{
			{"STATE_DB", "TRANSCEIVER_STATUS_SW"},
		}
	} else {
		queries = [][]string{
			{"STATE_DB", "TRANSCEIVER_STATUS_SW", intf},
		}
	}

	data, err := GetDataFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to get data from queries %v, got err: %v", queries, err)
		return nil, err
	}
	return data, nil
}

func getInterfaceTransceiverPresence(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	intf := args.At(0)

	// Get STATE_DB transceiver info
	queries := [][]string{
		{"STATE_DB", "TRANSCEIVER_INFO"},
	}
	data, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to get transceiver data from STATE_DB queries %v, got err: %v", queries, err)
		return nil, err
	}
	log.V(6).Infof("TRANSCEIVER_INFO Data from STATE_DB: %v", data)

	status := make(map[string]string)

	if intf != "" {
		// If specific interface provided, skip ConfigDB check
		if _, exist := data[intf]; exist {
			status[intf] = "Present"
		} else {
			status[intf] = "Not Present"
		}
	} else {
		// No specific interface provided, get all from ConfigDB
		ports, err := getAllPortsFromConfigDB()
		if err != nil {
			log.Errorf("Unable to get all ports from CONFIG_DB, %v", err)
			return nil, err
		}

		for _, port := range ports {
			if _, exist := data[port]; exist {
				status[port] = "Present"
			} else {
				status[port] = "Not Present"
			}
		}
	}

	log.V(6).Infof("Transceiver presence status: %v", status)
	return json.Marshal(status)
}

func getEEPROM(args sdc.CmdArgs, options sdc.OptionMap) (map[string]interface{}, error) {
	intf := args.At(0)

	var dumpDom bool
	if v, ok := options["dom"].Bool(); ok {
		dumpDom = v
	}

	var queries [][]string
	queries = [][]string{
		{"APPL_DB", "PORT_TABLE"},
	}

	portTable, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to pull data for queries %v, got err %v", queries, err)
		return nil, err
	}

	intfEEPROM := make(map[string]interface{})
	for iface := range portTable {
		if intf != "" && iface != intf {
			continue
		}

		role := GetFieldValueString(portTable, iface, defaultMissingCounterValue, "role")
		if common.IsFrontPanelPort(iface, role) {
			intfEEPROM[iface] = convertInterfaceSfpInfoToCliOutputString(iface, dumpDom)
		} else {
			continue
		}

	}
	return intfEEPROM, nil
}

func getTransceiverInfo(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	intfEEPROM, err := getEEPROM(args, options)
	if err != nil {
		return nil, err
	}

	return json.Marshal(intfEEPROM)
}

type portLpmode struct {
	Port   string `json:"Port"`
	Lpmode string `json:"Low-power Mode"`
}

func getInterfaceTransceiverLpMode(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	intf := args.At(0)
	cmdParts := []string{"sudo", "sfputil", "show", "lpmode"}
	if intf != "" {
		cmdParts = append(cmdParts, "-p", intf)
	}
	cmdStr := strings.Join(cmdParts, " ")

	output, err := common.GetDataFromHostCommand(cmdStr)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(output, "\n")
	entries := make([]portLpmode, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Port") || strings.HasPrefix(line, "---") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		port := fields[0]
		mode := fields[1]
		ml := strings.ToLower(mode)
		if ml == "on" || ml == "off" {
			mode = strings.Title(ml)
		}
		entries = append(entries, portLpmode{Port: port, Lpmode: mode})
	}

	return json.Marshal(entries)
}

func getInterfaceTransceiverStatus(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	intfArg := args.At(0)
	namingMode, _ := options[SonicCliIfaceMode].String()

	// APPL_DB PORT_TABLE -> determine valid ports
	portTable, err := GetMapFromQueries([][]string{{ApplDb, AppDBPortTable}})
	if err != nil {
		return nil, fmt.Errorf("failed to read PORT_TABLE: %w", err)
	}

	var ports []string
	if intfArg != "" {
		interfaceName, err := TryConvertInterfaceNameFromAlias(intfArg, namingMode)
		if err != nil {
			return nil, fmt.Errorf("alias conversion failed for %s: %w", intfArg, err)
		}
		if _, ok := portTable[interfaceName]; !ok {
			return nil, fmt.Errorf("invalid interface name %s", intfArg)
		}
		ports = []string{interfaceName}
	} else {
		for p := range portTable {
			ports = append(ports, p)
		}
		ports = NatsortInterfaces(ports)
	}

	result := make(map[string]string, len(ports))

	for _, p := range ports {
		if ok, _ := common.IsValidPhysicalPort(p); !ok {
			continue
		}
		statusStr := convertInterfaceSfpStatusToCliOutputString(p)
		result[p] = statusStr
	}

	return json.Marshal(result)
}
