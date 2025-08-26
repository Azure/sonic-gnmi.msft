package ipinterfaces

// IPAddressDetail holds information about a single IP address, including BGP data.
type IPAddressDetail struct {
	Address         string `json:"address"`
	BGPNeighborIP   string `json:"bgp_neighbor_ip,omitempty"`
	BGPNeighborName string `json:"bgp_neighbor_name,omitempty"`
}

// IPInterfaceDetail holds all the consolidated information for a network interface.
type IPInterfaceDetail struct {
	Name        string            `json:"name"`
	IPAddresses []IPAddressDetail `json:"ip_addresses"`
	AdminStatus string            `json:"admin_status"`
	OperStatus  string            `json:"oper_status"`
	Master      string            `json:"master,omitempty"`
}

// BGPNeighborInfo holds the minimal BGP data needed, matching the ipintutil script.
type BGPNeighborInfo struct {
	Name       string // The descriptive name of the neighbor
	NeighborIP string // The IP address of the neighbor
}

// NamespacesByRole holds categorized lists of network namespaces based on their
// sub-role defined in the DEVICE_METADATA table in ConfigDB.
type NamespacesByRole struct {
	Frontend []string
	Backend  []string
	Fabric   []string
}

// Logger defines a simple logging interface that can be implemented by callers
// to integrate with their own logging framework (e.g., glog, zerolog, slog).
type Logger interface {
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Debugf(format string, args ...any)
}

// DBQueryFunc defines the signature for a function that can query the SONiC DB.
// This allows callers to inject their own database client implementation.
type DBQueryFunc func(q [][]string) (map[string]interface{}, error)

// Dependencies holds all the external dependencies required by the ipinterfaces package.
// This struct is passed to the main functions, making dependencies explicit.
type Dependencies struct {
	Logger  Logger
	DBQuery DBQueryFunc
}

// noOpLogger is a logger that discards all messages.
type noOpLogger struct{}

func (l *noOpLogger) Infof(format string, args ...any)  {}
func (l *noOpLogger) Warnf(format string, args ...any)  {}
func (l *noOpLogger) Debugf(format string, args ...any) {}

// DiscardLogger is a ready-to-use logger instance that performs no actions.
var DiscardLogger Logger = &noOpLogger{}

// GetInterfacesOptions holds the optional parameters for the GetIPInterfaces function.
// A nil pointer is valid and will result in default behavior.
type GetInterfacesOptions struct {
	Namespace *string
	Display   *string
}
