package gnmi

// interface_switchport_cli_test.go
// Tests SHOW interface/switchport/config and SHOW interface/switchport/status

import (
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"

	sccommon "github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// Test SHOW interface switchport config
func TestGetShowInterfaceSwitchportConfig(t *testing.T) {
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

	portsFileName := "../testdata/PORTS_SWITCHPORT.txt"
	vlanMemberFileName := "../testdata/VLAN_MEMBER_SWITCHPORT.txt"

	expectedConfig := `[{"Interface":"Ethernet0","Mode":"trunk","Tagged":"","Untagged":"1000"},{"Interface":"Ethernet1","Mode":"trunk","Tagged":"","Untagged":"1000"},{"Interface":"Ethernet2","Mode":"trunk","Tagged":"","Untagged":"1000"},{"Interface":"Ethernet3","Mode":"trunk","Tagged":"","Untagged":"1000"},{"Interface":"Ethernet4","Mode":"trunk","Tagged":"","Untagged":"1000"},{"Interface":"Ethernet5","Mode":"trunk","Tagged":"","Untagged":"1000"},{"Interface":"Ethernet6","Mode":"trunk","Tagged":"1000","Untagged":""},{"Interface":"Ethernet7","Mode":"routed","Tagged":"","Untagged":""}]`

	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal interface{}
		valTest     bool
		mockSleep   bool
		testInit    func()
		mockPatch   func() *gomonkey.Patches
		teardown    func()
	}{
		{
			desc:       "query SHOW interfaces switchport config NO DATA",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "switchport" >
                elem: <name: "config" >
            `,
			wantRetCode: codes.OK,
			valTest:     false,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
			},
		},
		{
			desc:       "query SHOW interfaces switchport config (load ports + vlan_member)",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "switchport" >
                elem: <name: "config" >
            `,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedConfig),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, portsFileName)
				AddDataSet(t, ConfigDbNum, vlanMemberFileName)
			},
		},
		{
			desc:       "query SHOW interfaces switchport config - GetMapFromQueries returns error",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "switchport" >
                elem: <name: "config" >
            `,
			wantRetCode: codes.NotFound, // server wraps getter errors as NotFound
			valTest:     false,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
			},
			mockPatch: func() *gomonkey.Patches {
				// inject GetMapFromQueries failure when table == "PORT"
				return gomonkey.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					if len(queries) > 0 && len(queries[0]) > 1 && queries[0][1] == "PORT" {
						return nil, fmt.Errorf("injected GetMapFromQueries failure")
					}
					// otherwise return empty but successful map
					return map[string]interface{}{}, nil
				})
			},
		},
		{
			desc:       "query SHOW interfaces switchport config - portchannel membership and colon-delimiter key",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "switchport" >
                elem: <name: "config" >
            `,
			wantRetCode: codes.OK,
			valTest:     false,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
			},
			mockPatch: func() *gomonkey.Patches {
				// Provide deterministic table content for PORT / PORTCHANNEL / PORTCHANNEL_MEMBER / VLAN_MEMBER
				return gomonkey.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					if len(queries) == 0 || len(queries[0]) < 2 {
						return map[string]interface{}{}, nil
					}
					table := queries[0][1]
					switch table {
					case "PORT":
						return map[string]interface{}{
							"Ethernet0": map[string]string{"alias": "etp0"},
							"Ethernet2": map[string]string{"alias": "etp2"},
							"Ethernet6": map[string]string{"alias": "etp6"},
						}, nil
					case "PORTCHANNEL":
						return map[string]interface{}{
							"PortChannel1": map[string]string{"mode": "trunk"},
						}, nil
					case "PORTCHANNEL_MEMBER":
						// use colon delimiter in member key
						return map[string]interface{}{
							"PortChannel1:Ethernet2": map[string]string{},
						}, nil
					case "VLAN_MEMBER":
						// include both '|' style and ':' style keys (the code's SplitCompositeKey supports both)
						return map[string]interface{}{
							"Vlan100|Ethernet0":  map[string]string{"tagging_mode": "untagged"},
							"Vlan2000:Ethernet6": map[string]string{"tagging_mode": "tagged"},
						}, nil
					default:
						return map[string]interface{}{}, nil
					}
				})
			},
		},
		{
			desc:       "query SHOW interfaces switchport config - path option SONIC_CLI_IFACE_MODE=alias",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "switchport" >
                elem: <name: "config" key: { key: "SONIC_CLI_IFACE_MODE" value: "alias" } >
            `,
			wantRetCode: codes.OK,
			valTest:     false,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, portsFileName)
				AddDataSet(t, ConfigDbNum, vlanMemberFileName)
			},
			mockPatch: func() *gomonkey.Patches {
				return gomonkey.ApplyFunc(sdc.PortToAliasNameMap, func() map[string]string {
					return map[string]string{
						"Ethernet0": "etp0",
						"Ethernet1": "etp1",
					}
				})
			},
		},
	}

	for _, test := range tests {
		if test.testInit != nil {
			test.testInit()
		}
		var patchesSlice []*gomonkey.Patches
		if test.mockSleep {
			patchesSlice = append(patchesSlice, gomonkey.ApplyFunc(time.Sleep, func(d time.Duration) {}))
		}
		if test.mockPatch != nil {
			if p := test.mockPatch(); p != nil {
				patchesSlice = append(patchesSlice, p)
			}
		}
		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})
		for _, p := range patchesSlice {
			p.Reset()
		}
		if test.teardown != nil {
			test.teardown()
		}
	}
}

// Test SHOW interface switchport status
func TestGetShowInterfaceSwitchportStatus(t *testing.T) {
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

	portsFileName := "../testdata/PORTS_SWITCHPORT.txt"
	vlanMemberFileName := "../testdata/VLAN_MEMBER_SWITCHPORT.txt"

	expectedStatus := `[{"Interface":"Ethernet0","Mode":"trunk"},{"Interface":"Ethernet1","Mode":"trunk"},{"Interface":"Ethernet2","Mode":"trunk"},{"Interface":"Ethernet3","Mode":"trunk"},{"Interface":"Ethernet4","Mode":"trunk"},{"Interface":"Ethernet5","Mode":"trunk"},{"Interface":"Ethernet6","Mode":"trunk"},{"Interface":"Ethernet7","Mode":"routed"}]`

	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal interface{}
		valTest     bool
		mockSleep   bool
		testInit    func()
		mockPatch   func() *gomonkey.Patches
		teardown    func()
	}{
		{
			desc:       "query SHOW interfaces switchport status NO DATA",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "switchport" >
                elem: <name: "status" >
            `,
			wantRetCode: codes.OK,
			valTest:     false,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
			},
		},
		{
			desc:       "query SHOW interfaces switchport status (load ports + vlan_member)",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "switchport" >
                elem: <name: "status" >
            `,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedStatus),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, portsFileName)
				AddDataSet(t, ConfigDbNum, vlanMemberFileName)
			},
		},
		{
			desc:       "query SHOW interfaces switchport status - GetMapFromQueries returns error",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "switchport" >
                elem: <name: "status" >
            `,
			wantRetCode: codes.NotFound,
			valTest:     false,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
			},
			mockPatch: func() *gomonkey.Patches {
				return gomonkey.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					if len(queries) > 0 && len(queries[0]) > 1 && queries[0][1] == "PORT" {
						return nil, fmt.Errorf("injected failure")
					}
					return map[string]interface{}{}, nil
				})
			},
		},
		{
			desc:       "query SHOW interfaces switchport status - portchannel membership and colon-delimiter key",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "switchport" >
                elem: <name: "status" >
            `,
			wantRetCode: codes.OK,
			valTest:     false,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
			},
			mockPatch: func() *gomonkey.Patches {
				return gomonkey.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					if len(queries) == 0 || len(queries[0]) < 2 {
						return map[string]interface{}{}, nil
					}
					table := queries[0][1]
					switch table {
					case "PORT":
						return map[string]interface{}{
							"Ethernet0": map[string]string{"alias": "etp0"},
							"Ethernet2": map[string]string{"alias": "etp2"},
						}, nil
					case "PORTCHANNEL":
						return map[string]interface{}{
							"PortChannel1": map[string]string{"mode": "trunk"},
						}, nil
					case "PORTCHANNEL_MEMBER":
						return map[string]interface{}{
							"PortChannel1:Ethernet2": map[string]string{},
						}, nil
					case "VLAN_MEMBER":
						return map[string]interface{}{
							"Vlan100|Ethernet0": map[string]string{"tagging_mode": "untagged"},
						}, nil
					default:
						return map[string]interface{}{}, nil
					}
				})
			},
		},
		{
			desc:       "query SHOW interfaces switchport status - path option SONIC_CLI_IFACE_MODE=alias",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "switchport" >
                elem: <name: "status" key: { key: "SONIC_CLI_IFACE_MODE" value: "alias" } >
            `,
			wantRetCode: codes.OK,
			valTest:     false,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, portsFileName)
				AddDataSet(t, ConfigDbNum, vlanMemberFileName)
			},
			mockPatch: func() *gomonkey.Patches {
				return gomonkey.ApplyFunc(sdc.PortToAliasNameMap, func() map[string]string {
					return map[string]string{
						"Ethernet0": "etp0",
						"Ethernet1": "etp1",
					}
				})
			},
		},
	}

	for _, test := range tests {
		if test.testInit != nil {
			test.testInit()
		}
		var patchesSlice []*gomonkey.Patches
		if test.mockSleep {
			patchesSlice = append(patchesSlice, gomonkey.ApplyFunc(time.Sleep, func(d time.Duration) {}))
		}
		if test.mockPatch != nil {
			if p := test.mockPatch(); p != nil {
				patchesSlice = append(patchesSlice, p)
			}
		}
		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})
		for _, p := range patchesSlice {
			p.Reset()
		}
		if test.teardown != nil {
			test.teardown()
		}
	}
}
