package gnmi

// platform_cli_test.go

// Tests SHOW platform summary, psustatus, fan, temperature, voltage, current, ssdhealth

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"

	"github.com/agiledragon/gomonkey/v2"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"

	sccommon "github.com/sonic-net/sonic-gnmi/show_client/common"
	"github.com/sonic-net/sonic-gnmi/show_client/helpers"
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

func TestGetShowPlatformSsdhealth(t *testing.T) {
	// Mock SsdInfo values — matches helpers.SsdInfo struct
	nvmeSsdInfo := &helpers.SsdInfo{
		Model:        "SAMSUNG MZQLB960HAJR-00007",
		Firmware:     "EDA5602Q",
		Serial:       "S439NA0M900123",
		Health:       "98",
		Temperature:  "33",
		VendorOutput: "Extended SMART info",
	}
	sataSsdInfo := &helpers.SsdInfo{
		Model:        "InnoDisk Corp. - mSATA 3IE3",
		Firmware:     "S16425cG",
		Serial:       "BCA11712190600081",
		Health:       "85",
		Temperature:  "43",
		VendorOutput: "",
	}
	nonNumericSsdInfo := &helpers.SsdInfo{
		Model:        "SAMSUNG MZQLB960HAJR-00007",
		Firmware:     "EDA5602Q",
		Serial:       "S439NA0M900123",
		Health:       "N/A",
		Temperature:  "N/A",
		VendorOutput: "",
	}

	// Expected outputs match SsdHealthInfo struct with omitempty:
	//   default: disk_type, device_model, health, temperature
	//   verbose: + firmware, serial
	//   vendor:  + vendor_output (only when non-empty)
	expectedNVMeDefault := `{"disk_type":"NVME","device_model":"SAMSUNG MZQLB960HAJR-00007","health":"98%","temperature":"33C"}`
	expectedNVMeVerbose := `{"disk_type":"NVME","device_model":"SAMSUNG MZQLB960HAJR-00007","firmware":"EDA5602Q","serial":"S439NA0M900123","health":"98%","temperature":"33C"}`
	expectedNVMeVendor := `{"disk_type":"NVME","device_model":"SAMSUNG MZQLB960HAJR-00007","health":"98%","temperature":"33C","vendor_output":"Extended SMART info"}`
	expectedNVMeVerboseVendor := `{"disk_type":"NVME","device_model":"SAMSUNG MZQLB960HAJR-00007","firmware":"EDA5602Q","serial":"S439NA0M900123","health":"98%","temperature":"33C","vendor_output":"Extended SMART info"}`
	expectedSATADefault := `{"disk_type":"SATA","device_model":"InnoDisk Corp. - mSATA 3IE3","health":"85%","temperature":"43C"}`
	expectedSATAVerbose := `{"disk_type":"SATA","device_model":"InnoDisk Corp. - mSATA 3IE3","firmware":"S16425cG","serial":"BCA11712190600081","health":"85%","temperature":"43C"}`
	expectedSATAVendor := `{"disk_type":"SATA","device_model":"InnoDisk Corp. - mSATA 3IE3","health":"85%","temperature":"43C"}`
	expectedNotDetected := `{"message":"SSD not detected"}`
	expectedPlatformJson := `{"disk_type":"NVME","device_model":"SAMSUNG MZQLB960HAJR-00007","health":"98%","temperature":"33C"}`
	expectedNonNumeric := `{"disk_type":"NVME","device_model":"SAMSUNG MZQLB960HAJR-00007","health":"N/A","temperature":"N/A"}`

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
			desc:       "query SHOW platform ssdhealth NVMe default (no options)",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "ssdhealth" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedNVMeDefault),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				ResetDataSetsAndMappings(t)
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(helpers.ImportSsdApi, func(device string) (*helpers.SsdInfo, error) {
					return nvmeSsdInfo, nil
				})
				patches.ApplyFunc(helpers.GetDefaultDisk, func() (string, string) {
					return "/dev/nvme0n1", "nvme"
				})
				patches.ApplyFunc(sccommon.GetPlatformJsonData, func() (map[string]interface{}, error) {
					return nil, fmt.Errorf("not found")
				})
				return patches
			},
		},
		{
			desc:       "query SHOW platform ssdhealth NVMe verbose",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "ssdhealth" key: { key: "verbose" value: "true" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedNVMeVerbose),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				ResetDataSetsAndMappings(t)
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(helpers.ImportSsdApi, func(device string) (*helpers.SsdInfo, error) {
					return nvmeSsdInfo, nil
				})
				patches.ApplyFunc(helpers.GetDefaultDisk, func() (string, string) {
					return "/dev/nvme0n1", "nvme"
				})
				patches.ApplyFunc(sccommon.GetPlatformJsonData, func() (map[string]interface{}, error) {
					return nil, fmt.Errorf("not found")
				})
				return patches
			},
		},
		{
			desc:       "query SHOW platform ssdhealth NVMe vendor",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "ssdhealth" key: { key: "vendor" value: "true" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedNVMeVendor),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				ResetDataSetsAndMappings(t)
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(helpers.ImportSsdApi, func(device string) (*helpers.SsdInfo, error) {
					return nvmeSsdInfo, nil
				})
				patches.ApplyFunc(helpers.GetDefaultDisk, func() (string, string) {
					return "/dev/nvme0n1", "nvme"
				})
				patches.ApplyFunc(sccommon.GetPlatformJsonData, func() (map[string]interface{}, error) {
					return nil, fmt.Errorf("not found")
				})
				return patches
			},
		},
		{
			desc:       "query SHOW platform ssdhealth NVMe verbose+vendor",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "ssdhealth" key: { key: "verbose" value: "true" } key: { key: "vendor" value: "true" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedNVMeVerboseVendor),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				ResetDataSetsAndMappings(t)
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(helpers.ImportSsdApi, func(device string) (*helpers.SsdInfo, error) {
					return nvmeSsdInfo, nil
				})
				patches.ApplyFunc(helpers.GetDefaultDisk, func() (string, string) {
					return "/dev/nvme0n1", "nvme"
				})
				patches.ApplyFunc(sccommon.GetPlatformJsonData, func() (map[string]interface{}, error) {
					return nil, fmt.Errorf("not found")
				})
				return patches
			},
		},
		{
			desc:       "query SHOW platform ssdhealth SATA disk",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "ssdhealth" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedSATADefault),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				ResetDataSetsAndMappings(t)
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(helpers.ImportSsdApi, func(device string) (*helpers.SsdInfo, error) {
					return sataSsdInfo, nil
				})
				patches.ApplyFunc(helpers.GetDefaultDisk, func() (string, string) {
					return "/dev/sda", "sata"
				})
				patches.ApplyFunc(sccommon.GetPlatformJsonData, func() (map[string]interface{}, error) {
					return nil, fmt.Errorf("not found")
				})
				return patches
			},
		},
		{
			desc:       "query SHOW platform ssdhealth SSD not detected",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "ssdhealth" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedNotDetected),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				ResetDataSetsAndMappings(t)
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(helpers.ImportSsdApi, func(device string) (*helpers.SsdInfo, error) {
					return nil, fmt.Errorf("SsdUtil import failed")
				})
				patches.ApplyFunc(helpers.GetDefaultDisk, func() (string, string) {
					return "/dev/sda", "sata"
				})
				patches.ApplyFunc(sccommon.GetPlatformJsonData, func() (map[string]interface{}, error) {
					return nil, fmt.Errorf("not found")
				})
				return patches
			},
		},
		{
			desc:       "query SHOW platform ssdhealth device from platform.json",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "ssdhealth" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedPlatformJson),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				ResetDataSetsAndMappings(t)
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(helpers.ImportSsdApi, func(device string) (*helpers.SsdInfo, error) {
					return nvmeSsdInfo, nil
				})
				patches.ApplyFunc(helpers.GetDefaultDisk, func() (string, string) {
					return "/dev/nvme0n1", "nvme"
				})
				patches.ApplyFunc(sccommon.GetPlatformJsonData, func() (map[string]interface{}, error) {
					return map[string]interface{}{
						"chassis": map[string]interface{}{
							"disk": map[string]interface{}{
								"device": "/dev/nvme0n1",
							},
						},
					}, nil
				})
				return patches
			},
		},
		{
			desc:       "query SHOW platform ssdhealth SATA verbose",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "ssdhealth" key: { key: "verbose" value: "true" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedSATAVerbose),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				ResetDataSetsAndMappings(t)
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(helpers.ImportSsdApi, func(device string) (*helpers.SsdInfo, error) {
					return sataSsdInfo, nil
				})
				patches.ApplyFunc(helpers.GetDefaultDisk, func() (string, string) {
					return "/dev/sda", "sata"
				})
				patches.ApplyFunc(sccommon.GetPlatformJsonData, func() (map[string]interface{}, error) {
					return nil, fmt.Errorf("not found")
				})
				return patches
			},
		},
		{
			desc:       "query SHOW platform ssdhealth SATA vendor (empty vendor_output omitted)",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "ssdhealth" key: { key: "vendor" value: "true" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedSATAVendor),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				ResetDataSetsAndMappings(t)
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(helpers.ImportSsdApi, func(device string) (*helpers.SsdInfo, error) {
					return sataSsdInfo, nil
				})
				patches.ApplyFunc(helpers.GetDefaultDisk, func() (string, string) {
					return "/dev/sda", "sata"
				})
				patches.ApplyFunc(sccommon.GetPlatformJsonData, func() (map[string]interface{}, error) {
					return nil, fmt.Errorf("not found")
				})
				return patches
			},
		},
		{
			desc:       "query SHOW platform ssdhealth non-numeric health and temperature",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "ssdhealth" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedNonNumeric),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				ResetDataSetsAndMappings(t)
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(helpers.ImportSsdApi, func(device string) (*helpers.SsdInfo, error) {
					return nonNumericSsdInfo, nil
				})
				patches.ApplyFunc(helpers.GetDefaultDisk, func() (string, string) {
					return "/dev/nvme0n1", "nvme"
				})
				patches.ApplyFunc(sccommon.GetPlatformJsonData, func() (map[string]interface{}, error) {
					return nil, fmt.Errorf("not found")
				})
				return patches
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

func TestGetShowPlatformSyseeprom(t *testing.T) {
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
			desc:       "query SHOW platform syseeprom - full EEPROM data",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "syseeprom" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: func() []byte {
				expected := helpers.SysEepromInfo{
					TlvInfoHeader: helpers.SysEepromHeader{
						IdString:    "TlvInfo",
						Version:     "1",
						TotalLength: "169",
					},
					TlvList: []helpers.SysEepromTlv{
						{Name: "Product Name", Code: "0x21", Length: "12", Value: "DCS-7060CX-32"},
						{Name: "Part Number", Code: "0x22", Length: "14", Value: "FP-T3048-C32-R"},
						{Name: "Serial Number", Code: "0x23", Length: "11", Value: "JPE20381234"},
						{Name: "Base MAC Address", Code: "0x24", Length: "6", Value: "00:1C:73:01:23:45"},
						{Name: "Manufacture Date", Code: "0x25", Length: "19", Value: "01/01/2024 00:00:00"},
						{Name: "Device Version", Code: "0x26", Length: "1", Value: "2"},
						{Name: "Label Revision", Code: "0x27", Length: "3", Value: "R01"},
						{Name: "Platform Name", Code: "0x28", Length: "30", Value: "x86_64-mlnx_msn3700-r0"},
						{Name: "ONIE Version", Code: "0x29", Length: "12", Value: "2024.02.01.0"},
						{Name: "MAC Addresses", Code: "0x2A", Length: "2", Value: "256"},
						{Name: "Manufacturer", Code: "0x2B", Length: "8", Value: "Mellanox"},
						{Name: "Vendor Extension", Code: "0xFD", Length: "36", Value: "0x00 0x00 0x81 0x19 0x02 0x40 0x44 0x65"},
						{Name: "Vendor Extension", Code: "0xFD", Length: "36", Value: "0x00 0x00 0x81 0x19 0x02 0x40 0x44 0x66"},
						{Name: "CRC-32", Code: "0xFE", Length: "4", Value: "0xABCDEF01"},
					},
					ChecksumValid: true,
				}
				jsonData, _ := json.Marshal(expected)
				return jsonData
			}(),
			valTest: true,
			testInit: func() *gomonkey.Patches {
				ResetDataSetsAndMappings(t)
				AddDataSet(t, StateDbNum, "../testdata/SYSEEPROM.txt")
				return gomonkey.ApplyFunc(sccommon.GetPlatform, func() string {
					return "x86_64-mlnx_msn3700-r0"
				})
			},
		},
		{
			desc:       "query SHOW platform syseeprom - not initialized",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "syseeprom" >
			`,
			wantRetCode: codes.NotFound,
			valTest:     false,
			testInit: func() *gomonkey.Patches {
				ResetDataSetsAndMappings(t)
				AddDataSet(t, StateDbNum, "../testdata/SYSEEPROM_NOT_INITIALIZED.txt")
				return gomonkey.ApplyFunc(sccommon.GetPlatform, func() string {
					return "x86_64-mlnx_msn3700-r0"
				})
			},
		},
		{
			desc:       "query SHOW platform syseeprom - no data",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "syseeprom" >
			`,
			wantRetCode: codes.NotFound,
			valTest:     false,
			testInit: func() *gomonkey.Patches {
				ResetDataSetsAndMappings(t)
				return gomonkey.ApplyFunc(sccommon.GetPlatform, func() string {
					return "x86_64-mlnx_msn3700-r0"
				})
			},
		},
		{
			desc:       "query SHOW platform syseeprom - KVM platform not supported",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "syseeprom" >
			`,
			wantRetCode: codes.NotFound,
			valTest:     false,
			testInit: func() *gomonkey.Patches {
				ResetDataSetsAndMappings(t)
				return gomonkey.ApplyFunc(sccommon.GetPlatform, func() string {
					return "x86_64-kvm_x86_64-r0"
				})
			},
		},
		{
			desc:       "query SHOW platform syseeprom - Arista platform uses nsenter fallback",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "syseeprom" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: func() []byte {
				result := map[string]string{"eeprom_raw": "SKU: DCS-7060X6-64PE-B\nSerialNumber: HBG251204WB\nMAC: d8:06:f3:5a:a9:b1\nHwRev: 11.00"}
				jsonData, _ := json.Marshal(result)
				return jsonData
			}(),
			valTest: true,
			testInit: func() *gomonkey.Patches {
				ResetDataSetsAndMappings(t)
				patches := gomonkey.ApplyFunc(sccommon.GetPlatform, func() string {
					return "x86_64-arista_7050-r0"
				})
				eepromText := "SKU: DCS-7060X6-64PE-B\nSerialNumber: HBG251204WB\nMAC: d8:06:f3:5a:a9:b1\nHwRev: 11.00"
				patches.ApplyFunc(sccommon.GetDataFromHostCommand, func(command string) (string, error) {
					return eepromText, nil
				})
				return patches
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
