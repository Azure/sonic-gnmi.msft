package show_client

import (
	"sort"
	"strings"

	log "github.com/golang/glog"
	"github.com/olekukonko/tablewriter"
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
	Getter func(cfg VlanConfig, vlan string) string
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

func getVlanId(cfg VlanConfig, vlan string) string {
	return strings.TrimPrefix(vlan, "Vlan")
}

func getVlanIpAddress(cfg VlanConfig, vlan string) string {
	ipAddress := ""
	for key, _ := range cfg.VlanIpData {
		if isIPPrefixInKey(key) {
			ifname, address := parseKey(key)
			if vlan == ifname {
				ipAddress += "\n" + address
			}
		}
	}
	return ipAddress
}

func getVlanPorts(cfg VlanConfig, vlan string) string {
	var vlanPorts []string
	for key := range cfg.VlanPortsData {
		portsKey, portsValue := parseKey(key)
		if vlan != portsKey {
			continue
		}
		vlanPorts = append(vlanPorts, portsValue)
	}
	return strings.Join(vlanPorts, "\n")
}

func getVlanPortsTagging(cfg VlanConfig, vlan string) string {
	var vlanPortsTagging []string
	for key, value := range cfg.VlanPortsData {
		portsKey, _ := parseKey(key)
		if vlan != portsKey {
			continue
		}
		taggingMode := value.(map[string]interface{})["tagging_mode"].(string)
		vlanPortsTagging = append(vlanPortsTagging, taggingMode)
	}
	return strings.Join(vlanPortsTagging, "\n")
}

func getProxyArp(cfg VlanConfig, vlan string) string {
	proxyArp := "disabled"
	for key, value := range cfg.VlanIpData {
		if vlan == key {
			if v, ok := value.(map[string]interface{})["proxy_arp"]; ok {
				proxyArp = v.(string)
			}
		}
	}
	return proxyArp
}

func parseKey(key interface{}) (string, string) {
	return "", ""
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

	vlanInterfaceData, ierr := GetMapFromQueries(queriesVlanInterface)
	if ierr != nil {
		log.Errorf("Unable to get data from queries %v, got err: %v", queriesVlanInterface, ierr)
		return nil, ierr
	}

	vlanMemberData, merr := GetMapFromQueries(queriesVlanMember)
	if merr != nil {
		log.Errorf("Unable to get data from queries %v, got err: %v", queriesVlanMember, merr)
		return nil, merr
	}

	vlanCfg := VlanConfig{vlanData, vlanInterfaceData, vlanMemberData}

	vlans := getSortedKeys(vlanData)
	for _, vlan := range vlans {
		row := []string{}
		for _, col := range VlanBriefColumns {
			row = append(row, col.Getter(vlanCfg, vlan))
		}
		body = append(body, row)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	for _, v := range body {
		table.Append(v)
	}
	table.Render()

	return data, nil
}
