package gnmi

import (
	"crypto/tls"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	pb "github.com/openconfig/gnmi/proto/gnmi"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"

	"github.com/sonic-net/sonic-gnmi/show_client/common"
	"github.com/sonic-net/sonic-gnmi/show_client/helpers"
)

func TestGetShowPlatformFirmwareStatus(t *testing.T) {
	// Expected output matching actual device output (MSN2700)
	expectedOutput := `{"chassis":"MSN2700","module":"N/A","components":[{"name":"ONIE","version":"2018.05-5.2.0004-9600","description":"ONIE - Open Network Install Environment"},{"name":"SSD","version":"0115-000","description":"SSD - Solid-State Drive"},{"name":"BIOS","version":"0ABZS017_02.02.002","description":"BIOS - Basic Input/Output System"},{"name":"CPLD1","version":"CPLD000085_REV2000","description":"CPLD - Complex Programmable Logic Device"},{"name":"CPLD2","version":"CPLD000128_REV0600","description":"CPLD - Complex Programmable Logic Device"},{"name":"CPLD3","version":"CPLD000000_REV0300","description":"CPLD - Complex Programmable Logic Device"}]}`

	// Expected output for modular chassis test
	expectedModularOutput := `{"chassis":"ModularChassis","module":"Module1","components":[{"name":"BIOS","version":"1.0.0","description":"System BIOS"},{"name":"FPGA","version":"2.1.0","description":"Module FPGA"},{"name":"CPLD","version":"3.0.0","description":"Module CPLD"}]}`

	expectedEmptyOutput := `{"chassis":"N/A","module":"N/A","components":[]}`

	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal interface{}
		valTest     bool
		testInit    func() func()
	}{
		{
			desc:       "query SHOW platform firmware status success",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "firmware" >
				elem: <name: "status" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedOutput),
			valTest:     true,
			testInit: func() func() {
				ResetDataSetsAndMappings(t)

				// Mock helper functions for MSN2700 non-modular chassis
				patches := gomonkey.NewPatches()

				// Mock GetAllFirmwareData to return MSN2700 components
				patches.ApplyFunc(helpers.GetAllFirmwareData, func() ([]helpers.FirmwareData, error) {
					return []helpers.FirmwareData{
						{
							Chassis:     "MSN2700",
							Module:      "N/A",
							Component:   "ONIE",
							Version:     "2018.05-5.2.0004-9600",
							Description: "ONIE - Open Network Install Environment",
						},
						{
							Chassis:     "",
							Module:      "",
							Component:   "SSD",
							Version:     "0115-000",
							Description: "SSD - Solid-State Drive",
						},
						{
							Chassis:     "",
							Module:      "",
							Component:   "BIOS",
							Version:     "0ABZS017_02.02.002",
							Description: "BIOS - Basic Input/Output System",
						},
						{
							Chassis:     "",
							Module:      "",
							Component:   "CPLD1",
							Version:     "CPLD000085_REV2000",
							Description: "CPLD - Complex Programmable Logic Device",
						},
						{
							Chassis:     "",
							Module:      "",
							Component:   "CPLD2",
							Version:     "CPLD000128_REV0600",
							Description: "CPLD - Complex Programmable Logic Device",
						},
						{
							Chassis:     "",
							Module:      "",
							Component:   "CPLD3",
							Version:     "CPLD000000_REV0300",
							Description: "CPLD - Complex Programmable Logic Device",
						},
					}, nil
				})

				return func() {
					patches.Reset()
				}
			},
		},
		{
			desc:       "query SHOW platform firmware status modular chassis",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "firmware" >
				elem: <name: "status" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedModularOutput),
			valTest:     true,
			testInit: func() func() {
				ResetDataSetsAndMappings(t)

				// Mock helper function for modular chassis
				patches := gomonkey.NewPatches()

				patches.ApplyFunc(helpers.GetAllFirmwareData, func() ([]helpers.FirmwareData, error) {
					return []helpers.FirmwareData{
						{
							Chassis:     "ModularChassis",
							Module:      "",
							Component:   "BIOS",
							Version:     "1.0.0",
							Description: "System BIOS",
						},
						{
							Chassis:     "",
							Module:      "Module1",
							Component:   "FPGA",
							Version:     "2.1.0",
							Description: "Module FPGA",
						},
						{
							Chassis:     "",
							Module:      "",
							Component:   "CPLD",
							Version:     "3.0.0",
							Description: "Module CPLD",
						},
					}, nil
				})

				return func() {
					patches.Reset()
				}
			},
		},
		{
			desc:       "query SHOW platform firmware status no components",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "firmware" >
				elem: <name: "status" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedEmptyOutput),
			valTest:     true,
			testInit: func() func() {
				ResetDataSetsAndMappings(t)

				// Mock helper function returning empty data
				patches := gomonkey.NewPatches()

				patches.ApplyFunc(helpers.GetAllFirmwareData, func() ([]helpers.FirmwareData, error) {
					return []helpers.FirmwareData{}, nil
				})

				return func() {
					patches.Reset()
				}
			},
		},
		{
			desc:       "test helper functions coverage - chassis name and components",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "firmware" >
				elem: <name: "status" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{"chassis":"TestChassis","module":"N/A","components":[{"name":"BIOS","version":"1.0.0","description":"Test BIOS"},{"name":"CPLD","version":"2.0.0","description":"Test CPLD"}]}`),
			valTest:     true,
			testInit: func() func() {
				ResetDataSetsAndMappings(t)

				// Add test data for CHASSIS_INFO to STATE_DB
				stateDbClient := getRedisClientN(t, StateDbNum, "")
				stateDbClient.HSet(context.Background(), "CHASSIS_INFO|chassis 1", "model", "TestChassis")
				stateDbClient.Close()

				// Mock individual helper functions to test integration logic
				patches := gomonkey.NewPatches()

				patches.ApplyFunc(helpers.GetChassisComponents, func() ([]helpers.ComponentInfo, error) {
					return []helpers.ComponentInfo{
						{
							Name:            "BIOS",
							FirmwareVersion: "1.0.0",
							Description:     "Test BIOS",
						},
						{
							Name:            "CPLD",
							FirmwareVersion: "2.0.0",
							Description:     "Test CPLD",
						},
					}, nil
				})

				patches.ApplyFunc(helpers.GetModuleComponents, func() ([]helpers.ModuleComponentInfo, error) {
					// Return empty for non-modular chassis
					return []helpers.ModuleComponentInfo{}, nil
				})

				return func() {
					patches.Reset()
				}
			},
		},
		{
			desc:       "test helper functions coverage - modular chassis with modules",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "firmware" >
				elem: <name: "status" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{"chassis":"Modular","module":"LineCard1","components":[{"name":"BIOS","version":"1.0","description":"Chassis BIOS"},{"name":"FPGA","version":"2.0","description":"Module FPGA"},{"name":"CPLD","version":"3.0","description":"Module CPLD"}]}`),
			valTest:     true,
			testInit: func() func() {
				ResetDataSetsAndMappings(t)

				// Add test data for CHASSIS_INFO to STATE_DB
				stateDbClient := getRedisClientN(t, StateDbNum, "")
				stateDbClient.HSet(context.Background(), "CHASSIS_INFO|chassis 1", "model", "Modular")
				stateDbClient.Close()

				// Mock helper functions for modular chassis with modules
				patches := gomonkey.NewPatches()

				patches.ApplyFunc(helpers.GetChassisComponents, func() ([]helpers.ComponentInfo, error) {
					return []helpers.ComponentInfo{
						{
							Name:            "BIOS",
							FirmwareVersion: "1.0",
							Description:     "Chassis BIOS",
						},
					}, nil
				})

				patches.ApplyFunc(helpers.GetModuleComponents, func() ([]helpers.ModuleComponentInfo, error) {
					return []helpers.ModuleComponentInfo{
						{
							ModuleName:      "LineCard1",
							Name:            "FPGA",
							FirmwareVersion: "2.0",
							Description:     "Module FPGA",
						},
						{
							ModuleName:      "LineCard1",
							Name:            "CPLD",
							FirmwareVersion: "3.0",
							Description:     "Module CPLD",
						},
					}, nil
				})

				return func() {
					patches.Reset()
				}
			},
		},
		{
			desc:       "test helper functions error handling - platform API failures",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "firmware" >
				elem: <name: "status" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedEmptyOutput),
			valTest:     true,
			testInit: func() func() {
				ResetDataSetsAndMappings(t)

				// Mock helper functions returning errors to test error handling
				patches := gomonkey.NewPatches()

				// Mock database query failure for chassis info
				patches.ApplyFunc(common.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					return nil, fmt.Errorf("database query error")
				})

				patches.ApplyFunc(helpers.GetChassisComponents, func() ([]helpers.ComponentInfo, error) {
					return nil, fmt.Errorf("chassis components error")
				})

				patches.ApplyFunc(helpers.GetModuleComponents, func() ([]helpers.ModuleComponentInfo, error) {
					return nil, fmt.Errorf("module components error")
				})

				return func() {
					patches.Reset()
				}
			},
		},
		{
			desc:       "test helper functions coverage - platform API command failures",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "firmware" >
				elem: <name: "status" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedEmptyOutput),
			valTest:     true,
			testInit: func() func() {
				ResetDataSetsAndMappings(t)

				// Mock Platform API command failures to test error handling
				patches := gomonkey.NewPatches()

				patches.ApplyFunc(common.GetDataFromHostCommand, func(command string) (string, error) {
					// All Platform API calls fail
					return "", fmt.Errorf("platform API error")
				})

				return func() {
					patches.Reset()
				}
			},
		},
		{
			desc:       "test helper functions coverage - invalid JSON response",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "firmware" >
				elem: <name: "status" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedEmptyOutput),
			valTest:     true,
			testInit: func() func() {
				ResetDataSetsAndMappings(t)

				// Mock invalid JSON responses to test parsing error handling
				patches := gomonkey.NewPatches()

				patches.ApplyFunc(common.GetDataFromHostCommand, func(command string) (string, error) {
					// Return invalid JSON to test error handling in GetChassisComponents and GetModuleComponents
					if strings.Contains(command, "json.dumps") {
						return "invalid json}", nil
					}
					// Return valid chassis name
					return "TestChassis", nil
				})

				return func() {
					patches.Reset()
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var cleanup func()
			if tt.testInit != nil {
				cleanup = tt.testInit()
			}
			defer func() {
				if cleanup != nil {
					cleanup()
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
