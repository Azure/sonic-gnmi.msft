package gnmi

// mac_cli_test.go

// Tests SHOW mac CLI command

import (
	"crypto/tls"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestShowMacCommand(t *testing.T) {
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	stateDbContentFileNameForShowMac := "../testdata/ShowMacStateDb.txt"
	configDbContentFileNameForShowMac := "../testdata/ShowMacConfigDB.txt"

	FlushDataSet(t, StateDbNum)
	FlushDataSet(t, ConfigDbNum)
	AddDataSet(t, StateDbNum, stateDbContentFileNameForShowMac)
	AddDataSet(t, ConfigDbNum, configDbContentFileNameForShowMac)

	t.Run("query SHOW mac", func(t *testing.T) {
		textPbPath := `
			elem: <name: "mac" >
		`
		wantRespVal := []byte(`{
			"entries":[
        {"macAddress": "e8:eb:d3:32:f0:08", "port": "Ethernet320", "type": "dynamic", "vlan": 1000},
        {"macAddress": "e8:eb:d3:32:f0:1b", "port": "Ethernet108", "type": "dynamic", "vlan": 1000},
        {"macAddress": "e8:eb:d3:32:f0:1e", "port": "Ethernet120", "type": "dynamic", "vlan": 1000},
        {"macAddress": "e8:eb:d3:32:f0:25", "port": "Ethernet148", "type": "static", "vlan": 1000},
        {"macAddress": "e8:eb:d3:32:f0:28", "port": "Ethernet160", "type": "dynamic", "vlan": 1001}
			],
			"total": 5
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	t.Run("query SHOW mac -c", func(t *testing.T) {
		textPbPath := `
			elem: <name: "mac"  key: { key: "count" value: "True" } >
		`
		wantRespVal := []byte(`{
														  "total": 5
													 }`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	t.Run("query SHOW mac -a e8:eb:d3:32:f0:08", func(t *testing.T) {
		textPbPath := `
			elem: <name: "mac" 
				key: { key: "address" value: "e8:eb:d3:32:f0:08" }
				>
		`
		wantRespVal := []byte(`{
				"entries":[
						{"macAddress": "e8:eb:d3:32:f0:08", "port": "Ethernet320", "type": "dynamic", "vlan": 1000}
					],
				"total": 1
		}`)
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
														"total": 1
														}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	t.Run("query SHOW mac -v 1000", func(t *testing.T) {
		textPbPath := `
			elem: <name: "mac" 
				key: { key: "vlan" value: "1000" }
				>
		`
		wantRespVal := []byte(`{
			"entries":[
									{"macAddress": "e8:eb:d3:32:f0:08", "port": "Ethernet320", "type": "dynamic", "vlan": 1000},
									{"macAddress": "e8:eb:d3:32:f0:1b", "port": "Ethernet108", "type": "dynamic", "vlan": 1000},
									{"macAddress": "e8:eb:d3:32:f0:1e", "port": "Ethernet120", "type": "dynamic", "vlan": 1000},
									{"macAddress": "e8:eb:d3:32:f0:25", "port": "Ethernet148", "type": "static", "vlan": 1000}
								],
								"total": 4
				}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	t.Run("query SHOW mac -t dynamic", func(t *testing.T) {
		textPbPath := `
			elem: <name: "mac" 
				key: { key: "type" value: "dynamic" }
				>
		`
		wantRespVal := []byte(`{
			"entries":[
					{"macAddress": "e8:eb:d3:32:f0:08", "port": "Ethernet320", "type": "dynamic", "vlan": 1000},
					{"macAddress": "e8:eb:d3:32:f0:1b", "port": "Ethernet108", "type": "dynamic", "vlan": 1000},
					{"macAddress": "e8:eb:d3:32:f0:1e", "port": "Ethernet120", "type": "dynamic", "vlan": 1000},
					{"macAddress": "e8:eb:d3:32:f0:28", "port": "Ethernet160", "type": "dynamic", "vlan": 1001}
			],
		"total":4
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	t.Run("query SHOW mac -p Ethernet320", func(t *testing.T) {
		textPbPath := `
			elem: <name: "mac" 
			key: { key: "port" value: "Ethernet320" }
			>
		`
		wantRespVal := []byte(`{
			"entries":[
				{"macAddress": "e8:eb:d3:32:f0:08", "port": "Ethernet320", "type": "dynamic", "vlan": 1000}
			],
			"total": 1
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	// Ethernet121 port didn't exist in STATE_DB, it exist in CONFIG_DB
	t.Run("query SHOW mac -p Ethernet121", func(t *testing.T) {
		textPbPath := `
			elem: <name: "mac" 
			key: { key: "port" value: "Ethernet121" }
			>
		`
		wantRespVal := []byte(`{
			"entries":[],
			"total": 0
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	// Invalid Port
	t.Run("query SHOW mac -p Ethernet999", func(t *testing.T) {
		textPbPath := `
			elem: <name: "mac" 
			key: { key: "port" value: "Ethernet999" }
			>
		`
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
	})

	// Invalid Type
	t.Run("query SHOW mac -t DDStatic", func(t *testing.T) {
		textPbPath := `
			elem: <name: "mac" 
			key: { key: "type" value: "DDStatic" }
			>
		`
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
	})

	// Invalid mac address
	t.Run("query SHOW mac -a g8:eb:d3:32:f0:08", func(t *testing.T) {
		textPbPath := `
			elem: <name: "mac" 
				key: { key: "address" value: "g8:eb:d3:32:f0:08" }
				>
		`
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
	})

	// Invalid vlan
	t.Run("query SHOW mac -v 4096", func(t *testing.T) {
		textPbPath := `
			elem: <name: "mac" 
				key: { key: "vlan" value: "4096" }
				>
		`
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
	})
}
