package gnmi

// interface_transceiver_cli_test.go

// Tests SHOW interface transceiver commands

import (
	"crypto/tls"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"

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

	// send CONFIG_DB sample data
	portDbDataFilename := "../testdata/CONFIG_DB_PORT.txt"
	portLpmodeDefault := "../testdata/LPMODE_DEFAULT.txt"
	portLpmodeError := "../testdata/LPMODE_ERROR.txt"
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
			desc:       "query SHOW interface transceiver lpmode no interface option",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interface" >
				elem: <name: "transceiver" >
				elem: <name: "lpmode" >
			`,
			wantRetCode: codes.OK,
			mockOutputFile: portLpmodeDefault,
			valTest:     true,
			wantRespVal: `{"Ethernet0":"Off","Ethernet2":"Off","Ethernet40":"Off","Ethernet80":"On","Ethernet120":"Off"}`,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, portDbDataFilename)
			},
		},
		{
			desc: 		"query SHOW interface transceiver lpmode with valid interface option",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interface" >
				elem: <name: "transceiver" >
				elem: <name: "lpmode" key: { key: "interface" value: "Ethernet80" }>
			`,
			wantRetCode: codes.OK,
			mockOutputFile: "Port         Low-power Mode\n-----------  ----------------\nEthernet80   On\n",
			valTest:     true,
			wantRespVal: `{"Ethernet80":"On"}`,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, portDbDataFilename)
			},
		},
		{
			desc:       "query SHOW interface transceiver lpmode with valid interface but no lpmode option",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interface" >
				elem: <name: "transceiver" >
				elem: <name: "lpmode" key: { key: "interface" value: "Ethernet120" }>
			`,
			wantRetCode: codes.OK,
			mockOutputFile: "Port         Low-power Mode\n-----------  ----------------\n",
			valTest:     true,
			wantRespVal: `{"Ethernet120":"N/A"}`,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, portDbDataFilename)
			},
		},
		{
			desc:       "query SHOW interface transceiver lpmode with invalid interface option",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interface" >
				elem: <name: "transceiver" >
				elem: <name: "lpmode" key: { key: "interface" value: "InvalidPort" }>
			`,
			wantRetCode: codes.NotFound,
			mockOutputFile: portLpmodeError,
			valTest:     false,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, portDbDataFilename)
			},
		},
	}

	for _, test := range tests {
		if test.testInit != nil {
			test.testInit()
		}
		var patches *gomonkey.Patches
		if test.mockOutputFile != "" {
			patches = MockNSEnterOutput(t, test.mockOutputFile)
		}

		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})
		if patches != nil {
			patches.Reset()
		}
	}
}