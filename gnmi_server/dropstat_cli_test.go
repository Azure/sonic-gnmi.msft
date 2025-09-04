package gnmi

// dropcounters_cli_test.go

// Tests SHOW dropcounters counts CLI command

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

func TestGetDropCounters(t *testing.T) {
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

	// Test data files
	portOidMappingFileName := "../testdata/PORT_COUNTERS_MAPPING.txt"
	portCountersFileName := "../testdata/PORT_COUNTERS.txt"
	portTableFileName := "../testdata/PORT_TABLE.txt"
	debugMapFileName := "../testdata/COUNTERS_DEBUG_NAME_PORT_STAT_MAP.txt"
	debugCountersFileName := "../testdata/PORT_DEBUG_COUNTERS.txt"
	configDebugCounterFileName := "../testdata/CONFIG_DEBUG_COUNTER.txt"

	// Expected JSON outputs
	dropCountersAll := `{"Ethernet0":{"State":"U","RX_ERR":"0","RX_DROPS":"957","TX_ERR":"0","TX_DROPS":"2"},"Ethernet40":{"State":"U","RX_ERR":"0","RX_DROPS":"0","TX_ERR":"0","TX_DROPS":"1"},"Ethernet80":{"State":"U","RX_ERR":"0","RX_DROPS":"0","TX_ERR":"0","TX_DROPS":"1"}}`
	dropCountersIngress := `{"Ethernet0":{"State":"U","RX_ERR":"0","RX_DROPS":"957"},"Ethernet40":{"State":"U","RX_ERR":"0","RX_DROPS":"0"},"Ethernet80":{"State":"U","RX_ERR":"0","RX_DROPS":"0"}}`
	dropCountersIngressWithDebug := `{"Ethernet0":{"State":"U","RX_ERR":"0","RX_DROPS":"957","DEBUG_2":"5"},"Ethernet40":{"State":"U","RX_ERR":"0","RX_DROPS":"0","DEBUG_2":"0"},"Ethernet80":{"State":"U","RX_ERR":"0","RX_DROPS":"0","DEBUG_2":"0"}}`
	dropCountersGroupBAD := `{"Ethernet0":{"State":"U","DEBUG_2":"5"},"Ethernet40":{"State":"U","DEBUG_2":"0"},"Ethernet80":{"State":"U","DEBUG_2":"0"}}`

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
			desc:       "query SHOW dropcounters counts NO DATA",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "dropcounters" >
				elem: <name: "counts" >
			`,
			wantRetCode: codes.OK,
		},
		{
			desc:       "query SHOW dropcounters counts",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "dropcounters" >
				elem: <name: "counts" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(dropCountersAll),
			valTest:     true,
			testInit: func() {
				AddDataSet(t, CountersDbNum, portOidMappingFileName)
				AddDataSet(t, CountersDbNum, portCountersFileName)
				AddDataSet(t, ApplDbNum, portTableFileName)
			},
		},
		{
			desc:       "query SHOW dropcounters counts[counter_type=PORT_INGRESS_DROPS]",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "dropcounters" >
				elem: <name: "counts"
					  key: { key: "counter_type" value: "PORT_INGRESS_DROPS" }>
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(dropCountersIngress),
			valTest:     true,
		},
		{
			desc:       "query SHOW dropcounters counts[counter_type=PORT_INGRESS_DROPS] includes configured debug counter",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "dropcounters" >
				elem: <name: "counts"
				      key: { key: "counter_type" value: "PORT_INGRESS_DROPS" }>
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(dropCountersIngressWithDebug),
			valTest:     true,
			testInit: func() {
				AddDataSet(t, CountersDbNum, portOidMappingFileName)
				AddDataSet(t, CountersDbNum, portCountersFileName)
				AddDataSet(t, ApplDbNum, portTableFileName)
				AddDataSet(t, CountersDbNum, debugMapFileName)
				AddDataSet(t, CountersDbNum, debugCountersFileName)
				AddDataSet(t, ConfigDbNum, configDebugCounterFileName)
			},
		},
		{
			desc:       "query SHOW dropcounters counts[group=foo] filters out std counters (no config)",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "dropcounters" >
				elem: <name: "counts"
					  key: { key: "group" value: "foo" }>
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{}`),
			valTest:     true,
		},
		{
			desc:       "query SHOW dropcounters counts[group=BAD] filters out std counters and includes only debug group",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "dropcounters" >
				elem: <name: "counts"
				      key: { key: "group" value: "BAD" }>
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(dropCountersGroupBAD),
			valTest:     true,
			testInit: func() {
				AddDataSet(t, CountersDbNum, portOidMappingFileName)
				AddDataSet(t, CountersDbNum, debugMapFileName)
				AddDataSet(t, CountersDbNum, debugCountersFileName)
				AddDataSet(t, ConfigDbNum, configDebugCounterFileName)
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
