package gnmi

// lldp_neighbors_cli_test.go
// Tests SHOW lldp neighbors

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

func TestGetLLDPNeighbors(t *testing.T) {
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
	expectedEmptyDBResponse := `{"title": "LLDP neighbors","interfaces": {}}`

	// Expected output for standalone device
	expectedLLDPNeighborsOnlyOneResponseFileName := "../testdata/lldp/Expected_show_lldp_neighbors_response_only_one.txt"
	expectedLLDPNeighborsOnlyOneResponse, err := ioutil.ReadFile(expectedLLDPNeighborsOnlyOneResponseFileName)
	if err != nil {
		t.Fatalf("Failed to read file %v err: %v", expectedLLDPNeighborsOnlyOneResponseFileName, err)
	}

	// Expected output for normal device
	expectedLLDPNeighborsResponseFileName := "../testdata/lldp/Expected_show_lldp_neighbors_response.txt"
	expectedLLDPNeighborsResponse, err := ioutil.ReadFile(expectedLLDPNeighborsResponseFileName)
	if err != nil {
		t.Fatalf("Failed to read file %v err: %v", expectedLLDPNeighborsResponseFileName, err)
	}

	// Expected output for normal device with path option SONIC_CLI_IFACE_MODE=alias
	expectedLLDPNeighborsAliasResponseFileName := "../testdata/lldp/Expected_show_lldp_neighbors_alias_response.txt"
	expectedLLDPNeighborsAliasResponse, err := ioutil.ReadFile(expectedLLDPNeighborsAliasResponseFileName)
	if err != nil {
		t.Fatalf("Failed to read file %v err: %v", expectedLLDPNeighborsAliasResponseFileName, err)
	}

	// Expected output for normal device with specified interface name
	expectedLLDPNeighborsWithIfNameResponseFileName := "../testdata/lldp/Expected_show_lldp_neighbors_filtered_response.txt"
	expectedLLDPNeighborsWithIfNameResponse, err := ioutil.ReadFile(expectedLLDPNeighborsWithIfNameResponseFileName)
	if err != nil {
		t.Fatalf("Failed to read file %v err: %v", expectedLLDPNeighborsWithIfNameResponseFileName, err)
	}

	// Expected output for normal device with specified interface alias name
	expectedLLDPNeighborsWithIfAliasNameResponseFileName := "../testdata/lldp/Expected_show_lldp_neighbors_filtered_alias_response.txt"
	expectedLLDPNeighborsWithIfAliasNameResponse, err := ioutil.ReadFile(expectedLLDPNeighborsWithIfAliasNameResponseFileName)
	if err != nil {
		t.Fatalf("Failed to read file %v err: %v", expectedLLDPNeighborsWithIfAliasNameResponseFileName, err)
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
		initAliasMap    bool
	}{
		{
			desc:       "query SHOW lldp neighbors - empty json output",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "lldp" >
				elem: <name: "neighbors" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedEmptyDBResponse),
			valTest:     true,
			mockOutputFile: map[string]string{
				"docker": "../testdata/lldp/lldpctl_empty_json.txt",
			},
		},
		{
			desc:       "query SHOW lldp neighbors - standalone",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "lldp" >
				elem: <name: "neighbors" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedLLDPNeighborsOnlyOneResponse),
			valTest:     true,
			mockOutputFile: map[string]string{
				"docker": "../testdata/lldp/lldpctl_only_one_interface_json.txt",
			},
			ignoreValOrder: true,
		},
		{
			desc:       "query SHOW lldp neighbors - normal device",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "lldp" >
				elem: <name: "neighbors" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedLLDPNeighborsResponse),
			valTest:     true,
			mockOutputFile: map[string]string{
				"docker": "../testdata/lldp/lldpctl_json.txt",
			},
			ignoreValOrder: true,
		},
		{
			desc:       "query SHOW lldp neighbors - normal device - path option SONIC_CLI_IFACE_MODE=alias",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "lldp" >
                elem: <name: "neighbors" key: { key: "SONIC_CLI_IFACE_MODE" value: "alias" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedLLDPNeighborsAliasResponse),
			valTest:     true,
			mockOutputFile: map[string]string{
				"docker": "../testdata/lldp/lldpctl_json.txt",
			},
			ignoreValOrder: true,
			initAliasMap:  true,
		},
		{
			desc:       "query SHOW lldp neighbors - normal device - path option SONIC_CLI_IFACE_MODE=alias-No alias map",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "lldp" >
                elem: <name: "neighbors" key: { key: "SONIC_CLI_IFACE_MODE" value: "alias" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedLLDPNeighborsResponse),
			valTest:     true,
			mockOutputFile: map[string]string{
				"docker": "../testdata/lldp/lldpctl_json.txt",
			},
			ignoreValOrder: true,
		},
		{
			desc:       "query SHOW lldp neighbors - normal device with specified interface name",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "lldp" >
				elem: <name: "neighbors" >
				elem: <name: "Ethernet354" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedLLDPNeighborsWithIfNameResponse),
			valTest:     true,
			mockOutputFile: map[string]string{
				"docker": "../testdata/lldp/lldpctl_specified_interface_json.txt",
			},
			ignoreValOrder: true,
		},
		{
			desc:       "query SHOW lldp neighbors - normal device with specified interface alias -path option SONIC_CLI_IFACE_MODE=alias-No alias map",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "lldp" >
				elem: <name: "neighbors" >
                elem: <name: "etp354" key: { key: "SONIC_CLI_IFACE_MODE" value: "alias" } >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: []byte(expectedLLDPNeighborsWithIfAliasNameResponse),
			valTest:     false,
			mockOutputFile: map[string]string{
				"docker": "../testdata/lldp/lldpctl_specified_interface_json.txt",
			},
			ignoreValOrder: true,
		},
		{
			desc:       "query SHOW lldp neighbors - normal device with specified interface alias -path option SONIC_CLI_IFACE_MODE=wrongNameMode",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "lldp" >
				elem: <name: "neighbors" >
                elem: <name: "etp354" key: { key: "SONIC_CLI_IFACE_MODE" value: "wrongNameMode" } >
			`,
			wantRetCode: codes.InvalidArgument,
		},
		{
			desc:       "query SHOW lldp neighbors - normal device with specified interface alias -path option SONIC_CLI_IFACE_MODE=alias",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "lldp" >
				elem: <name: "neighbors" >
                elem: <name: "etp354" key: { key: "SONIC_CLI_IFACE_MODE" value: "alias" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedLLDPNeighborsWithIfAliasNameResponse),
			valTest:     true,
			mockOutputFile: map[string]string{
				"docker": "../testdata/lldp/lldpctl_specified_interface_json.txt",
			},
			ignoreValOrder: true,
			initAliasMap:  true,
		},
		{
			desc:       "query SHOW lldp neighbors - normal device with specified interface name -path option SONIC_CLI_IFACE_MODE=alias",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "lldp" >
				elem: <name: "neighbors" >
                elem: <name: "Ethernet354" key: { key: "SONIC_CLI_IFACE_MODE" value: "alias" } >
			`,
			wantRetCode: codes.NotFound,
			valTest:     false,
			mockOutputFile: map[string]string{
				"docker": "../testdata/lldp/lldpctl_specified_interface_json.txt",
			},
			ignoreValOrder: true,
			initAliasMap:  true,
		},
		{
			desc:       "query SHOW lldp neighbors - normal device with non-existing interface name",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "lldp" >
				elem: <name: "neighbors" >
				elem: <name: "nonexist0" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedEmptyDBResponse),
			valTest:     true,
			mockOutputFile: map[string]string{
				"docker": "../testdata/lldp/lldpctl_non_exists_interface_json0.txt",
			},
			ignoreValOrder: true,
		},
	}

	for _, test := range tests {	
		var patchesSlice []*gomonkey.Patches
		if test.initAliasMap == false {
			patchesSlice = append(patchesSlice, gomonkey.ApplyFunc(sdc.PortToAliasNameMap, func() map[string]string {
				return map[string]string{}
			}))

			patchesSlice = append(patchesSlice, gomonkey.ApplyFunc(sdc.AliasToPortNameMap, func() map[string]string {
				return map[string]string{}
			}))
		} else {
			patchesSlice = append(patchesSlice, gomonkey.ApplyFunc(sdc.PortToAliasNameMap, func() map[string]string {
				return map[string]string{
					"eth0": "etp0",
					"Ethernet353": "etp353",
					"Ethernet354": "etp354",
					"Ethernet355": "etp355",
					"Ethernet356": "etp356",
				}
			}))

			patchesSlice = append(patchesSlice, gomonkey.ApplyFunc(sdc.AliasToPortNameMap, func() map[string]string {
				return map[string]string{
					"etp0": "eth0",
					"etp353": "Ethernet353",
					"etp354": "Ethernet354",
					"etp355": "Ethernet355",
					"etp356": "Ethernet356",
				}
			}))
		}

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
