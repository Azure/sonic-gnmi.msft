package gnmi

// interface_flap_cli_test.go
// Tests SHOW interface/flap

import (
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	show_client "github.com/sonic-net/sonic-gnmi/show_client"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestGetShowInterfaceFlap(t *testing.T) {
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

	expectedAll := `[{"Interface":"Ethernet0","Flap Count":"3","Admin":"Up","Oper":"Down","Link Down TimeStamp(UTC)":"2000","Link Up TimeStamp(UTC)":"1000"},{"Interface":"Ethernet1","Flap Count":"0","Admin":"Down","Oper":"Down","Link Down TimeStamp(UTC)":"1234","Link Up TimeStamp(UTC)":"Never"}]`
	expectedSingle := `[{"Interface":"Ethernet0","Flap Count":"3","Admin":"Up","Oper":"Down","Link Down TimeStamp(UTC)":"2000","Link Up TimeStamp(UTC)":"1000"}]`
	expectedNamingModeAliasWithInterface := `[{"Interface":"Ethernet0","Flap Count":"3","Admin":"Up","Oper":"Down","Link Down TimeStamp(UTC)":"2000","Link Up TimeStamp(UTC)":"1000"}]`

	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal interface{}
		valTest     bool
		testInit    func()
		mockPatch   func() []*gomonkey.Patches
		teardown    func()
	}{
		{
			desc:       "query SHOW interfaces flap NO DATA",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "flap" >
            `,
			wantRetCode: codes.OK,
			valTest:     false,
			testInit: func() {
				// Ensure APPL DB empty
				FlushDataSet(t, ApplDbNum)
			},
		},
		{
			desc:       "query SHOW interfaces flap (load appl port_table - all)",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "flap" >
            `,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedAll),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ApplDbNum)
				AddDataSet(t, ApplDbNum, "../testdata/APPL_PORT_TABLE_FLAP.txt")
			},
		},
		{
			desc:       "query SHOW interfaces flap (load appl port_table - single interface)",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "flap" key: { key: "interface" value: "Ethernet0" } >
            `,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedSingle),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ApplDbNum)
				AddDataSet(t, ApplDbNum, "../testdata/APPL_PORT_TABLE_FLAP.txt")
			},
		},
		{
			desc:       "query SHOW interfaces flap - invalid interface",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "flap" key: { key: "interface" value: "Ethernet999" } >
            `,
			wantRetCode: codes.NotFound,
			valTest:     false,
			// ensure port table empty -> invalid interface
			testInit: func() {
				FlushDataSet(t, ApplDbNum)
				AddDataSet(t, ApplDbNum, "../testdata/APPL_PORT_TABLE_FLAP.txt")
			},
		},
		{
			desc:       "query SHOW interfaces flap - GetMapFromQueries error",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "flap" >
            `,
			wantRetCode: codes.NotFound,
			valTest:     false,
			mockPatch: func() []*gomonkey.Patches {
				p := gomonkey.ApplyFunc(show_client.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					if len(queries) > 0 && len(queries[0]) > 1 && queries[0][0] == "APPL_DB" && queries[0][1] == show_client.AppDBPortTable {
						return nil, fmt.Errorf("injected failure")
					}
					return map[string]interface{}{}, nil
				})
				return []*gomonkey.Patches{p}
			},
		},
		{
			desc:       "query SHOW interfaces flap with interface=etp0 and SONIC_CLI_IFACE_MODE=alias",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "flap" key: { key: "interface" value: "etp0" } key: { key: "SONIC_CLI_IFACE_MODE" value: "alias" } >
            `,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedNamingModeAliasWithInterface),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ApplDbNum)
				AddDataSet(t, ApplDbNum, "../testdata/APPL_PORT_TABLE_FLAP.txt")
			},
			mockPatch: func() []*gomonkey.Patches {
				p1 := gomonkey.ApplyFunc(sdc.AliasToPortNameMap, func() map[string]string {
					return map[string]string{"etp0": "Ethernet0"}
				})
				p2 := gomonkey.ApplyFunc(sdc.PortToAliasNameMap, func() map[string]string {
					return map[string]string{"Ethernet0": "etp0"}
				})
				return []*gomonkey.Patches{p1, p2}
			},
		},
	}

	for _, test := range tests {
		// setup test dataset if provided
		if test.testInit != nil {
			test.testInit()
		}

		var patches []*gomonkey.Patches
		if test.mockPatch != nil {
			patches = test.mockPatch()
		}

		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})

		for _, p := range patches {
			if p != nil {
				p.Reset()
			}
		}
		if test.teardown != nil {
			test.teardown()
		}
	}
}
