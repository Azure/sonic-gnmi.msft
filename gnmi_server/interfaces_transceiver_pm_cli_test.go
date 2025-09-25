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
	ConfigDbFile := "../testdata/CONFIG_DB.json"
	StateDbFile := "../testdata/STATE_DB.json"
	transceiverPM := `[{"interface": "Ethernet0","description": "Transceiver performance monitoring not applicable"}, {"interface": "Ethernet40","description": "Transceiver performance monitoring not applicable"},{"interface": "Ethernet80","description": "Transceiver performance monitoring not applicable"},{"interface": "Ethernet120","description": "Transceiver performance monitoring not applicable"}]`
	transceiverPMPort := `[{"interface": "Ethernet0","description": "Transceiver performance monitoring not applicable"}]`
	transceiverPMNonExistPort := `[{"interface": "Ethernet1","description": "Transceiver performance monitoring not applicable"}]`
	transceiverPMWithData := `[{"interface": "Ethernet0","description": "Min,Avg,Max,Threshold High Alarm,Threshold High Warning,Threshold Crossing Alert-High,Threshold Low Alarm,Threshold Low Warning,Threshold Crossing Alert-Low",
	"Tx Power":        "-8.22dBm,-8.23dBm,-8.24dBm,-5dBm,-6dBm,false,-16.99dBm,-16.003dBm,false",
	"Rx Total Power":  "-10.61dBm,-10.62dBm,-10.62dBm,2dBm,0dBm,false,-21dBm,-18dBm,false",
	"Rx Signal Power": "-40dBm,0dBm,40dBm,13dBm,10dBm,true,-18dBm,-15dBm,true",
	"CD-short link":   "0ps/nm,0ps/nm,0ps/nm,1000ps/nm,500ps/nm,false,-1000ps/nm,-500ps/nm,false",
	"PDL":             "0.5dB,0.6dB,0.6dB,4dB,4dB,false,0dB,0dB,false",
	"OSNR":            "36.5dB,36.5dB,36.5dB,99dB,99dB,false,0dB,0dB,false",
	"eSNR":            "30.5dB,30.5dB,30.5dB,99dB,99dB,false,0dB,0dB,false",
	"CFO":             "54MHz,70MHz,121MHz,3800MHz,3800MHz,false,-3800MHz,-3800MHz,false",
	"DGD":             "5.37ps,5.56ps,5.81ps,7ps,7ps,false,0ps,0ps,false",
	"SOPMD":           "0ps^2,0ps^2,0ps^2,655.35ps^2,655.35ps^2,false,0ps^2,0ps^2,false",
	"SOP ROC":         "1krad/s,1krad/s,2krad/s,N/A,N/A,N/A,N/A,N/A,N/A",
	"Pre-FEC BER":     "4.58E-04,4.66E-04,5.76E-04,1.25E-02,1.10E-02,false,0,0,false",
	"Post-FEC BER":    "0,0,0,1000,1,false,0,0,false",
	"EVM":             "100%,100%,100%,N/A,N/A,N/A,N/A,N/A,N/A"},{"interface": "Ethernet40","description": "Transceiver performance monitoring not applicable"},{"interface": "Ethernet80","description": "Transceiver performance monitoring not applicable"},{"interface": "Ethernet120","description": "Transceiver performance monitoring not applicable"}]`
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
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, ConfigDbFile)
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
				AddDataSet(t, ConfigDbNum, ConfigDbFile)
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
			wantRetCode: codes.OK,
			wantRespVal: []byte(transceiverPMNonExistPort),
			valTest:     true,
			testInit: func() {
				AddDataSet(t, ApplDbNum, ApplDbFile)
				AddDataSet(t, ConfigDbNum, ConfigDbFile)
			},
		},
		{
			desc:       "query SHOW interfaces transceiver pm -- with PM data",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "transceiver" >
				elem: <name: "pm" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(transceiverPMWithData),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ApplDbNum)
				FlushDataSet(t, ConfigDbNum)
				FlushDataSet(t, StateDbNum)
				AddDataSet(t, ApplDbNum, ApplDbFile)
				AddDataSet(t, ConfigDbNum, ConfigDbFile)
				AddDataSet(t, StateDbNum, StateDbFile)
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
