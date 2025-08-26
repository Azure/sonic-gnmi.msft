
/* show_client/lldp_cli.go
 * This file contains the implementation of the 'show lldp table' command for the Sonic gNMI client.

    Example output of 'show lldp table' command:

    admin@sonic:~$ show lldp table
    Capability codes: (R) Router, (B) Bridge, (O) Other
    LocalPort    RemoteDevice           RemotePortID     Capability  RemotePortDescr
    ------------ ---------------------  ---------------- ----------- ----------------------------------------
    Ethernet0    <neighbor0_hostname>    Ethernet1/51    BR          <my_hostname>:fortyGigE0/0
    Ethernet4    <neighbor1_hostname>    Ethernet1/51    BR          <my_hostname>:fortyGigE0/4
    Ethernet8    <neighbor2_hostname>    Ethernet1/51    BR          <my_hostname>:fortyGigE0/8
    Ethernet12   <neighbor3_hostname>    Ethernet1/51    BR          <my_hostname>:fortyGigE0/12
    ...          ...                     ...             ...         ...
    Ethernet124  <neighborN_hostname>    Ethernet4/20/1  BR          <my_hostname>:fortyGigE0/124
    eth0         <mgmt_neighbor_name>    Ethernet1/25    BR          Ethernet1/25
    -----------------------------------------------------
    Total entries displayed:  33
*/

package show_client

import (
	"encoding/json"	
	"fmt"
	"strconv"
	    
	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// lldpTableResponse represents the response structure for show llpd table command.
type lldpTableResponse struct {
	CapabilityCodesHelper string           `json:"capability_codes_helper"`
    Neighbors               []lldpNeighbor `json:"neighbors"`
    Total                   uint           `json:"total"`
}

// lldpNeighbor represents a single LLDP table entry.
type lldpNeighbor struct {
    LocalPort       string `json:"localPort"`
    RemoteDevice    string `json:"remoteDevice"`
    RemotePortID    string `json:"remotePortId"`
    Capability      string `json:"capability"`
    RemotePortDescr string `json:"remotePortDescr"`
}

var capabilityMap = map[int]string{
	0: "other",
	1: "repeater",
	2: "bridge",
	3: "wlanAccessPoint",
	4: "router",
	5: "telephone",
	6: "docsisCableDevice",
	7: "stationOnly",
}

// So far only find Router and Bridge two capabilities in lldpctl,
// so any other capacility types will be read as Other
// https://github.com/sonic-net/sonic-utilities/blob/master/scripts/lldpshow#L49
var capabilityCodeMap = map[string]string{
	"bridge": "B",
	"router": "R",
}

// Decodes the hex string representing LLDP capabilities into a slice of capability names.
// The hex string is expected to be in the format where each bit represents a capability.
// For example, Hex string "28 00" would indicate "bridge" and "router".
func decodeCapabilities(hexStr string) ([]string, error) {
	if hexStr == "" {
		return nil, fmt.Errorf("Hex string is empty, cannot decode capabilities")
	}

	// Ensure the hex string is at least 2 characters long
	if len(hexStr) < 2 {
		return nil, fmt.Errorf("Hex string %v is too short to decode capabilities", hexStr)
	}

	// Parse the hex string (only the first byte)
	val, err := strconv.ParseUint(hexStr[:2], 16, 8)
	if err != nil {
		log.Errorf("Unable to parse hex string %v to unint, got err %v", hexStr, err)
		return nil, err
	}

	// Decode the capabilities based on the bits set in the first byte
	// Each bit corresponds to a capability defined in the capabilityMap
	// See:
	// http://www.ieee802.org/1/files/public/MIBs/LLDP-MIB-200505060000Z.txt
	// https://github.com/sonic-net/sonic-dbsyncd/blob/master/src/lldp_syncd/conventions.py#L73
	var capabilities []string
	for i := 0; i < 8; i++ {
		if val&(1<<uint(7-i)) != 0 {
			if name, ok := capabilityMap[i]; ok {
				capabilities = append(capabilities, name)
			}
		}
	}
	return capabilities, nil
}

// Parses the hex string representing LLDP capabilities and returns a string of capability codes.
func parseCapabilityCodes(hexStr string) (string, error) {
	capabilities, err := decodeCapabilities(hexStr)
	if err != nil {
		log.Errorf("Failed to decode capability from hex string %v, got err %v", hexStr, err)
		return DefaultEmptyString, err
	}

	capabilityCodes := ""
	for _, cap := range capabilities {
		if tag, ok := capabilityCodeMap[cap]; ok {
			capabilityCodes += tag
		} else {
			capabilityCodes += "O"
		}
	}
	return capabilityCodes, nil
}

// Gets the LLDP table from the APPL_DB and returns it as a JSON byte slice.
// If any error occurs during the process, it logs the error and returns nil.
func getLLDPTable(options sdc.OptionMap) ([]byte, error) {
	queries := [][]string{
		{"APPL_DB", "LLDP_ENTRY_TABLE"},
	}

	lldpTableOutput, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to pull data for queries %v, got err %v", queries, err)
		return nil, err
	}
	log.V(6).Infof("LLDP Table output count: %v", len(lldpTableOutput))

	// LLDP_ENTRY_TABLE keys are like "LLDP_ENTRY_TABLE:Ethernet0"
	var neighbors []lldpNeighbor = make([]lldpNeighbor, 0, len(lldpTableOutput))

	for key, lldpTableItem := range lldpTableOutput {
		log.V(6).Infof("LLDP Table item: %v, %+v", key, lldpTableItem)

		enabledCapHexString := GetFieldValueString(lldpTableOutput, key, DefaultEmptyString, "lldp_rem_sys_cap_enabled")
		capabilitiesCode, err := parseCapabilityCodes(enabledCapHexString)
		if err != nil {
			log.Errorf("Unable to parse capability %v, got err %v", enabledCapHexString, err)
			capabilitiesCode = DefaultEmptyString
		}

		// Create lldpNeighbor instance
		neighbor := lldpNeighbor{
			LocalPort:       key,
			RemoteDevice: GetFieldValueString(lldpTableOutput, key, DefaultEmptyString, "lldp_rem_sys_name"),
			RemotePortID: GetFieldValueString(lldpTableOutput, key, DefaultEmptyString, "lldp_rem_port_id"),
			Capability: capabilitiesCode,
			RemotePortDescr: GetFieldValueString(lldpTableOutput, key, DefaultEmptyString, "lldp_rem_port_desc"),
		}
		neighbors = append(neighbors, neighbor)
	}

	// Create response structure
	var response = lldpTableResponse{
		CapabilityCodesHelper: "Capability codes: (R) Router, (B) Bridge, (O) Other",
		Neighbors:              neighbors,
		Total:                  uint(len(neighbors)),
	}
	log.V(6).Infof("LLDP Table response: %+v", response)
	return json.Marshal(response)	
}