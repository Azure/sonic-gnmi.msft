package show_client

import (
	"encoding/json"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
)

// lldpData represents the overall LLDP data structure from lldpctl -f json0.
// why use json0 not json
// Because json0 is verbose and consistent structure, ideal for automated parsing
// while json is more compact and human-readable but can have variable structures
type lldpData struct {
	LLDP []lldpEntry `json:"lldp"`
}

type lldpEntry struct {
	Interface []interfaceEntry `json:"interface"`
}

type interfaceEntry struct {
	Name        string          `json:"name"`
	Via         string          `json:"via"`
	RID         string          `json:"rid"`
	Age         string          `json:"age"`
	Chassis     []chassis       `json:"chassis"`
	Port        []port          `json:"port"`
	VLAN        []vlan          `json:"vlan"`
	LLDPMed     []lldpMed       `json:"lldp-med"`
	UnknownTLVs []unknownTLVSet `json:"unknown-tlvs"`
}

type chassis struct {
	ID         []idEntry    `json:"id"`
	Name       []valueEntry `json:"name"`
	Descr      []valueEntry `json:"descr"`
	MgmtIp     []valueEntry `json:"mgmt-ip"`
	MgmtIface  []valueEntry `json:"mgmt-iface"`
	Capability []capability `json:"capability"`
}

type idEntry struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type valueEntry struct {
	Value string `json:"value"`
}

type capability struct {
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
}

type port struct {
	ID              []idEntry         `json:"id"`
	Descr           []valueEntry      `json:"descr"`
	TTL             []valueEntry      `json:"ttl"`
	MFS             []valueEntry      `json:"mfs"`
	AutoNegotiation []autoNegotiation `json:"auto-negotiation"`
}

type autoNegotiation struct {
	Supported  bool              `json:"supported"`
	Enabled    bool              `json:"enabled"`
	Advertised []advertisedSpeed `json:"advertised"`
	Current    []valueEntry      `json:"current"`
}

type advertisedSpeed struct {
	Type string `json:"type"`
	HD   bool   `json:"hd"`
	FD   bool   `json:"fd"`
}

type vlan struct {
	VLANID string `json:"vlan-id"`
	PVID   bool   `json:"pvid"`
	Value  string `json:"value"`
}

type lldpMed struct {
	DeviceType []valueEntry    `json:"device-type"`
	Capability []medCapability `json:"capability"`
}

type medCapability struct {
	Type      string `json:"type"`
	Available bool   `json:"available"`
}

type unknownTLVSet struct {
	UnknownTLV []unknownTLV `json:"unknown-tlv"`
}

type unknownTLV struct {
	OUI     string `json:"oui"`
	Subtype string `json:"subtype"`
	Len     string `json:"len"`
	Value   string `json:"value"`
}

// get full lldp data via docker exec lldp lldpctl -f json0
func getLLDPDataFromHostCommand() (lldpData, error) {
	log.V(2).Infof("Start to get lldp data from lldpctl")
	lldpShowJsonCommand := "docker exec lldp lldpctl -f json0"
	lldpOutput, err := common.GetDataFromHostCommand(lldpShowJsonCommand)
	if err != nil {
		log.Errorf("Unable to successfully execute command %v, get err %v", lldpShowJsonCommand, err)
		return lldpData{}, err
	}

	log.V(6).Infof("lldp output for cmd %s, output: %+v", lldpShowJsonCommand, lldpOutput)

	var data lldpData
	err = json.Unmarshal([]byte(lldpOutput), &data)
	if err != nil {
		log.Errorf("failed to unmarshal lldp output, get err %v", err)
		return lldpData{}, err
	}
	return data, nil
}

func boolToYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func boolToOnOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
