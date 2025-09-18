package show_client

import (
	"encoding/json"
	"net"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

const (
	vlanTable          = "VLAN"
	vlanInterfaceTable = "VLAN_INTERFACE"
	vlanMemberTable    = "VLAN_MEMBER"
	proxyArpKey        = "proxy_arp"
	vlanKey            = "Vlan"
	pipeDelimiter      = "|"
	dhcpServers        = "dhcp_servers@"
)

type vlanConfig struct {
	VlanData      map[string]interface{}
	VlanIpData    map[string]interface{}
	VlanPortsData map[string]interface{}
}

type vlanBriefColumn struct {
	Name   string
	Getter func(cfg vlanConfig, vlan string) interface{}
}

var vlanBriefColumns = []vlanBriefColumn{
	{"vlan_id", getVlanId},
	{"ip_address", getVlanIpAddress},
	{"ports", getVlanPortsAndTagging},
	{"proxy_arp", getProxyArp},
	{"dhcp_helper_addresses", getVlanDhcpHelperAddress},
}

type portAndTagging struct {
	Name         string `json:"name"`
	Port_tagging string `json:"port_tagging"`
}

// Function to check if given key is having valid IP, IP CIDR
func isIPPrefixInKey(key interface{}) bool {
	if keyStr, ok := key.(string); ok {
		vlanId, ip := common.ParseKey(keyStr, pipeDelimiter)

		if vlanId == "" {
			return false
		}

		_, _, err := net.ParseCIDR(ip)
		if err != nil {
			parsedIp := net.ParseIP(ip)
			if parsedIp != nil {
				return true
			} else {
				return false
			}
		}
		return true
	} else {
		log.Info("Unable to parse the key")
		return false
	}
}

func getVlanId(cfg vlanConfig, vlan string) interface{} {
	id := strings.TrimPrefix(vlan, vlanKey)
	return id
}

func getVlanIpAddress(cfg vlanConfig, vlan string) interface{} {
	var ipAddress []string
	for key, _ := range cfg.VlanIpData {
		if isIPPrefixInKey(key) {
			ifname, address := common.ParseKey(key, pipeDelimiter)
			if vlan == ifname {
				ipAddress = append(ipAddress, address)
			}
		}
	}
	return ipAddress
}

func getVlanDhcpHelperAddress(cfg vlanConfig, vlan string) interface{} {
	var ipAddress []string
	for key, value := range cfg.VlanData {
		if key == vlan {
			if dhcpHelperIps, ok := value.(map[string]interface{})[dhcpServers]; ok {
				ipAddress = strings.Split(dhcpHelperIps.(string), ",")
				break
			}
		}
	}
	ipAddress = common.NatsortInterfaces(ipAddress)
	return ipAddress
}

func getVlanPortsAndTagging(cfg vlanConfig, vlan string) interface{} {
	var vlanPorts []portAndTagging
	for key, value := range cfg.VlanPortsData {
		portsKey, portsValue := common.ParseKey(key, pipeDelimiter)
		if vlan != portsKey {
			continue
		}

		vlanPorts = append(vlanPorts, portAndTagging{portsValue, value.(map[string]interface{})["tagging_mode"].(string)})
	}

	return vlanPorts
}

func getProxyArp(cfg vlanConfig, vlan string) interface{} {
	proxyArp := "disabled"
	for key, value := range cfg.VlanIpData {
		if vlan == key {
			if v, ok := value.(map[string]interface{})[proxyArpKey]; ok {
				proxyArp = v.(string)
			}
		}
	}

	return proxyArp
}

func getVlanBrief(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	queriesVlan := [][]string{
		{"CONFIG_DB", vlanTable},
	}

	queriesVlanInterface := [][]string{
		{"CONFIG_DB", vlanInterfaceTable},
	}

	queriesVlanMember := [][]string{
		{"CONFIG_DB", vlanMemberTable},
	}

	vlanData, err := common.GetMapFromQueries(queriesVlan)
	if err != nil {
		log.Errorf("Unable to get data from queries %v, got err: %v", queriesVlan, err)
		return nil, err
	}

	vlanInterfaceData, err := common.GetMapFromQueries(queriesVlanInterface)
	if err != nil {
		log.Errorf("Unable to get data from queries %v, got err: %v", queriesVlanInterface, err)
		return nil, err
	}

	vlanMemberData, err := common.GetMapFromQueries(queriesVlanMember)
	if err != nil {
		log.Errorf("Unable to get data from queries %v, got err: %v", queriesVlanMember, err)
		return nil, err
	}

	vlanCfg := vlanConfig{vlanData, vlanInterfaceData, vlanMemberData}

	vlans := common.GetSortedKeys(vlanData)
	vlanBriefData := make(map[string]interface{})

	for _, vlan := range vlans {
		data := make(map[string]interface{})
		for _, col := range vlanBriefColumns {
			data[col.Name] = col.Getter(vlanCfg, vlan)
		}
		vlanBriefData[vlan] = data
	}

	return json.Marshal(vlanBriefData)
}
