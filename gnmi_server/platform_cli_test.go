package gnmi

// platform_cli_test.go

// Tests SHOW platform summary and psustatus

import (
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"

	"github.com/agiledragon/gomonkey/v2"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"

	sccommon "github.com/sonic-net/sonic-gnmi/show_client/common"
)

func TestGetShowPlatformSummary(t *testing.T) {
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
				AddDataSet(t, StateDbNum, chassisDataFilename)
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
	expectedOutput := `[{"index":"1","name":"PSU 1","presence":"true","status":"OK","led_status":"green","model":"PWR-500AC-F","serial":"ABC12345678","revision":"A1","voltage":"12.00","current":"5.50","power":"66.00"},{"index":"2","name":"PSU 2","presence":"false","status":"NOT PRESENT","led_status":"off","model":"N/A","serial":"N/A","revision":"N/A","voltage":"N/A","current":"N/A","power":"N/A"}]`

	expectedOutputWarning := `[{"index":"1","name":"PSU 1","presence":"true","status":"WARNING","led_status":"amber","model":"PWR-500AC-F","serial":"XYZ98765432","revision":"B2","voltage":"12.50","current":"8.00","power":"100.00"}]`

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

func TestGetShowPlatformFan(t *testing.T) {
	expectedOutput := `[{"drawer":"Drawer 1","led":"green","fan":"FAN 1","speed":"60%","direction":"intake","presence":"Present","status":"OK","timestamp":"20250217 12:00:00"},{"drawer":"Drawer 1","led":"green","fan":"FAN 2","speed":"65%","direction":"intake","presence":"Present","status":"OK","timestamp":"20250217 12:00:00"},{"drawer":"Drawer 2","led":"red","fan":"FAN 3","speed":"0%","direction":"N/A","presence":"Not Present","status":"Not OK","timestamp":"20250217 12:00:00"}]`
	expectedHighSpeedOutput := `[{"drawer":"Drawer 1","led":"green","fan":"FAN 1","speed":"5000RPM","direction":"exhaust","presence":"Present","status":"OK","timestamp":"20250217 13:00:00"}]`

	fanDataFilename := "../testdata/FAN_STATUS.json"
	fanEmptyFilename := "../testdata/EMPTY_JSON.txt"
	fanHighSpeedFilename := "../testdata/FAN_STATUS_HIGH_SPEED.json"

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
			desc:       "query SHOW platform fan with no fans",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "fan" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{"message":"Fan not detected"}`),
			valTest:     true,
			testInit: func() {
				ResetDataSetsAndMappings(t)
				AddDataSet(t, StateDbNum, fanEmptyFilename)
			},
		},
		{
			desc:       "query SHOW platform fan with multiple fans",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "fan" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedOutput),
			valTest:     true,
			testInit: func() {
				ResetDataSetsAndMappings(t)
				AddDataSet(t, StateDbNum, fanDataFilename)
			},
		},
		{
			desc:       "query SHOW platform fan with high speed (RPM format)",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "fan" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedHighSpeedOutput),
			valTest:     true,
			testInit: func() {
				ResetDataSetsAndMappings(t)
				AddDataSet(t, StateDbNum, fanHighSpeedFilename)
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

func TestGetShowPlatformTemperature(t *testing.T) {
	expectedOutput := `[{"sensor":"Sensor 1","temperature":"45.5","high_th":"75.0","low_th":"5.0","crit_high_th":"85.0","crit_low_th":"0.0","warning":"False","timestamp":"20250217 12:00:00"},{"sensor":"Sensor 2","temperature":"50.0","high_th":"75.0","low_th":"5.0","crit_high_th":"85.0","crit_low_th":"0.0","warning":"False","timestamp":"20250217 12:00:00"},{"sensor":"Sensor 3","temperature":"78.0","high_th":"75.0","low_th":"5.0","crit_high_th":"85.0","crit_low_th":"0.0","warning":"True","timestamp":"20250217 12:00:00"}]`

	tempDataFilename := "../testdata/TEMPERATURE_STATUS.json"
	tempEmptyFilename := "../testdata/EMPTY_JSON.txt"

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
			desc:       "query SHOW platform temperature with no sensors",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "temperature" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{"message":"Sensor not detected"}`),
			valTest:     true,
			testInit: func() {
				ResetDataSetsAndMappings(t)
				AddDataSet(t, StateDbNum, tempEmptyFilename)
			},
		},
		{
			desc:       "query SHOW platform temperature",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "temperature" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedOutput),
			valTest:     true,
			testInit: func() {
				ResetDataSetsAndMappings(t)
				AddDataSet(t, StateDbNum, tempDataFilename)
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

func TestGetShowPlatformVoltage(t *testing.T) {
	expectedOutput := `[{"sensor":"VoltSensor 1","voltage":"12.0 V","high_th":"13.2","low_th":"10.8","crit_high_th":"14.0","crit_low_th":"10.0","warning":"False","timestamp":"20250217 12:00:00"},{"sensor":"VoltSensor 2","voltage":"5.0 V","high_th":"5.5","low_th":"4.5","crit_high_th":"6.0","crit_low_th":"4.0","warning":"False","timestamp":"20250217 12:00:00"}]`

	voltageDataFilename := "../testdata/VOLTAGE_STATUS.json"
	voltageEmptyFilename := "../testdata/EMPTY_JSON.txt"

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
			desc:       "query SHOW platform voltage with no sensors",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "voltage" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{"message":"Sensor not detected"}`),
			valTest:     true,
			testInit: func() {
				ResetDataSetsAndMappings(t)
				AddDataSet(t, StateDbNum, voltageEmptyFilename)
			},
		},
		{
			desc:       "query SHOW platform voltage",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "voltage" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedOutput),
			valTest:     true,
			testInit: func() {
				ResetDataSetsAndMappings(t)
				AddDataSet(t, StateDbNum, voltageDataFilename)
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

func TestGetShowPlatformCurrent(t *testing.T) {
	expectedOutput := `[{"sensor":"CurrentSensor 1","current":"5.5 A","high_th":"10.0","low_th":"0.5","crit_high_th":"12.0","crit_low_th":"0.0","warning":"False","timestamp":"20250217 12:00:00"},{"sensor":"CurrentSensor 2","current":"3.2 A","high_th":"8.0","low_th":"0.5","crit_high_th":"10.0","crit_low_th":"0.0","warning":"False","timestamp":"20250217 12:00:00"}]`

	currentDataFilename := "../testdata/CURRENT_STATUS.json"
	currentEmptyFilename := "../testdata/EMPTY_JSON.txt"

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
			desc:       "query SHOW platform current with no sensors",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "current" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{"message":"Sensor not detected"}`),
			valTest:     true,
			testInit: func() {
				ResetDataSetsAndMappings(t)
				AddDataSet(t, StateDbNum, currentEmptyFilename)
			},
		},
		{
			desc:       "query SHOW platform current",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "current" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedOutput),
			valTest:     true,
			testInit: func() {
				ResetDataSetsAndMappings(t)
				AddDataSet(t, StateDbNum, currentDataFilename)
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

func TestGetShowPlatformPcieinfo(t *testing.T) {
	showOutputFilename := "../testdata/PCIEINFO_SHOW.json"
	checkOutputFilename := "../testdata/PCIEINFO_CHECK.json"
	invalidOutputFilename := "../testdata/INVALID_JSON.txt"

	showOutputBytes, err := os.ReadFile(showOutputFilename)
	if err != nil {
		t.Fatalf("read file %v err: %v", showOutputFilename, err)
	}
	checkOutputBytes, err := os.ReadFile(checkOutputFilename)
	if err != nil {
		t.Fatalf("read file %v err: %v", checkOutputFilename, err)
	}

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
			desc:       "query SHOW platform pcieinfo",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "pcieinfo" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: showOutputBytes,
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				return gomonkey.ApplyFunc(sccommon.GetDataFromHostCommand, func(cmd string) (string, error) {
					if strings.Contains(cmd, "get_pcie_device") {
						return string(showOutputBytes), nil
					}
					return "", fmt.Errorf("unexpected command: %s", cmd)
				})
			},
		},
		{
			desc:       "query SHOW platform pcieinfo check",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "pcieinfo" key: { key: "check" value: "true" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: checkOutputBytes,
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				return gomonkey.ApplyFunc(sccommon.GetDataFromHostCommand, func(cmd string) (string, error) {
					if strings.Contains(cmd, "get_pcie_check") {
						return string(checkOutputBytes), nil
					}
					return "", fmt.Errorf("unexpected command: %s", cmd)
				})
			},
		},
		{
			desc:       "query SHOW platform pcieinfo invalid JSON output",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "pcieinfo" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() *gomonkey.Patches {
				return MockNSEnterOutput(t, invalidOutputFilename)
			},
		},
		{
			desc:       "query SHOW platform pcieinfo host command error",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "pcieinfo" >
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
			desc:       "query SHOW platform pcieinfo with verbose flag",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "pcieinfo" key: { key: "verbose" value: "true" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: showOutputBytes,
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				return gomonkey.ApplyFunc(sccommon.GetDataFromHostCommand, func(cmd string) (string, error) {
					if strings.Contains(cmd, "get_pcie_device") {
						return string(showOutputBytes), nil
					}
					return "", fmt.Errorf("unexpected command: %s", cmd)
				})
			},
		},
		{
			desc:       "query SHOW platform pcieinfo check with verbose flag",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "pcieinfo" key: { key: "check" value: "true" } key: { key: "verbose" value: "true" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: checkOutputBytes,
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				return gomonkey.ApplyFunc(sccommon.GetDataFromHostCommand, func(cmd string) (string, error) {
					if strings.Contains(cmd, "get_pcie_check") {
						return string(checkOutputBytes), nil
					}
					return "", fmt.Errorf("unexpected command: %s", cmd)
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
