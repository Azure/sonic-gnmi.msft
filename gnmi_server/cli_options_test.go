package gnmi

// reboot_cause_cli_test.go

// Tests SHOW reboot-cause and SHOW reboot-cause history

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

func TestShowClientOptions(t *testing.T) {
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

	portsFileName := "../testdata/PORTS.txt"
	portOidMappingFileName := "../testdata/PORT_COUNTERS_MAPPING.txt"
	portCountersFileName := "../testdata/PORT_COUNTERS.txt"
	portRatesFileName := "../testdata/PORT_RATES.txt"
	portTableFileName := "../testdata/PORT_TABLE.txt"

	showInterfaceCountersHelp := `{"options":{"display":"[display=all] No-op since no-multi-asic support","help":"[help=true]Show this message","interfaces":"[interfaces=TEXT] Filter by interfaces name","json":"[json=true] No-op since response is in json format","namespace":"UNIMPLEMENTED","period":"[period=INTEGER] Display statistics over a specified period (in seconds)","verbose":"[verbose=true] Enable verbose output"},"subcommands":{"detailed":"show/interfaces/counters/detailed: Show interface counters detailed","errors":"show/interfaces/counters/errors: Show interface counters errors","fec-histogram":"show/interfaces/counters/fec-histogram: Show interface counters fec-histogram","fec-stats":"show/interfaces/counters/fec-stats: Show interface counters rates","rates":"show/interfaces/counters/rates: Show interface counters rates","rif":"show/interfaces/counters/rif: Show interface counters rif","trim":"show/interfaces/counters/trim: Show interface counters trim"},"usage":{"desc":"SHOW/interfaces/counters[OPTIONS]: Show interface counters"}}`
	interfaceCountersSelectPorts := `{"Ethernet0":{"State":"U","RxOk":"149903","RxBps":"25.12 B/s","RxUtil":"0.00%","RxErr":"0","RxDrp":"957","RxOvr":"0","TxOk":"144782","TxBps":"773.23 KB/s","TxUtil":"0.01%","TxErr":"0","TxDrp":"2","TxOvr":"0"}}`
	interfaceCountersAll := `{"Ethernet0":{"State":"U","RxOk":"149903","RxBps":"25.12 B/s","RxUtil":"0.00%","RxErr":"0","RxDrp":"957","RxOvr":"0","TxOk":"144782","TxBps":"773.23 KB/s","TxUtil":"0.01%","TxErr":"0","TxDrp":"2","TxOvr":"0"},"Ethernet40":{"State":"U","RxOk":"7295","RxBps":"0.00 B/s","RxUtil":"0.00%","RxErr":"0","RxDrp":"0","RxOvr":"0","TxOk":"50184","TxBps":"633.66 KB/s","TxUtil":"0.01%","TxErr":"0","TxDrp":"1","TxOvr":"0"},"Ethernet80":{"State":"U","RxOk":"76555","RxBps":"0.37 B/s","RxUtil":"0.00%","RxErr":"0","RxDrp":"0","RxOvr":"0","TxOk":"144767","TxBps":"631.94 KB/s","TxUtil":"0.01%","TxErr":"0","TxDrp":"1","TxOvr":"0"}}`
	intfErrorsEmpty := `[{"Port Errors": "oper error status","Count": "0","Last timestamp(UTC)": "Never"},{"Port Errors": "mac local fault","Count": "0","Last timestamp(UTC)": "Never"},{"Port Errors": "mac remote fault","Count": "0","Last timestamp(UTC)": "Never"},{"Port Errors": "fec sync loss","Count": "0","Last timestamp(UTC)": "Never"},{"Port Errors": "fec alignment loss","Count": "0","Last timestamp(UTC)": "Never"},{"Port Errors": "high ser error","Count": "0","Last timestamp(UTC)": "Never"},{"Port Errors": "high ber error","Count": "0","Last timestamp(UTC)": "Never"},{"Port Errors": "data unit crc error","Count": "0","Last timestamp(UTC)": "Never"},{"Port Errors": "data unit misalignment error","Count": "0","Last timestamp(UTC)": "Never"},{"Port Errors": "signal local error","Count": "0","Last timestamp(UTC)": "Never"},{"Port Errors": "crc rate","Count": "0","Last timestamp(UTC)": "Never"},{"Port Errors": "data unit size","Count": "0","Last timestamp(UTC)": "Never"},{"Port Errors": "code group error","Count": "0","Last timestamp(UTC)": "Never"},{"Port Errors": "no rx reachability","Count": "0","Last timestamp(UTC)": "Never"}]`

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
			desc:       "query SHOW interfaces counters[help=True]",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" key: { key: "help" value: "True" }>
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(showInterfaceCountersHelp),
			valTest:     true,
		},
		{
			desc:       "query SHOW interfaces counters[interfaces=Ethernet0][help=False]",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
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
			desc:       "query SHOW interfaces[help=True] counters[interfaces=Ethernet0]",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces"
				      key: { key: "help" value: "true" }>
				elem: <name: "counters" 
				      key: { key: "interfaces" value: "Ethernet0" }>
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersSelectPorts),
			valTest:     true,
		},
		{
			desc:       "query SHOW interfaces[dummy=test] counters[interfaces=Ethernet0]",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces"
				      key: { key: "dummy" value: "test" }>
				elem: <name: "counters" 
				      key: { key: "interfaces" value: "Ethernet0" }>
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersSelectPorts),
			valTest:     true,
		},
		{
			desc:       "query SHOW interfaces[interfaces=Ethernet0] counters",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces"
				      key: { key: "interfaces" value: "Ethernet0" }>
				elem: <name: "counters" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersAll),
			valTest:     true,
		},
		{
			desc:       "query SHOW interfaces errors[dummy=test] Ethernet0",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "errors" 
				      key: { key: "dummy" value: "test" }>
				elem: <name: "Ethernet0" >
			`,
			wantRetCode: codes.InvalidArgument,
		},
		{
			desc:       "query SHOW interfaces[dummy=test] errors Ethernet0",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces"
				      key: { key: "dummy" value: "test" }>
				elem: <name: "errors" >
				elem: <name: "Ethernet0" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(intfErrorsEmpty),
			valTest:     true,
		},
		{
			desc:       "query SHOW interfaces counters[interfaces-Ethernet0][period=foobar]",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters"
				      key: { key: "interfaces" value: "Ethernet0" }
				      key: { key: "period" value: "foobar" }>
			`,
			wantRetCode: codes.InvalidArgument,
		},

		{
			desc:       "query SHOW interfaces counters[interfaces-Ethernet0][period=5][foo=bar]",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters"
				      key: { key: "interfaces" value: "Ethernet0" }
				      key: { key: "period" value: "5" }
				      key: { key: "foo" value: "bar" }>
			`,
			wantRetCode: codes.InvalidArgument,
		},
		{
			desc:       "query SHOW interfaces errors missing interface",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "errors">
			`,
			wantRetCode: codes.InvalidArgument,
		},
		{
			desc:       "query SHOW interfaces counters[interfaces-Ethernet0][period=5][namespace=all]",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters"
				      key: { key: "interfaces" value: "Ethernet0" }
				      key: { key: "period" value: "5" }
				      key: { key: "namespace" value: "all" }>
			`,
			wantRetCode: codes.Unimplemented,
		},
		{
			desc:       "query SHOW ipv6 bgp neighbors[info_type] - wrong info_type",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "bgp" >
				elem: <name: "neighbors"
				      key: { key: "info_type" value: "unknown" } key: {key: "ipaddress" value: "fc00::72"}>
			`,
			wantRetCode: codes.InvalidArgument,
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
