package gnmi

// bgp_device-global_cli_test.go

// Tests SHOW bgp device-global

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

func TestGetShowBgpDeviceGlobal(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()

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

	bgpDeviceGlobalExpected := `{"tsa":"enabled","w-ecmp":"disabled"}`

	bgpDeviceGlobalDbDataFilename := "../testdata/BGP_DEVICE_GLOBAL_DB_DATA.txt"
	bgpDeviceGlobalDbDataEmptyFilename := "../testdata/EMPTY_JSON.txt"

	ResetDataSetsAndMappings(t)

	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal []byte
		valTest     bool
		testInit    func()
	}{
		{
			desc:       "query SHOW bgp device-global with no data",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "bgp" >
				elem: <name: "device-global" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, bgpDeviceGlobalDbDataEmptyFilename)
			},
		},
		{
			desc:       "query SHOW bgp device-global",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "bgp" >
				elem: <name: "device-global" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(bgpDeviceGlobalExpected),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, bgpDeviceGlobalDbDataFilename)
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

func TestGetShowBgpDeviceGlobalErrorCases(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()

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

	bgpDeviceGlobalDbDataEmptyFilename := "../testdata/EMPTY_JSON.txt"
	bgpDeviceGlobalDbDataWrongKeyFilename := "../testdata/BGP_DEVICE_GLOBAL_WRONG_KEY.txt"

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
			desc:       "query SHOW bgp device-global with missing BGP_DEVICE_GLOBAL table",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "bgp" >
				elem: <name: "device-global" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, bgpDeviceGlobalDbDataEmptyFilename)
			},
		},
		{
			desc:       "query SHOW bgp device-global with no data",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "bgp" >
				elem: <name: "device-global" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, bgpDeviceGlobalDbDataEmptyFilename)
			},
		},
		{
			desc:       "query SHOW bgp device-global with wrong BGP_DEVICE_GLOBAL key",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "bgp" >
				elem: <name: "device-global" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, bgpDeviceGlobalDbDataWrongKeyFilename)
			},
		},
		{
			desc:       "query SHOW bgp device-global with no CONFIG_DB",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "bgp" >
				elem: <name: "device-global" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
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
