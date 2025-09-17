package gnmi

// interface_cli_test.go

// Tests SHOW interface/counters

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

func TestGetInterfaceCounters(t *testing.T) {
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
	portCountersTwoFileName := "../testdata/PORT_COUNTERS_TWO.txt"
	portRatesFileName := "../testdata/PORT_RATES.txt"
	portRatesTwoFileName := "../testdata/PORT_RATES_TWO.txt"
	portTableFileName := "../testdata/PORT_TABLE.txt"
	interfaceCountersAll := `{"Ethernet0":{"State":"U","RxOk":"149903","RxBps":"25.12 B/s","RxUtil":"0.00%","RxErr":"0","RxDrp":"957","RxOvr":"0","TxOk":"144782","TxBps":"773.23 KB/s","TxUtil":"0.01%","TxErr":"0","TxDrp":"2","TxOvr":"0"},"Ethernet40":{"State":"U","RxOk":"7295","RxBps":"0.00 B/s","RxUtil":"0.00%","RxErr":"0","RxDrp":"0","RxOvr":"0","TxOk":"50184","TxBps":"633.66 KB/s","TxUtil":"0.01%","TxErr":"0","TxDrp":"1","TxOvr":"0"},"Ethernet80":{"State":"U","RxOk":"76555","RxBps":"0.37 B/s","RxUtil":"0.00%","RxErr":"0","RxDrp":"0","RxOvr":"0","TxOk":"144767","TxBps":"631.94 KB/s","TxUtil":"0.01%","TxErr":"0","TxDrp":"1","TxOvr":"0"}}`
	interfaceCountersSelectPorts := `{"Ethernet0":{"State":"U","RxOk":"149903","RxBps":"25.12 B/s","RxUtil":"0.00%","RxErr":"0","RxDrp":"957","RxOvr":"0","TxOk":"144782","TxBps":"773.23 KB/s","TxUtil":"0.01%","TxErr":"0","TxDrp":"2","TxOvr":"0"}}`
	interfaceCountersDiff := `{"Ethernet0":{"State":"U","RxOk":"11658","RxBps":"21.39 B/s","RxUtil":"0.00%","RxErr":"0","RxDrp":"76","RxOvr":"0","TxOk":"11270","TxBps":"634.00 KB/s","TxUtil":"0.01%","TxErr":"0","TxDrp":"0","TxOvr":"0"}}`

	ResetDataSetsAndMappings(t)

	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal interface{}
		valTest     bool
		mockSleep   bool
		testInit    func()
	}{
		{
			desc:       "query SHOW interfaces counters NO DATA",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" >
			`,
			wantRetCode: codes.OK,
		},
		{
			desc:       "query SHOW interfaces counters",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersAll),
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
			desc:       "query SHOW interfaces counters interfaces option",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" key: { key: "interfaces" value: "Ethernet0" }>
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersSelectPorts),
			valTest:     true,
		},
		{
			desc:       "query SHOW interfaces counters period option",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters"
				      key: { key: "interfaces" value: "Ethernet0" }
				      key: { key: "period" value: "5" }>
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersDiff),
			valTest:     true,
			mockSleep:   true,
		},
	}

	for _, test := range tests {
		if test.testInit != nil {
			test.testInit()
		}
		var patches *gomonkey.Patches
		if test.mockSleep {
			patches = gomonkey.ApplyFunc(time.Sleep, func(d time.Duration) {
				AddDataSet(t, CountersDbNum, portCountersTwoFileName)
				AddDataSet(t, CountersDbNum, portRatesTwoFileName)
			})
		}

		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})
		if patches != nil {
			patches.Reset()
		}
	}
}

func TestGetInterfaceRifCounters(t *testing.T) {
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

	FlushDataSet(t, CountersDbNum)
	interfacesCountersRifTestData := "../testdata/InterfacesCountersRifTestData.txt"
	AddDataSet(t, CountersDbNum, interfacesCountersRifTestData)

	t.Run("query SHOW interfaces counters rif", func(t *testing.T) {
		textPbPath := `
			elem: <name: "interfaces" >
			elem: <name: "counters" >
			elem: <name: "rif" >
		`
		wantRespVal := []byte(`{
  "PortChannel101": {
    "RxBps": "4214812716.943851",
    "RxErrBits": "17866494",
    "RxErrPackets": "172078",
    "RxOkPackets": "43864767060035",
    "RxOkBits": "4561966927266923",
    "RxPps": "40527122.163856164",
    "TxBps": "4214792810.2678127",
    "TxErrBits": "52942226547142352",
    "TxErrPackets": "509056042421691",
    "TxOkBits": "4561964553298733",
    "TxOkPackets": "43864743789853",
    "TxPps": "40526901.803920366"
  },
  "PortChannel102": {
    "RxBps": "1.2202977000824049",
    "RxErrBits": "0",
    "RxErrPackets": "0",
    "RxOkPackets": "5937",
    "RxOkBits": "N/A",
    "RxPps": "0.013699805079217392",
    "TxBps": "0",
    "TxErrBits": "0",
    "TxErrPackets": "0",
    "TxOkBits": "0",
    "TxOkPackets": "0",
    "TxPps": "0"
  },
  "PortChannel103": {
    "RxBps": "6.0568048649819142",
    "RxErrBits": "0",
    "RxErrPackets": "0",
    "RxOkPackets": "5943",
    "RxOkBits": "1048821",
    "RxPps": "0.058547265917178126",
    "TxBps": "0",
    "TxErrBits": "0",
    "TxErrPackets": "0",
    "TxOkBits": "0",
    "TxOkPackets": "0",
    "TxPps": "0"
  },
  "PortChannel104": {
    "RxBps": "20.260496891870496",
    "RxErrBits": "0",
    "RxErrPackets": "0",
    "RxOkPackets": "5950",
    "RxOkBits": "1049477",
    "RxPps": "0.24715843207997978",
    "TxBps": "0",
    "TxErrBits": "N/A",
    "TxErrPackets": "0",
    "TxOkBits": "0",
    "TxOkPackets": "0",
    "TxPps": "0"
  },
  "Vlan1000": {
    "RxBps": "0.0003231896674387374",
    "RxErrBits": "0",
    "RxErrPackets": "0",
    "RxOkPackets": "17856",
    "RxOkBits": "1865088",
    "RxPps": "3.2330838487270913e-06",
    "TxBps": "0",
    "TxErrBits": "0",
    "TxErrPackets": "0",
    "TxOkBits": "0",
    "TxOkPackets": "0",
    "TxPps": "0"
  }
}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	t.Run("query SHOW interfaces counters rif PortChannel101", func(t *testing.T) {
		textPbPath := `
			elem: <name: "interfaces" >
			elem: <name: "counters" >
			elem: <name: "rif" >
			elem: <name: "PortChannel101" >
		`
		wantRespVal := []byte(`{
			"PortChannel101": {
				"RxBps": "4214812716.943851",
				"RxErrBits": "17866494",
				"RxErrPackets": "172078",
				"RxOkPackets": "43864767060035",
				"RxOkBits": "4561966927266923",
				"RxPps": "40527122.163856164",
				"TxBps": "4214792810.2678127",
				"TxErrBits": "52942226547142352",
				"TxErrPackets": "509056042421691",
				"TxOkBits": "4561964553298733",
				"TxOkPackets": "43864743789853",
				"TxPps": "40526901.803920366"
			}
	  }`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	t.Run("query SHOW interfaces counters rif PortChannel104 -p 2", func(t *testing.T) {
		textPbPath := `
			elem: <name: "interfaces" >
			elem: <name: "counters" >
			elem: <name: "rif" >
			elem: <name: "PortChannel104" key: {key: "period" value: "1"} >
		`
		wantRespVal := []byte(`{
			"PortChannel104": {
				"RxBps": "20.260496891870496",
				"RxErrBits": "0",
				"RxErrPackets": "0",
				"RxOkPackets": "0",
				"RxOkBits": "0",
				"RxPps": "0.24715843207997978",
				"TxBps": "0",
				"TxErrBits": "N/A",
				"TxErrPackets": "0",
				"TxOkBits": "0",
				"TxOkPackets": "0",
				"TxPps": "0"
			}
	  }`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	t.Run("query SHOW interfaces counters rif PortChannel101 -p 1", func(t *testing.T) {
		textPbPath := `
			elem: <name: "interfaces" >
			elem: <name: "counters" >
			elem: <name: "rif" >
			elem: <name: "PortChannel101"  key: {key: "period" value: "1"} >
		`
		wantRespVal := []byte(`{
			"PortChannel101": {
				"RxBps": "4214812716.943851",
				"RxErrBits": "0",
				"RxErrPackets": "0",
				"RxOkPackets": "0",
				"RxOkBits": "0",
				"RxPps": "40527122.163856164",
				"TxBps": "4214792810.2678127",
				"TxErrBits": "0",
				"TxErrPackets": "0",
				"TxOkBits": "0",
				"TxOkPackets": "0",
				"TxPps": "40526901.803920366"
			}
	  }`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	t.Run("query SHOW interfaces counters rif PortChannel102 -p 1", func(t *testing.T) {
		textPbPath := `
			elem: <name: "interfaces" >
			elem: <name: "counters" >
			elem: <name: "rif" >
			elem: <name: "PortChannel102"  key: {key: "period" value: "1"} >
		`
		wantRespVal := []byte(`{
			"PortChannel102": {
				"RxBps": "1.2202977000824049",
				"RxErrBits": "0",
				"RxErrPackets": "0",
				"RxOkPackets": "0",
				"RxOkBits": "N/A",
				"RxPps": "0.013699805079217392",
				"TxBps": "0",
				"TxErrBits": "0",
				"TxErrPackets": "0",
				"TxOkBits": "0",
				"TxOkPackets": "0",
				"TxPps": "0"
			}
	  }`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	// invalid interface name
	t.Run("query SHOW interfaces counters rif PortChannel11 -p 1", func(t *testing.T) {
		textPbPath := `
			elem: <name: "interfaces" >
			elem: <name: "counters" >
			elem: <name: "rif" >
			elem: <name: "PortChannel11"  key: {key: "period" value: "1"} >
		`
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.InvalidArgument, nil, false)
	})
}
