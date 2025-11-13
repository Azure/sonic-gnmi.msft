package gnmi

import (
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"
	sccommon "github.com/sonic-net/sonic-gnmi/show_client/common"

	"github.com/agiledragon/gomonkey/v2"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestGetEnvironment(t *testing.T) {
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

	expectedSensorOutput := `
	{
		"devices": [
		{
			"device": "pmbus-i2c-5-27",
			"adapter": "i2c-1-mux (chan_id 5)",
			"readings": [
			{
				"label": "PMB-2 PSU 12V Rail (in)",
				"value": "11.89",
				"unit": "V",
				"thresholds": {
					"crit min": "+9.00 V",
					"min": "+10.00 V",
					"max": "+15.00 V",
					"crit max": "+16.00 V"
				}
			},
			{
				"label": "PMB-2 3.3V Rail (out)",
				"value": "3.32",
				"unit": "V",
				"thresholds": {
					"crit min": "+2.97 V",
					"min": "+3.04 V",
					"max": "+3.56 V",
					"crit max": "+3.63 V"
				}
			},
			{
				"label": "PMB-2 1.2V Rail (out)",
				"value": "1.21",
				"unit": "V",
				"thresholds": {
					"crit min": "+1.08 V",
					"min": "+1.10 V",
					"max": "+1.29 V",
					"crit max": "+1.32 V"
				}
			},
			{
				"label": "PMB-2 Temp 1",
				"value": "+25.5",
				"unit": "C",
				"thresholds": {
					"high": "+80.0 C",
					"crit": "+90.0 C"
				}
			},
			{
				"label": "PMB-2 Temp 2",
				"value": "+25.5",
				"unit": "C",
				"thresholds": {
					"high": "+80.0 C",
					"crit": "+90.0 C"
				}
			},
			{
				"label": "PMB-2 3.3V Rail Pwr (out)",
				"value": "5.44",
				"unit": "W",
				"thresholds": {}
			},
			{
				"label": "PMB-2 1.2V Rail Pwr (out)",
				"value": "25.19",
				"unit": "W",
				"thresholds": {}
			},
			{
				"label": "PMB-2 3.3V Rail Curr (out)",
				"value": "843.00",
				"unit": "mA",
				"thresholds": {
					"crit min": "-13.53 A",
					"max": "+43.06 A",
					"crit max": "+53.31 A"
				}
			},
			{
				"label": "PMB-2 1.2V Rail Curr (out)",
				"value": "10.83",
				"unit": "A",
				"thresholds": {
					"crit min": "-14.84 A",
					"max": "+47.25 A",
					"crit max": "+58.50 A"
				}
			}
			]
		}
		],
		"total_devices": 1
	}
	`

	expectedSensorOutputWithError := `
	{
		"devices": [
		{
			"device": "ucd90160a-i2c-58-34",
			"adapter": "i2c-41-mux (chan_id 2)",
			"readings": [
			{
				"label": "vout1",
				"value": "831.00",
				"unit": "mV",
				"thresholds": {
					"crit min": "+0.77 V",
					"min": "+0.78 V",
					"max": "+0.91 V",
					"crit max": "+0.95 V"
				}
			},
			{
				"label": "vout2",
				"value": "1.15",
				"unit": "V",
				"thresholds": {
					"crit min": "+1.06 V",
					"min": "+1.09 V",
					"max": "+1.21 V",
					"crit max": "+1.24 V"
				}
			},
			{
				"label": "vout3",
				"value": "753.00",
				"unit": "mV",
				"thresholds": {
					"crit min": "+0.68 V",
					"min": "+0.70 V",
					"max": "+0.80 V",
					"crit max": "+0.82 V"
				}
			},
			{
				"label": "vout4",
				"value": "962.00",
				"unit": "mV",
				"thresholds": {
					"crit min": "+0.89 V",
					"min": "+0.91 V",
					"max": "+1.01 V",
					"crit max": "+1.03 V"
				}
			},
			{
				"label": "vout5",
				"value": "1.81",
				"unit": "V",
				"thresholds": {
					"crit min": "+1.64 V",
					"min": "+1.67 V",
					"max": "+1.93 V",
					"crit max": "+1.96 V"
				}
			},
			{
				"label": "vout6",
				"value": "1.81",
				"unit": "V",
				"thresholds": {
					"crit min": "+1.64 V",
					"min": "+1.67 V",
					"max": "+1.93 V",
					"crit max": "+1.96 V"
				}
			},
			{
				"label": "vout7",
				"value": "1.20",
				"unit": "V",
				"thresholds": {
					"crit min": "+1.09 V",
					"min": "+1.12 V",
					"max": "+1.28 V",
					"crit max": "+1.31 V"
				}
			},
			{
				"label": "vout8",
				"value": "760.00",
				"unit": "mV",
				"thresholds": {
					"crit min": "+0.68 V",
					"min": "+0.70 V",
					"max": "+0.80 V",
					"crit max": "+0.82 V"
				}
			},
			{
				"label": "vout10",
				"value": "126.00",
				"unit": "mV",
				"thresholds": {
					"crit min": "+2.64 V",
					"min": "+2.80 V",
					"max": "+3.53 V",
					"crit max": "+3.60 V"
				}
			},
			{
				"label": "vout11",
				"value": "3.32",
				"unit": "V",
				"thresholds": {
					"crit min": "+3.00 V",
					"min": "+3.07 V",
					"max": "+3.53 V",
					"crit max": "+3.60 V"
				}
			},
			{
				"label": "vout12",
				"value": "3.35",
				"unit": "V",
				"thresholds": {
					"crit min": "+3.04 V",
					"min": "+3.10 V",
					"max": "+3.50 V",
					"crit max": "+3.56 V"
				}
			},
			{
				"label": "vout13",
				"value": "3.35",
				"unit": "V",
				"thresholds": {
					"crit min": "+3.04 V",
					"min": "+3.10 V",
					"max": "+3.50 V",
					"crit max": "+3.56 V"
				}
			},
			{
				"label": "vout14",
				"value": "3.28",
				"unit": "V",
				"thresholds": {
					"crit min": "+3.00 V",
					"min": "+3.07 V",
					"max": "+3.53 V",
					"crit max": "+3.60 V"
				}
			},
			{
				"label": "vout15",
				"value": "5.02",
				"unit": "V",
				"thresholds": {
					"crit min": "+4.55 V",
					"min": "+4.65 V",
					"max": "+5.35 V",
					"crit max": "+5.45 V"
				}
			},
			{
				"label": "vout16",
				"value": "11.90",
				"unit": "V",
				"thresholds": {
					"crit min": "+10.92 V",
					"min": "+11.16 V",
					"max": "+12.84 V",
					"crit max": "+13.08 V"
				}
			},
			{
				"label": "temp1",
				"value": null,
				"unit": "N/A",
				"thresholds": {}
			}
			]
		},
		{
			"device": "jc42-i2c-1-18",
			"adapter": "iMC socket 0 for channel pair 0-1",
			"readings": []
		}
		],
		"total_devices": 2
	}	
	`

	ResetDataSetsAndMappings(t)

	tests := []struct {
		desc           string
		pathTarget     string
		textPbPath     string
		wantRetCode    codes.Code
		wantRespVal    interface{}
		valTest        bool
		mockOutputFile string
		testInit       func() *gomonkey.Patches
	}{
		{
			desc:       "query show environment",
			pathTarget: "SHOW",
			textPbPath: `
			elem: <name: "environment" >
			`,
			wantRetCode:    codes.OK,
			wantRespVal:    []byte(expectedSensorOutput),
			valTest:        true,
			mockOutputFile: "../testdata/environment_data.txt",
		},
		{
			desc:       "query show environmenti with ERROR data",
			pathTarget: "SHOW",
			textPbPath: `
			elem: <name: "environment" >
			`,
			wantRetCode:    codes.OK,
			wantRespVal:    []byte(expectedSensorOutputWithError),
			valTest:        true,
			mockOutputFile: "../testdata/environment_data_with_error.txt",
		},
		{
			desc:       "query show environment with blank output",
			pathTarget: "SHOW",
			textPbPath: `
			elem: <name: "environment" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() *gomonkey.Patches {
				return gomonkey.ApplyFunc(sccommon.GetDataFromHostCommand, func(cmd string) (string, error) {
					return "", nil
				})
			},
		},
		{
			desc:       "query show environment with error from command",
			pathTarget: "SHOW",
			textPbPath: `
			elem: <name: "environment" >
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
	}

	for _, test := range tests {
		var patch1, patch2 *gomonkey.Patches
		if test.testInit != nil {
			patch1 = test.testInit()
		}

		if len(test.mockOutputFile) > 0 {
			patch2 = MockNSEnterOutput(t, test.mockOutputFile)
		}

		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})

		if patch1 != nil {
			patch1.Reset()
		}
		if patch2 != nil {
			patch2.Reset()
		}
	}
}
