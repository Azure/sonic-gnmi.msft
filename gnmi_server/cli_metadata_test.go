package gnmi

// reboot_cause_cli_test.go

// Tests SHOW reboot-cause and SHOW reboot-cause history

import (
	"crypto/tls"
	"os"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"

	"github.com/sonic-net/sonic-gnmi/metadata"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestShowClientVersionMetadata(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	defer func() {
		metadata.SetVersionTest(metadata.DefaultVersion)
		os.Setenv(metadata.VersionEnvVar, "")
		os.Setenv(metadata.EnableMetadataEnvVar, "false")
		os.Setenv(metadata.EnableVersionEnvVar, "false")
	}()

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

	portsFileName := "../testdata/PORTS.txt"
	portOidMappingFileName := "../testdata/PORT_COUNTERS_MAPPING.txt"
	portCountersFileName := "../testdata/PORT_COUNTERS.txt"
	portRatesFileName := "../testdata/PORT_RATES.txt"
	portTableFileName := "../testdata/PORT_TABLE.txt"

	showInterfaceCountersHelp := `{"options":{"display":"[display=all] No-op since no-multi-asic support","help":"[help=true]Show this message","interfaces":"[interfaces=TEXT] Filter by interfaces name","json":"[json=true] No-op since response is in json format","namespace":"UNIMPLEMENTED","period":"[period=INTEGER] Display statistics over a specified period (in seconds)","verbose":"[verbose=true] Enable verbose output"},"subcommands":null}`
	interfaceCountersSelectPorts := `{"Ethernet0":{"State":"U","RxOk":"149903","RxBps":"25.12 B/s","RxUtil":"0.00%","RxErr":"0","RxDrp":"957","RxOvr":"0","TxOk":"144782","TxBps":"773.23 KB/s","TxUtil":"0.01%","TxErr":"0","TxDrp":"2","TxOvr":"0"}}`

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
			desc:       "Default version test when specifying help, no version added",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interface" >
				elem: <name: "counters" key: { key: "help" value: "True" }>
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(showInterfaceCountersHelp),
			valTest:     true,
			testInit: func() {
				os.Setenv(metadata.EnableMetadataEnvVar, "false")
				os.Setenv(metadata.EnableVersionEnvVar, "false")
			},
		},
		{
			desc:       "Default version test when specifying help",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interface" >
				elem: <name: "counters" key: { key: "help" value: "True" }>
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(showInterfaceCountersHelp),
			valTest:     true,
			testInit: func() {
				os.Setenv(metadata.EnableMetadataEnvVar, "true")
				os.Setenv(metadata.EnableVersionEnvVar, "true")
			},
		},
		{
			desc:       "Default version test when not specifying help but other args",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interface" >
				elem: <name: "counters" 
				      key: { key: "interfaces" value: "Ethernet0" }
				      key: { key: "help" value: "false" }>
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersSelectPorts),
			valTest:     true,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, portsFileName)
				AddDataSet(t, CountersDbNum, portOidMappingFileName)
				AddDataSet(t, CountersDbNum, portCountersFileName)
				AddDataSet(t, CountersDbNum, portRatesFileName)
				AddDataSet(t, ApplDbNum, portTableFileName)
			},
		},
		{
			desc:       "Default version test when there is error",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interface" >
				elem: <name: "counters"
				      key: { key: "interfaces" value: "Ethernet0" }
				      key: { key: "period" value: "foobar" }>
			`,
			wantRetCode: codes.InvalidArgument,
		},
		{
			desc:       "Default version test when there is unimplemented",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interface" >
				elem: <name: "counters"
				      key: { key: "interfaces" value: "Ethernet0" }
				      key: { key: "period" value: "5" }
				      key: { key: "namespace" value: "all" }>
			`,
			wantRetCode: codes.Unimplemented,
		},
		{
			desc:       "Set version test",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interface" >
				elem: <name: "counters" 
				      key: { key: "interfaces" value: "Ethernet0" }
				      key: { key: "help" value: "false" }>
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersSelectPorts),
			valTest:     true,
			testInit: func() {
				metadata.SetVersionTest("1.0.0")
				AddDataSet(t, ConfigDbNum, portsFileName)
				AddDataSet(t, CountersDbNum, portOidMappingFileName)
				AddDataSet(t, CountersDbNum, portCountersFileName)
				AddDataSet(t, CountersDbNum, portRatesFileName)
				AddDataSet(t, ApplDbNum, portTableFileName)
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
