package gnmi

// mac_cli_test.go

// Tests SHOW mac CLI command

import (
	"crypto/tls"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/sonic-net/sonic-gnmi/show_client"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestParseKey(t *testing.T) {
	tests := []struct {
		key      string
		expected show_client.MacEntry
	}{
		{
			key: "Vlan1000:e8:eb:d3:32:f0:1e",
			expected: show_client.MacEntry{
				Vlan:       "1000",
				MacAddress: "e8:eb:d3:32:f0:1e",
			},
		},
		{
			key: "Vlan1000:e8:eb:d3:32:f0:1a",
			expected: show_client.MacEntry{
				Vlan:       "1000",
				MacAddress: "e8:eb:d3:32:f0:1a",
			},
		},
	}

	for _, test := range tests {
		vlan, mac, success := show_client.ParseKey(test.key)
		if !success {
			t.Errorf("Failed to ParseKey(%q)", test.key)
			continue
		}
		if vlan != test.expected.Vlan || mac != test.expected.MacAddress {
			t.Errorf("parseKey(%q) = %+v; want %+v", test.key, show_client.MacEntry{Vlan: vlan, MacAddress: mac}, test.expected)
		}
	}
}

func TestProcessFDBData(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}

	conn, err := grpc.Dial(TargetAddr, opts...)
	if err != nil {
		t.Fatalf("Dailing to %q failed: %v", TargetAddr, err)
	}
	defer conn.Close()

	gClient := pb.NewGNMIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	stateDbContentFileNameForShowMac := "../testdata/ShowMacStateDb.txt"

	FlushDataSet(t, StateDbNum)
	AddDataSet(t, StateDbNum, stateDbContentFileNameForShowMac)
	t.Run("query SHOW mac", func(t *testing.T) {
		textPbPath := `
			elem: <name: "mac" >
		`
		wantRespVal := []byte(`[
        {"macAddress": "e8:eb:d3:32:f0:08", "port": "Ethernet320", "type": "dynamic", "vlan": "1000"},
        {"macAddress": "e8:eb:d3:32:f0:1b", "port": "Ethernet108", "type": "dynamic", "vlan": "1000"},
        {"macAddress": "e8:eb:d3:32:f0:1e", "port": "Ethernet120", "type": "dynamic", "vlan": "1000"},
        {"macAddress": "e8:eb:d3:32:f0:25", "port": "Ethernet148", "type": "static", "vlan": "1000"},
        {"macAddress": "e8:eb:d3:32:f0:28", "port": "Ethernet160", "type": "dynamic", "vlan": "1001"}
    ]`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	t.Run("query SHOW mac -c", func(t *testing.T) {
		textPbPath := `
			elem: <name: "mac"  key: { key: "count" value: "True" } >
		`
		wantRespVal := []byte(`{
														"count": 5
														}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	t.Run("query SHOW mac -a e8:eb:d3:32:f0:08", func(t *testing.T) {
		textPbPath := `
			elem: <name: "mac" 
				key: { key: "address" value: "e8:eb:d3:32:f0:08" }
				>
		`
		wantRespVal := []byte(`[
        {"macAddress": "e8:eb:d3:32:f0:08", "port": "Ethernet320", "type": "dynamic", "vlan": "1000"}
    ]`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	t.Run("query SHOW mac -a e8:eb:d3:32:f0:08 -c", func(t *testing.T) {
		textPbPath := `
			elem: <name: "mac" 
				key: { key: "address" value: "e8:eb:d3:32:f0:08" }
				key: { key: "count" value: "True" }
				>
		`
		wantRespVal := []byte(`{
														"count": 1
														}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	t.Run("query SHOW mac -v 1000", func(t *testing.T) {
		textPbPath := `
			elem: <name: "mac" 
				key: { key: "vlan" value: "1000" }
				>
		`
		wantRespVal := []byte(`[
        {"macAddress": "e8:eb:d3:32:f0:08", "port": "Ethernet320", "type": "dynamic", "vlan": "1000"},
        {"macAddress": "e8:eb:d3:32:f0:1b", "port": "Ethernet108", "type": "dynamic", "vlan": "1000"},
        {"macAddress": "e8:eb:d3:32:f0:1e", "port": "Ethernet120", "type": "dynamic", "vlan": "1000"},
        {"macAddress": "e8:eb:d3:32:f0:25", "port": "Ethernet148", "type": "static", "vlan": "1000"}
    ]`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	t.Run("query SHOW mac -t dynamic", func(t *testing.T) {
		textPbPath := `
			elem: <name: "mac" 
				key: { key: "type" value: "dynamic" }
				>
		`
		wantRespVal := []byte(`[
        {"macAddress": "e8:eb:d3:32:f0:08", "port": "Ethernet320", "type": "dynamic", "vlan": "1000"},
        {"macAddress": "e8:eb:d3:32:f0:1b", "port": "Ethernet108", "type": "dynamic", "vlan": "1000"},
        {"macAddress": "e8:eb:d3:32:f0:1e", "port": "Ethernet120", "type": "dynamic", "vlan": "1000"},
        {"macAddress": "e8:eb:d3:32:f0:28", "port": "Ethernet160", "type": "dynamic", "vlan": "1001"}
    ]`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	t.Run("query SHOW mac -p Ethernet320", func(t *testing.T) {
		textPbPath := `
			elem: <name: "mac" 
			key: { key: "port" value: "Ethernet320" }
			>
		`
		wantRespVal := []byte(`[
        {"macAddress": "e8:eb:d3:32:f0:08", "port": "Ethernet320", "type": "dynamic", "vlan": "1000"}
    ]`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})
}
