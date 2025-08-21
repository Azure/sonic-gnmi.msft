package gnmi

// mac_aging_time_cli_test.go

// Tests SHOW mac aging-time CLI command

import (
	"crypto/tls"
	"encoding/json"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestMacAgingTime(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}

	conn, err := grpc.Dial(TargetAddr, opts...)
	if err != nil {
		t.Fatalf("Dailing to %q failed: %v", TargetAddr, err)
	}
	defer conn.Close()

	gClient := pb.NewGNMIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	macAgingTimeDefaultMap := map[string]string{
		"fdb_aging_time": "N/A",
	}
	macAgingTimeSetMap := map[string]string{
		"fdb_aging_time": "600s",
	}
	// Convert to JSON bytes for comparison
	macAgingTimeDefault, _ := json.Marshal(macAgingTimeDefaultMap)
	macAgingTimeSet, _ := json.Marshal(macAgingTimeSetMap)

	macAgingTimeSetFileName := "../testdata/MAC_AGING_TIME_SET.txt"
	macAgingTimeNotSetFileName := "../testdata/MAC_AGING_TIME_NOT_SET.txt"

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
			desc:       "query SHOW mac aging-time read error",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "mac" >
				elem: <name: "aging-time" >
			`,
			wantRetCode: codes.OK,
		},
		{
			desc:       "query SHOW mac aging-time not set in APPL_DB",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "mac" >
				elem: <name: "aging-time" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: macAgingTimeDefault,
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ApplDbNum)
				AddDataSet(t, ApplDbNum, macAgingTimeNotSetFileName)
			},
		},
		{
			desc:       "query SHOW mac aging-time Set in APPL_DB",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "mac" >
				elem: <name: "aging-time" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: macAgingTimeSet,
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ApplDbNum)
				AddDataSet(t, ApplDbNum, macAgingTimeSetFileName)
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
