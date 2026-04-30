package show_client

import (
	"encoding/json"
	"sort"
	"strings"

	natural "github.com/maruel/natural"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// ManagementInterfaceAddress represents a single management interface address entry
type ManagementInterfaceAddress struct {
	ManagementIPAddress             string `json:"Management IP address"`
	ManagementNetworkDefaultGateway string `json:"Management Network Default Gateway"`
}

const (
	MgmtInterfaceTable = "MGMT_INTERFACE"
)

// getManagementInterfaceAddress retrieves management interface IP addresses and gateways
func getManagementInterfaceAddress(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	// Query CONFIG_DB for MGMT_INTERFACE table
	queries := [][]string{
		{common.ConfigDb, MgmtInterfaceTable},
	}

	mgmtData, err := common.GetMapFromQueries(queries)
	if err != nil {
		return nil, err
	}

	addresses := make([]ManagementInterfaceAddress, 0)

	// Process the management interface data
	// The key format is typically: "eth0|10.0.0.1/24" where the IP is after |
	// The value contains gateway information

	// Get keys and sort them naturally (matches Python natsorted behavior)
	keys := make([]string, 0, len(mgmtData))
	for key := range mgmtData {
		keys = append(keys, key)
	}
	sort.Sort(natural.StringSlice(keys))

	for _, key := range keys {
		// Parse the key to extract interface and IP address
		// Format: "interface|ip_address"
		keyParts := strings.Split(key, "|")
		if len(keyParts) == 2 {
			ipAddress := keyParts[1]

			// Extract gateway from the value
			defaultGateway := ""
			if valueData, exists := mgmtData[key]; exists {
				if valueMap, isMap := valueData.(map[string]interface{}); isMap {
					if gwAddr, hasGw := valueMap["gwaddr"]; hasGw {
						if gwStr, isString := gwAddr.(string); isString {
							defaultGateway = gwStr
						}
					}
				}
			}

			addresses = append(addresses, ManagementInterfaceAddress{
				ManagementIPAddress:             ipAddress,
				ManagementNetworkDefaultGateway: defaultGateway,
			})
		}
	}

	return json.Marshal(addresses)
}
