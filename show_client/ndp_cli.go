package show_client

import (
	"strconv"
	"strings"
	"syscall/js"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

type NeighborEntry struct {
	Address    string `json:"address"`     // IP address (IPv4 or IPv6)
	MacAddress string `json:"mac_address"` // MAC address of the neighbor
	Iface      string `json:"iface"`       // Interface name (e.g., Ethernet64, eth0)
	Vlan       string `json:"vlan"`        // VLAN ID (or "-" if not applicable)
	Status     string `json:"status"`      // Neighbor state (REACHABLE, STALE, etc.)
}

type NeighborTable struct {
	TotalEntries int             `json:"total_entries"` // Number of entries
	Entries      []NeighborEntry `json:"entries"`       // List of neighbor entries
}

// TODO: handle error cases
/*
show ndp [OPTIONS] [IP6ADDRESS] -> nbrshow -6 [-ip IPADDR] [-if IFACE] -> ip -6 neigh show [IPADDR] dev [IFACE]
admin@str4-7060x6-512-1:~$ show ndp --help
Usage: show ndp [OPTIONS] [IP6ADDRESS]

  Show IPv6 Neighbour table

Options:
  -if, --iface TEXT
  --verbose          Enable verbose output
  -h, -?, --help     Show this message and exit.
admin@str4-7060x6-512-1:~$ /bin/ip -6 neigh show fc00::5a2 dev Ethernet360 lladdr 0a:80:32:98:97:95 router REACHABLE fe80::d494:e8ff:fe96:e188 dev Ethernet392 lladdr d6:94:e8:96:e1:88 REACHABLE fc00::202 dev Ethernet128 lladdr a6:da:cf:f5:6a:e6 router REACHABLE fe80::7a5f:6cff:fe30:d7dc dev Vlan1000 lladdr 78:5f:6c:30:d7:dc router STALE fe80::bace:f6ff:fee5:51c0 dev Vlan1000 lladdr b8:ce:f6:e5:51:c0 REACHABLE fe80::acaf:aeff:fe2e:4080 dev Ethernet128 lladdr ae:af:ae:2e:40:80 REACHABLE fe80::bace:f6ff:fee5:51c8 dev Vlan1000 lladdr b8:ce:f6:e5:51:c8 REACHABLE fe80::7c4f:56ff:feb2:61b8 dev Ethernet440 lladdr 7e:4f:56:b2:61:b8
admin@str4-7060x6-512-2:~$ show ndp
Address                       MacAddress         Iface           Vlan    Status
----------------------------  -----------------  --------------  ------  ---------
2a01:111:e210:b000::a40:f66f  dc:f4:01:e6:54:a9  eth0            -       STALE
2a01:111:e210:b000::a40:f77e  56:aa:a6:3f:f4:91  eth0            -       STALE
fc00::1b2                     e2:85:9a:1a:43:a1  Ethernet120     -       REACHABLE
fc00::1e2                     02:8a:73:68:05:d8  Ethernet144     -       REACHABLE
fc00::2                       6e:1f:37:5f:bf:26  Ethernet0       -       REACHABLE
fc00::2a2                     f2:f7:cb:68:43:2d  Ethernet192     -       REACHABLE
fc00::2c2                     ae:1c:2f:2f:ab:60  Ethernet200     -       REACHABLE
fc00::2e2                     a6:f0:40:18:a6:a5  Ethernet208     -       REACHABL
*/

// show ndp is read from 'ip -6 neigh show' output from kernel
var (
	baseNdpCmd = "/bin/ip -6 neigh show"
)

func contains(arr []string, val string) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}

func parseNDPOutput(output string) NeighborTable {
	table := NeighborTable{}

	if strings.TrimSpace(output) == "" {
		return table
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if !contains(fields, "lladdr") {
			continue
		}

		var address, mac, iface, vlan, status string

		address = fields[0]
		for i := 0; i < len(fields); i++ {
			if fields[i] == "dev" && i+1 < len(fields) {
				iface = fields[i+1]
			}
			if fields[i] == "lladdr" && i+1 < len(fields) {
				mac = fields[i+1]
			}
		}

		// Derive VLAN from interface name if it starts with "Vlan"
		vlan = "-"
		if strings.HasPrefix(iface, "Vlan") {
			vlanNumStr := strings.TrimPrefix(iface, "Vlan")
			if vlanNum, err := strconv.Atoi(vlanNumStr); err == nil {
				vlan = strconv.Itoa(vlanNum)
			}
		}

		// Get Status (last field)
		status = fields[len(fields)-1]

		entry := NeighborEntry{
			Address:    address,
			MacAddress: mac,
			Iface:      iface,
			Vlan:       vlan,
			Status:     status,
		}

		table.Entries = append(table.Entries, entry)
	}

	table.TotalEntries = len(table.Entries)
	return table
}

func getNDP(options sdc.OptionMap) ([]byte, error) {
	intf, _ := options["interface"].String()
	ip, _ := options["ipaddress"].String()

	queries := [][]string{
		{"ASIC_DB", "ASIC_STATE", "SAI_OBJECT_TYPE_BRIDGE_PORT"},
	}
	brPortStr, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get SAI_OBJECT_TYPE_BRIDGE_PORT list from ASIC_DB: %v", err)
		return nil, err
	}
	log.Infof("data from query: %v", brPortStr)
	cmd := baseNdpCmd
	if ip != "" {
		cmd += " " + ip
	}
	if intf != "" {
		cmd += " dev " + intf
	}
	log.V(2).Infof("Running command: %s", cmd)

	cmdOutput, err := GetDataFromHostCommand(cmd)
	if err != nil {
		log.Errorf("Error getting NDP data: %v", err)
		return nil, err
	}

	// If cmdOutput is empty
	if strings.TrimSpace(cmdOutput) == "" {
		return []byte(`{"TotalEntries":0,"Entries":[]}`), nil
	}

	log.Infof("ndp output: %s", cmdOutput)
	// Parse the output
	table := parseNDPOutput(cmdOutput)
	log.Infof("parsed table: %v", table)
	// Convert to JSON
	jsonData, err := js.Marshal(table)
	if err != nil {
		return nil, err
	}
	return jsonData, nil
}
