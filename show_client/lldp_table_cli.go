/* show_client/lldp_table_cli.go
* This file contains the implementation of the 'show lldp table' command for the Sonic gNMI client.

   Example output of 'show lldp table' command:
       admin@sonic:~$ show lldp table
       Capability codes: (R) Router, (B) Bridge, (O) Other
       LocalPort    RemoteDevice           RemotePortID     Capability  RemotePortDescr
       ------------ ---------------------  ---------------- ----------- ----------------------------------------
       Ethernet0    <neighbor0_hostname>    Ethernet1/51    BR          <my_hostname>:fortyGigE0/0
       Ethernet4    <neighbor1_hostname>    Ethernet1/51    BR          <my_hostname>:fortyGigE0/4
       ...          ...                     ...             ...         ...
       Ethernet124  <neighborN_hostname>    Ethernet4/20/1  BR          <my_hostname>:fortyGigE0/124
       eth0         <mgmt_neighbor_name>    Ethernet1/25    BR          Ethernet1/25
       -----------------------------------------------------
       Total entries displayed:  33
*/

package show_client

import (
	"encoding/json"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// lldpTableResponse represents the response structure for show lldp table command.
type lldpTableResponse struct {
	CapabilityCodesHelper string           `json:"capability_codes_helper"`
	Neighbors             []lldpTableEntry `json:"neighbors"`
	Total                 uint             `json:"total"`
}

// lldpTableEntry represents a single LLDP table entry.
type lldpTableEntry struct {
	LocalPort       string `json:"localPort"`
	RemoteDevice    string `json:"remoteDevice"`
	RemotePortID    string `json:"remotePortId"`
	Capability      string `json:"capability"`
	RemotePortDescr string `json:"remotePortDescr"`
}

// So far only find Router and Bridge two capabilities in lldpctl,
// so any other capacility types will be read as Other
// https://github.com/sonic-net/sonic-utilities/blob/master/scripts/lldpshow#L49
var capabilityCodeMap = map[string]string{
	"Bridge": "B",
	"Router": "R",
}

// Parses the enabled LLDP capability codes and returns a string of capability tags.
// Capability codes: (R) Router, (B) Bridge, (O) Other
func parseEnabledCapabilityCodes(enabledCapabilities []string) string {
	capabilityCodes := ""
	for _, cap := range enabledCapabilities {
		if tag, ok := capabilityCodeMap[cap]; ok {
			capabilityCodes += tag
		} else {
			capabilityCodes += "O"
		}
	}
	return capabilityCodes
}

// Extracts the LLDP table entries from the LLDP data.
func extractAllTableEntries(data lldpData) []lldpTableEntry {
	neighbors := make([]lldpTableEntry, 0)

	for _, entry := range data.LLDP {
		for _, iface := range entry.Interface {
			neighbor := extractTableEntry(iface)
			neighbors = append(neighbors, neighbor)
		}
	}

	return neighbors
}

func extractTableEntry(iface interfaceEntry) lldpTableEntry {
	localPort := iface.Name
	remoteDeviceName := ""
	remotePortID := ""
	remotePortDescr := ""
	var caps []string = make([]string, 0)

	for _, chassis := range iface.Chassis {
		if len(chassis.Name) > 0 {
			remoteDeviceName = chassis.Name[0].Value
		}

		for _, cap := range chassis.Capability {
			if cap.Enabled {
				caps = append(caps, cap.Type)
			}
		}
	}

	for _, port := range iface.Port {
		if len(port.ID) > 0 {
			remotePortID = port.ID[0].Value
		}
		if len(port.Descr) > 0 {
			remotePortDescr = port.Descr[0].Value
		}
	}

	neighbor := lldpTableEntry{
		LocalPort:       localPort,
		RemoteDevice:    remoteDeviceName,
		RemotePortID:    remotePortID,
		Capability:      parseEnabledCapabilityCodes(caps),
		RemotePortDescr: remotePortDescr,
	}

	return neighbor
}

func getLLDPTable(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	// get lldp data
	data, err := getLLDPDataFromHostCommand()
	if err != nil {
		log.Errorf("Failed to get lldp data, get err %v", err)
		return nil, err
	}

	// parse neighbors summary from full lldp data
	neighbors := extractAllTableEntries(data)

	// create response structure
	var response = lldpTableResponse{
		CapabilityCodesHelper: "Capability codes: (R) Router, (B) Bridge, (O) Other",
		Neighbors:             neighbors,
		Total:                 uint(len(neighbors)),
	}
	log.V(6).Infof("LLDP Table response: %+v", response)
	return json.Marshal(response)
}
