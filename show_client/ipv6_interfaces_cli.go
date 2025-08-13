package show_client

import (
	"encoding/json"
	"fmt"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/internal/ipinterfaces"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// getIPv6Interfaces is the handler for the "show ipv6 interfaces" command.
// It uses the ipinterfaces library to get all interface details and returns them
// as a JSON byte slice.
func getIPv6Interfaces(options sdc.OptionMap) ([]byte, error) {
	log.V(2).Info("Executing 'show ipv6 interfaces' command via ipinterfaces library.")

	// Ensure ipinterfaces can access ConfigDB via our show_client DB helper.
	ipinterfaces.DBQuery = GetMapFromQueries

	// Extract optional namespace and display options from validated options.
	var namespacePtr *string
	if ns, ok := options["namespace"].String(); ok {
		namespacePtr = &ns
	}
	var displayPtr *string
	if dv, ok := options["display"].String(); ok {
		displayPtr = &dv
	}

	allIPv6Interfaces, err := ipinterfaces.GetIPInterfaces(ipinterfaces.AddressFamilyIPv6, namespacePtr, displayPtr)
	if err != nil {
		nsLog := "<auto>"
		if namespacePtr != nil {
			nsLog = *namespacePtr
		}
		dispLog := "<auto>"
		if displayPtr != nil {
			dispLog = *displayPtr
		}
		log.Errorf("Failed to get IP interface details (ns=%s display=%s): %v", nsLog, dispLog, err)
		return nil, fmt.Errorf("error retrieving interface information: %w", err)
	}

	jsonOutput, err := json.Marshal(allIPv6Interfaces)
	if err != nil {
		log.Errorf("Failed to marshal interface details to JSON: %v", err)
		return nil, fmt.Errorf("error formatting output: %w", err)
	}

	return jsonOutput, nil
}
