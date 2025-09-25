package show_client

import (
	"encoding/json"
	"strings"

	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// Struct to represent each ARP entry
type ArpEntry struct {
	Address    string `json:"address"`
	MacAddress string `json:"mac_address"`
	Iface      string `json:"iface"`
	Vlan       string `json:"vlan"`
}

var (
	CmdPrefix         = "nbrshow -4"
	IPFlag            = "-ip"
	IFaceFlag         = "-if"
	OutputFieldsCount = 4
)

func getArpTable(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	namingModeStr, _ := options[SonicCliIfaceMode].String()
	namingMode, err := common.ParseInterfaceNamingMode(namingModeStr)
	if err != nil {
		return nil, err
	}
	cmd := CmdPrefix

	if len(args) > 0 && args[0] != "" {
		ip, err := common.ParseIPv4(args[0])
		if err != nil {
			return nil, err
		}
		cmd += " " + IPFlag + " " + ip.String()
	}

	if ifaceVal, ok := options["iface"]; ok {
		if ifaceStr, ok := ifaceVal.String(); ok && ifaceStr != "" {
			if !strings.HasPrefix(ifaceStr, "PortChannel") && !strings.HasPrefix(ifaceStr, "eth") {
				var err error
				ifaceStr, err = common.TryConvertInterfaceNameFromAlias(ifaceStr, namingMode)
				if err != nil {
					return nil, err
				}
			}
			cmd += " " + IFaceFlag + " " + ifaceStr
		}
	}

	output, err := common.GetDataFromHostCommand(cmd)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(output, "\n")
	entries := []ArpEntry{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		fields := strings.Fields(line)

		// Skip headers, separators, summaries, or malformed lines
		if len(fields) < OutputFieldsCount ||
			strings.HasPrefix(fields[0], "-") ||
			strings.Contains(line, "Address") ||
			strings.Contains(line, "Total number of entries") {
			continue
		}

		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}

		entries = append(entries, ArpEntry{
			Address:    fields[0],
			MacAddress: fields[1],
			Iface:      fields[2],
			Vlan:       fields[3],
		})
	}

	return json.Marshal(map[string][]ArpEntry{
		"arp_entries": entries,
	})
}
