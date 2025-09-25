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

// GetInterfacesOptions holds the optional parameters for the GetIPInterfaces function.
// A nil pointer is valid and will result in default behavior.
type GetInterfacesOptions struct {
	Namespace *string
	Display   *string
}
