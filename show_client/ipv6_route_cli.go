package show_client

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// getIPv6Route is the getter function for the "show ipv6 route" command.
// This command is only supported on single-ASIC devices. It directly
// returns the JSON output from vtysh.
func getIPv6Route(options sdc.OptionMap) ([]byte, error) {
	if IsMultiAsic() {
		log.Errorf("Attempted to execute 'show ipv6 route' on a multi-ASIC platform")
		return nil, fmt.Errorf("'show ipv6 route' is not supported on multi-ASIC platforms")
	}

	cmdArgs := []string{"show", "ipv6", "route"}
	jsonArgPresent := false

	if val, ok := options["args"]; ok {
		if frrArgs, ok := val.String(); ok {
			userArgs := strings.Fields(frrArgs)
			for _, arg := range userArgs {
				if arg == "json" {
					jsonArgPresent = true
				}
				cmdArgs = append(cmdArgs, arg)
			}
		}
	}

	if !jsonArgPresent {
		cmdArgs = append(cmdArgs, "json")
	}

	vtyshCmdArgs := strings.Join(cmdArgs, " ")

	// For single-ASIC, run in the default BGP instance on the host.
	vtyshIPv6RouteCmd := fmt.Sprintf("vtysh -c \"%s\"", vtyshCmdArgs)

	output, err := GetDataFromHostCommand(vtyshIPv6RouteCmd)
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
