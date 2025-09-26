package common

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	log "github.com/golang/glog"
)

type BridgeMacEntry struct {
	VlanID int
	Mac    string
	IfName string
}

const oidPrefixLen = len("oid:0x")

func FetchFDBData() ([]BridgeMacEntry, error) {
	queries := [][]string{
		{"ASIC_DB", "ASIC_STATE:SAI_OBJECT_TYPE_FDB_ENTRY:*"},
	}

	// "ASIC_STATE:SAI_OBJECT_TYPE_FDB_ENTRY:{\"bvid\":\"oid:0x2600000000063f\",\"mac\":\"B8:CE:F6:E5:50:05\",\"switch_id\":\"oid:0x21000000000000\"}"
	brPortStr, err := GetMapFromQueries(queries)
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

func getInterfaceOidMap() (map[string]string, error) {
	portQueries := [][]string{
		{"COUNTERS_DB", "COUNTERS_PORT_NAME_MAP"},
	}
	lagQueries := [][]string{
		{"COUNTERS_DB", "COUNTERS_LAG_NAME_MAP"},
	}

	portMap, err := GetMapFromQueries(portQueries)
	if err != nil {
		return nil, err
	}
	lagMap, err := GetMapFromQueries(lagQueries)
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

func getBridgePortMap() (map[string]string, error) {
	queries := [][]string{
		{"ASIC_DB", "ASIC_STATE:SAI_OBJECT_TYPE_BRIDGE_PORT:*"},
	}
	brPortStr, err := GetMapFromQueries(queries)
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

func buildBvidToVlanMap() (map[string]string, error) {
	queries := [][]string{
		{"ASIC_DB", "ASIC_STATE:SAI_OBJECT_TYPE_VLAN:*"},
	}

	vlanData, err := GetMapFromQueries(queries)
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
