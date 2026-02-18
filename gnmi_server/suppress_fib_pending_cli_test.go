package gnmi

// suppress_fib_pending_cli_test.go

// Tests for SHOW suppress-fib-pending

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

func TestGetShowSuppressFibPending(t *testing.T) {
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

	suppressFibPendingExpectedEnabled := `{"status":"Enabled"}`
	suppressFibPendingExpectedDisabled := `{"status":"Disabled"}`
	suppressFibPendingExpectedDefault := `{"status":"Disabled"}`

	suppressFibPendingEnabledFilename := "../testdata/SUPPRESS_FIB_PENDING_CONFIG_DB_DATA_ENABLED.txt"
	suppressFibPendingDisabledFilename := "../testdata/SUPPRESS_FIB_PENDING_CONFIG_DB_DATA_DISABLED.txt"
	suppressFibPendingEmptyFilename := "../testdata/EMPTY_JSON.txt"

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
			desc:       "Test suppress-fib-pending enabled",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "suppress-fib-pending" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(suppressFibPendingExpectedEnabled),
			valTest:     true,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, suppressFibPendingEnabledFilename)
			},
		},
		{
			desc:       "Test suppress-fib-pending disabled",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "suppress-fib-pending" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(suppressFibPendingExpectedDisabled),
			valTest:     true,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, suppressFibPendingDisabledFilename)
			},
		},
		{
			desc:       "Test suppress-fib-pending default when field missing",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "suppress-fib-pending" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(suppressFibPendingExpectedDefault),
			valTest:     true,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, suppressFibPendingEmptyFilename)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ResetDataSetsAndMappings(t)
			if test.testInit != nil {
				test.testInit()
			}
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})
	}
}

