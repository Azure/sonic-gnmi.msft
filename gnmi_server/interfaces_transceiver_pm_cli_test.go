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
	transceiverPM := `[{"name": "Ethernet0","description": "Transceiver performance monitoring not applicable"}, {"name": "Ethernet40","description": "Transceiver performance monitoring not applicable"},{"name": "Ethernet80","description": "Transceiver performance monitoring not applicable"},{"name": "Ethernet120","description": "Transceiver performance monitoring not applicable"}]`
	transceiverPMPort := `[{"name": "Ethernet0","description": "Transceiver performance monitoring not applicable"}]`
	transceiverPMWithData := `[{"name": "Ethernet0","description": "Min,Avg,Max,Threshold High Alarm,Threshold High Warning,Threshold Crossing Alert-High,Threshold Low Alarm,Threshold Low Warning,Threshold Crossing Alert-Low",
	"Tx Power":        "-8.22dBm,-8.23dBm,-8.24dBm,-5.0dBm,-6.0dBm,False,-16.99dBm,-16.003dBm,False"},
	"Rx Total Power":  "-10.61dBm,-10.62dBm,-10.62dBm,2.0dBm,0.0dBm,False,-21.0dBm,-18.0dBm,False"},
	"Rx Signal Power": "-40.0dBm,0.0dBm,40.0dBm,13.0dBm,10.0dBm,True,-18.0dBm,-15.0dBm,True"},
	"CD-short link":   "0.0ps/nm,0.0ps/nm,0.0ps/nm,1000.0ps/nm,500.0ps/nm,False,-1000.0ps/nm,-500.0ps/nm,False"},
	"PDL":             "0.5dB,0.6dB,0.6dB,4.0dB,4.0dB,False,0.0dB,0.0dB,False"},
	"OSNR":            "36.5dB,36.5dB,36.5dB,99.0dB,99.0dB,False,0.0dB,0.0dB,FalsedB"},
	"eSNR":            "30.5dB,30.5dB,30.5dB,99.0dB,99.0dB,False,0.0dB,0.0dB,FalsedB"},
	"CFO":             "54.0MHz,70.0MHz,121.0MHz,3800.0MHz,3800.0MHz,False,-3800.0MHz,-3800.0MHz,False"},
	"DGD":             "5.37ps,5.56ps,5.81ps,7.0ps,7.0ps,False,0.0ps,0.0ps,False"},
	"SOPMD":           "0.0ps^2,0.0ps^2,0.0ps^2,655.35ps^2,655.35ps^2,False,0.0ps^2,0.0ps^2,False"},
	"SOP ROC":         "1.0krad/s,1.0krad/s,2.0krad/s,N/A,N/A,N/A,N/A,N/A,N/A"},
	"Pre-FEC BER":     "4.58E-04,4.66E-04,5.76E-04,1.25E-02,1.10E-02,0.0,0.0,0.0,0.0"},
	"Post-FEC BER":    "0.0,0.0,0.0,1000.0,1.0,False,0.0,0.0,False"},
	"EVM":             "100.0%,100.0%,100.0%,N/A,N/A,N/A,N/A,N/A,N/A"},{"name": "Ethernet40","description": "Transceiver performance monitoring not applicable"},{"name": "Ethernet80","description": "Transceiver performance monitoring not applicable"},{"name": "Ethernet120","description": "Transceiver performance monitoring not applicable"}]`
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
