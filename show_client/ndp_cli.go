package show_client

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
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

const oidPrefixLen = len("oid:0x")

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

type BridgeMacEntry struct {
	VlanID int
	Mac    string
	IfName string
}

func parseNDPOutput(output string, intf string) NeighborTable {
	table := NeighborTable{}

	// Fetch FDB entries
	bridgeMacList, err := fetchFdbData()
	if err != nil {
		log.Warningf("Failed to fetch FDB data: %v", err)
		bridgeMacList = []BridgeMacEntry{} // fallback to empty
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if !common.ContainsString(fields, "lladdr") {
			continue
		}

		var address, mac, iface, vlan, status string
		address = fields[0]

		// Extract iface and mac
		for i := 0; i < len(fields); i++ {
			if fields[i] == "dev" && i+1 < len(fields) {
				iface = fields[i+1]
			}
			if fields[i] == "lladdr" && i+1 < len(fields) {
				mac = strings.ToUpper(fields[i+1])
			}
		}

		// When iface is explicitly specified, the kernel output omits the 'dev <iface>' field
		if iface == "" && intf != "" {
			iface = intf
		}

		// Derive VLAN from interface name if it starts with "Vlan"
		vlan = "-"
		vlanID := 0
		if strings.HasPrefix(iface, "Vlan") {
			vlanNumStr := strings.TrimPrefix(iface, "Vlan")
			if n, err := strconv.Atoi(vlanNumStr); err == nil {
				vlanID = n
				vlan = strconv.Itoa(n)
			}
		}

		// Try to match FDB entry to replace iface
		if vlanID != 0 && mac != "" {
			for _, fdb := range bridgeMacList {
				if fdb.VlanID == vlanID && strings.EqualFold(fdb.Mac, mac) {
					iface = fdb.IfName
					vlan = strconv.Itoa(fdb.VlanID)
					break
				}
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

func getInterfaceOidMap() (map[string]string, error) {
	portQueries := [][]string{
		{"COUNTERS_DB", "COUNTERS_PORT_NAME_MAP"},
	}
	lagQueries := [][]string{
		{"COUNTERS_DB", "COUNTERS_LAG_NAME_MAP"},
	}

	portMap, err := common.GetMapFromQueries(portQueries)
	if err != nil {
		return nil, err
	}
	lagMap, err := common.GetMapFromQueries(lagQueries)
	if err != nil {
		return nil, err
	}

	// SONiC interface regex patterns
	ethRe := regexp.MustCompile(`^Ethernet(\d+)$`)
	lagRe := regexp.MustCompile(`^PortChannel(\d+)$`)
	vlanRe := regexp.MustCompile(`^Vlan(\d+)$`)
	mgmtRe := regexp.MustCompile(`^eth(\d+)$`)

	ifOidMap := make(map[string]string)

	// helper closure to check valid names
	isValidIfName := func(name string) bool {
		return ethRe.MatchString(name) ||
			lagRe.MatchString(name) ||
			vlanRe.MatchString(name) ||
			mgmtRe.MatchString(name)
	}

	for portName, oidVal := range portMap {
		oidStr, ok := oidVal.(string)
		if !ok || len(oidStr) <= oidPrefixLen {
			continue
		}
		if isValidIfName(portName) {
			ifOidMap[oidStr[oidPrefixLen:]] = portName
		}
	}
	for lagName, oidVal := range lagMap {
		oidStr, ok := oidVal.(string)
		if !ok || len(oidStr) <= oidPrefixLen {
			continue
		}
		if isValidIfName(lagName) {
			ifOidMap[oidStr[oidPrefixLen:]] = lagName
		}
	}

	return ifOidMap, nil
}

func buildBvidToVlanMap() (map[string]string, error) {
	queries := [][]string{
		{"ASIC_DB", "ASIC_STATE:SAI_OBJECT_TYPE_VLAN:*"},
	}

	vlanData, err := common.GetMapFromQueries(queries)
	if err != nil {
		return nil, err
	}

	const prefix = "SAI_OBJECT_TYPE_VLAN:"
	result := make(map[string]string)

	for key, val := range vlanData {
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		bvid := strings.TrimPrefix(key, prefix) // "oid:..."

		ent, ok := val.(map[string]interface{})
		if !ok {
			log.Warningf("Unexpected format for VLAN entry %s: %#v", key, val)
			continue
		}

		if vlanIDRaw, ok := ent["SAI_VLAN_ATTR_VLAN_ID"]; ok {
			if vlanIDStr, ok := vlanIDRaw.(string); ok {
				result[bvid] = vlanIDStr
			}
		}
	}

	return result, nil
}

func getVlanIDFromBvid(bvid string, bvidMap map[string]string) (string, error) {
	if vlanID, ok := bvidMap[bvid]; ok {
		return vlanID, nil
	}
	return "", fmt.Errorf("BVID %s not found in VLAN map", bvid)
}

func getBridgePortMap() (map[string]string, error) {
	queries := [][]string{
		{"ASIC_DB", "ASIC_STATE:SAI_OBJECT_TYPE_BRIDGE_PORT:*"},
	}
	brPortStr, err := common.GetMapFromQueries(queries)
	if err != nil {
		return nil, err
	}
	log.V(6).Infof("SAI_OBJECT_TYPE_BRIDGE_PORT data from query: %v", brPortStr)

	ifBrOidMap := make(map[string]string)

	// key SAI_OBJECT_TYPE_BRIDGE_PORT:oid:0x2600000000063f
	for key, val := range brPortStr {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) < 2 {
			continue
		}
		if len(parts[1]) < oidPrefixLen {
			// not long enough to contain "oid:0x...", skip
			continue
		}
		bridgePortOid := parts[1][oidPrefixLen:] // strip "oid:0x"

		attrs, ok := val.(map[string]string)
		if !ok {
			// sometimes it might be map[string]interface{}, so try that
			if m, ok2 := val.(map[string]interface{}); ok2 {
				attrs = make(map[string]string)
				for k, v := range m {
					attrs[k] = fmt.Sprintf("%v", v)
				}
			} else {
				log.Warningf("Unexpected type for attrs: %T", val)
				continue
			}
		}
		// attrs is map[string]string
		portIdRaw, ok := attrs["SAI_BRIDGE_PORT_ATTR_PORT_ID"]
		if !ok {
			continue
		}
		portId := portIdRaw[oidPrefixLen:] // strip "oid:0x"
		// Map bridge port OID to port ID
		ifBrOidMap[bridgePortOid] = portId
	}
	return ifBrOidMap, nil
}

func fetchFdbData() ([]BridgeMacEntry, error) {
	queries := [][]string{
		{"ASIC_DB", "ASIC_STATE:SAI_OBJECT_TYPE_FDB_ENTRY:*"},
	}

	// "ASIC_STATE:SAI_OBJECT_TYPE_FDB_ENTRY:{\"bvid\":\"oid:0x2600000000063f\",\"mac\":\"B8:CE:F6:E5:50:05\",\"switch_id\":\"oid:0x21000000000000\"}"
	brPortStr, err := common.GetMapFromQueries(queries)
	if err != nil {
		return nil, err
	}
	log.V(6).Infof("FDB_ENTRY list: %v", brPortStr)

	ifOidMap, err := getInterfaceOidMap()
	if err != nil {
		return nil, err
	}

	ifBrOidMap, err := getBridgePortMap()
	if err != nil {
		return nil, err
	}

	if ifBrOidMap == nil || ifOidMap == nil {
		return nil, fmt.Errorf("bridge/port maps not initialized")
	}

	bvidMap, err := buildBvidToVlanMap()
	if err != nil {
		log.Warningf("Failed to build BVID map: %v", err)
		return nil, err
	}

	bridgeMacList := []BridgeMacEntry{}

	// fdbKey is like SAI_OBJECT_TYPE_FDB_ENTRY:{"bvid":"oid:0x2600000000063f","mac":"B8:CE:F6:E5:50:05","switch_id":"oid:0x21000000000000"}
	for fdbKey, entryData := range brPortStr {
		// Split at first colon to separate top-level type from JSON
		idx := strings.Index(fdbKey, ":")
		if idx == -1 || idx+1 >= len(fdbKey) {
			continue
		}
		fdbJSON := fdbKey[idx+1:] // everything after the first colon

		fdb := map[string]string{}
		if err := json.Unmarshal([]byte(fdbJSON), &fdb); err != nil {
			continue
		}

		// Attributes map
		ent, ok := entryData.(map[string]interface{})
		if !ok {
			continue
		}

		brPortOidRaw, ok := ent["SAI_FDB_ENTRY_ATTR_BRIDGE_PORT_ID"].(string)
		if !ok || len(brPortOidRaw) <= oidPrefixLen {
			continue
		}
		brPortOid := brPortOidRaw[oidPrefixLen:]

		portID, ok := ifBrOidMap[brPortOid]
		if !ok {
			continue
		}

		ifName, ok := ifOidMap[portID]
		if !ok {
			ifName = portID
		}

		var vlanIDStr string
		if v, ok := fdb["vlan"]; ok {
			vlanIDStr = v
		} else if bvid, ok := fdb["bvid"]; ok {
			vlanIDStr, err = getVlanIDFromBvid(bvid, bvidMap)
			if err != nil || vlanIDStr == "" {
				continue
			}
		} else {
			continue
		}

		vlanID, err := strconv.Atoi(vlanIDStr)
		if err != nil {
			continue
		}

		bridgeMacList = append(bridgeMacList, BridgeMacEntry{
			VlanID: vlanID,
			Mac:    fdb["mac"],
			IfName: ifName,
		})
	}

	return bridgeMacList, nil
}

func getNDP(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	intf, _ := options["iface"].String()
	ip := args.At(0)

	cmd := baseNdpCmd
	if ip != "" {
		if addr, _, err := net.ParseCIDR(ip); err == nil {
			if addr.To4() != nil {
				return nil, fmt.Errorf("IPv4 prefixes not allowed: %s", ip)
			}
		} else {
			p := net.ParseIP(ip)
			if p == nil || p.To4() != nil {
				return nil, fmt.Errorf("invalid IPv6 address: %s", ip)
			}
		}
		cmd += " " + ip
	}
	if intf != "" {
		cmd += " dev " + intf
	}
	log.V(6).Infof("Running command: %s", cmd)

	cmdOutput, err := common.GetDataFromHostCommand(cmd)
	if err != nil {
		log.Errorf("Error getting NDP data: %v", err)
		return nil, err
	}

	// If cmdOutput is empty
	if strings.TrimSpace(cmdOutput) == "" {
		return []byte(`{"total_entries":0,"entries":[]}`), nil
	}

	log.V(6).Infof("ndp output: %s", cmdOutput)
	// Parse the output
	table := parseNDPOutput(cmdOutput, intf)
	log.V(6).Infof("parsed table: %v", table)
	// Convert to JSON
	jsonData, err := json.Marshal(table)
	if err != nil {
		return nil, err
	}
	return jsonData, nil
}
