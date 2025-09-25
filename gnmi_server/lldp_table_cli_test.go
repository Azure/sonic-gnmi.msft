package gnmi

// lldp_table_cli_test.go
// Tests SHOW lldp table

import (
	"crypto/tls"
	"io/ioutil"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"

	"github.com/agiledragon/gomonkey/v2"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

func TestGetLLDPTable(t *testing.T) {
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

	ResetDataSetsAndMappings(t)

	// expected empty response
	expectedEmptyDBResponse := `{"capability_codes_helper":"Capability codes: (R) Router, (B) Bridge, (O) Other","neighbors": [],"total": 0}`

	// Expected output for standalone device
	expectedLLDPTableOnlyOneResponseFileName := "../testdata/lldp/Expected_show_lldp_table_response_only_one.txt"
	expectedLLDPTableOnlyResponse, err := ioutil.ReadFile(expectedLLDPTableOnlyOneResponseFileName)
	if err != nil {
		t.Fatalf("Failed to read file %v err: %v", expectedLLDPTableOnlyOneResponseFileName, err)
	}

	// Expected output for normal device
	expectedLLDPTableResponseFileName := "../testdata/lldp/Expected_show_lldp_table_response.txt"
	expectedLLDPTableResponse, err := ioutil.ReadFile(expectedLLDPTableResponseFileName)
	if err != nil {
		t.Fatalf("Failed to read file %v err: %v", expectedLLDPTableResponseFileName, err)
	}

	// Expected output for normal device with path option SONIC_CLI_IFACE_MODE=alias
	expectedLLDPTableResponseAliasFileName := "../testdata/lldp/Expected_show_lldp_table_alias_response.txt"
	expectedLLDPTableResponseAlias, err := ioutil.ReadFile(expectedLLDPTableResponseAliasFileName)
	if err != nil {
		t.Fatalf("Failed to read file %v err: %v", expectedLLDPTableResponseAliasFileName, err)
	}

	tests := []struct {
		desc           string
		pathTarget     string
		textPbPath     string
		wantRetCode    codes.Code
		wantRespVal    interface{}
		valTest        bool
		mockOutputFile map[string]string
		ignoreValOrder bool
	}{
		{
			desc:       "query SHOW lldp table - empty json output",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "lldp" >
				elem: <name: "table" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedEmptyDBResponse),
			valTest:     true,
			mockOutputFile: map[string]string{
				"docker": "../testdata/lldp/lldpctl_empty_json.txt",
			},
		},
		{
			desc:       "query SHOW lldp table - standalone",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "lldp" >
				elem: <name: "table" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedLLDPTableOnlyResponse),
			valTest:     true,
			mockOutputFile: map[string]string{
				"docker": "../testdata/lldp/lldpctl_only_one_interface_json.txt",
			},
			ignoreValOrder: true,
		},
		{
			desc:       "query SHOW lldp table - normal device",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "lldp" >
				elem: <name: "table" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedLLDPTableResponse),
			valTest:     true,
			mockOutputFile: map[string]string{
				"docker": "../testdata/lldp/lldpctl_json.txt",
			},
			ignoreValOrder: true,
		},
		{
			desc:       "query SHOW lldp table - normal device - path option SONIC_CLI_IFACE_MODE=alias",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "lldp" >
				elem: <name: "table" key: { key: "SONIC_CLI_IFACE_MODE" value: "alias" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedLLDPTableResponseAlias),
			valTest:     true,
			mockOutputFile: map[string]string{
				"docker": "../testdata/lldp/lldpctl_json.txt",
			},
			ignoreValOrder: true,
		},
		{
			desc:       "query SHOW lldp table - normal device - path option SONIC_CLI_IFACE_MODE=wrongNameMode",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "lldp" >
				elem: <name: "table" key: { key: "SONIC_CLI_IFACE_MODE" value: "wrongNameMode" } >
			`,
			wantRetCode: codes.InvalidArgument,
		},
	}

	for _, test := range tests {
		var patchesSlice []*gomonkey.Patches
		patchesSlice = append(patchesSlice, gomonkey.ApplyFunc(sdc.PortToAliasNameMap, func() map[string]string {
			return map[string]string{
				"eth0": "etp0",
				"Ethernet353": "etp353",
				"Ethernet354": "etp354",
				"Ethernet355": "etp355",
				"Ethernet356": "etp356",
			}
		}))

		if len(test.mockOutputFile) > 0 {
			patchesSlice = append(patchesSlice, MockExecCmds(t, test.mockOutputFile))
		}

		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest, test.ignoreValOrder)
		})

		for _, patches := range patchesSlice {
			patches.Reset()
		}
	}
}
