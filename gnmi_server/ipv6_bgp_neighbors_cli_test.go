package gnmi

// ipv6_bgp_neighbors_cli_test.go

// Tests SHOW ipv6 bgp neighbors

import (
	"crypto/tls"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"

	"github.com/agiledragon/gomonkey/v2"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestGetIPv6BGPNeighbors(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}

	conn, err := grpc.Dial(TargetAddr, opts...)
	if err != nil {
		t.Fatalf("Dialing to %q failed: %v", TargetAddr, err)
	}
	defer conn.Close()

	gClient := pb.NewGNMIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()

	bgpNeighborsFileName := "../testdata/ipv6_bgp_neighbors/BGP_NEIGHBOR.txt"
	bgpNeighborsTwoFileName := "../testdata/ipv6_bgp_neighbors/BGP_NEIGHBOR_2.txt"
	ipv6BGPNeighborsAll := `{"fc00::7a":{"remoteAs":64600,"localAs":65100,"nbrExternalLink":true,"localRole":"undefined","remoteRole":"undefined","nbrDesc":"ARISTA03T1","hostname":"Unknown","peerGroup":"TIER1_V6","bgpVersion":4,"remoteRouterId":"100.1.0.3","localRouterId":"10.1.0.32","bgpState":"Established","bgpTimerUpMsec":77574000,"bgpTimerUpString":"21:32:54","bgpTimerUpEstablishedEpoch":1756274149,"bgpTimerLastRead":4000,"bgpTimerLastWrite":54000,"bgpInUpdateElapsedTimeMsecs":77570000,"bgpTimerConfiguredHoldTimeMsecs":180000,"bgpTimerConfiguredKeepAliveIntervalMsecs":60000,"bgpTimerHoldTimeMsecs":180000,"bgpTimerKeepAliveIntervalMsecs":60000,"bgpTcpMssConfigured":0,"bgpTcpMssSynced":9028,"extendedOptionalParametersLength":false,"bgpTimerConfiguredConditionalAdvertisementsSec":60,"neighborCapabilities":{"4byteAs":"advertisedAndReceived","extendedMessage":"advertised","addPath":{"ipv6Unicast":{"rxAdvertisedAndReceived":true}},"longLivedGracefulRestart":"advertised","routeRefresh":"advertisedAndReceived","enhancedRouteRefresh":"advertisedAndReceived","multiprotocolExtensions":{"ipv6Unicast":{"advertisedAndReceived":true}},"hostName":{"advHostName":"str4-7050cx3-c28s4-1","advDomainName":"n\/a"},"softwareVersion":{},"gracefulRestart":"advertisedAndReceived","gracefulRestartRemoteTimerMsecs":300000,"addressFamiliesByPeer":"none"},"gracefulRestartInfo":{"endOfRibSend":{"ipv6Unicast":true},"endOfRibRecv":{"ipv6Unicast":true},"localGrMode":"Restart*","remoteGrMode":"Helper","rBit":false,"nBit":true,"timers":{"configuredRestartTimer":240,"configuredLlgrStaleTime":0,"receivedRestartTimer":300},"ipv6Unicast":{"fBit":false,"endOfRibStatus":{"endOfRibSend":true,"endOfRibSentAfterUpdate":false,"endOfRibRecv":true},"timers":{"stalePathTimer":360,"llgrStaleTime":0,"selectionDeferralTimer":360}}},"messageStats":{"depthInq":0,"depthOutq":0,"opensSent":1,"opensRecv":0,"notificationsSent":1,"notificationsRecv":0,"updatesSent":3209,"updatesRecv":3202,"keepalivesSent":1292,"keepalivesRecv":1512,"routeRefreshSent":0,"routeRefreshRecv":0,"capabilitySent":0,"capabilityRecv":0,"totalSent":4503,"totalRecv":4714},"minBtwnAdvertisementRunsTimerMsecs":0,"addressFamilyInfo":{"ipv6Unicast":{"peerGroupMember":"TIER1_V6","updateGroupId":2,"subGroupId":3,"packetQueueLength":0,"inboundSoftConfigPermit":true,"allowAsInCount":1,"commAttriSentToNbr":"extendedAndStandard","inboundPathPolicyConfig":true,"outboundPathPolicyConfig":true,"routeMapForIncomingAdvertisements":"FROM_TIER1_V6","routeMapForOutgoingAdvertisements":"TO_TIER1_V6","acceptedPrefixCounter":6400,"sentPrefixCounter":6412,"prefixAllowedMax":8000,"prefixAllowedMaxWarning":true,"prefixAllowedWarningThresh":90}},"connectionsEstablished":1,"connectionsDropped":0,"lastResetTimerMsecs":77591000,"lastResetDueTo":"No AFI\/SAFI activated for peer","lastResetCode":30,"softwareVersion":"n\/a","externalBgpNbrMaxHopsAway":1,"hostLocal":"fc00::79","portLocal":179,"hostForeign":"fc00::7a","portForeign":39985,"nexthop":"10.0.0.60","nexthopGlobal":"fc00::79","nexthopLocal":"fe80::6a8b:f4ff:fe87:9ddc","bgpConnection":"sharedNetwork","connectRetryTimer":120,"estimatedRttInMsecs":7,"readThread":"on","writeThread":"on"},"fc00::7e":{"remoteAs":64600,"localAs":65100,"nbrExternalLink":true,"localRole":"undefined","remoteRole":"undefined","nbrDesc":"ARISTA04T1","hostname":"Unknown","peerGroup":"TIER1_V6","bgpVersion":4,"remoteRouterId":"100.1.0.4","localRouterId":"10.1.0.32","bgpState":"Established","bgpTimerUpMsec":77573000,"bgpTimerUpString":"21:32:53","bgpTimerUpEstablishedEpoch":1756274150,"bgpTimerLastRead":30000,"bgpTimerLastWrite":53000,"bgpInUpdateElapsedTimeMsecs":77570000,"bgpTimerConfiguredHoldTimeMsecs":180000,"bgpTimerConfiguredKeepAliveIntervalMsecs":60000,"bgpTimerHoldTimeMsecs":180000,"bgpTimerKeepAliveIntervalMsecs":60000,"bgpTcpMssConfigured":0,"bgpTcpMssSynced":9028,"extendedOptionalParametersLength":false,"bgpTimerConfiguredConditionalAdvertisementsSec":60,"neighborCapabilities":{"4byteAs":"advertisedAndReceived","extendedMessage":"advertised","addPath":{"ipv6Unicast":{"rxAdvertisedAndReceived":true}},"longLivedGracefulRestart":"advertised","routeRefresh":"advertisedAndReceived","enhancedRouteRefresh":"advertisedAndReceived","multiprotocolExtensions":{"ipv6Unicast":{"advertisedAndReceived":true}},"hostName":{"advHostName":"str4-7050cx3-c28s4-1","advDomainName":"n\/a"},"softwareVersion":{},"gracefulRestart":"advertisedAndReceived","gracefulRestartRemoteTimerMsecs":300000,"addressFamiliesByPeer":"none"},"gracefulRestartInfo":{"endOfRibSend":{"ipv6Unicast":true},"endOfRibRecv":{"ipv6Unicast":true},"localGrMode":"Restart*","remoteGrMode":"Helper","rBit":false,"nBit":true,"timers":{"configuredRestartTimer":240,"configuredLlgrStaleTime":0,"receivedRestartTimer":300},"ipv6Unicast":{"fBit":false,"endOfRibStatus":{"endOfRibSend":true,"endOfRibSentAfterUpdate":false,"endOfRibRecv":true},"timers":{"stalePathTimer":360,"llgrStaleTime":0,"selectionDeferralTimer":360}}},"messageStats":{"depthInq":0,"depthOutq":0,"opensSent":2,"opensRecv":1,"notificationsSent":1,"notificationsRecv":2,"updatesSent":3209,"updatesRecv":3202,"keepalivesSent":1293,"keepalivesRecv":1517,"routeRefreshSent":0,"routeRefreshRecv":0,"capabilitySent":0,"capabilityRecv":0,"totalSent":4505,"totalRecv":4722},"minBtwnAdvertisementRunsTimerMsecs":0,"addressFamilyInfo":{"ipv6Unicast":{"peerGroupMember":"TIER1_V6","updateGroupId":2,"subGroupId":3,"packetQueueLength":0,"inboundSoftConfigPermit":true,"allowAsInCount":1,"commAttriSentToNbr":"extendedAndStandard","inboundPathPolicyConfig":true,"outboundPathPolicyConfig":true,"routeMapForIncomingAdvertisements":"FROM_TIER1_V6","routeMapForOutgoingAdvertisements":"TO_TIER1_V6","acceptedPrefixCounter":6400,"sentPrefixCounter":6412,"prefixAllowedMax":8000,"prefixAllowedMaxWarning":true,"prefixAllowedWarningThresh":90}},"connectionsEstablished":1,"connectionsDropped":0,"lastResetTimerMsecs":77591000,"lastResetDueTo":"No AFI\/SAFI activated for peer","lastResetCode":30,"softwareVersion":"n\/a","externalBgpNbrMaxHopsAway":1,"hostLocal":"fc00::7d","portLocal":179,"hostForeign":"fc00::7e","portForeign":32935,"nexthop":"10.0.0.62","nexthopGlobal":"fc00::7d","nexthopLocal":"fe80::6a8b:f4ff:fe87:9ddc","bgpConnection":"sharedNetwork","connectRetryTimer":120,"estimatedRttInMsecs":3,"readThread":"on","writeThread":"on"}}`
	ipv6BGPNeighborsIPSpecific := `{"fc00::7a":{"remoteAs":64600,"localAs":65100,"nbrExternalLink":true,"localRole":"undefined","remoteRole":"undefined","nbrDesc":"ARISTA03T1","hostname":"Unknown","peerGroup":"TIER1_V6","bgpVersion":4,"remoteRouterId":"100.1.0.3","localRouterId":"10.1.0.32","bgpState":"Established","bgpTimerUpMsec":77574000,"bgpTimerUpString":"21:32:54","bgpTimerUpEstablishedEpoch":1756274149,"bgpTimerLastRead":4000,"bgpTimerLastWrite":54000,"bgpInUpdateElapsedTimeMsecs":77570000,"bgpTimerConfiguredHoldTimeMsecs":180000,"bgpTimerConfiguredKeepAliveIntervalMsecs":60000,"bgpTimerHoldTimeMsecs":180000,"bgpTimerKeepAliveIntervalMsecs":60000,"bgpTcpMssConfigured":0,"bgpTcpMssSynced":9028,"extendedOptionalParametersLength":false,"bgpTimerConfiguredConditionalAdvertisementsSec":60,"neighborCapabilities":{"4byteAs":"advertisedAndReceived","extendedMessage":"advertised","addPath":{"ipv6Unicast":{"rxAdvertisedAndReceived":true}},"longLivedGracefulRestart":"advertised","routeRefresh":"advertisedAndReceived","enhancedRouteRefresh":"advertisedAndReceived","multiprotocolExtensions":{"ipv6Unicast":{"advertisedAndReceived":true}},"hostName":{"advHostName":"str4-7050cx3-c28s4-1","advDomainName":"n\/a"},"softwareVersion":{},"gracefulRestart":"advertisedAndReceived","gracefulRestartRemoteTimerMsecs":300000,"addressFamiliesByPeer":"none"},"gracefulRestartInfo":{"endOfRibSend":{"ipv6Unicast":true},"endOfRibRecv":{"ipv6Unicast":true},"localGrMode":"Restart*","remoteGrMode":"Helper","rBit":false,"nBit":true,"timers":{"configuredRestartTimer":240,"configuredLlgrStaleTime":0,"receivedRestartTimer":300},"ipv6Unicast":{"fBit":false,"endOfRibStatus":{"endOfRibSend":true,"endOfRibSentAfterUpdate":false,"endOfRibRecv":true},"timers":{"stalePathTimer":360,"llgrStaleTime":0,"selectionDeferralTimer":360}}},"messageStats":{"depthInq":0,"depthOutq":0,"opensSent":1,"opensRecv":0,"notificationsSent":1,"notificationsRecv":0,"updatesSent":3209,"updatesRecv":3202,"keepalivesSent":1292,"keepalivesRecv":1512,"routeRefreshSent":0,"routeRefreshRecv":0,"capabilitySent":0,"capabilityRecv":0,"totalSent":4503,"totalRecv":4714},"minBtwnAdvertisementRunsTimerMsecs":0,"addressFamilyInfo":{"ipv6Unicast":{"peerGroupMember":"TIER1_V6","updateGroupId":2,"subGroupId":3,"packetQueueLength":0,"inboundSoftConfigPermit":true,"allowAsInCount":1,"commAttriSentToNbr":"extendedAndStandard","inboundPathPolicyConfig":true,"outboundPathPolicyConfig":true,"routeMapForIncomingAdvertisements":"FROM_TIER1_V6","routeMapForOutgoingAdvertisements":"TO_TIER1_V6","acceptedPrefixCounter":6400,"sentPrefixCounter":6412,"prefixAllowedMax":8000,"prefixAllowedMaxWarning":true,"prefixAllowedWarningThresh":90}},"connectionsEstablished":1,"connectionsDropped":0,"lastResetTimerMsecs":77591000,"lastResetDueTo":"No AFI\/SAFI activated for peer","lastResetCode":30,"softwareVersion":"n\/a","externalBgpNbrMaxHopsAway":1,"hostLocal":"fc00::79","portLocal":179,"hostForeign":"fc00::7a","portForeign":39985,"nexthop":"10.0.0.60","nexthopGlobal":"fc00::79","nexthopLocal":"fe80::6a8b:f4ff:fe87:9ddc","bgpConnection":"sharedNetwork","connectRetryTimer":120,"estimatedRttInMsecs":7,"readThread":"on","writeThread":"on"}}`
	ipv6BGPNeighborsRoutes := `{"vrfId":0,"tableVersion":6412,"localAS":65100,"routerId":"10.1.0.32","routes":{"20c1:9c8::/64":[{"origin":"IGP","pathFrom":"external","network":"20c1:9c8::/64","weight":0,"bestpath":true,"selectionReason":"Older Path","prefix":"20c1:9c8::","valid":true,"peerId":"fc00::72","version":6267,"prefixLen":64,"path":"64600 65534 64795 65509","nexthops":[{"ip":"fc00::72","used":true,"afi":"ipv6","scope":"global"}]}],"20c0:df78::/64":[{"origin":"IGP","pathFrom":"external","network":"20c0:df78::/64","weight":0,"bestpath":true,"selectionReason":"Older Path","prefix":"20c0:df78::","valid":true,"peerId":"fc00::72","version":3559,"prefixLen":64,"path":"64600 65534 64710 65515","nexthops":[{"ip":"fc00::72","used":true,"afi":"ipv6","scope":"global"}]}]},"vrfName":"default","defaultLocPrf":100}`
	ipv6BGPNeighborsAdvertisedRoutes := `{"bgpTableVersion":6412,"filteredPrefixCounter":0,"totalPrefixCounter":6412,"localAS":65100,"advertisedRoutes":{"20c1:9c8::/64":{"origin":"IGP","network":"20c1:9c8::/64","weight":0,"valid":true,"nextHopGlobal":"::","addrPrefix":"20c1:9c8::","prefixLen":64,"path":"64600 65534 64795 65509","multipath":true,"best":true},"20c0:df78::/64":{"origin":"IGP","network":"20c0:df78::/64","weight":0,"valid":true,"nextHopGlobal":"::","addrPrefix":"20c0:df78::","prefixLen":64,"path":"64600 65534 64710 65515","multipath":true,"best":true}},"bgpLocalRouterId":"10.1.0.32","defaultLocPrf":100}`
	ipv6BGPNeighborsReceivedRoutes := `{"bgpTableVersion":6412,"receivedRoutes":{"20c1:9c8::/64":{"origin":"IGP","network":"20c1:9c8::/64","weight":0,"valid":true,"nextHopGlobal":"fc00::72","addrPrefix":"20c1:9c8::","prefixLen":64,"path":"64600 65534 64795 65509","multipath":true,"best":true},"20c0:df78::/64":{"origin":"IGP","network":"20c0:df78::/64","weight":0,"valid":true,"nextHopGlobal":"fc00::72","addrPrefix":"20c0:df78::","prefixLen":64,"path":"64600 65534 64710 65515","multipath":true,"best":true}},"filteredPrefixCounter":0,"totalPrefixCounter":6400,"localAS":65100,"bgpLocalRouterId":"10.1.0.32","defaultLocPrf":100}`
	ipv6BGPNeighborsAddressFamiliesByPeer := `{"fc00::2":{"remoteAs":65200,"localAs":65100,"nbrExternalLink":true,"localRole":"undefined","remoteRole":"undefined","nbrDesc":"ARISTA01T2","hostname":"Unknown","peerGroup":"TIER2_V6","bgpVersion":4,"remoteRouterId":"100.1.0.1","localRouterId":"10.1.0.32","bgpState":"Established","bgpTimerUpMsec":6.1609e+07,"bgpTimerUpString":"17:06:49","bgpTimerUpEstablishedEpoch":1.759279837e+09,"bgpTimerLastRead":22000,"bgpTimerLastWrite":49000,"bgpInUpdateElapsedTimeMsecs":6.1609e+07,"bgpTimerConfiguredHoldTimeMsecs":180000,"bgpTimerConfiguredKeepAliveIntervalMsecs":60000,"bgpTimerHoldTimeMsecs":180000,"bgpTimerKeepAliveIntervalMsecs":60000,"bgpTcpMssConfigured":0,"bgpTcpMssSynced":0,"extendedOptionalParametersLength":false,"bgpTimerConfiguredConditionalAdvertisementsSec":60,"neighborCapabilities":{"4byteAs":"advertisedAndReceived","extendedMessage":"advertised","addPath":{"ipv6Unicast":{"rxAdvertisedAndReceived":true}},"longLivedGracefulRestart":"advertised","routeRefresh":"advertisedAndReceivedNew","enhancedRouteRefresh":"advertisedAndReceived","multiprotocolExtensions":{"ipv6Unicast":{"advertisedAndReceived":true}},"hostName":{"advHostName":"unknown","advDomainName":"n/a"},"softwareVersion":null,"gracefulRestart":"advertisedAndReceived","gracefulRestartRemoteTimerMsecs":300000,"addressFamiliesByPeer":"none"},"gracefulRestartInfo":{"endOfRibSend":{"ipv6Unicast":true},"endOfRibRecv":{"ipv6Unicast":true},"localGrMode":"Helper*","remoteGrMode":"Helper","rBit":false,"nBit":true,"timers":{"configuredRestartTimer":120,"configuredLlgrStaleTime":0,"receivedRestartTimer":300},"ipv6Unicast":{"fBit":false,"endOfRibStatus":{"endOfRibSend":true,"endOfRibSentAfterUpdate":true,"endOfRibRecv":true},"timers":{"stalePathTimer":360,"llgrStaleTime":0,"selectionDeferralTimer":0}}},"messageStats":{"depthInq":0,"depthOutq":0,"opensSent":1,"opensRecv":1,"notificationsSent":0,"notificationsRecv":0,"updatesSent":18,"updatesRecv":3187,"keepalivesSent":1027,"keepalivesRecv":1206,"routeRefreshSent":0,"routeRefreshRecv":0,"capabilitySent":0,"capabilityRecv":0,"totalSent":1046,"totalRecv":4394},"minBtwnAdvertisementRunsTimerMsecs":0,"addressFamilyInfo":{"ipv6Unicast":{"peerGroupMember":"TIER2_V6","updateGroupId":2,"subGroupId":2,"packetQueueLength":0,"inboundSoftConfigPermit":true,"allowAsInCount":0,"commAttriSentToNbr":"extendedAndStandard","inboundPathPolicyConfig":true,"outboundPathPolicyConfig":true,"routeMapForIncomingAdvertisements":"FROM_TIER2_V6","routeMapForOutgoingAdvertisements":"TO_TIER2_V6","acceptedPrefixCounter":6370,"sentPrefixCounter":33,"prefixAllowedMax":8000,"prefixAllowedMaxWarning":true,"prefixAllowedWarningThresh":90}},"connectionsEstablished":1,"connectionsDropped":0,"lastResetTimerMsecs":6.1668e+07,"lastResetDueTo":"No AFI/SAFI activated for peer","lastResetCode":30,"softwareVersion":"","externalBgpNbrMaxHopsAway":1,"hostLocal":"fc00::1","portLocal":179,"hostForeign":"fc00::2","portForeign":39219,"nexthop":"10.0.0.0","nexthopGlobal":"fc00::1","nexthopLocal":"fe80::1e34:daff:febb:8300","bgpConnection":"sharedNetwork","connectRetryTimer":120,"estimatedRttInMsecs":4,"readThread":"on","writeThread":"on"},"fc00::42":{"remoteAs":64001,"localAs":65100,"nbrExternalLink":true,"localRole":"undefined","remoteRole":"undefined","nbrDesc":"ARISTA01T0","hostname":"Unknown","peerGroup":"TIER0_V6_DEPLOYMENT_ID_0","bgpVersion":4,"remoteRouterId":"100.1.0.17","localRouterId":"10.1.0.32","bgpState":"Established","bgpTimerUpMsec":6.1609e+07,"bgpTimerUpString":"17:06:49","bgpTimerUpEstablishedEpoch":1.759279837e+09,"bgpTimerLastRead":17000,"bgpTimerLastWrite":49000,"bgpInUpdateElapsedTimeMsecs":6.1609e+07,"bgpTimerConfiguredHoldTimeMsecs":180000,"bgpTimerConfiguredKeepAliveIntervalMsecs":60000,"bgpTimerHoldTimeMsecs":180000,"bgpTimerKeepAliveIntervalMsecs":60000,"bgpTcpMssConfigured":0,"bgpTcpMssSynced":0,"extendedOptionalParametersLength":false,"bgpTimerConfiguredConditionalAdvertisementsSec":60,"neighborCapabilities":{"4byteAs":"advertisedAndReceived","extendedMessage":"advertised","addPath":{"ipv6Unicast":{"rxAdvertisedAndReceived":true}},"longLivedGracefulRestart":"advertised","routeRefresh":"advertisedAndReceivedNew","enhancedRouteRefresh":"advertisedAndReceived","multiprotocolExtensions":{"ipv6Unicast":{"advertisedAndReceived":true}},"hostName":{"advHostName":"unknown","advDomainName":"n/a"},"softwareVersion":null,"gracefulRestart":"advertisedAndReceived","gracefulRestartRemoteTimerMsecs":700000,"addressFamiliesByPeer":{"ipv6Unicast":{}}},"gracefulRestartInfo":{"endOfRibSend":{"ipv6Unicast":true},"endOfRibRecv":{"ipv6Unicast":true},"localGrMode":"Helper*","remoteGrMode":"Restart","rBit":false,"nBit":true,"timers":{"configuredRestartTimer":120,"configuredLlgrStaleTime":0,"receivedRestartTimer":700},"ipv6Unicast":{"fBit":false,"endOfRibStatus":{"endOfRibSend":true,"endOfRibSentAfterUpdate":true,"endOfRibRecv":true},"timers":{"stalePathTimer":360,"llgrStaleTime":0,"selectionDeferralTimer":0}}},"messageStats":{"depthInq":0,"depthOutq":0,"opensSent":1,"opensRecv":1,"notificationsSent":0,"notificationsRecv":0,"updatesSent":3204,"updatesRecv":3,"keepalivesSent":1027,"keepalivesRecv":1205,"routeRefreshSent":0,"routeRefreshRecv":0,"capabilitySent":0,"capabilityRecv":0,"totalSent":4232,"totalRecv":1209},"minBtwnAdvertisementRunsTimerMsecs":0,"addressFamilyInfo":{"ipv6Unicast":{"peerGroupMember":"TIER0_V6_DEPLOYMENT_ID_0","updateGroupId":1,"subGroupId":1,"packetQueueLength":0,"inboundSoftConfigPermit":true,"allowAsInCount":1,"commAttriSentToNbr":"extendedAndStandard","inboundPathPolicyConfig":true,"outboundPathPolicyConfig":true,"routeMapForIncomingAdvertisements":"FROM_TIER0_V6_DEPLOYMENT_ID_0","routeMapForOutgoingAdvertisements":"TO_TIER0_V6","acceptedPrefixCounter":2,"sentPrefixCounter":6406,"prefixAllowedMax":8000,"prefixAllowedMaxWarning":true,"prefixAllowedWarningThresh":90}},"connectionsEstablished":1,"connectionsDropped":0,"lastResetTimerMsecs":6.1668e+07,"lastResetDueTo":"Waiting for peer OPEN","lastResetCode":32,"softwareVersion":"","externalBgpNbrMaxHopsAway":1,"hostLocal":"fc00::41","portLocal":179,"hostForeign":"fc00::42","portForeign":34651,"nexthop":"10.0.0.32","nexthopGlobal":"fc00::41","nexthopLocal":"fe80::1e34:daff:febb:8300","bgpConnection":"sharedNetwork","connectRetryTimer":120,"estimatedRttInMsecs":0,"readThread":"on","writeThread":"on"},"fc00::46":{"remoteAs":64002,"localAs":65100,"nbrExternalLink":true,"localRole":"undefined","remoteRole":"undefined","nbrDesc":"ARISTA02T0","hostname":"Unknown","peerGroup":"TIER0_V6_DEPLOYMENT_ID_0","bgpVersion":4,"remoteRouterId":"100.1.0.18","localRouterId":"10.1.0.32","bgpState":"Established","bgpTimerUpMsec":6.1608e+07,"bgpTimerUpString":"17:06:48","bgpTimerUpEstablishedEpoch":1.759279838e+09,"bgpTimerLastRead":47000,"bgpTimerLastWrite":48000,"bgpInUpdateElapsedTimeMsecs":6.1608e+07,"bgpTimerConfiguredHoldTimeMsecs":180000,"bgpTimerConfiguredKeepAliveIntervalMsecs":60000,"bgpTimerHoldTimeMsecs":180000,"bgpTimerKeepAliveIntervalMsecs":60000,"bgpTcpMssConfigured":0,"bgpTcpMssSynced":0,"extendedOptionalParametersLength":false,"bgpTimerConfiguredConditionalAdvertisementsSec":60,"neighborCapabilities":{"4byteAs":"advertisedAndReceived","extendedMessage":"advertised","addPath":{"ipv6Unicast":{"rxAdvertisedAndReceived":true}},"longLivedGracefulRestart":"advertised","routeRefresh":"advertisedAndReceivedNew","enhancedRouteRefresh":"advertisedAndReceived","multiprotocolExtensions":{"ipv6Unicast":{"advertisedAndReceived":true}},"hostName":{"advHostName":"unknown","advDomainName":"n/a"},"softwareVersion":null,"gracefulRestart":"advertisedAndReceived","gracefulRestartRemoteTimerMsecs":700000,"addressFamiliesByPeer":{"ipv6Unicast":{}}},"gracefulRestartInfo":{"endOfRibSend":{"ipv6Unicast":true},"endOfRibRecv":{"ipv6Unicast":true},"localGrMode":"Helper*","remoteGrMode":"Restart","rBit":false,"nBit":true,"timers":{"configuredRestartTimer":120,"configuredLlgrStaleTime":0,"receivedRestartTimer":700},"ipv6Unicast":{"fBit":false,"endOfRibStatus":{"endOfRibSend":true,"endOfRibSentAfterUpdate":true,"endOfRibRecv":true},"timers":{"stalePathTimer":360,"llgrStaleTime":0,"selectionDeferralTimer":0}}},"messageStats":{"depthInq":0,"depthOutq":0,"opensSent":1,"opensRecv":1,"notificationsSent":0,"notificationsRecv":0,"updatesSent":3204,"updatesRecv":3,"keepalivesSent":1027,"keepalivesRecv":1211,"routeRefreshSent":0,"routeRefreshRecv":0,"capabilitySent":0,"capabilityRecv":0,"totalSent":4232,"totalRecv":1215},"minBtwnAdvertisementRunsTimerMsecs":0,"addressFamilyInfo":{"ipv6Unicast":{"peerGroupMember":"TIER0_V6_DEPLOYMENT_ID_0","updateGroupId":1,"subGroupId":1,"packetQueueLength":0,"inboundSoftConfigPermit":true,"allowAsInCount":1,"commAttriSentToNbr":"extendedAndStandard","inboundPathPolicyConfig":true,"outboundPathPolicyConfig":true,"routeMapForIncomingAdvertisements":"FROM_TIER0_V6_DEPLOYMENT_ID_0","routeMapForOutgoingAdvertisements":"TO_TIER0_V6","acceptedPrefixCounter":2,"sentPrefixCounter":6406,"prefixAllowedMax":8000,"prefixAllowedMaxWarning":true,"prefixAllowedWarningThresh":90}},"connectionsEstablished":1,"connectionsDropped":0,"lastResetTimerMsecs":6.1668e+07,"lastResetDueTo":"Waiting for peer OPEN","lastResetCode":32,"softwareVersion":"","externalBgpNbrMaxHopsAway":1,"hostLocal":"fc00::45","portLocal":37728,"hostForeign":"fc00::46","portForeign":179,"nexthop":"10.0.0.34","nexthopGlobal":"fc00::45","nexthopLocal":"fe80::1e34:daff:febb:8300","bgpConnection":"sharedNetwork","connectRetryTimer":120,"estimatedRttInMsecs":1,"readThread":"on","writeThread":"on"},"fc00::6":{"remoteAs":65200,"localAs":65100,"nbrExternalLink":true,"localRole":"undefined","remoteRole":"undefined","nbrDesc":"ARISTA03T2","hostname":"Unknown","peerGroup":"TIER2_V6","bgpVersion":4,"remoteRouterId":"100.1.0.3","localRouterId":"10.1.0.32","bgpState":"Established","bgpTimerUpMsec":6.1608e+07,"bgpTimerUpString":"17:06:48","bgpTimerUpEstablishedEpoch":1.759279838e+09,"bgpTimerLastRead":17000,"bgpTimerLastWrite":48000,"bgpInUpdateElapsedTimeMsecs":6.1607e+07,"bgpTimerConfiguredHoldTimeMsecs":180000,"bgpTimerConfiguredKeepAliveIntervalMsecs":60000,"bgpTimerHoldTimeMsecs":180000,"bgpTimerKeepAliveIntervalMsecs":60000,"bgpTcpMssConfigured":0,"bgpTcpMssSynced":0,"extendedOptionalParametersLength":false,"bgpTimerConfiguredConditionalAdvertisementsSec":60,"neighborCapabilities":{"4byteAs":"advertisedAndReceived","extendedMessage":"advertised","addPath":{"ipv6Unicast":{"rxAdvertisedAndReceived":true}},"longLivedGracefulRestart":"advertised","routeRefresh":"advertisedAndReceivedNew","enhancedRouteRefresh":"advertisedAndReceived","multiprotocolExtensions":{"ipv6Unicast":{"advertisedAndReceived":true}},"hostName":{"advHostName":"unknown","advDomainName":"n/a"},"softwareVersion":null,"gracefulRestart":"advertisedAndReceived","gracefulRestartRemoteTimerMsecs":300000,"addressFamiliesByPeer":"none"},"gracefulRestartInfo":{"endOfRibSend":{"ipv6Unicast":true},"endOfRibRecv":{"ipv6Unicast":true},"localGrMode":"Helper*","remoteGrMode":"Helper","rBit":false,"nBit":true,"timers":{"configuredRestartTimer":120,"configuredLlgrStaleTime":0,"receivedRestartTimer":300},"ipv6Unicast":{"fBit":false,"endOfRibStatus":{"endOfRibSend":true,"endOfRibSentAfterUpdate":true,"endOfRibRecv":true},"timers":{"stalePathTimer":360,"llgrStaleTime":0,"selectionDeferralTimer":0}}},"messageStats":{"depthInq":0,"depthOutq":0,"opensSent":1,"opensRecv":1,"notificationsSent":0,"notificationsRecv":0,"updatesSent":18,"updatesRecv":3187,"keepalivesSent":1027,"keepalivesRecv":1211,"routeRefreshSent":0,"routeRefreshRecv":0,"capabilitySent":0,"capabilityRecv":0,"totalSent":1046,"totalRecv":4399},"minBtwnAdvertisementRunsTimerMsecs":0,"addressFamilyInfo":{"ipv6Unicast":{"peerGroupMember":"TIER2_V6","updateGroupId":2,"subGroupId":2,"packetQueueLength":0,"inboundSoftConfigPermit":true,"allowAsInCount":0,"commAttriSentToNbr":"extendedAndStandard","inboundPathPolicyConfig":true,"outboundPathPolicyConfig":true,"routeMapForIncomingAdvertisements":"FROM_TIER2_V6","routeMapForOutgoingAdvertisements":"TO_TIER2_V6","acceptedPrefixCounter":6370,"sentPrefixCounter":33,"prefixAllowedMax":8000,"prefixAllowedMaxWarning":true,"prefixAllowedWarningThresh":90}},"connectionsEstablished":1,"connectionsDropped":0,"lastResetTimerMsecs":6.1668e+07,"lastResetDueTo":"Waiting for peer OPEN","lastResetCode":32,"softwareVersion":"","externalBgpNbrMaxHopsAway":1,"hostLocal":"fc00::5","portLocal":42232,"hostForeign":"fc00::6","portForeign":179,"nexthop":"10.0.0.4","nexthopGlobal":"fc00::5","nexthopLocal":"fe80::1e34:daff:febb:8300","bgpConnection":"sharedNetwork","connectRetryTimer":120,"estimatedRttInMsecs":1,"readThread":"on","writeThread":"on"}}`

	ResetDataSetsAndMappings(t)

	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal interface{}
		valTest     bool
		mockFile    string
		testInit    func()
	}{
		{
			desc:       "query SHOW ipv6 bgp neighbors - read error",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "neighbors" >
			`,
			wantRetCode: codes.NotFound,
		},
		{
			desc:       "query SHOW ipv6 bgp neighbors - empty vtysh output",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "neighbors" >
			`,
			wantRetCode: codes.NotFound,
			mockFile:    "../testdata/ipv6_bgp_neighbors/EMPTY_VTYSH_JSON.txt",
		},
		{
			desc:       "query SHOW ipv6 bgp neighbors - invalid vtysh output",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "neighbors" >
			`,
			wantRetCode: codes.NotFound,
			mockFile:    "../testdata/ipv6_bgp_neighbors/INVALID_VTYSH_JSON.txt",
		},
		{
			desc:       "query SHOW ipv6 bgp neighbors <ipaddress> routes - invalid vtysh output",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "neighbors" >
				elem: <name: "fc00::7a" >
				elem: <name: "routes" >
			`,
			wantRetCode: codes.NotFound,
			mockFile:    "../testdata/ipv6_bgp_neighbors/INVALID_VTYSH_JSON.txt",
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, bgpNeighborsFileName)
			},
		},
		{
			desc:       "query SHOW ipv6 bgp neighbors <ipaddress> advertised-routes - invalid vtysh output",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "neighbors" >
				elem: <name: "fc00::7a" >
				elem: <name: "advertised-routes" >
			`,
			wantRetCode: codes.NotFound,
			mockFile:    "../testdata/ipv6_bgp_neighbors/INVALID_VTYSH_JSON.txt",
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, bgpNeighborsFileName)
			},
		},
		{
			desc:       "query SHOW ipv6 bgp neighbors <ipaddress> received-routes - invalid vtysh output",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "neighbors" >
				elem: <name: "fc00::7a" >
				elem: <name: "received-routes" >
			`,
			wantRetCode: codes.NotFound,
			mockFile:    "../testdata/ipv6_bgp_neighbors/INVALID_VTYSH_JSON.txt",
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, bgpNeighborsFileName)
			},
		},
		{
			desc:       "query SHOW ipv6 bgp neighbors - no ipv6 address specified",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "neighbors" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(ipv6BGPNeighborsAll),
			valTest:     true,
			mockFile:    "../testdata/ipv6_bgp_neighbors/VTYSH_SHOW_IPV6_BGP_NEIGHBORS_JSON.txt",
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, bgpNeighborsFileName)
			},
		},
		{
			desc:       "query SHOW ipv6 bgp neighbors - no ipv6 address specified - addressFamiliesByPeer as an object",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "neighbors" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(ipv6BGPNeighborsAddressFamiliesByPeer),
			valTest:     true,
			mockFile:    "../testdata/ipv6_bgp_neighbors/VTYSH_SHOW_IPV6_BGP_NEIGHBORS_ADDRESS_FAMILIES_OBJECT.txt",
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, bgpNeighborsTwoFileName)
			},
		},
		{
			desc:       "query SHOW ipv6 bgp neighbors <ipaddress> - ipv6 address specified",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "neighbors" >
				elem: <name: "fc00::7a" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(ipv6BGPNeighborsIPSpecific),
			valTest:     true,
			mockFile:    "../testdata/ipv6_bgp_neighbors/VTYSH_SHOW_IPV6_BGP_NEIGHBORS_SPECIFIC_IP_JSON.txt",
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, bgpNeighborsFileName)
			},
		},
		{
			desc:       "query SHOW ipv6 bgp neighbors <ipaddress> - ipv6 address not in CONFIG_DB",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "neighbors" >
				elem: <name: "fa00::7e" >
			`,
			wantRetCode: codes.NotFound,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, bgpNeighborsFileName)
			},
		},
		{
			desc:       "query SHOW ipv6 bgp neighbors <ipaddress> - Not valid ipv6 address",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "neighbors" >
				elem: <name: "2001:db8:::1" >
			`,
			wantRetCode: codes.NotFound,
		},
		{
			desc:       "query SHOW ipv6 bgp neighbors <ipaddress> <info_type> - no ipv6 address specified",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "neighbors" >
				elem: <name: "" >
				elem: <name: "routes" >
			`,
			wantRetCode: codes.NotFound,
		},
		{
			desc:       "query SHOW ipv6 bgp neighbors <ipaddress> routes",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "neighbors" >
				elem: <name: "fc00::7a" >
				elem: <name: "routes" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(ipv6BGPNeighborsRoutes),
			valTest:     true,
			mockFile:    "../testdata/ipv6_bgp_neighbors/VTYSH_SHOW_IPV6_BGP_NEIGHBORS_ROUTES_JSON.txt",
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, bgpNeighborsFileName)
			},
		},
		{
			desc:       "query SHOW ipv6 bgp neighbors <ipaddress> advertised-routes",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "neighbors" >
				elem: <name: "fc00::72" >
				elem: <name: "advertised-routes" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(ipv6BGPNeighborsAdvertisedRoutes),
			valTest:     true,
			mockFile:    "../testdata/ipv6_bgp_neighbors/VTYSH_SHOW_IPV6_BGP_NEIGHBORS_ADVERTISED_ROUTES_JSON.txt",
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, bgpNeighborsFileName)
			},
		},
		{
			desc:       "query SHOW ipv6 bgp neighbors <ipaddress> received-routes",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "neighbors">
				elem: <name: "fc00::72" >
				elem: <name: "received-routes" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(ipv6BGPNeighborsReceivedRoutes),
			valTest:     true,
			mockFile:    "../testdata/ipv6_bgp_neighbors/VTYSH_SHOW_IPV6_BGP_NEIGHBORS_RECEIVED_ROUTES_JSON.txt",
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, bgpNeighborsFileName)
			},
		},
	}

	for _, test := range tests {
		if test.testInit != nil {
			test.testInit()
		}
		var patches *gomonkey.Patches
		if test.mockFile != "" {
			patches = MockNSEnterOutput(t, test.mockFile)
		}

		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})
		if patches != nil {
			patches.Reset()
		}
	}
}
