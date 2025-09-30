package show_client

import (
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

type ARPEntry struct {
	Address    string `json:"address"`
	MacAddress string `json:"mac_address"`
	Iface      string `json:"iface"`
	Vlan       string `json:"vlan"`
}

type ARPResponse struct {
	Entries         []ARPEntry `json:"entries"`
	TotalEntryCount int        `json:"total_entries"`
}

var (
	CmdPrefix         = "/usr/sbin/arp -n"
	IFaceFlag         = "-i"
	OutputFieldsCount = 4
)

func getARP(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
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
		cmd += " " + ip.String()
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

	rawOutput, err := common.GetDataFromHostCommand(cmd)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(rawOutput) == "" {
		return []byte(`{"entries":[],"total_entries":0}`), nil
	}
	nbrdata := parseNbrData(rawOutput)

	bridgeMacList, err := common.FetchFDBData()
	if err != nil {
		return nil, err
	}

	entries := mergeNbrWithFDB(nbrdata, bridgeMacList)

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Address < entries[j].Address
	})

	response := ARPResponse{
		Entries:         entries,
		TotalEntryCount: len(entries),
	}

	if response.Entries == nil {
		response.Entries = []ARPEntry{}
	}

	return json.Marshal(response)
}

func parseNbrData(output string) [][]string {
	var nbrdata [][]string
	for _, line := range strings.Split(output, "\n")[1:] {
		if !strings.Contains(line, "ether") {
			continue
		}
		var ent []string
		fields := strings.Fields(line)
		for i := 0; i < len(fields); i += 2 {
			ent = append(ent, fields[i])
		}
		nbrdata = append(nbrdata, ent)
	}
	return nbrdata
}

func mergeNbrWithFDB(nbrdata [][]string, bridgeMacList []common.BridgeMacEntry) []ARPEntry {
	var output []ARPEntry

	for _, ent := range nbrdata {
		vlan := "-"
		if strings.Contains(ent[2], "Vlan") {
			re := regexp.MustCompile(`\d+`)
			vlanMatch := re.FindString(ent[2])
			vlanid := vlanMatch
			mac := strings.ToUpper(ent[1])

			port := "-"
			for _, fdb := range bridgeMacList {
				if strconv.Itoa(fdb.VlanID) == vlanid && strings.ToUpper(fdb.Mac) == mac {
					port = fdb.IfName
					break
				}
			}

			entry := ARPEntry{
				Address:    ent[0],
				MacAddress: ent[1],
				Iface:      port,
				Vlan:       vlanid,
			}
			output = append(output, entry)
		} else {
			entry := ARPEntry{
				Address:    ent[0],
				MacAddress: ent[1],
				Iface:      ent[2],
				Vlan:       vlan,
			}
			output = append(output, entry)
		}
	}

	return output
}
