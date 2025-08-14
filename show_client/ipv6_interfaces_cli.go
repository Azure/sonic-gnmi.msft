package show_client

import (
	"encoding/json"
	"fmt"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/internal/ipinterfaces"
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
func getIPv6Interfaces(options sdc.OptionMap) ([]byte, error) {
	log.V(2).Info("Executing 'show ipv6 interfaces' command via ipinterfaces library.")

	// Instantiate the provider with its dependencies.
	deps := ipinterfaces.Dependencies{
		Logger:  &glogAdapter{},
		DBQuery: GetMapFromQueries,
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

	jsonOutput, err := json.Marshal(allIPv6Interfaces)
	if err != nil {
		log.Errorf("Failed to marshal interface details to JSON: %v", err)
		return nil, fmt.Errorf("error formatting output: %w", err)
	}

	return jsonOutput, nil
}
