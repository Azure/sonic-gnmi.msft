package show_client

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

const VlanTable = "VLAN"
const VlanInterfaceTable = "VLAN_INTERFACE"
const VlanMemberTable = "VLAN_MEMBER"

type VlanConfig struct {
	VlanData      map[string]interface{}
	VlanIpData    map[string]interface{}
	VlanPortsData map[string]interface{}
}

type VlanBriefColumn struct {
	Name   string
	Getter func(cfg VlanConfig, vlan string) []string
}

var VlanBriefColumns = []VlanBriefColumn{
	{"VLAN ID", getVlanId},
	{"IP Address", getVlanIpAddress},
	{"Ports", getVlanPorts},
	{"Port Tagging", getVlanPortsTagging},
	{"Proxy ARP", getProxyArp},
}

func getSortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func isIPPrefixInKey(key interface{}) bool {
	switch key.(type) {
	case []interface{}:
		return true
	default:
		return false
	}
}

func getVlanId(cfg VlanConfig, vlan string) []string {
	var ids []string
	ids = append(ids, strings.TrimPrefix(vlan, "Vlan"))
	return ids
}

func getVlanIpAddress(cfg VlanConfig, vlan string) []string {
	var ipAddress []string
	for key, _ := range cfg.VlanIpData {
		if isIPPrefixInKey(key) {
			ifname, address := parseKey(key)
			if vlan == ifname {
				ipAddress = append(ipAddress, address)
			}
		}
	}
	return ipAddress
}

func getVlanPorts(cfg VlanConfig, vlan string) []string {
	var vlanPorts []string
	for key := range cfg.VlanPortsData {
		fmt.Println("Key ==>" + key)
		portsKey, portsValue := parseKey(key)
		if vlan != portsKey {
			continue
		}
		vlanPorts = append(vlanPorts, portsValue)
	}
	return vlanPorts
}

func getVlanPortsTagging(cfg VlanConfig, vlan string) []string {
	var vlanPortsTagging []string
	for key, value := range cfg.VlanPortsData {
		portsKey, _ := parseKey(key)
		if vlan != portsKey {
			continue
		}
		taggingMode := value.(map[string]interface{})["tagging_mode"].(string)
		vlanPortsTagging = append(vlanPortsTagging, taggingMode)
	}
	return vlanPortsTagging
}

func getProxyArp(cfg VlanConfig, vlan string) []string {
	proxyArp := "disabled"
	for key, value := range cfg.VlanIpData {
		if vlan == key {
			if v, ok := value.(map[string]interface{})["proxy_arp"]; ok {
				proxyArp = v.(string)
			}
		}
	}

	var arp []string
	arp = append(arp, proxyArp)
	return arp
}

func parseKey(key interface{}) (string, string) {
	keyStr, ok := key.(string)
	if !ok {
		log.Errorf("parse Key failure to convert key as string:")
	}

	parts := strings.Split(keyStr, "|")
	if len(parts) < 2 {
		log.Errorf("Unable to parse the string")
		return "", ""
	}
	return parts[0], parts[1]
}

func getVlanBrief(options sdc.OptionMap) ([]byte, error) {
	queriesVlan := [][]string{
		{"CONFIG_DB", VlanTable},
	}

	queriesVlanInterface := [][]string{
		{"CONFIG_DB", VlanInterfaceTable},
	}

	queriesVlanMember := [][]string{
		{"CONFIG_DB", VlanMemberTable},
	}

	vlanData, derr := GetMapFromQueries(queriesVlan)
	if derr != nil {
		log.Errorf("Unable to get data from queries %v, got err: %v", queriesVlan, derr)
		return nil, derr
	}
	fmt.Println("vlanData")
	fmt.Println(vlanData)

	vlanInterfaceData, ierr := GetMapFromQueries(queriesVlanInterface)
	if ierr != nil {
		log.Errorf("Unable to get data from queries %v, got err: %v", queriesVlanInterface, ierr)
		return nil, ierr
	}
	fmt.Println("vlanIntData")
	fmt.Println(vlanInterfaceData)

	vlanMemberData, merr := GetMapFromQueries(queriesVlanMember)
	if merr != nil {
		log.Errorf("Unable to get data from queries %v, got err: %v", queriesVlanMember, merr)
		return nil, merr
	}
	fmt.Println("vlanMemData")
	fmt.Println(vlanMemberData)

	vlanCfg := VlanConfig{vlanData, vlanInterfaceData, vlanMemberData}

	vlans := getSortedKeys(vlanData)
	vlanBriefData := make(map[string]interface{})

	for _, vlan := range vlans {
		data := make(map[string]interface{})
		for _, col := range VlanBriefColumns {
			data[col.Name] = col.Getter(vlanCfg, vlan)
		}
		vlanBriefData[vlan] = data
	}

	for _, innerSlice := range vlanBriefData {
		fmt.Println("==>")
		fmt.Println(innerSlice)
	}

	jsonVlanBrief, jsonErr := json.Marshal(vlanBriefData)
	if jsonErr != nil {
		log.Errorf("Unable to parse the json: %v", jsonErr)
		return nil, jsonErr
	}

	return jsonVlanBrief, nil
}
