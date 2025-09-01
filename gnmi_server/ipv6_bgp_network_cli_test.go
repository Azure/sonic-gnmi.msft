package gnmi

// ipv6_cli_test.go

// Tests SHOW ipv6 bgp network

import (
	"crypto/tls"
	"os"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestGetIPv6BGPNetwork(t *testing.T) {
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

	// info_type incorrect case
	t.Run("SHOW ipv6 bgp network 2064:100::1 longer", func(t *testing.T) {
		textPbPath := `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "network" 
					key: { key: "ipaddress" value: "2064:100::2" } 
					key: { key: "info_type" value: "longer" } >
			`
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
	})

	// address incorrect case
	t.Run("SHOW ipv6 bgp network 2064:100::1 longer-prefixes", func(t *testing.T) {
		textPbPath := `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "network" 
					key: { key: "ipaddress" value: "2064:100::2" } 
					key: { key: "info_type" value: "longer-prefixes" } >
			`
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
	})

	showIpv6BgpNetworkMockFile := "../testdata/show_ipv6_bgp_network.txt"
	patches := MockNSEnterOutput(t, showIpv6BgpNetworkMockFile)
	t.Run("SHOW ipv6 bgp network", func(t *testing.T) {
		textPbPath := `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "network" >
			`
		wantRespVal := `
		{
			"output": "BGP table version is 6405, local router ID is 10.1.0.32, vrf id 0\nDefault local pref 100, local AS 64601\nStatus codes:  s suppressed, d damped, h history, * valid, > best, = multipath,\n               i internal, r RIB-failure, S Stale, R Removed\nNexthop codes: @NNN nexthop's vrf id, < announce-nh-self\nOrigin codes:  i - IGP, e - EGP, ? - incomplete\nRPKI validation codes: V valid, I invalid, N Not found\n\n    Network          Next Hop            Metric LocPrf Weight Path\n *= ::/0             fc00::12                               0 64802 65534 6666 6667 i\n *>                  fc00::2                                0 64802 65534 6666 6667 i\n *=                  fc00::1a                               0 64802 65534 6666 6667 i\n *=                  fc00::a                                0 64802 65534 6666 6667 i\n *> 2064:100::1/128  fc00::2                                0 64802 i\n *> 2064:100::2/128  fc00::a                                0 64802 i\n *> 2064:100::3/128  fc00::12                               0 64802 i\n *> 2064:100::4/128  fc00::1a                               0 64802 i\n *= 20c0:a808::/64   fc00::12                               0 64802 64602 i\n *>                  fc00::2                                0 64802 64602 i\n *=                  fc00::1a                               0 64802 64602 i\n *=                  fc00::a                                0 64802 64602 i\n *= 20c0:a808:0:80::/64\n                    fc00::12                               0 64802 64602 i\n *>                  fc00::2                                0 64802 64602 i\n *=                  fc00::1a                               0 64802 64602 i\n *=                  fc00::a                                0 64802 64602 i\n *= 20c0:a810::/64   fc00::12                               0 64802 64603 i\n *>                  fc00::2                                0 64802 64603 i\n *=                  fc00::1a                               0 64802 64603 i\n *=                  fc00::a                                0 64802 64603 i\n\nDisplayed 8 routes and 20 total paths"
		}
		`
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})
	patches.Reset()

	showIpv6BgpNetworkAddressMockFile := "../testdata/show_ipv6_bgp_network_address.txt"
	patches = MockNSEnterOutput(t, showIpv6BgpNetworkAddressMockFile)
	t.Run("SHOW ipv6 bgp network 2064:100::2/128", func(t *testing.T) {
		textPbPath := `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "network" key: { key: "ipaddress" value: "2064:100::2/128" } >
			`
		wantRespVal := `
		{
			"output": "BGP routing table entry for 2064:100::2/128, version 5\nPaths: (1 available, best #1, table default)\n  Advertised to non peer-group peers:\n  fc00::2 fc00::a fc00::12 fc00::1a\n  64802\n    fc00::a from fc00::a (100.1.0.2)\n      Origin IGP, valid, external, best (First path received)\n      Last update: Wed Aug 27 22:46:48 2025"
		}
		`
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})
	patches.Reset()

	showIpv6BgpNetworkAddressIBestpathMockFile := "../testdata/show_ipv6_bgp_network_address_json.txt"
	patches = MockNSEnterOutput(t, showIpv6BgpNetworkAddressIBestpathMockFile)
	t.Run("SHOW ipv6 bgp network 2064:100::2/128 json", func(t *testing.T) {
		textPbPath := `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "network" 
					key: { key: "ipaddress" value: "2064:100::2/128" } 
					key: { key: "info_type" value: "json" } >
			`
		wantRespVal, err := os.ReadFile("../testdata/show_ipv6_bgp_network_address_json.txt")
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})
	patches.Reset()
}
