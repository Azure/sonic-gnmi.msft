package show_client

import (
	"encoding/json"
	"fmt"
	"net"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

type IPv6BGPSummaryResponse struct {
	IPv6Unicast IPv6Unicast `json:"ipv6Unicast"`
}

type IPv6Unicast struct {
	RouterID        string          `json:"routerId"`
	LocalAS         int             `json:"as"`
	VRFId           int             `json:"vrfId"`
	TableVersion    int             `json:"tableVersion"`
	RibCount        int             `json:"ribCount"`
	RibMemory       int             `json:"ribMemory"`
	PeerCount       int             `json:"peerCount"`
	PeerMemory      int             `json:"peerMemory"`
	PeerGroupCount  int             `json:"peerGroupCount"`
	PeerGroupMemory int             `json:"peerGroupMemory"`
	Peers           map[string]Peer `json:"peers"`
}

type Peer struct {
	Version      int    `json:"version"`
	RemoteAS     int    `json:"remoteAs"`
	MsgRcvd      int    `json:"msgRcvd"`
	MsgSent      int    `json:"msgSent"`
	TableVersion int    `json:"tableVersion"`
	InQ          int    `json:"inq"`
	OutQ         int    `json:"outq"`
	UpDown       string `json:"peerUptime"`
	State        string `json:"state"`
	PfxRcd       int    `json:"pfxRcd"`
	NeighborName string
}

var (
	vtyshBGPIPv6SummaryCommand      = "vtysh -c \"show bgp ipv6 summary json\""
	vtyshBGPIPv6BGPNeighborsCommand = "vtysh -c \"show bgp ipv6 neighbors json\""
)

func getIPv6BGPSummary(options sdc.OptionMap) ([]byte, error) {
	// Get data from vtysh command
	vtyshOutput, err := GetDataFromHostCommand(vtyshBGPIPv6SummaryCommand)
	if err != nil {
		log.Errorf("Unable to succesfully execute command %v, get err %v", vtyshBGPIPv6SummaryCommand, err)
		return nil, err
	}
	var vtyshResponse IPv6BGPSummaryResponse
	if err := json.Unmarshal([]byte(vtyshOutput), &vtyshResponse); err != nil {
		log.Errorf("Unable to create response from vtysh output %v", err)
		return nil, err
	}

	// Fetch neighbor name from CONFIG DB
	queries := [][]string{
		{"CONFIG_DB", "BGP_NEIGHBOR"},
	}

	bgpNeighborTableOutput, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to pull data for queries %v, got err %v", queries, err)
		return nil, err
	}

	// Modify vtysh data to use neighbor name from CONFIG DB
	for ip, peer := range vtyshResponse.IPv6Unicast.Peers {
		// If unable to find name in CONFIG_DB/BGP_NEIGHBOR using show command default of NotAvailable
		neighborName := "NotAvailable"
		if neighbor, found := bgpNeighborTableOutput[ip]; found {
			if entry, ok := neighbor.(map[string]interface{}); ok {
				if name, exists := entry["name"]; exists {
					if nameVal, ok := name.(string); ok {
						neighborName = nameVal
					}
				}
			}
		}
		peer.NeighborName = neighborName
		vtyshResponse.IPv6Unicast.Peers[ip] = peer
	}

	ipv6BGPSummaryJSON, err := json.Marshal(vtyshResponse)
	if err != nil {
		log.Errorf("Unable to create json data from modified vtysh response %v, got err %v", vtyshResponse, err)
		return nil, err
	}
	return ipv6BGPSummaryJSON, nil
}

func isIPv6Address(ip string) bool {
	// Check if the given string is a valid IPv6 address
	parsedIP := net.ParseIP(ip)
	return parsedIP != nil && parsedIP.To4() == nil
}

func isBGPNeighborPresent(ip string) bool {
	// Fetch neighbor name from CONFIG DB
	queries := [][]string{
		{"CONFIG_DB", "BGP_NEIGHBOR"},
	}

	bgpNeighborTableOutput, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to pull data for queries %v, got err %v", queries, err)
		return false
	}

	// Check if the IP exists in the neighbor table
	_, exists := bgpNeighborTableOutput[ip]
	return exists
}

func getIPv6BGPNeighbors(ip string) ([]byte, error) {
	var cmd string
	if ip != "" {
		// Construct command with specific neighbor
		cmd = fmt.Sprintf("vtysh -c \"show bgp ipv6 neighbors %s json\"", ip)
	} else {
		// Default command for all neighbors
		cmd = vtyshBGPIPv6BGPNeighborsCommand
	}

	// Get data from vtysh shell
	vtyshOutput, err := GetDataFromHostCommand(cmd)
	if err != nil {
		log.Errorf("Unable to successfully execute command %v, got err %v",
			cmd, err)
		return nil, err
	}

	var neighbors map[string]IPv6BGPPeer
	// Unmarshal JSON response
	if err := json.Unmarshal([]byte(vtyshOutput), &neighbors); err != nil {
		log.Errorf("Unable to create IPv6 BGP Neighbors response from vtysh output %v", err)
		return nil, err
	}

	// Marshal back to JSON to return clean structured data
	result, err := json.Marshal(neighbors)
	if err != nil {
		log.Errorf("Failed to marshal IPv6 BGP neighbors response: %v", err)
		return nil, err
	}

	return result, nil
}

func getIPv6BGPNeighborsRoutes(ip string) ([]byte, error) {
	// Construct command with specific neighbor
	cmd := fmt.Sprintf("vtysh -c \"show bgp ipv6 neighbors %s routes json\"", ip)

	// Get data from vtysh shell
	vtyshOutput, err := GetDataFromHostCommand(cmd)
	if err != nil {
		log.Errorf("Unable to successfully execute command %v, got err %v",
			cmd, err)
		return nil, err
	}

	// Define struct for unmarshalling
	var routesResp IPv6BGPNeighborRoutes

	// Unmarshal raw JSON into struct
	if err := json.Unmarshal([]byte(vtyshOutput), &routesResp); err != nil {
		log.Errorf("Failed to unmarshal vtysh output for %v: %v", cmd, err)
		return nil, fmt.Errorf("failed to parse routes response: %v", err)
	}

	// Marshal back to JSON to return clean structured data
	result, err := json.Marshal(routesResp)
	if err != nil {
		log.Errorf("Failed to marshal IPv6 BGP routes response: %v", err)
		return nil, err
	}

	return result, nil
}

func getIPv6BGPNeighborsAdvertisedRoutes(ip string) ([]byte, error) {
	// Construct vtysh command for advertised routes
	cmd := fmt.Sprintf("vtysh -c \"show bgp ipv6 neighbors %s advertised-routes json\"", ip)

	// Run the command
	vtyshOutput, err := GetDataFromHostCommand(cmd)
	if err != nil {
		log.Errorf("Unable to execute command %v, got err %v", cmd, err)
		return nil, err
	}

	// Unmarshal JSON response
	var advRoutesResp IPv6BGPAdvertisedRoutesResponse
	if err := json.Unmarshal([]byte(vtyshOutput), &advRoutesResp); err != nil {
		log.Errorf("Failed to unmarshal vtysh output: %v", err)
		return nil, fmt.Errorf("failed to parse advertised routes response: %v", err)
	}

	// Marshal back to JSON
	result, err := json.Marshal(advRoutesResp)
	if err != nil {
		log.Errorf("Failed to marshal advertised routes response: %v", err)
		return nil, err
	}

	return result, nil
}

func getIPv6BGPNeighborsReceivedRoutes(ip string) ([]byte, error) {
	// Construct vtysh command for received routes
	cmd := fmt.Sprintf("vtysh -c \"show bgp ipv6 neighbors %s received-routes json\"", ip)

	// Run the command
	vtyshOutput, err := GetDataFromHostCommand(cmd)
	if err != nil {
		log.Errorf("Unable to execute command %v, got err %v", cmd, err)
		return nil, err
	}

	// Unmarshal JSON response
	var recRoutesResp IPv6BGPReceivedRoutesResponse
	if err := json.Unmarshal([]byte(vtyshOutput), &recRoutesResp); err != nil {
		log.Errorf("Failed to unmarshal vtysh output: %v", err)
		return nil, fmt.Errorf("failed to parse received routes response: %v", err)
	}

	// Marshal back to JSON
	result, err := json.Marshal(recRoutesResp)
	if err != nil {
		log.Errorf("Failed to marshal received routes response: %v", err)
		return nil, err
	}

	return result, nil
}

// show ipv6 bgp neighbors -> list all neighbors
// show ipv6 bgp neighbors <ipaddress> -> show neighbor info
// show ipv6 bgp neighbors <ipaddress> routes|advertised-routes|received-routes → show specific option
func getIPv6BGPNeighborsHandler(options sdc.OptionMap) ([]byte, error) {
	ip, _ := options["ipaddress"].String()
	info_type, _ := options["info_type"].String()

	// Validate IPv6 address if provided
	if ip != "" && !isIPv6Address(ip) {
		log.Errorf("Invalid IPv6 address: %v", ip)
		return nil, fmt.Errorf("Invalid IPv6 address: %v", ip)
	}

	// If info_type is provided, ip becomes required
	if info_type != "" && ip == "" {
		log.Errorf("IPv6 address is required when info_type %v is specified", info_type)
		return nil, fmt.Errorf("IPv6 address is required when info_type %v is specified", info_type)
	}

	// Check neighbor exists if IP is provided
	if ip != "" && !isBGPNeighborPresent(ip) {
		log.Errorf("IPv6 BGP neighbor %v does not exist in CONFIG_DB", ip)
		return nil, fmt.Errorf("neighbor %v not found in CONFIG_DB", ip)
	}

	// Dispatch based on info_type
	switch info_type {
	case "routes":
		return getIPv6BGPNeighborsRoutes(ip)
	case "advertised-routes":
		return getIPv6BGPNeighborsAdvertisedRoutes(ip)
	case "received-routes":
		return getIPv6BGPNeighborsReceivedRoutes(ip)
	case "":
		return getIPv6BGPNeighbors(ip) // ip may be empty → list all
	default:
		log.Errorf("Invalid info_type: %v", info_type)
		return nil, fmt.Errorf("Invalid info_type: %v", info_type)
	}
}
