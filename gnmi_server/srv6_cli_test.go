package gnmi

// interface_cli_test.go

// Tests SHOW interface/counters

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

func TestGetSRv6Stats(t *testing.T) {
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

	counterDbFileName := "../testdata/SRV6_COUNTER_DB.json"

	srv6Counters := `[{"MySID":"2001:db8:1::/48","Packets":"12345","Bytes":"67890"},{"MySID":"2001:db8:2::/48","Packets":"23456","Bytes":"78901"}]`

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
			desc:       "query SHOW srv6 stats NO DATA",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "srv6" >
				elem: <name: "stats" >
			`,
			wantRetCode: codes.OK,
		},
		{
			desc:       "query SHOW srv6 stats",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "srv6" >
				elem: <name: "stats" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(srv6Counters),
			valTest:     true,
			testInit: func() {
				AddDataSet(t, CountersDbNum, counterDbFileName)
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
