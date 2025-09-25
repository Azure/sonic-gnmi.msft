package gnmi

import (
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	sccommon "github.com/sonic-net/sonic-gnmi/show_client/common"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestGetArpTable(t *testing.T) {
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

	expectedArp := `
	{
		"arp_entries": [
		{
			"address": "10.0.0.1",
			"mac_address": "aa:bb:cc:dd:ee:ff",
			"iface": "eth0",
			"vlan": "Vlan100"
		}
		]
	}
	`

	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal interface{}
		valTest     bool
		testInit    func() *gomonkey.Patches
	}{
		{
			desc:       "query show arp with valid entry",
			pathTarget: "SHOW",
			textPbPath: `
			elem: <name: "arp" >
			elem: <name: "10.0.0.1"
			key: { key: "iface" value: "eth0" }
			key: { key: "SONIC_CLI_IFACE_MODE" value: "alias" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedArp),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				return gomonkey.ApplyFunc(sccommon.GetDataFromHostCommand, func(cmd string) (string, error) {
					return `
					Address    MacAddress    Iface    Vlan
					---------  ------------  -------  ------
					10.0.0.1   aa:bb:cc:dd:ee:ff  eth0  Vlan100
					Total number of entries 1
					`, nil
				})
			},
		},
		{
			desc:       "query show arp with alias conversion",
			pathTarget: "SHOW",
			textPbPath: `
			elem: <name: "arp" >
			elem: <name: "10.0.0.1"
			key: { key: "iface" value: "Ethernet0" }
			key: { key: "SONIC_CLI_IFACE_MODE" value: "alias" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedArp),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(sccommon.GetDataFromHostCommand, func(cmd string) (string, error) {
					return `
					Address    MacAddress    Iface    Vlan
					---------  ------------  -------  ------
					10.0.0.1   aa:bb:cc:dd:ee:ff  eth0  Vlan100
					Total number of entries 1
					`, nil
				})
				patches.ApplyFunc(sccommon.TryConvertInterfaceNameFromAlias, func(interfaceName string, namingMode sccommon.InterfaceNamingMode) (string, error) {
					if interfaceName == "Ethernet0" && namingMode == sccommon.Alias {
						return "eth0", nil
					}
					return "", fmt.Errorf("mocked conversion failure")
				})
				return patches
			},
		},
		{
			desc:       "query show arp with alias conversion failure",
			pathTarget: "SHOW",
			textPbPath: `
			elem: <name: "arp" >
			elem: <name: "10.0.0.1"
			key: { key: "iface" value: "Ethernet0" }
			key: { key: "SONIC_CLI_IFACE_MODE" value: "alias" } >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(sccommon.GetDataFromHostCommand, func(cmd string) (string, error) {
					return `
					Address    MacAddress    Iface    Vlan
					---------  ------------  -------  ------
					10.0.0.1   aa:bb:cc:dd:ee:ff  eth0  Vlan100
					Total number of entries 1
					`, nil
				})
				patches.ApplyFunc(sccommon.TryConvertInterfaceNameFromAlias, func(interfaceName string, namingMode sccommon.InterfaceNamingMode) (string, error) {
					return "", fmt.Errorf("Cannot find interface name for alias %s", interfaceName)
				})
				return patches
			},
		},
		{
			desc:       "query show arp with empty output",
			pathTarget: "SHOW",
			textPbPath: `
			elem: <name: "arp" key: { key: "SONIC_CLI_IFACE_MODE" value: "default" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{"arp_entries":[]}`),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				return gomonkey.ApplyFunc(sccommon.GetDataFromHostCommand, func(cmd string) (string, error) {
					return `
					Address    MacAddress    Iface    Vlan
					---------  ------------  -------  ------
					Total number of entries 0
					`, nil
				})
			},
		},
		{
			desc:       "query show arp with command error",
			pathTarget: "SHOW",
			textPbPath: `
			elem: <name: "arp" key: { key: "SONIC_CLI_IFACE_MODE" value: "default" } >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() *gomonkey.Patches {
				return gomonkey.ApplyFunc(sccommon.GetDataFromHostCommand, func(cmd string) (string, error) {
					return "", fmt.Errorf("simulated command failure")
				})
			},
		},
		{
			desc:       "query show arp with invalid IPv4 address",
			pathTarget: "SHOW",
			textPbPath: `
			elem: <name: "arp" >
			elem: <name: "10.0.0.999"
			key: { key: "iface" value: "eth0" }
			key: { key: "SONIC_CLI_IFACE_MODE" value: "alias" } >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() *gomonkey.Patches {
				return nil
			},
		},
	}

	for _, test := range tests {
		var patch *gomonkey.Patches
		if test.testInit != nil {
			patch = test.testInit()
		}

		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})

		if patch != nil {
			patch.Reset()
		}
	}
}
