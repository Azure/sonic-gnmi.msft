package gnmi

// ipv6_interfaces_cli_test.go
// Tests SHOW ipv6 interfaces

import (
	"crypto/tls"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/sonic-net/sonic-gnmi/internal/ipinterfaces"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

// TestGetIPv6InterfacesCLI
// Simple smoke test: exercise the gRPC path for `show ipv6 interfaces` without
// any mocking to ensure the server accepts the request and returns codes.OK.
// We intentionally do NOT assert the body because the real system state (and
// thus interface inventory) is environment-dependent and may vary across CI.
func TestGetIPv6InterfacesCLI(t *testing.T) {
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

	tests := []struct {
		desc        string
		textPbPath  string
		wantRetCode codes.Code
	}{
		{
			desc: "show ipv6 interfaces default",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "interfaces" >
			`,
			wantRetCode: codes.OK,
		},
		{
			desc: "show ipv6 interface --namespace all (option currently unimplemented)",
			textPbPath: `
					elem: <name: "ipv6" >
					elem: <name: "interfaces"
						key: { key: "namespace" value: "all" } >
				`,
			wantRetCode: codes.Unimplemented,
		},
		{
			desc: "show ipv6 interface --display all",
			textPbPath: `
					elem: <name: "ipv6" >
					elem: <name: "interfaces"
					    key: { key: "display" value: "all" } >
				`,
			wantRetCode: codes.OK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, "SHOW", tc.textPbPath, tc.wantRetCode, nil, false)
		})
	}
}

// TestGetIPv6InterfacesCLIMixedBGPFields
// Mocks the underlying ipinterfaces library to provide a deterministic set of
// addresses: one with real BGP neighbor data (enriched) and two without (will
// be backfilled to "N/A" by the CLI layer). Validates JSON shaping rules:
//   - master field is always present (even when empty)
//   - missing neighbor info replaced with N/A
//
// This focuses on conversion / presentation logic, not data sourcing.
func TestGetIPv6InterfacesCLIMixedBGPFields(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	// Deterministic dataset: one enriched, two defaulted ("N/A") after CLI backfill.
	patches := gomonkey.ApplyFunc(ipinterfaces.GetIPInterfaces, func(deps ipinterfaces.Dependencies, addressFamily string, opts *ipinterfaces.GetInterfacesOptions) ([]ipinterfaces.IPInterfaceDetail, error) {
		return []ipinterfaces.IPInterfaceDetail{
			{Name: "Ethernet8", AdminStatus: "up", OperStatus: "up", Master: "", IPAddresses: []ipinterfaces.IPAddressDetail{
				{Address: "fc00::1/64", BGPNeighborIP: "aa00::1", BGPNeighborName: "ARISTA01T1"}, // enriched
				{Address: "fc00::ff/64"}, // default -> N/A
				{Address: "fc00::aa/64"}, // default -> N/A
			}},
		}, nil
	})
	defer patches.Reset()

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

	// Expected JSON after CLI layer transforms (master always present, defaults applied).
	// Expected JSON after CLI layer applies naming, master emission and N/A defaults.
	expected := `{"Ethernet8":{"ipv6_addresses":[{"address":"fc00::1/64","bgp_neighbor_ip":"aa00::1","bgp_neighbor_name":"ARISTA01T1"},{"address":"fc00::ff/64","bgp_neighbor_ip":"N/A","bgp_neighbor_name":"N/A"},{"address":"fc00::aa/64","bgp_neighbor_ip":"N/A","bgp_neighbor_name":"N/A"}],"admin_status":"up","oper_status":"up","master":""}}`

	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal interface{}
		valTest     bool
	}{
		{
			desc:       "query SHOW ipv6 interfaces mixed enrichment/default",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "interfaces" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expected),
			valTest:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})
	}
}
