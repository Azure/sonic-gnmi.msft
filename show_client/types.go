package show_client

// show ipv6 bgp neighbors output
type IPv6BGPPeer struct {
	RemoteAs                                 int64  `json:"remoteAs"`
	LocalAs                                  int64  `json:"localAs"`
	NbrExternalLink                          bool   `json:"nbrExternalLink"`
	LocalRole                                string `json:"localRole"`
	RemoteRole                               string `json:"remoteRole"`
	NbrDescription                           string `json:"nbrDesc"`
	Hostname                                 string `json:"hostname"`
	PeerGroup                                string `json:"peerGroup"`
	BGPVersion                               int    `json:"bgpVersion"`
	RemoteRouterId                           string `json:"remoteRouterId"`
	LocalRouterId                            string `json:"localRouterId"`
	BGPState                                 string `json:"bgpState"`
	BGPTimerUpMsec                           int64  `json:"bgpTimerUpMsec"`
	BGPTimerUpString                         string `json:"bgpTimerUpString"`
	BGPTimerUpEstablishedEpoch               int64  `json:"bgpTimerUpEstablishedEpoch"`
	BGPTimerLastRead                         int64  `json:"bgpTimerLastRead"`
	BGPTimerLastWrite                        int64  `json:"bgpTimerLastWrite"`
	BGPInUpdateElapsedTimeMsecs              int64  `json:"bgpInUpdateElapsedTimeMsecs"`
	BGPTimerConfiguredHoldTimeMsecs          int64  `json:"bgpTimerConfiguredHoldTimeMsecs"`
	BGPTimerConfiguredKeepAliveIntervalMsecs int64  `json:"bgpTimerConfiguredKeepAliveIntervalMsecs"`
	BGPTimerHoldTimeMsecs                    int64  `json:"bgpTimerHoldTimeMsecs"`
	BGPTimerKeepAliveIntervalMsecs           int64  `json:"bgpTimerKeepAliveIntervalMsecs"`
	BGPTcpMssConfigured                      int    `json:"bgpTcpMssConfigured"`
	BGPTcpMssSynced                          int    `json:"bgpTcpMssSynced"`
	ExtendedOptionalParametersLength         bool   `json:"extendedOptionalParametersLength"`
	BGPTimerConfiguredConditionalAdvSec      int    `json:"bgpTimerConfiguredConditionalAdvertisementsSec"`

	NeighborCapabilities NeighborCapabilities `json:"neighborCapabilities"`
	GracefulRestartInfo  GracefulRestartInfo  `json:"gracefulRestartInfo"`
	MessageStats         MessageStats         `json:"messageStats"`

	MinBtwnAdvertisementRunsTimerMsecs int `json:"minBtwnAdvertisementRunsTimerMsecs"`

	AddressFamilyInfo map[string]AddressFamilyDetails `json:"addressFamilyInfo"`

	ConnectionsEstablished    int    `json:"connectionsEstablished"`
	ConnectionsDropped        int    `json:"connectionsDropped"`
	LastResetTimerMsecs       int64  `json:"lastResetTimerMsecs"`
	LastResetDueTo            string `json:"lastResetDueTo"`
	LastResetCode             int    `json:"lastResetCode"`
	SoftwareVersion           string `json:"softwareVersion"`
	ExternalBgpNbrMaxHopsAway int    `json:"externalBgpNbrMaxHopsAway"`
	HostLocal                 string `json:"hostLocal"`
	PortLocal                 int    `json:"portLocal"`
	HostForeign               string `json:"hostForeign"`
	PortForeign               int    `json:"portForeign"`
	Nexthop                   string `json:"nexthop"`
	NexthopGlobal             string `json:"nexthopGlobal"`
	NexthopLocal              string `json:"nexthopLocal"`
	BGPConnection             string `json:"bgpConnection"`
	ConnectRetryTimer         int    `json:"connectRetryTimer"`
	EstimatedRttInMsecs       int    `json:"estimatedRttInMsecs"`
	ReadThread                string `json:"readThread"`
	WriteThread               string `json:"writeThread"`
}

// Nested structs
type NeighborCapabilities struct {
	FourByteAs                      string              `json:"4byteAs"`
	ExtendedMessage                 string              `json:"extendedMessage"`
	AddPath                         AddPathCapabilities `json:"addPath"`
	LongLivedGracefulRestart        string              `json:"longLivedGracefulRestart"`
	RouteRefresh                    string              `json:"routeRefresh"`
	EnhancedRouteRefresh            string              `json:"enhancedRouteRefresh"`
	MultiprotocolExtensions         MultiProtocolCaps   `json:"multiprotocolExtensions"`
	HostName                        HostNameInfo        `json:"hostName"`
	SoftwareVersion                 map[string]string   `json:"softwareVersion"`
	GracefulRestart                 string              `json:"gracefulRestart"`
	GracefulRestartRemoteTimerMsecs int                 `json:"gracefulRestartRemoteTimerMsecs"`
	AddressFamiliesByPeer           string              `json:"addressFamiliesByPeer"`
}

type AddPathCapabilities struct {
	IPv6Unicast struct {
		RxAdvertisedAndReceived bool `json:"rxAdvertisedAndReceived"`
	} `json:"ipv6Unicast"`
}

type MultiProtocolCaps struct {
	IPv6Unicast struct {
		AdvertisedAndReceived bool `json:"advertisedAndReceived"`
	} `json:"ipv6Unicast"`
}

type HostNameInfo struct {
	AdvHostName   string `json:"advHostName"`
	AdvDomainName string `json:"advDomainName"`
}

type GracefulRestartInfo struct {
	EndOfRibSend struct {
		IPv6Unicast bool `json:"ipv6Unicast"`
	} `json:"endOfRibSend"`
	EndOfRibRecv struct {
		IPv6Unicast bool `json:"ipv6Unicast"`
	} `json:"endOfRibRecv"`
	LocalGrMode  string `json:"localGrMode"`
	RemoteGrMode string `json:"remoteGrMode"`
	RBit         bool   `json:"rBit"`
	NBit         bool   `json:"nBit"`
	Timers       struct {
		ConfiguredRestartTimer  int `json:"configuredRestartTimer"`
		ConfiguredLlgrStaleTime int `json:"configuredLlgrStaleTime"`
		ReceivedRestartTimer    int `json:"receivedRestartTimer"`
	} `json:"timers"`
	IPv6Unicast struct {
		FBit           bool `json:"fBit"`
		EndOfRibStatus struct {
			EndOfRibSend            bool `json:"endOfRibSend"`
			EndOfRibSentAfterUpdate bool `json:"endOfRibSentAfterUpdate"`
			EndOfRibRecv            bool `json:"endOfRibRecv"`
		} `json:"endOfRibStatus"`
		Timers struct {
			StalePathTimer         int `json:"stalePathTimer"`
			LlgrStaleTime          int `json:"llgrStaleTime"`
			SelectionDeferralTimer int `json:"selectionDeferralTimer"`
		} `json:"timers"`
	} `json:"ipv6Unicast"`
}

type MessageStats struct {
	DepthInq          int `json:"depthInq"`
	DepthOutq         int `json:"depthOutq"`
	OpensSent         int `json:"opensSent"`
	OpensRecv         int `json:"opensRecv"`
	NotificationsSent int `json:"notificationsSent"`
	NotificationsRecv int `json:"notificationsRecv"`
	UpdatesSent       int `json:"updatesSent"`
	UpdatesRecv       int `json:"updatesRecv"`
	KeepalivesSent    int `json:"keepalivesSent"`
	KeepalivesRecv    int `json:"keepalivesRecv"`
	RouteRefreshSent  int `json:"routeRefreshSent"`
	RouteRefreshRecv  int `json:"routeRefreshRecv"`
	CapabilitySent    int `json:"capabilitySent"`
	CapabilityRecv    int `json:"capabilityRecv"`
	TotalSent         int `json:"totalSent"`
	TotalRecv         int `json:"totalRecv"`
}

type AddressFamilyDetails struct {
	PeerGroupMember                   string `json:"peerGroupMember"`
	UpdateGroupId                     int    `json:"updateGroupId"`
	SubGroupId                        int    `json:"subGroupId"`
	PacketQueueLength                 int    `json:"packetQueueLength"`
	InboundSoftConfigPermit           bool   `json:"inboundSoftConfigPermit"`
	AllowAsInCount                    int    `json:"allowAsInCount"`
	CommAttriSentToNbr                string `json:"commAttriSentToNbr"`
	InboundPathPolicyConfig           bool   `json:"inboundPathPolicyConfig"`
	OutboundPathPolicyConfig          bool   `json:"outboundPathPolicyConfig"`
	RouteMapForIncomingAdvertisements string `json:"routeMapForIncomingAdvertisements"`
	RouteMapForOutgoingAdvertisements string `json:"routeMapForOutgoingAdvertisements"`
	AcceptedPrefixCounter             int    `json:"acceptedPrefixCounter"`
	SentPrefixCounter                 int    `json:"sentPrefixCounter"`
	PrefixAllowedMax                  int    `json:"prefixAllowedMax"`
	PrefixAllowedMaxWarning           bool   `json:"prefixAllowedMaxWarning"`
	PrefixAllowedWarningThresh        int    `json:"prefixAllowedWarningThresh"`
}

// show ipv6 bgp neighbors <ipaddress> routes output
type IPv6BGPNeighborRoutes struct {
	VrfID         int                            `json:"vrfId"`
	VrfName       string                         `json:"vrfName"`
	TableVersion  int                            `json:"tableVersion"`
	RouterID      string                         `json:"routerId"`
	DefaultLocPrf int                            `json:"defaultLocPrf"`
	LocalAS       int                            `json:"localAS"`
	Routes        map[string][]IPv6BGPRouteEntry `json:"routes"`
}

type IPv6BGPRouteEntry struct {
	Valid           bool                  `json:"valid"`
	Multipath       bool                  `json:"multipath,omitempty"`
	BestPath        bool                  `json:"bestpath,omitempty"`
	SelectionReason string                `json:"selectionReason,omitempty"`
	PathFrom        string                `json:"pathFrom"`
	Prefix          string                `json:"prefix"`
	PrefixLen       int                   `json:"prefixLen"`
	Network         string                `json:"network"`
	Version         int                   `json:"version"`
	Weight          int                   `json:"weight"`
	PeerID          string                `json:"peerId"`
	Path            string                `json:"path"`
	Origin          string                `json:"origin"`
	Nexthops        []IPv6BGPNexthopEntry `json:"nexthops"`
}

type IPv6BGPNexthopEntry struct {
	IP    string `json:"ip"`
	AFI   string `json:"afi"`
	Scope string `json:"scope"`
	Used  bool   `json:"used"`
}

// show ipv6 bgp neighbors <ipaddress> advertised-routes output
type IPv6BGPAdvertisedRoutesResponse struct {
	BGPTableVersion       int                                    `json:"bgpTableVersion"`
	BGPLocalRouterID      string                                 `json:"bgpLocalRouterId"`
	DefaultLocPrf         int                                    `json:"defaultLocPrf"`
	LocalAS               int                                    `json:"localAS"`
	AdvertisedRoutes      map[string]IPv6BGPAdvertisedRouteEntry `json:"advertisedRoutes"`
	TotalPrefixCounter    int                                    `json:"totalPrefixCounter"`
	FilteredPrefixCounter int                                    `json:"filteredPrefixCounter"`
}

type IPv6BGPAdvertisedRouteEntry struct {
	AddrPrefix    string `json:"addrPrefix"`
	PrefixLen     int    `json:"prefixLen"`
	Network       string `json:"network"`
	NextHopGlobal string `json:"nextHopGlobal"`
	Weight        int    `json:"weight"`
	Path          string `json:"path"`
	Origin        string `json:"origin"`
	Valid         bool   `json:"valid"`
	Best          bool   `json:"best,omitempty"`
	Multipath     bool   `json:"multipath,omitempty"`
}

// show ipv6 bgp neighbors <ipaddress> received-routes output
type IPv6BGPReceivedRoutesResponse struct {
	BGPTableVersion       int                                  `json:"bgpTableVersion"`
	BGPLocalRouterID      string                               `json:"bgpLocalRouterId"`
	DefaultLocPrf         int                                  `json:"defaultLocPrf"`
	LocalAS               int                                  `json:"localAS"`
	ReceivedRoutes        map[string]IPv6BGPReceivedRouteEntry `json:"receivedRoutes"`
	TotalPrefixCounter    int                                  `json:"totalPrefixCounter"`
	FilteredPrefixCounter int                                  `json:"filteredPrefixCounter"`
}

type IPv6BGPReceivedRouteEntry struct {
	AddrPrefix      string `json:"addrPrefix"`
	PrefixLen       int    `json:"prefixLen"`
	Network         string `json:"network"`
	NextHopGlobal   string `json:"nextHopGlobal"`
	Weight          int    `json:"weight"`
	Path            string `json:"path"`
	Origin          string `json:"origin"`
	Valid           bool   `json:"valid"`
	Best            bool   `json:"best,omitempty"`
	Multipath       bool   `json:"multipath,omitempty"`
	SelectionReason string `json:"selectionReason,omitempty"`
}
