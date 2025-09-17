/* show_client/lldp_neighbors_cli.go
* This file contains the implementation of the 'show lldp neighbors' command for the Sonic gNMI client.

   Example output of 'show lldp neighbors' command:

       admin@sonic:~$ show lldp neighbors
       -------------------------------------------------------------------------------
       LLDP neighbors:
       -------------------------------------------------------------------------------
       Interface:    eth0, via: LLDP, RID: 1, Time: 10 days, 00:58:57
         Chassis:
           ChassisID:    mac <mac-address>
           SysName:      <sonic-device-name>
           SysDescr:     Juniper Networks, Inc. ex2200-48t-4g , version 12.3R4.6 Build date: 2013-09-13 02:53:16 UTC
           Capability:   Bridge, on
           Capability:   Router, on
         Port:
           PortID:       local 508
           PortDescr:    ge-0/0/11.0
           TTL:          90
           MFS:          1514
           PMD autoneg:  supported: yes, enabled: yes
             Adv:          10Base-T, HD: yes, FD: yes
             Adv:          100Base-TX, HD: yes, FD: yes
             Adv:          1000Base-T, HD: no, FD: yes
             MAU oper type: unknown
         VLAN:         147, pvid: yes labuse
         LLDP-MED:
           Device Type:  Network Connectivity Device
           Capability:   Capabilities, yes
           Capability:   Policy, yes
           Capability:   Location, yes
           Capability:   MDI/PSE, yes
         Unknown TLVs:
           TLV:          OUI: 00,90,69, SubType: 1, Len: 12 43,55,30,32,31,33,35,31,30,36,36,33
       -------------------------------------------------------------------------------
       ...
       -------------------------------------------------------------------------------
*/

package show_client

import (
	"encoding/json"
	"fmt"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// lldpNeighborsResponse represents the response structure for show lldp neighbors command.
type lldpNeighborsResponse struct {
	Title      string                        `json:"title"`
	Interfaces map[string]lldpNeighborsEntry `json:"interfaces"`
}

// lldpNeighborsEntry represents a single LLDP neighbors entry.
type lldpNeighborsEntry struct {
	Via         string             `json:"via"`
	RID         string             `json:"RID"`
	Age         string             `json:"Time"`
	Chassis     neighborsChassis   `json:"Chassis"`
	Port        neighborsPort      `json:"Port"`
	VLAN        []neighborsVLAN    `json:"VLAN,omitempty"`
	LLDPMed     []neighborsLLDPMed `json:"LLDP-MED,omitempty"`
	UnknownTLVs []unknownTLVSet    `json:"UnknownTLVs,omitempty"`
}

type neighborsChassis struct {
	ID         string   `json:"ChassisID"`
	Name       string   `json:"SysName"`
	Descr      string   `json:"SysDescr"`
	MgmtIp     string   `json:"MgmtIP,omitempty"`
	MgmtIface  string   `json:"MgmtIface,omitempty"`
	Capability []string `json:"Capability,omitempty"`
}

type neighborsPort struct {
	ID              string          `json:"PortID"`
	Descr           string          `json:"PortDescr"`
	TTL             string          `json:"TTL"`
	MFS             string          `json:"MFS,omitempty"`
	AutoNegotiation *pmdNegotiation `json:"PMD-autoneg,omitempty"`
}

type pmdNegotiation struct {
	Supported  string `json:"supported,omitempty"`
	Enabled    string `json:"enabled,omitempty"`
	Advertised []adv  `json:"Adv,omitempty"`
	Current    string `json:"MAU-oper-type,omitempty"`
}

type adv struct {
	Type string `json:"Type"`
	HD   string `json:"HD"`
	FD   string `json:"FD"`
}

type neighborsVLAN struct {
	VLANID string `json:"vlan-id"`
	PVID   string `json:"pvid"`
	Value  string `json:"value,omitempty"`
}

type neighborsLLDPMed struct {
	DeviceType string   `json:"Device-type"`
	Capability []string `json:"Capability"`
}

// Extracts the LLDP neighbors entries from the LLDP data.
func extractAllNeighborsEntry(data lldpData) map[string]lldpNeighborsEntry {
	neighbors := make(map[string]lldpNeighborsEntry)

	for _, entry := range data.LLDP {
		for _, iface := range entry.Interface {
			neighbor := extractNeighborsEntry(iface)
			neighbors[iface.Name] = neighbor
		}
	}

	return neighbors
}

func extractNeighborsEntry(iface interfaceEntry) lldpNeighborsEntry {
	neighbor := lldpNeighborsEntry{
		Via:     iface.Via,
		RID:     iface.RID,
		Age:     iface.Age,
		Chassis: neighborsChassis{},
		Port: neighborsPort{
			AutoNegotiation: &pmdNegotiation{},
		},
		VLAN:        make([]neighborsVLAN, 0),
		LLDPMed:     make([]neighborsLLDPMed, 0),
		UnknownTLVs: make([]unknownTLVSet, 0),
	}

	// Populate chassis (take the first chassis if multiple)
	if len(iface.Chassis) > 0 {
		c := iface.Chassis[0]
		if len(c.ID) > 0 {
			neighbor.Chassis.ID = fmt.Sprintf("%s %s", c.ID[0].Type, c.ID[0].Value)
		}
		if len(c.Name) > 0 {
			neighbor.Chassis.Name = c.Name[0].Value
		}
		if len(c.Descr) > 0 {
			neighbor.Chassis.Descr = c.Descr[0].Value
		}

		// Mgmt IP and Mgmt interface (if present)
		if len(c.MgmtIp) > 0 {
			neighbor.Chassis.MgmtIp = c.MgmtIp[0].Value
		}
		if len(c.MgmtIface) > 0 {
			neighbor.Chassis.MgmtIface = c.MgmtIface[0].Value
		}

		// Capabilities
		for _, cp := range c.Capability {
			capabilityDesc := fmt.Sprintf("%s, %s", cp.Type, boolToOnOff(cp.Enabled))
			neighbor.Chassis.Capability = append(neighbor.Chassis.Capability, capabilityDesc)
		}
	}

	// Populate port (take the first port if multiple)
	if len(iface.Port) > 0 {
		p := iface.Port[0]
		if len(p.ID) > 0 {
			neighbor.Port.ID = fmt.Sprintf("%s %s", p.ID[0].Type, p.ID[0].Value)
		}
		if len(p.Descr) > 0 {
			neighbor.Port.Descr = p.Descr[0].Value
		}

		if len(p.TTL) > 0 {
			neighbor.Port.TTL = p.TTL[0].Value
		}
		if len(p.MFS) > 0 {
			neighbor.Port.MFS = p.MFS[0].Value
		}

		// Parse PMD autonegotiation info (if present)
		if len(p.AutoNegotiation) > 0 {
			an := p.AutoNegotiation[0]
			neighbor.Port.AutoNegotiation.Supported = boolToYesNo(an.Supported)
			neighbor.Port.AutoNegotiation.Enabled = boolToYesNo(an.Enabled)

			// Parse advertised capabilities (Adv)
			for _, advEntry := range an.Advertised {
				var advItem adv
				advItem.Type = advEntry.Type
				advItem.HD = boolToYesNo(advEntry.HD)
				advItem.FD = boolToYesNo(advEntry.FD)
				neighbor.Port.AutoNegotiation.Advertised = append(neighbor.Port.AutoNegotiation.Advertised, advItem)
			}

			// Current MAU oper type (if present)
			if len(an.Current) > 0 {
				neighbor.Port.AutoNegotiation.Current = an.Current[0].Value
			}
		} else {
			// This to omit AutoNegotiation in json
			neighbor.Port.AutoNegotiation = nil
		}
	}

	// Populate VLANs (if present)
	if len(iface.VLAN) > 0 {
		for _, v := range iface.VLAN {
			nv := neighborsVLAN{
				VLANID: v.VLANID,
				PVID:   boolToYesNo(v.PVID),
				Value:  v.Value,
			}
			neighbor.VLAN = append(neighbor.VLAN, nv)
		}
	}

	// Populate LLDP-MED entries (if present)
	if len(iface.LLDPMed) > 0 {
		for _, m := range iface.LLDPMed {
			med := neighborsLLDPMed{}

			if len(m.DeviceType) > 0 {
				med.DeviceType = m.DeviceType[0].Value
			}

			// Capability
			for _, cp := range m.Capability {
				capabilityDesc := fmt.Sprintf("%s, %s", cp.Type, boolToYesNo(cp.Available))
				med.Capability = append(med.Capability, capabilityDesc)
			}

			neighbor.LLDPMed = append(neighbor.LLDPMed, med)
		}
	}

	// Populate Unknown TLVs (if present)
	// Keep same as lldpctl json data
	if len(iface.UnknownTLVs) > 0 {
		for _, t := range iface.UnknownTLVs {
			var unSet unknownTLVSet
			if b, err := json.Marshal(t); err == nil {
				if err := json.Unmarshal(b, &unSet); err == nil {
					neighbor.UnknownTLVs = append(neighbor.UnknownTLVs, unSet)
				}
			}
		}
	}

	return neighbor
}

func getLLDPNeighbors(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	// Get interface name from args, if provided, default to ""
	ifaceName := args.At(0)

	data, err := getLLDPDataFromHostCommand()
	if err != nil {
		log.Errorf("Failed to get lldp data, get err %v", err)
		return nil, err
	}

	// parse neighbors summary from full lldp data
	neighbors := extractAllNeighborsEntry(data)

	// If interface name is provided, filter to return only that interface
	if ifaceName != "" {
		if entry, ok := neighbors[ifaceName]; ok {
			neighbors = map[string]lldpNeighborsEntry{ifaceName: entry}
		} else {
			// Interface name provided but not found; return empty map
			neighbors = make(map[string]lldpNeighborsEntry)
		}
	}

	// create response structure
	var response = lldpNeighborsResponse{
		Title:      "LLDP neighbors",
		Interfaces: neighbors,
	}
	log.V(6).Infof("LLDP Neighbors data: %+v", response)
	return json.Marshal(response)
}
