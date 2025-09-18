package show_client

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// getIPv6Route is the getter function for the "show ipv6 route" command.
// This command is only supported on single-ASIC devices. It directly
// returns the JSON output from vtysh.
func getIPv6Route(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	if IsMultiAsic() {
		log.Errorf("Attempted to execute 'show ipv6 route' on a multi-ASIC platform")
		return nil, fmt.Errorf("'show ipv6 route' is not supported on multi-ASIC platforms")
	}

	vtyshCmdArgs := []string{"show", "ipv6", "route"}

	for _, a := range args {
		// Skip empty args
		// Skip "nexthop-group" since NHG ID already included in JSON output
		// Skip "json" since we will add it at the end
		if a == "" || a == "nexthop-group" || a == "json" {
			continue
		}
		vtyshCmdArgs = append(vtyshCmdArgs, a)
	}

	vtyshCmdArgs = append(vtyshCmdArgs, "json")

	// For single-ASIC, run in the default BGP instance on the host.
	vtyshIPv6RouteCmd := fmt.Sprintf("vtysh -c \"%s\"", strings.Join(vtyshCmdArgs, " "))

	output, err := common.GetDataFromHostCommand(vtyshIPv6RouteCmd)
	if err != nil {
		log.Errorf("Unable to successfully execute command %v, get err %v", vtyshIPv6RouteCmd, err)
		return nil, err
	}

	// Validate & compact JSON: unmarshal then directly marshal
	var parsed interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		log.Errorf("Invalid JSON from vtysh command '%s': %v", vtyshIPv6RouteCmd, err)
		return nil, err
	}
	return json.Marshal(parsed)
}
