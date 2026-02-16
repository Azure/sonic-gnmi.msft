package gnmi

// platform_cli_test.go

// Tests SHOW platform summary and psustatus

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
	sccommon "github.com/sonic-net/sonic-gnmi/show_client/common"
	show_client "github.com/sonic-net/sonic-gnmi/show_client"
)

func TestGetShowPlatformSummary(t *testing.T) {
	versionInfo := `
build_version: test_branch.1-a8fbac59d
asic_type: mellanox
`
	deviceMetadataFilename := "../testdata/PLATFORM_METADATA.txt"
	chassisDataFilename := "../testdata/PLATFORM_CHASSIS.txt"

	expectedOutput := `{"platform":"x86_64-mlnx_msn2700-r0","hwsku":"Mellanox-SN2700","asic_type":"mellanox","asic_count":"1","serial":"MT1234X56789","model":"MSN2700-CS2FO","revision":"A1"}`
	expectedOutputWithNA := `{"platform":"","hwsku":"","asic_type":"N/A","asic_count":"1","serial":"N/A","model":"N/A","revision":"N/A"}`

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
			desc:       "query SHOW platform summary with missing data",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "summary" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedOutputWithNA),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				ResetDataSetsAndMappings(t)
				sdc.ImplIoutilReadFile = ioutil.ReadFile
				return gomonkey.ApplyFunc(sccommon.GetPlatformInfo, func(versionInfo map[string]interface{}) (map[string]interface{}, error) {
					return map[string]interface{}{
						"platform":   "",
						"hwsku":      "",
						"asic_type":  "N/A",
						"asic_count": "1",
					}, nil
				})
			},
		},
		{
			desc:       "query SHOW platform summary success",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "summary" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedOutput),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				ResetDataSetsAndMappings(t)
				sdc.ImplIoutilReadFile = ioutil.ReadFile
				MockReadFile(show_client.SonicVersionYamlPath, versionInfo, nil)
				MockReadFile(deviceMetadataFilename, "onie_platform=x86_64-mlnx_msn2700-r0", nil)
				MockReadFile(chassisDataFilename, `{"chassis 1": {"serial": "MT1234X56789", "model": "MSN2700-CS2FO", "revision": "A1"}}`, nil)
				return gomonkey.ApplyFunc(sccommon.GetPlatformInfo, func(versionInfo map[string]interface{}) (map[string]interface{}, error) {
					return map[string]interface{}{
						"platform":   "x86_64-mlnx_msn2700-r0",
						"hwsku":      "Mellanox-SN2700",
						"asic_type":  "mellanox",
						"asic_count": "1",
					}, nil
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var patches *gomonkey.Patches
			if tt.testInit != nil {
				patches = tt.testInit()
			}
			defer func() {
				if patches != nil {
					patches.Reset()
				}
			}()

			s := createServer(t, ServerPort)
			go runServer(t, s)
			defer s.ForceStop()

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

			runTestGet(t, ctx, gClient, tt.pathTarget, tt.textPbPath, tt.wantRetCode, tt.wantRespVal, tt.valTest)
		})
	}
}

func TestGetShowPlatformPsustatus(t *testing.T) {
	expectedOutput := `[{"index":"1","name":"PSU 1","presence":"true","model":"PWR-500AC-F","serial":"ABC12345678","revision":"A1","voltage":"12.00","current":"5.50","power":"66.00","status":"OK","led_status":"green"},{"index":"2","name":"PSU 2","presence":"false","model":"N/A","serial":"N/A","revision":"N/A","voltage":"N/A","current":"N/A","power":"N/A","status":"NOT PRESENT","led_status":"off"}]`

	expectedOutputWarning := `[{"index":"1","name":"PSU 1","presence":"true","model":"PWR-500AC-F","serial":"XYZ98765432","revision":"B2","voltage":"12.50","current":"8.00","power":"100.00","status":"WARNING","led_status":"amber"}]`

	psuMixedFilename := "../testdata/PSU_STATUS_MIXED.json"
	psuWarningFilename := "../testdata/PSU_STATUS_WARNING.json"
	psuNoPsuFilename := "../testdata/PSU_STATUS_NO_PSU.json"

	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal interface{}
		valTest     bool
		testInit    func()
	}{
		{
			desc:       "query SHOW platform psustatus no PSUs",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "psustatus" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`[]`),
			valTest:     true,
			testInit: func() {
				ResetDataSetsAndMappings(t)
				AddDataSet(t, StateDbNum, psuNoPsuFilename)
			},
		},
		{
			desc:       "query SHOW platform psustatus with mixed PSU states",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "psustatus" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedOutput),
			valTest:     true,
			testInit: func() {
				ResetDataSetsAndMappings(t)
				AddDataSet(t, StateDbNum, psuMixedFilename)
			},
		},
		{
			desc:       "query SHOW platform psustatus with power overload warning",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "psustatus" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedOutputWarning),
			valTest:     true,
			testInit: func() {
				ResetDataSetsAndMappings(t)
				AddDataSet(t, StateDbNum, psuWarningFilename)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if tt.testInit != nil {
				tt.testInit()
			}

			s := createServer(t, ServerPort)
			go runServer(t, s)
			defer s.ForceStop()

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

			runTestGet(t, ctx, gClient, tt.pathTarget, tt.textPbPath, tt.wantRetCode, tt.wantRespVal, tt.valTest)
		})
	}
}
