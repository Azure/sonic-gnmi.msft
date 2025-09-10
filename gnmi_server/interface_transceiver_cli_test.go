package gnmi

// interface_transceiver_cli_test.go

// Tests SHOW interface transceiver commands

import (
	"crypto/tls"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"

	"github.com/agiledragon/gomonkey/v2"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestGetTransceiverErrorStatus(t *testing.T) {
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

	transceiverErrorStatusFileName := "../testdata/TRANSCEIVER_STATUS_SW.txt"
	transceiverErrorStatus := `{"Ethernet90":{"cmis_state": "READY","error": "N/A","status": "1"},"Ethernet91": {"cmis_state": "READY","error": "N/A","status": "1"},"Ethernet92": {"cmis_state": "READY","error": "N/A","status": "1"},"Ethernet93": {"cmis_state": "READY","error": "N/A","status": "1"},"Ethernet94": {"cmis_state": "READY","error": "N/A","status": "1"},"Ethernet95": {"cmis_state": "READY","error": "N/A","status": "1"},"Ethernet96": {"cmis_state": "READY","error": "N/A","status": "1"},"Ethernet97": {"cmis_state": "READY","error": "N/A","status": "1"},"Ethernet98": {"cmis_state": "READY","error": "N/A","status": "1"},"Ethernet99": {"cmis_state": "READY","error": "N/A","status": "1"}}`
	transceiverErrorStatusPort := `{"cmis_state": "READY","error": "N/A","status": "1"}`
	ResetDataSetsAndMappings(t)

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
			desc:       "query SHOW interface transceiver error-status read error",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "" >
				elem: <name: "transceiver" >
				elem: <name: "error-status" >
			`,
			wantRetCode: codes.NotFound,
		},
		{
			desc:       "query SHOW interface transceiver error-status NO interface dataset",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interface" >
				elem: <name: "transceiver" >
				elem: <name: "error-status" >
			`,
			wantRetCode: codes.OK,
		},
		{
			desc:       "query SHOW interface transceiver error-status",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interface" >
				elem: <name: "transceiver" >
				elem: <name: "error-status" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(transceiverErrorStatus),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, StateDbNum)
				AddDataSet(t, StateDbNum, transceiverErrorStatusFileName)
			},
		},
		{
			desc:       "query SHOW interface transceiver error-status port option",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interface" >
				elem: <name: "transceiver" >
				elem: <name: "error-status" key: { key: "interface" value: "Ethernet90" }>
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(transceiverErrorStatusPort),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, StateDbNum)
				AddDataSet(t, StateDbNum, transceiverErrorStatusFileName)
			},
		},
	}

	for _, test := range tests {
		if test.testInit != nil {
			test.testInit()
		}

		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})

	}
}

func TestGetInterfaceTransceiverLpmode(t *testing.T) {
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

	lpmodeDefaultFile := "../testdata/LPMODE_DEFAULT.txt"
	lpmodeErrorFile := "../testdata/LPMODE_ERROR.txt"

	FlushDataSet(t, StateDbNum)
	AddDataSet(t, StateDbNum, "../testdata/CONFIG_DB_PORT.txt")

	tests := []struct {
		desc          string
		pathTarget    string
		textPbPath    string
		wantRetCode   codes.Code
		wantRespVal   interface{}
		valTest       bool
		mockOutputFile string
		testInit      func()
	}{
		{
			desc:       "lpmode all ports success",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interface" >
				elem: <name: "transceiver" >
				elem: <name: "lpmode" >
			`,
			wantRetCode: codes.OK,
			mockOutputFile: lpmodeDefaultFile,
			valTest:    false,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, configDbPortFile)
			},
		},
		{
			desc:       "lpmode single valid interface (returns all ports)",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interface" >
				elem: <name: "transceiver" >
				elem: <name: "lpmode" key: { key: "interface" value: "Ethernet80" }>
			`,
			wantRetCode: codes.OK,
			mockOutputFile: lpmodeDefaultFile,
			valTest:    false,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, configDbPortFile)
			},
		},
		{
			desc:       "lpmode invalid interface option",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interface" >
				elem: <name: "transceiver" >
				elem: <name: "lpmode" key: { key: "interface" value: "InvalidPort" }>
			`,
			wantRetCode: codes.NotFound,
			mockOutputFile: lpmodeErrorFile,
			valTest:    false,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, configDbPortFile)
			},
		},
	}

	for _, tc := range tests {
		if tc.testInit != nil {
			tc.testInit()
		}
		var patches *gomonkey.Patches
		if tc.mockOutputFile != "" {
			patches = MockNSEnterOutput(t, tc.mockOutputFile)
		}
		t.Run(tc.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, tc.pathTarget, tc.textPbPath, tc.wantRetCode, tc.wantRespVal, tc.valTest)
		})
		if patches != nil {
			patches.Reset()
		}
	}
}