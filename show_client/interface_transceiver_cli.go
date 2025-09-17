package show_client

import (
	"encoding/json"
	"strings"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

func getAllPortsFromConfigDB() ([]string, error) {
	queries := [][]string{
		{"CONFIG_DB", "PORT"},
	}
	data, err := GetMapFromQueries(queries)
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
	// TODO
	var intf string
	if v, ok := options["interface"].String(); ok {
		intf = v
	}

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
	// TODO
	var intf string
	if v, ok := options["interface"].String(); ok {
		intf = v
	}

	// Get STATE_DB transceiver info
	queries := [][]string{
		{"STATE_DB", "TRANSCEIVER_INFO"},
	}
	data, err := GetMapFromQueries(queries)
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

	output, err := GetDataFromHostCommand(cmdStr)
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

func querySfpPM(intf string) map[string]string {
	return map[string]string{
		"name":   intf,
		"status": "Transceiver status info not applicable",
	}
	// TODO: Implement the logic after we find a device that has transceiver performance monitoring enabled
	// firstSubport := getFirstSubPort(intf)
	// if firstSubport == "" {
	// 	log.Errorf("Unable to get first subport for %v while converting SFP status", intf)
	// 	return map[string]string{
	// 		"name":   intf,
	// 		"status": "Transceiver status info not applicable",
	// 	}
	// }

	// // Query PM info from STATE_DB
	// queries := [][]string{
	// 	{"STATE_DB", "TRANSCEIVER_PM", intf},
	// }
	// sfpPM, err := GetMapFromQueries(queries)
	// if err != nil {
	// 	log.Errorf("Failed to get PM dict from STATE_DB: %v", err)
	// 	return nil, err
	// }

	// // Query threshold info from STATE_DB
	// queries = [][]string{
	// 	{"STATE_DB", "TRANSCEIVER_DOM_THRESHOLD", intf},
	// }
	// sfpThreshold, err := GetMapFromQueries(queries)
	// if err != nil {
	// 	log.Errorf("Failed to get PM dict from STATE_DB: %v", err)
	// 	return nil, err
	// }
}

func getInterfaceTransceiverPM(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	intf := args.At(0)

	result := make([]map[string]string, 0)
	if intf != "" {
		result = append(result, querySfpPM(intf))
	} else {
		queries := [][]string{
			{"APPL_DB", AppDBPortTable},
		}
		portTable, err := GetMapFromQueries(queries)
		if err != nil {
			log.Errorf("Failed to get interface list from APPL_DB: %v", err)
			return nil, err
		}

		ports := make([]string, 0, len(portTable))
		for key := range portTable {
			ports = append(ports, key)
		}
		ports = NatsortInterfaces(ports)

		for _, p := range ports {
			if ok, _ := isValidPhysicalPort(p); ok {
				result = append(result, querySfpPM(p))
			}
		}
	}

	return json.Marshal(result)
}
