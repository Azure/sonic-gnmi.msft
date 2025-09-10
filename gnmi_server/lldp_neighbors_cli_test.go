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

	// Expected output for normal device with specified interface name
	expectedLLDPNeighborsWithIfNameResponseFileName := "../testdata/lldp/Expected_show_lldp_neighbors_filtered_response.txt"
	expectedLLDPNeighborsWithIfNameResponse, err := ioutil.ReadFile(expectedLLDPNeighborsWithIfNameResponseFileName)
	if err != nil {
		t.Fatalf("Failed to read file %v err: %v", expectedLLDPNeighborsWithIfNameResponseFileName, err)
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
				"docker": "../testdata/lldp/lldpctl_json.txt",
			},
			ignoreValOrder: true,
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
				"docker": "../testdata/lldp/lldpctl_json.txt",
			},
			ignoreValOrder: true,
		},
	}

	for _, test := range tests {
		var patches *gomonkey.Patches
		if len(test.mockOutputFile) > 0 {
			patches = MockExecCmds(t, test.mockOutputFile)
		}

		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest, test.ignoreValOrder)
		})

		if patches != nil {
			patches.Reset()
		}
	}
}
