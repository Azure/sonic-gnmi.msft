package ipinterfaces

// IPAddressDetail holds information about a single IP address, including BGP data.
type IPAddressDetail struct {
	Address         string `json:"address"`
	BGPNeighborIP   string `json:"bgp_neighbor_ip,omitempty"`
	BGPNeighborName string `json:"bgp_neighbor_name,omitempty"`
}

// NamespacesByRole holds categorized lists of network namespaces based on their
// sub-role defined in the DEVICE_METADATA table in ConfigDB.
type NamespacesByRole struct {
	Frontend []string
	Backend  []string
	Fabric   []string
}
