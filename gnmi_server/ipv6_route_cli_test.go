package gnmi

// ipv6_route_cli_test.go
// Tests SHOW ipv6 route (pass-through JSON) with args option

import (
	"crypto/tls"
	"os"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"

	"github.com/agiledragon/gomonkey/v2"
	show_client "github.com/sonic-net/sonic-gnmi/show_client"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestGetIPv6Route(t *testing.T) {
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

	// Expected JSON strings loaded from files via MockNSEnterOutput*; we still compare raw bytes.
	tests := []struct {
		desc           string
		pathTarget     string
		textPbPath     string
		wantRetCode    codes.Code
		wantRespFile   string // expected body (same as mock output)
		mockOutputFile string // file to feed to nsenter mock
		multiAsic      bool
		valTest        bool
	}{
		{
			desc:       "query SHOW ipv6 route base",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "route" >
			`,
			wantRetCode:    codes.OK,
			wantRespFile:   "../testdata/VTYSH_SHOW_IPV6_ROUTE.json",
			mockOutputFile: "../testdata/VTYSH_SHOW_IPV6_ROUTE.json",
			valTest:        true,
		},
		{
			desc:       "query SHOW ipv6 route bgp",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "route" key <key: "args" value: "bgp" > >
			`,
			wantRetCode:    codes.OK,
			wantRespFile:   "../testdata/VTYSH_SHOW_IPV6_ROUTE_BGP.json",
			mockOutputFile: "../testdata/VTYSH_SHOW_IPV6_ROUTE_BGP.json",
			valTest:        true,
		},
		{
			desc:       "query SHOW ipv6 route prefix arg",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "route" key <key: "args" value: "2001:db8::/64" > >
			`,
			wantRetCode:    codes.OK,
			wantRespFile:   "../testdata/VTYSH_SHOW_IPV6_ROUTE_PREFIX.json",
			mockOutputFile: "../testdata/VTYSH_SHOW_IPV6_ROUTE_PREFIX.json",
			valTest:        true,
		},
		{
			desc:       "query SHOW ipv6 route args include json (no duplicate)",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "route" key <key: "args" value: "bgp json" > >
			`,
			wantRetCode:    codes.OK,
			wantRespFile:   "../testdata/VTYSH_SHOW_IPV6_ROUTE_BGP.json",
			mockOutputFile: "../testdata/VTYSH_SHOW_IPV6_ROUTE_BGP.json",
			valTest:        true,
		},
		{
			desc:       "query SHOW ipv6 route invalid JSON output",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "route" >
			`,
			wantRetCode:    codes.NotFound,
			mockOutputFile: "../testdata/INVALID_JSON.txt",
			valTest:        false,
		},
		{
			desc:       "query SHOW ipv6 route multi-asic unsupported",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "route" >
			`,
			wantRetCode: codes.NotFound,
			multiAsic:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			var allPatches []*gomonkey.Patches
			// Patch multi-asic behavior
			allPatches = append(allPatches, gomonkey.ApplyFunc(show_client.IsMultiAsic, func() bool { return test.multiAsic }))
			if test.mockOutputFile != "" {
				allPatches = append(allPatches, MockNSEnterOutput(t, test.mockOutputFile))
			}
			// Load expected file if needed
			expected := []byte{}
			if test.wantRespFile != "" {
				b, err := os.ReadFile(test.wantRespFile)
				if err != nil {
					t.Fatalf("read expected file: %v", err)
				}
				expected = b
			}
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, expected, test.valTest)
			for _, p := range allPatches {
				p.Reset()
			}
		})
	}
}
