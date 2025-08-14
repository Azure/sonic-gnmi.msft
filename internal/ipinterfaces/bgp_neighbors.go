package ipinterfaces

import (
	"fmt"
	"net"
)

// getBGPNeighborsFromDB retrieves all BGP_NEIGHBOR entries from the CONFIG_DB.
// It returns a map where the key is the local interface IP address, and the value
// contains the BGP peer's info.
func getBGPNeighborsFromDB(logger Logger, dbQuery DBQueryFunc, namespace string) (map[string]*BGPNeighborInfo, error) {
	const bgpNeighborTable = "BGP_NEIGHBOR"

	var dbName string
	if namespace == defaultNamespace {
		dbName = "CONFIG_DB"
	} else {
		dbName = fmt.Sprintf("CONFIG_DB/%s", namespace)
	}
	query := [][]string{{dbName, bgpNeighborTable}}

	if dbQuery == nil {
		logger.Warnf("DBQuery is not configured; cannot read BGP neighbors")
		return nil, fmt.Errorf("DBQuery is not configured")
	}

	nsData, err := dbQuery(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query DB for BGP_NEIGHBOR in namespace '%s': %w", namespace, err)
	}
	logger.Debugf("DBQuery returned %d entries for namespace '%s' (table=%s)", len(nsData), namespace, bgpNeighborTable)

	bgpNeighbors := make(map[string]*BGPNeighborInfo)
	for neighborIP, data := range nsData {
		if net.ParseIP(neighborIP) == nil {
			logger.Warnf("Skipping entry %q: neighborIP is not a valid IP address", neighborIP)
			continue
		}
		logger.Debugf("Inspecting BGP_NEIGHBOR entry with key(neighborIP)=%q", neighborIP)
		if neighborData, ok := data.(map[string]interface{}); ok {
			if localAddr, ok := neighborData["local_addr"].(string); ok {
				nameStr := ""
				if v, ok := neighborData["name"].(string); ok {
					nameStr = v
				} else {
					logger.Debugf("Entry %q: missing or non-string 'name'; defaulting to empty", neighborIP)
				}
				logger.Debugf("Adding BGP neighbor: local_addr=%s neighbor_ip=%s name=%q", localAddr, neighborIP, nameStr)
				bgpNeighbors[localAddr] = &BGPNeighborInfo{
					Name:       nameStr,
					NeighborIP: neighborIP,
				}
			} else {
				logger.Debugf("Skipping entry %q: missing or non-string local_addr", neighborIP)
			}
		} else {
			logger.Debugf("Skipping entry %q: unexpected value type %T", neighborIP, data)
		}
	}
	logger.Debugf("Built bgpNeighbors map with %d entries", len(bgpNeighbors))

	return bgpNeighbors, nil
}
