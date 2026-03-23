package gnmi

// Tests for SHOW asic-sdk-health-event suppress-configuration and received

import (
	"crypto/tls"
	"encoding/json"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestShowAsicSdkHealthEventSuppressConfiguration(t *testing.T) {
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

	switchCapSupportedFile := "../testdata/SWITCH_CAPABILITY_HEALTH_EVENT_SUPPORTED.txt"
	switchCapUnsupportedFile := "../testdata/SWITCH_CAPABILITY_HEALTH_EVENT_UNSUPPORTED.txt"
	suppressConfigFile := "../testdata/SUPPRESS_ASIC_SDK_HEALTH_EVENT_CONFIG.txt"
	emptyFile := "../testdata/EMPTY_JSON.txt"

	// Expected output when suppress config data is present
	wantSuppressConfig := map[string]interface{}{
		"suppress_configuration": []interface{}{
			map[string]interface{}{
				"severity":   "fatal",
				"categories": "software",
				"max_events": "0",
			},
			map[string]interface{}{
				"severity":   "notice",
				"categories": "none",
				"max_events": "1024",
			},
			map[string]interface{}{
				"severity":   "warning",
				"categories": "firmware,asic_hw",
				"max_events": "10240",
			},
		},
	}
	wantSuppressConfigBytes, _ := json.Marshal(wantSuppressConfig)

	// Expected output when suppress config table is empty
	wantEmptySuppressConfig := map[string]interface{}{
		"suppress_configuration": []interface{}{},
	}
	wantEmptySuppressConfigBytes, _ := json.Marshal(wantEmptySuppressConfig)

	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal []byte
		valTest     bool
		testInit    func()
	}{
		{
			desc:       "suppress-configuration with data",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "asic-sdk-health-event" >
				elem: <name: "suppress-configuration" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: wantSuppressConfigBytes,
			valTest:     true,
			testInit: func() {
				AddDataSet(t, StateDbNum, switchCapSupportedFile)
				AddDataSet(t, ConfigDbNum, suppressConfigFile)
			},
		},
		{
			desc:       "suppress-configuration with empty config table",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "asic-sdk-health-event" >
				elem: <name: "suppress-configuration" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: wantEmptySuppressConfigBytes,
			valTest:     true,
			testInit: func() {
				AddDataSet(t, StateDbNum, switchCapSupportedFile)
				AddDataSet(t, ConfigDbNum, emptyFile)
			},
		},
		{
			desc:       "suppress-configuration unsupported platform",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "asic-sdk-health-event" >
				elem: <name: "suppress-configuration" >
			`,
			wantRetCode: codes.NotFound,
			valTest:     false,
			testInit: func() {
				AddDataSet(t, StateDbNum, switchCapUnsupportedFile)
			},
		},
		{
			desc:       "suppress-configuration no capability data",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "asic-sdk-health-event" >
				elem: <name: "suppress-configuration" >
			`,
			wantRetCode: codes.NotFound,
			valTest:     false,
			testInit: func() {
				AddDataSet(t, StateDbNum, emptyFile)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ResetDataSetsAndMappings(t)
			if test.testInit != nil {
				test.testInit()
			}
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})
	}
}

func TestShowAsicSdkHealthEventReceived(t *testing.T) {
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

	switchCapSupportedFile := "../testdata/SWITCH_CAPABILITY_HEALTH_EVENT_SUPPORTED.txt"
	switchCapUnsupportedFile := "../testdata/SWITCH_CAPABILITY_HEALTH_EVENT_UNSUPPORTED.txt"
	eventTableFile := "../testdata/ASIC_SDK_HEALTH_EVENT_TABLE_DATA.txt"
	emptyFile := "../testdata/EMPTY_JSON.txt"

	// Expected output when events are present
	wantEvents := map[string]interface{}{
		"events": []interface{}{
			map[string]interface{}{
				"date":        "2023-11-22 09:18:12",
				"severity":    "fatal",
				"category":    "firmware",
				"description": "ASIC SDK health event occurred",
			},
			map[string]interface{}{
				"date":        "2023-11-23 10:30:00",
				"severity":    "warning",
				"category":    "software",
				"description": "SDK warning event detected",
			},
			map[string]interface{}{
				"date":        "2023-11-24 15:45:30",
				"severity":    "notice",
				"category":    "cpu_hw",
				"description": "CPU hardware notice",
			},
		},
	}
	wantEventsBytes, _ := json.Marshal(wantEvents)

	// Expected output when no events present
	wantEmptyEvents := map[string]interface{}{
		"events": []interface{}{},
	}
	wantEmptyEventsBytes, _ := json.Marshal(wantEmptyEvents)

	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal []byte
		valTest     bool
		testInit    func()
	}{
		{
			desc:       "received events with data",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "asic-sdk-health-event" >
				elem: <name: "received" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: wantEventsBytes,
			valTest:     true,
			testInit: func() {
				AddDataSet(t, StateDbNum, switchCapSupportedFile)
				AddDataSet(t, StateDbNum, eventTableFile)
			},
		},
		{
			desc:       "received events with no events",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "asic-sdk-health-event" >
				elem: <name: "received" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: wantEmptyEventsBytes,
			valTest:     true,
			testInit: func() {
				AddDataSet(t, StateDbNum, switchCapSupportedFile)
			},
		},
		{
			desc:       "received events unsupported platform",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "asic-sdk-health-event" >
				elem: <name: "received" >
			`,
			wantRetCode: codes.NotFound,
			valTest:     false,
			testInit: func() {
				AddDataSet(t, StateDbNum, switchCapUnsupportedFile)
			},
		},
		{
			desc:       "received events no capability data",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "asic-sdk-health-event" >
				elem: <name: "received" >
			`,
			wantRetCode: codes.NotFound,
			valTest:     false,
			testInit: func() {
				AddDataSet(t, StateDbNum, emptyFile)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ResetDataSetsAndMappings(t)
			if test.testInit != nil {
				test.testInit()
			}
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})
	}
}
