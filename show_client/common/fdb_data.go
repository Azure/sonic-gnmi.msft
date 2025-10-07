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

const oidPrefix = "oid:0x"

func FetchFDBData() ([]BridgeMacEntry, error) {
	queries := [][]string{
		{"ASIC_DB", "ASIC_STATE:SAI_OBJECT_TYPE_FDB_ENTRY:*"},
	}

	fdbEntries, err := GetMapFromQueries(queries)
	if err != nil {
		return nil, err
	}
	log.V(6).Infof("FDB_ENTRY list: %v", fdbEntries)

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

	for fdbKey, entryData := range fdbEntries {
		idx := strings.Index(fdbKey, ":")
		if idx == -1 || idx+1 >= len(fdbKey) {
			continue
		}
		fdbJSON := fdbKey[idx+1:]

		fdb := map[string]string{}
		if err := json.Unmarshal([]byte(fdbJSON), &fdb); err != nil {
			continue
		}

		ent, ok := entryData.(map[string]interface{})
		if !ok {
			continue
		}

		brPortOidRaw, ok := ent["SAI_FDB_ENTRY_ATTR_BRIDGE_PORT_ID"].(string)
		if !ok || !strings.HasPrefix(brPortOidRaw, oidPrefix) {
			continue
		}
		brPortOid := strings.TrimPrefix(brPortOidRaw, oidPrefix)

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

	ethRe := regexp.MustCompile(`^Ethernet(\d+)$`)
	lagRe := regexp.MustCompile(`^PortChannel(\d+)$`)
	vlanRe := regexp.MustCompile(`^Vlan(\d+)$`)
	mgmtRe := regexp.MustCompile(`^eth(\d+)$`)

	ifOidMap := make(map[string]string)

	isValidIfName := func(name string) bool {
		return ethRe.MatchString(name) ||
			lagRe.MatchString(name) ||
			vlanRe.MatchString(name) ||
			mgmtRe.MatchString(name)
	}

	for portName, oidVal := range portMap {
		oidStr, ok := oidVal.(string)
		if !ok || !strings.HasPrefix(oidStr, oidPrefix) {
			continue
		}
		if isValidIfName(portName) {
			ifOidMap[strings.TrimPrefix(oidStr, oidPrefix)] = portName
		}
	}
	for lagName, oidVal := range lagMap {
		oidStr, ok := oidVal.(string)
		if !ok || !strings.HasPrefix(oidStr, oidPrefix) {
			continue
		}
		if isValidIfName(lagName) {
			ifOidMap[strings.TrimPrefix(oidStr, oidPrefix)] = lagName
		}
	}

	return ifOidMap, nil
}

func getBridgePortMap() (map[string]string, error) {
	queries := [][]string{
		{"ASIC_DB", "ASIC_STATE:SAI_OBJECT_TYPE_BRIDGE_PORT:*"},
	}
	bridgePortEntries, err := GetMapFromQueries(queries)
	if err != nil {
		return nil, err
	}
	log.V(6).Infof("SAI_OBJECT_TYPE_BRIDGE_PORT data from query: %v", bridgePortEntries)

	ifBrOidMap := make(map[string]string)

	for key, val := range bridgePortEntries {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) < 2 || !strings.HasPrefix(parts[1], oidPrefix) {
			continue
		}
		bridgePortOid := strings.TrimPrefix(parts[1], oidPrefix)

		attrs, ok := val.(map[string]string)
		if !ok {
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

		portIdRaw, ok := attrs["SAI_BRIDGE_PORT_ATTR_PORT_ID"]
		if !ok || !strings.HasPrefix(portIdRaw, oidPrefix) {
			continue
		}
		portId := strings.TrimPrefix(portIdRaw, oidPrefix)
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

		bvid := strings.TrimPrefix(key, prefix)

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
