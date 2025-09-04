package show_client

import (
	"encoding/json"
	"sort"
	"strings"

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
*/
func getInterfacePortchannel(options sdc.OptionMap) ([]byte, error) {
	cfgPC, err := GetMapFromQueries([][]string{{"CONFIG_DB", "PORTCHANNEL"}})
	if err != nil {
		return nil, err
	}

	stateLag, err := GetMapFromQueries([][]string{{"STATE_DB", "LAG_TABLE"}})
	if err != nil {
		return nil, err
	}

	applLag, err := GetMapFromQueries([][]string{{"APPL_DB", "LAG_TABLE"}})
	if err != nil {
		return nil, err
	}

	stateLagMember, err := GetMapFromQueries([][]string{{"STATE_DB", "LAG_MEMBER_TABLE"}})
	if err != nil {
		return nil, err
	}

	applLagMember, err := GetMapFromQueries([][]string{{"APPL_DB", "LAG_MEMBER_TABLE"}})
	if err != nil {
		return nil, err
	}

	ts := &teamShow{
		cfgPC:       cfgPC,
		stateLag:    stateLag,
		applLag:     applLag,
		stateLagMem: stateLagMember,
		applLagMem:  applLagMember,
		aliasMode:   (GetInterfaceNamingMode() == "alias"),
	}

	result := ts.getTeamshowResult()
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

/*
Get the portchannel names from database.
admin@sonic:~$ redis-cli -n 4 KEYS 'PORTCHANNEL|*'
1) "PORTCHANNEL|PortChannel102"
2) "PORTCHANNEL|PortChannel113"
3) "PORTCHANNEL|PortChannel101"
*/
func (t *teamShow) getPortchannelNames() []string {
	var list []string
	for k := range t.cfgPC {
		if strings.HasPrefix(k, "PortChannel") {
			list = append(list, k)
		}
	}
	return list
}

/*
Get port channel status from database.
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
func (t *teamShow) getPortchannelStatus(pc string) string {
	active := GetFieldValueString(t.stateLag, pc, "", "runner.active") == "true"
	proto := "LACP"
	if active {
		proto += "(A)"
	} else {
		proto += "(I)"
	}

	suffix := "(N/A)"
	oper := strings.ToLower(GetFieldValueString(t.applLag, pc, "", "oper_status"))
	switch oper {
	case "up":
		suffix = "(Up)"
	case "down":
		suffix = "(Dw)"
	}

	return proto + suffix
}

/*
admin@sonic:~$ redis-cli -n 6 HGETALL "LAG_MEMBER_TABLE|PortChannel102|Ethernet332"
27) "runner.aggregator.selected"
28) "true"

admin@sonic:~$ redis-cli -n 0 HGETALL "LAG_MEMBER_TABLE:PortChannel102:Ethernet332"
1) "status"
2) "enabled"
*/
func (t *teamShow) getPortchannelMemberStatus(pc, member string) (bool, string) {
	selected := GetFieldValueString(t.stateLagMem, pc+"|"+member, "", "runner.aggregator.selected") == "true"
	status := GetFieldValueString(t.applLagMem, pc+":"+member, "", "status") // enabled/disabled/empty
	return selected, status
}

// Get teams raw data from teamdctl.
func (t *teamShow) getTeamdctl(pc string) []string {
	// Collect raw member names (STATE_DB key pattern PortChannelX|EthernetY)
	prefix := pc + "|"
	var members []string
	for k := range t.stateLagMem {
		if strings.HasPrefix(k, prefix) {
			members = append(members, strings.TrimPrefix(k, prefix))
		}
	}
	sort.Strings(members)

	var out []string
	for _, mem := range members {
		selected, status := t.getPortchannelMemberStatus(pc, mem)
		unsynced := status == "" ||
			(status == "enabled" && !selected) ||
			(status == "disabled" && selected)

		display := mem
		if t.aliasMode {
			display = GetInterfaceNameForDisplay(mem)
		}

		var sb strings.Builder
		sb.WriteString(display)
		sb.WriteString("(")
		if selected {
			sb.WriteString("S")
		} else {
			sb.WriteString("D")
		}
		if unsynced {
			sb.WriteString("*")
		}
		sb.WriteString(")")
		out = append(out, sb.String())
	}
	return out
}

// Skip the 'PortChannel' prefix and extract the team id.
func getTeamID(team string) string {
	const prefix = "PortChannel"
	if strings.HasPrefix(team, prefix) && len(team) > len(prefix) {
		return team[len(prefix):]
	}
	return team
}

// Get teamshow results by parsing the output of teamdctl and combining port channel status.
func (t *teamShow) getTeamshowResult() map[string]map[string]interface{} {
	names := t.getPortchannelNames()
	res := make(map[string]map[string]interface{}, len(names))
	for _, pc := range names {
		ports := t.getTeamdctl(pc)
		teamID := getTeamID(pc)
		res[teamID] = map[string]interface{}{
			"Team Dev": pc,
			"Protocol": t.getPortchannelStatus(pc),
			"Ports":    strings.Join(ports, " "),
		}
	}
	return res
}
