package show_client

type IPv6BGPNeighborsResponse struct {
	Neighbors map[string]IPv6BGPPeer `json:"neighbors"`
}

type IPv6BGPPeer struct {
	RemoteAS       int    `json:"remoteAs"`
	LocalAS        int    `json:"localAs"`
	LocalASDual    bool   `json:"localAsReplaceAsDualAs"`
	NbrExternal    bool   `json:"nbrExternalLink"`
	LocalRole      string `json:"localRole"`
	RemoteRole     string `json:"remoteRole"`
	NbrDesc        string `json:"nbrDesc"`
	Hostname       string `json:"hostname"`
	PeerGroup      string `json:"peerGroup"`
	BGPVersion     int    `json:"bgpVersion"`
	RemoteRouterID string `json:"remoteRouterId"`
	LocalRouterID  string `json:"localRouterId"`
	BGPState       string `json:"bgpState"`
	BGPTimerUpMsec int64  `json:"bgpTimerUpMsec"`
	BGPTimerUpStr  string `json:"bgpTimerUpString"`

	NeighborCapabilities NeighborCapabilities         `json:"neighborCapabilities"`
	GracefulRestartInfo  GracefulRestartInfo          `json:"gracefulRestartInfo"`
	MessageStats         MessageStats                 `json:"messageStats"`
	PrefixStats          PrefixStats                  `json:"prefixStats"`
	AddressFamilyInfo    map[string]AddressFamilyInfo `json:"addressFamilyInfo"`

	ConnectionsEstablished int    `json:"connectionsEstablished"`
	ConnectionsDropped     int    `json:"connectionsDropped"`
	LastResetTimerMsecs    int64  `json:"lastResetTimerMsecs"`
	LastNotificationReason string `json:"lastNotificationReason"`
	SoftwareVersion        string `json:"softwareVersion"`
	HostLocal              string `json:"hostLocal"`
	PortLocal              int    `json:"portLocal"`
	HostForeign            string `json:"hostForeign"`
	PortForeign            int    `json:"portForeign"`
	NextHop                string `json:"nexthop"`
	BGPConnection          string `json:"bgpConnection"`
}

// Nested structs
type NeighborCapabilities struct {
	FourByteAs              string                       `json:"4byteAs"`
	ExtendedMessage         string                       `json:"extendedMessage"`
	AddPath                 map[string]AddPathCapability `json:"addPath"`
	RouteRefresh            string                       `json:"routeRefresh"`
	MultiprotocolExtensions map[string]MultiprotocolExt  `json:"multiprotocolExtensions"`
	HostName                HostNameInfo                 `json:"hostName"`
}

type AddPathCapability struct {
	IPv6Unicast AddPathInfo `json:"ipv6Unicast"`
}

type AddPathInfo struct {
	RxAdvertisedAndReceived bool `json:"rxAdvertisedAndReceived"`
	RxAdvertised            bool `json:"rxAdvertised"`
	RxReceived              bool `json:"rxReceived"`
}

type MultiprotocolExt struct {
	IPv6Unicast bool `json:"ipv6Unicast"`
}

type HostNameInfo struct {
	AdvHostName   string `json:"advHostName"`
	AdvDomainName string `json:"advDomainName"`
}

type GracefulRestartInfo struct {
	EndOfRibSend map[string]bool   `json:"endOfRibSend"`
	EndOfRibRecv map[string]bool   `json:"endOfRibRecv"`
	LocalGrMode  string            `json:"localGrMode"`
	RemoteGrMode string            `json:"remoteGrMode"`
	Timers       map[string]int    `json:"timers"`
	IPv6Unicast  GracefulRestartAF `json:"ipv6Unicast"`
}

type GracefulRestartAF struct {
	FBit           bool            `json:"fBit"`
	EndOfRibStatus map[string]bool `json:"endOfRibStatus"`
	Timers         map[string]int  `json:"timers"`
}

type MessageStats struct {
	DepthInQ          int `json:"depthInq"`
	DepthOutQ         int `json:"depthOutq"`
	OpensSent         int `json:"opensSent"`
	OpensRecv         int `json:"opensRecv"`
	NotificationsSent int `json:"notificationsSent"`
	NotificationsRecv int `json:"notificationsRecv"`
	UpdatesSent       int `json:"updatesSent"`
	UpdatesRecv       int `json:"updatesRecv"`
	KeepalivesSent    int `json:"keepalivesSent"`
	KeepalivesRecv    int `json:"keepalivesRecv"`
}

type PrefixStats struct {
	InboundFiltered     int `json:"inboundFiltered"`
	AsPathLoop          int `json:"aspathLoop"`
	OriginatorLoop      int `json:"originatorLoop"`
	ClusterLoop         int `json:"clusterLoop"`
	InvalidNextHop      int `json:"invalidNextHop"`
	Withdrawn           int `json:"withdrawn"`
	AttributesDiscarded int `json:"attributesDiscarded"`
}

type AddressFamilyInfo struct {
	PeerGroupMember          string `json:"peerGroupMember"`
	UpdateGroupID            int    `json:"updateGroupId"`
	SubGroupID               int    `json:"subGroupId"`
	PacketQueueLength        int    `json:"packetQueueLength"`
	InboundSoftConfigPermit  bool   `json:"inboundSoftConfigPermit"`
	OutboundPathPolicyConfig bool   `json:"outboundPathPolicyConfig"`
	AcceptedPrefixCounter    int    `json:"acceptedPrefixCounter"`
	SentPrefixCounter        int    `json:"sentPrefixCounter"`
}
