package show_client

import (
	"encoding/json"
	"fmt"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/internal/ipinterfaces"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// glogAdapter provides a thin wrapper around glog to satisfy the ipinterfaces.Logger interface.
type glogAdapter struct{}

func (g *glogAdapter) Infof(format string, args ...any)  { log.Infof(format, args...) }
func (g *glogAdapter) Warnf(format string, args ...any)  { log.Warningf(format, args...) }
func (g *glogAdapter) Debugf(format string, args ...any) { log.V(6).Infof(format, args...) }

// getIPv6Interfaces is the handler for the "show ipv6 interfaces" command.
// It uses the ipinterfaces library to get all interface details and returns them
// as a JSON byte slice.
func getIPv6Interfaces(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	log.V(2).Info("Executing 'show ipv6 interfaces' command via ipinterfaces library.")

	// Instantiate the provider with its dependencies.
	deps := ipinterfaces.Dependencies{
		Logger:  &glogAdapter{},
		DBQuery: common.GetMapFromQueries,
	}

	// Extract optional namespace and display options from validated options.
	opts := &ipinterfaces.GetInterfacesOptions{}
	if ns, ok := options["namespace"].String(); ok {
		opts.Namespace = &ns
	}
	if dv, ok := options["display"].String(); ok {
		opts.Display = &dv
	}

	allIPv6Interfaces, err := ipinterfaces.GetIPInterfaces(deps, ipinterfaces.AddressFamilyIPv6, opts)
	if err != nil {
		nsLog := "<auto>"
		if opts.Namespace != nil {
			nsLog = *opts.Namespace
		}
		dispLog := "<auto>"
		if opts.Display != nil {
			dispLog = *opts.Display
		}
		log.Errorf("Failed to get IP interface details (ns=%s display=%s): %v", nsLog, dispLog, err)
		return nil, fmt.Errorf("error retrieving interface information: %w", err)
	}

	// Transform slice into a map keyed by interface name, omitting the redundant name field in the value.
	type ipv6InterfaceEntry struct {
		IPv6Addresses []ipinterfaces.IPAddressDetail `json:"ipv6_addresses"`
		AdminStatus   string                         `json:"admin_status"`
		OperStatus    string                         `json:"oper_status"`
		Master        string                         `json:"master"`
	}
	response := make(map[string]ipv6InterfaceEntry, len(allIPv6Interfaces))
	for _, d := range allIPv6Interfaces {
		// Backfill missing BGP neighbor details with "N/A" for presentation.
		addrs := make([]ipinterfaces.IPAddressDetail, 0, len(d.IPAddresses))
		for _, a := range d.IPAddresses {
			if a.BGPNeighborIP == "" {
				a.BGPNeighborIP = "N/A"
			}
			if a.BGPNeighborName == "" {
				a.BGPNeighborName = "N/A"
			}
			addrs = append(addrs, a)
		}
		response[d.Name] = ipv6InterfaceEntry{
			IPv6Addresses: addrs,
			AdminStatus:   d.AdminStatus,
			OperStatus:    d.OperStatus,
			Master:        d.Master,
		}
	}

	jsonOutput, err := json.Marshal(response)
	if err != nil {
		log.Errorf("Failed to marshal interface details to JSON: %v", err)
		return nil, fmt.Errorf("error formatting output: %w", err)
	}

	return jsonOutput, nil
}
