package gnmi

import (
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	sc "github.com/sonic-net/sonic-gnmi/show_client"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestShowIPv6Fib_HappyPath(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	conn, err := grpc.Dial(TargetAddr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	gClient := pb.NewGNMIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()

	// Load APPL_DB ROUTE_TABLE fixtures (includes 3 IPv6 routes + 1 IPv4 which should be ignored)
	AddDataSet(t, ApplDbNum, "../testdata/ROUTE_TABLE_IPV6_FIB.txt")

	textPbPath := `
        elem: <name: "ipv6" >
        elem: <name: "fib" >
    `

	// Expect entries sorted by route string; Index should be 1..N in sorted order.
	expected := []byte(`{
        "total": 3,
        "entries": [
            { "index": 1, "route": "fc00:1::/64", "nexthop": "::", "ifname": "Loopback0" },
            { "index": 2, "route": "fc00:1::32", "nexthop": "::", "ifname": "Loopback0" },
            { "index": 3, "vrf": "Vrf1000", "route": "fc02:1000::/64", "nexthop": "::", "ifname": "Vlan1000" }
        ]
    }`)
	runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, expected, true)
}

func TestShowIPv6Fib_FilterByPrefix(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	conn, err := grpc.Dial(TargetAddr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	gClient := pb.NewGNMIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()

	AddDataSet(t, ApplDbNum, "../testdata/ROUTE_TABLE_IPV6_FIB.txt")

	// Filter by exact prefix using option key "ipaddress"
	textPbPath := `
        elem: <name: "ipv6" >
        elem: <name: "fib"  >
		elem: <name: "fc00:1::/64" >
    `

	expected := []byte(`{
        "total": 1,
        "entries": [
            { "index": 1, "route": "fc00:1::/64", "nexthop": "::", "ifname": "Loopback0" }
        ]
    }`)
	runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, expected, true)
}

func TestShowIPv6Fib_EmptyApplDb(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	conn, err := grpc.Dial(TargetAddr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	gClient := pb.NewGNMIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()

	textPbPath := `
        elem: <name: "ipv6" >
        elem: <name: "fib" >
    `
	expected := []byte(`{"total":0,"entries":[]}`)
	runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, expected, true)
}

func TestShowIPv6Fib_ErrorOnRouteTable(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	patches := gomonkey.ApplyFunc(sc.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
		return nil, fmt.Errorf("error when read table ROUTE_TABLE")
	})
	defer patches.Reset()

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	conn, err := grpc.Dial(TargetAddr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	gClient := pb.NewGNMIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()

	textPbPath := `
        elem: <name: "ipv6" >
        elem: <name: "fib" >
    `
	runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
}
