package show_client

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

/*
'portchannel' subcommand ("show interfaces portchannel")
Example of the output:
admin@sonic:~$ show interfaces portchannel
Flags: A - active, I - inactive, Up - up, Dw - Down, N/A - not available,

	S - selected, D - deselected, * - not synced

No.  Team Dev         Protocol     Ports
-----  ---------------  -----------  -----------------------------
102  PortChannel102   LACP(A)(Dw)  Ethernet0(D) Ethernet8(D)
104  PortChannel104   LACP(A)(Dw)  Ethernet24(D) Ethernet16(D)
106  PortChannel106   LACP(A)(Dw)  Ethernet32(D) Ethernet40(D)

'show interfaces portchannel' (structured JSON output)

New JSON schema per PortChannel (key = numeric ID without 'PortChannel' prefix):

	{
	  "101": {
	    "Team Dev": "PortChannel101",
	    "Protocol": {
	      "name": "LACP",
	      "active": true,
	      "operational_status": "up"   // "up" | "down" | "N/A" (when unavailable)
	    },
	    "Ports": [
	      {
	        "name": "Ethernet0",
	        "selected": true,
	        "status": "enabled",         // "enabled" | "disabled" | ""
	        "in_sync": true              // false previously shown as '*'
	      }
	    ]
	  }
	}
*/
func getInterfacePortchannel(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	namingMode, _ := options[SonicCliIfaceMode].String()
	cfgPC, err := common.GetMapFromQueries([][]string{{"CONFIG_DB", "PORTCHANNEL"}})
	if err != nil {
		return nil, err
	}
	stateLag, err := common.GetMapFromQueries([][]string{{"STATE_DB", "LAG_TABLE"}})
	if err != nil {
		return nil, err
	}
	applLag, err := common.GetMapFromQueries([][]string{{"APPL_DB", "LAG_TABLE"}})
	if err != nil {
		return nil, err
	}
	stateLagMember, err := common.GetMapFromQueries([][]string{{"STATE_DB", "LAG_MEMBER_TABLE"}})
	if err != nil {
		return nil, err
	}
	applLagMember, err := common.GetMapFromQueries([][]string{{"APPL_DB", "LAG_MEMBER_TABLE"}})
	if err != nil {
		return nil, err
	}

	ts := &teamShow{
		cfgPC:       cfgPC,
		stateLag:    stateLag,
		applLag:     applLag,
		stateLagMem: stateLagMember,
		applLagMem:  applLagMember,
		aliasMode:   (namingMode == "alias"),
	}

	result := ts.getTeamshowResult(namingMode)
	return json.Marshal(result)
}

/************ teamShow struct & methods ************/

type teamShow struct {
	cfgPC       map[string]interface{}
	stateLag    map[string]interface{}
	applLag     map[string]interface{}
	stateLagMem map[string]interface{}
	applLagMem  map[string]interface{}
	aliasMode   bool
}

// Collect PortChannel names
func (t *teamShow) getPortchannelNames() []string {
	var list []string
	for k := range t.cfgPC {
		name := k
		if i := strings.Index(k, "|"); i >= 0 { // handle "PORTCHANNEL|PortChannel101"
			name = k[i+1:]
		}
		if strings.HasPrefix(name, "PortChannel") {
			list = append(list, name)
		}
	}
	return list
}

/*
Get port channel structured protocol status from database.
admin@sonic:~$ redis-cli -n 6 HGETALL 'LAG_TABLE|PortChannel102'
3) "oper_status"
4) "up"
17) "runner.active"
18) "true"

admin@sonic:~$ redis-cli -n 0 HGETALL "LAG_TABLE:PortChannel102"
1) "mtu"
2) "9100"
3) "tpid"
4) "0x8100"
5) "admin_status"
6) "up"
7) "oper_status"
8) "up"
*/
func (t *teamShow) getPortchannelStatus(pc string) map[string]interface{} {
	active := common.GetFieldValueString(t.stateLag, pc, "", "runner.active") == "true"
	oper := strings.ToLower(common.GetFieldValueString(t.applLag, pc, "", "oper_status"))
	if oper != "up" && oper != "down" {
		oper = "N/A"
	}
	return map[string]interface{}{
		"name":               "LACP",
		"active":             active,
		"operational_status": oper,
	}
}

/*
Get member selected / status

admin@sonic:~$ redis-cli -n 6 HGETALL "LAG_MEMBER_TABLE|PortChannel102|Ethernet332"
27) "runner.aggregator.selected"
28) "true"
admin@sonic:~$ redis-cli -n 0 HGETALL "LAG_MEMBER_TABLE:PortChannel102:Ethernet332"
1) "status"
2) "enabled"
*/
func (t *teamShow) getPortchannelMemberStatus(pc, member string) (bool, string) {
	selected := common.GetFieldValueString(t.stateLagMem, pc+"|"+member, "", "runner.aggregator.selected") == "true"
	status := common.GetFieldValueString(t.applLagMem, pc+":"+member, "", "status") // enabled/disabled/empty
	return selected, status
}

// Structured members list
func (t *teamShow) getPortchannelMembers(pc string, namingMode string) []map[string]interface{} {
	prefix := pc + "|"
	var members []string
	for k := range t.stateLagMem {
		if strings.HasPrefix(k, prefix) {
			members = append(members, strings.TrimPrefix(k, prefix))
		}
	}
	sort.Strings(members)

	out := make([]map[string]interface{}, 0, len(members))
	for _, mem := range members {
		selected, status := t.getPortchannelMemberStatus(pc, mem)
		unsynced := status == "" ||
			(status == "enabled" && !selected) ||
			(status == "disabled" && selected)

		display := mem
		if t.aliasMode {
			display = common.GetInterfaceNameForDisplay(mem, namingMode)
		}
		out = append(out, map[string]interface{}{
			"name":     display,
			"selected": selected,
			"status":   status,
			"in_sync":  !unsynced,
		})
	}
	return out
}

// Strip "PortChannel" prefix
func getTeamID(team string) string {
	const prefix = "PortChannel"
	if strings.HasPrefix(team, prefix) && len(team) > len(prefix) {
		return team[len(prefix):]
	}
	return team
}

// Build final structured result
func (t *teamShow) getTeamshowResult(namingMode string) map[string]map[string]interface{} {
	names := t.getPortchannelNames()
	res := make(map[string]map[string]interface{}, len(names))
	for _, pc := range names {
		teamID := getTeamID(pc)
		res[teamID] = map[string]interface{}{
			"Team Dev": pc,
			"Protocol": t.getPortchannelStatus(pc),
			"Ports":    t.getPortchannelMembers(pc, namingMode),
		}
	}
	return res
}
