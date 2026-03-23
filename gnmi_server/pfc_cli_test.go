package gnmi

// Tests SHOW pfc counters, SHOW pfc asymmetric, SHOW pfc priority

import (
	"crypto/tls"
	"os"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestShowPfcCounters(t *testing.T) {
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

	portOidMappingFileName := "../testdata/PORT_COUNTERS_MAPPING.txt"
	pfcCountersFileName := "../testdata/PFC_COUNTERS.txt"
	pfcCountersExpected, err := os.ReadFile("../testdata/PFC_COUNTERS_EXPECTED.txt")
	if err != nil {
		t.Fatalf("Failed to read expected PFC counters results: %v", err)
	}
	pfcHistoryFileName := "../testdata/PFC_HISTORY_COUNTERS.txt"
	pfcHistoryExpected, err := os.ReadFile("../testdata/PFC_HISTORY_EXPECTED.txt")
	if err != nil {
		t.Fatalf("Failed to read expected PFC history results: %v", err)
	}

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
			desc:       "query SHOW pfc counters NO DATA",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "pfc" >
				elem: <name: "counters" >
			`,
			wantRetCode: codes.OK,
		},
		{
			desc:       "query SHOW pfc counters with data",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "pfc" >
				elem: <name: "counters" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: pfcCountersExpected,
			valTest:     true,
			testInit: func() {
				AddDataSet(t, CountersDbNum, portOidMappingFileName)
				AddDataSet(t, CountersDbNum, pfcCountersFileName)
			},
		},
		{
			desc:       "query SHOW pfc counters with history flag",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "pfc" >
				elem: <name: "counters" key: { key: "history" value: "true" }>
			`,
			wantRetCode: codes.OK,
			wantRespVal: pfcHistoryExpected,
			valTest:     true,
			testInit: func() {
				ResetDataSetsAndMappings(t)
				AddDataSet(t, CountersDbNum, portOidMappingFileName)
				AddDataSet(t, CountersDbNum, pfcHistoryFileName)
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

func TestShowPfcAsymmetric(t *testing.T) {
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
	pfcAsymExpected, err := os.ReadFile("../testdata/PFC_ASYMMETRIC_EXPECTED.txt")
	if err != nil {
		t.Fatalf("Failed to read expected PFC asymmetric results: %v", err)
	}
	pfcAsymSingleExpected, err := os.ReadFile("../testdata/PFC_ASYMMETRIC_SINGLE_EXPECTED.txt")
	if err != nil {
		t.Fatalf("Failed to read expected PFC asymmetric single results: %v", err)
	}

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
			desc:       "query SHOW pfc asymmetric NO DATA",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "pfc" >
				elem: <name: "asymmetric" >
			`,
			wantRetCode: codes.OK,
		},
		{
			desc:       "query SHOW pfc asymmetric all ports",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "pfc" >
				elem: <name: "asymmetric" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: pfcAsymExpected,
			valTest:     true,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, portsFileName)
			},
		},
		{
			desc:       "query SHOW pfc asymmetric single port via arg",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "pfc" >
				elem: <name: "asymmetric" >
				elem: <name: "Ethernet0" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: pfcAsymSingleExpected,
			valTest:     true,
		},
		{
			desc:       "query SHOW pfc asymmetric non-existent port",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "pfc" >
				elem: <name: "asymmetric" >
				elem: <name: "EthernetNotExist" >
			`,
			wantRetCode: codes.OK,
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

func TestShowPfcPriority(t *testing.T) {
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

	portQosMapFileName := "../testdata/PFC_PORT_QOS_MAP.txt"
	pfcPriorityExpected, err := os.ReadFile("../testdata/PFC_PRIORITY_EXPECTED.txt")
	if err != nil {
		t.Fatalf("Failed to read expected PFC priority results: %v", err)
	}
	pfcPrioritySingleExpected, err := os.ReadFile("../testdata/PFC_PRIORITY_SINGLE_EXPECTED.txt")
	if err != nil {
		t.Fatalf("Failed to read expected PFC priority single results: %v", err)
	}

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
			desc:       "query SHOW pfc priority NO DATA",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "pfc" >
				elem: <name: "priority" >
			`,
			wantRetCode: codes.OK,
		},
		{
			desc:       "query SHOW pfc priority all ports",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "pfc" >
				elem: <name: "priority" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: pfcPriorityExpected,
			valTest:     true,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, portQosMapFileName)
			},
		},
		{
			desc:       "query SHOW pfc priority single port via arg",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "pfc" >
				elem: <name: "priority" >
				elem: <name: "Ethernet0" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: pfcPrioritySingleExpected,
			valTest:     true,
		},
		{
			desc:       "query SHOW pfc priority non-existent port",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "pfc" >
				elem: <name: "priority" >
				elem: <name: "EthernetNotExist" >
			`,
			wantRetCode: codes.NotFound,
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
