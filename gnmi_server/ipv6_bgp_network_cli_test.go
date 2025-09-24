package gnmi

// ipv6_cli_test.go

// Tests SHOW ipv6 bgp network

import (
	"crypto/tls"
	"fmt"
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
	t.Run("SHOW ipv6 bgp network 2064:100::2 longer", func(t *testing.T) {
		textPbPath := `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "network" >
				elem: <name: "2064:100::2" >
				elem: <name: "longer" >
			`
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
	})

	// address incorrect case
	t.Run("SHOW ipv6 bgp network 2064:100::2 longer-prefixes", func(t *testing.T) {
		textPbPath := `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "network" >
				elem: <name: "2064:100::2" >
				elem: <name: "longer-prefixes" >
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
		content, err := os.ReadFile(showIpv6BgpNetworkMockFile)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		wantRespVal := fmt.Sprintf(`
		{
			"output": %q
		}
		`, content)

		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})
	patches.Reset()

	showIpv6BgpNetworkAddressMockFile := "../testdata/show_ipv6_bgp_network_address.txt"
	patches = MockNSEnterOutput(t, showIpv6BgpNetworkAddressMockFile)
	t.Run("SHOW ipv6 bgp network 2064:100::2/128", func(t *testing.T) {
		textPbPath := `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "network" >
				elem: <name: "2064:100::2/128" >
			`
		content, err := os.ReadFile(showIpv6BgpNetworkAddressMockFile)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		wantRespVal := fmt.Sprintf(`
		{
			"output": %q
		}
		`, content)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})
	patches.Reset()

	showIpv6BgpNetworkAddressIJsonMockFile := "../testdata/show_ipv6_bgp_network_address_json.txt"
	patches = MockNSEnterOutput(t, showIpv6BgpNetworkAddressIJsonMockFile)
	t.Run("SHOW ipv6 bgp network 2064:100::2/128 json", func(t *testing.T) {
		textPbPath := `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "network" >
				elem: <name: "2064:100::2/128" >
				elem: <name: "json" >
			`
		wantRespVal, err := os.ReadFile(showIpv6BgpNetworkAddressIJsonMockFile)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})
	patches.Reset()

	showIpv6BgpNetworkAddressIBestpathMockFile := "../testdata/show_ipv6_bgp_network_address_bestpath.txt"
	patches = MockNSEnterOutput(t, showIpv6BgpNetworkAddressIBestpathMockFile)
	t.Run("SHOW ipv6 bgp network 20c0:a800:0:592::/64 bestpath", func(t *testing.T) {
		textPbPath := `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "network" >
				elem: <name: "20c0:a800:0:592::/64" >
				elem: <name: "bestpath" >
			`
		content, err := os.ReadFile(showIpv6BgpNetworkAddressIBestpathMockFile)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		wantRespVal := fmt.Sprintf(`
		{
			"output": %q
		}
		`, content)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})
	patches.Reset()
}
