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
