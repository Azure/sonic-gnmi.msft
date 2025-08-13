package ipinterfaces

import (
	"fmt"
	"strings"
)

// DBQuery is an injectable function to fetch data from SONiC DB.
// Callers (e.g., CLI) can set this to sonic_data_client.GetMapFromQueries.
// If left nil, getBGPNeighborsFromDB returns an error.
var DBQuery func(q [][]string) (map[string]interface{}, error)

// BGPNeighborInfo holds the minimal BGP data needed, matching the ipintutil script.
type BGPNeighborInfo struct {
	Name       string // The descriptive name of the neighbor
	NeighborIP string // The IP address of the neighbor
}

// getBGPNeighborsFromDB retrieves all BGP_NEIGHBOR entries from the CONFIG_DB.
// It returns a map where the key is the local interface IP address, and the value
// contains the BGP peer's info.
func getBGPNeighborsFromDB(namespace string) (map[string]*BGPNeighborInfo, error) {
	const bgpNeighborTable = "BGP_NEIGHBOR"

	var dbName string
	if namespace == "" {
		dbName = "CONFIG_DB"
	} else {
		dbName = fmt.Sprintf("CONFIG_DB/%s", namespace)
	}
	query := [][]string{{dbName, bgpNeighborTable}}

	if DBQuery == nil {
		return nil, fmt.Errorf("DBQuery is not configured")
	}
	// GetMapFromQueries can query multiple namespaces, but we'll do one at a time.
	nsData, err := DBQuery(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query DB for BGP_NEIGHBOR in namespace '%s': %w", namespace, err)
	}

	bgpNeighbors := make(map[string]*BGPNeighborInfo)

	// The result is map[table_key]map[field]value
	for key, data := range nsData {
		// Key is in the format "BGP_NEIGHBOR|10.0.0.1"
		parts := strings.Split(key, "|")
		if len(parts) < 2 {
			continue
		}
		neighborIP := parts[1]

		if neighborData, ok := data.(map[string]interface{}); ok {
			// The python script uses the 'local_addr' as the key to find the peer.
			if localAddr, ok := neighborData["local_addr"].(string); ok {
				bgpNeighbors[localAddr] = &BGPNeighborInfo{
					Name:       fmt.Sprintf("%v", neighborData["name"]),
					NeighborIP: neighborIP,
				}
			}
		}
	}

	return bgpNeighbors, nil
}
