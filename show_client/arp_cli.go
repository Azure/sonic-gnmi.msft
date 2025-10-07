package show_client

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
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
	CmdPrefix = "/usr/sbin/arp -n"
	IFaceFlag = "-i"
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

	fdbEntries, err := common.FetchFDBData()
	if err != nil {
		return nil, err
	}

	entries := mergeNbrWithFDB(nbrdata, fdbEntries)

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
	lines := strings.Split(output, "\n")
	if len(lines) <= 1 {
		return nbrdata
	}
	for _, line := range lines[1:] {
		if !strings.Contains(line, "ether") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		var address, mac, iface string
		address = fields[0]
		for i, f := range fields {
			if f == "ether" && i+1 < len(fields) {
				mac = fields[i+1]
			}
		}
		iface = fields[len(fields)-1]
		nbrdata = append(nbrdata, []string{address, mac, iface})
	}
	return nbrdata
}

func mergeNbrWithFDB(nbrdata [][]string, fdbEntries []common.BridgeMacEntry) []ARPEntry {
	var output []ARPEntry
	vlanRe := regexp.MustCompile(`^Vlan(\d+)$`)

	// Build lookup map: key = "vlanID|MAC", value = IfName
	lookup := make(map[string]string)
	for _, fdb := range fdbEntries {
		key := fmt.Sprintf("%d|%s", fdb.VlanID, strings.ToUpper(fdb.Mac))
		lookup[key] = fdb.IfName
	}

	for _, ent := range nbrdata {
		if len(ent) < 3 {
			continue
		}

		vlan := "-"
		if vlanRe.MatchString(ent[2]) {
			match := vlanRe.FindStringSubmatch(ent[2])
			if len(match) < 2 {
				continue
			}
			vlanid := match[1]
			mac := strings.ToUpper(ent[1])
			key := fmt.Sprintf("%s|%s", vlanid, mac)
			port, ok := lookup[key]
			if !ok {
				port = "-"
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
