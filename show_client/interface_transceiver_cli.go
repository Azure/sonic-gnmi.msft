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

func getTransceiverErrorStatus(options sdc.OptionMap) ([]byte, error) {
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

func getInterfaceTransceiverPresence(options sdc.OptionMap) ([]byte, error) {
	intf, _ := options["interface"].String()

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

// admin@str3-t0-8102-smartswitch-01:~$ show interfaces transceiver lpmode
// Port         Low-power Mode
// -----------  ----------------
// Ethernet0    Off
// Ethernet8    Off
// admin@str3-t0-8102-smartswitch-01:~$ sudo sfputil show lpmode -p Ethernet160
// Port         Low-power Mode
// -----------  ----------------
// Ethernet160  Off
func getInterfaceTransceiverLpmode(options sdc.OptionMap) ([]byte, error) {
	portNameToAlias, err := getLogicalToPhysicalPortMap()
	if err != nil {
		log.Errorf("Unable to get logical to physical port map, %v", err)
		return nil, err
	}

	v, ok := options["interface"].String()
	if !ok || v == "" {
		lpmode := getInterfaceTransceiverLpmode()
	} else {
		_, exist := portNameToAlias[v]
		if !exist {
			err = fmt.Errorf("Interface %s not found in CONFIG_DB PORT", v)
			log.Errorf(err.Error())
			return nil, err
		} else {
			lpmode := getInterfaceTransceiverLpmode(portNameToAlias[v])
		}
	}

	return json.Marshal(lpmode)
}

func getInterfaceTransceiverLpmode(physicalItfName string) ([]byte, error) {
	cmdStr := "sudo sfputil show lpmode"
	if physicalItfName != "" {
		cmdStr += " -p " + physicalItfName
	}

	out, err := GetDataFromHostCommand(cmdStr)
	if err != nil {
		log.Errorf("Unable to successfully execute command %v, got err %v", cmdStr, err)
		return nil, err
	}

	lpmode := make(map[string]string)
	lines := strings.Split(out, "\n")
	headerSkipped := false
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if !headerSkipped {
			headerSkipped = true
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		// Skip separator line like: "-----------  ----------------"
		if strings.Trim(fields[0], "-") == "" {
			continue
		}
		port := fields[0]
		status := fields[len(fields)-1]
		lpmode[port] = status
	}

	_, exist := lpmode[logicalItfName]
	if !exist {
		lpmode[logicalItfName] = "N/A"
	}

	return json.Marshal(lpmode)
}

func getLogicalToPhysicalPortMap() (map[string]string, error) {
	portMap := make(map[string]string)

	queries := [][]string{
		{"CONFIG_DB", "PORT"},
	}
	portEntries, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to get data from CONFIG_DB PORT with queries %v, got err: %v", queries, err)
		return nil, err
	}
	log.V(6).Infof("Data from CONFIG_DB.PORT: %v", portEntries)

	for name := range portEntries {
		alias := GetFieldValueString(portEntries, name, "", "alias")
		if alias == "" {
			alias = name
		}
		portMap[name] = alias
	}

	return portMap, nil
}
