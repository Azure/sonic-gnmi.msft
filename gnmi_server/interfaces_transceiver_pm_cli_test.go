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

func TestGetTransceiverPM(t *testing.T) {
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

	ApplDbFile := "../testdata/APPL_DB.json"
	transceiverPM := `[{"name": "Ethernet0","status": "Transceiver performance monitoring not applicable"}, {"name": "Ethernet40","status": "Transceiver performance monitoring not applicable"},{"name": "Ethernet80","status": "Transceiver performance monitoring not applicable"},{"name": "Ethernet120","status": "Transceiver performance monitoring not applicable"}]`
	transceiverPMPort := `[{"name": "Ethernet0","status": "Transceiver performance monitoring not applicable"}]`
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
			desc:       "query SHOW interfaces transceiver pm",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "transceiver" >
				elem: <name: "pm" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(transceiverPM),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ApplDbNum)
				AddDataSet(t, ApplDbNum, ApplDbFile)
			},
		},
		{
			desc:       "query SHOW interfaces transceiver pm -- single interface",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "transceiver" >
				elem: <name: "pm" >
				elem: <name: "Ethernet0">
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(transceiverPMPort),
			valTest:     true,
			testInit: func() {
				AddDataSet(t, ApplDbNum, ApplDbFile)
			},
		},
		{
			desc:       "query SHOW interfaces transceiver -- non-existent interface",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "transceiver" >
				elem: <name: "pm" >
				elem: <name: "Ethernet1">
			`,
			wantRetCode: codes.NotFound,
			testInit: func() {
				AddDataSet(t, ApplDbNum, ApplDbFile)
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
