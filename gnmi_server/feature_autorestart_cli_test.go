package gnmi

// feature_autorestart_cli_test.go

// Tests SHOW feature autorestart and SHOW feature autorestart <feature_name>

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

func TestGetShowFeatureAutoRestart(t *testing.T) {
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

	// expected output
	allFeaturesExpected := `{"features":[{"name":"bgp","auto_restart":"enabled"},{"name":"database","auto_restart":"disabled"},{"name":"lldp","auto_restart":"enabled"},{"name":"snmp","auto_restart":"enabled"},{"name":"swss","auto_restart":"enabled"},{"name":"syncd","auto_restart":"enabled"},{"name":"teamd","auto_restart":"enabled"}]}`

	// expected output for single feature (bgp)
	bgpFeatureExpected := `{"features":[{"name":"bgp","auto_restart":"enabled"}]}`

	featureAutoRestartDbDataFilename := "../testdata/FEATURE_DB_DATA.txt"
	featureAutoRestartDbDataEmptyFilename := "../testdata/EMPTY_JSON.txt"

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
			desc:       "query SHOW feature autorestart with no data",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "feature" >
				elem: <name: "autorestart" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, featureAutoRestartDbDataEmptyFilename)
			},
		},
		{
			desc:       "query SHOW feature autorestart all features",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "feature" >
				elem: <name: "autorestart" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(allFeaturesExpected),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, featureAutoRestartDbDataFilename)
			},
		},
		{
			desc:       "query SHOW feature autorestart bgp",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "feature" >
				elem: <name: "autorestart" >
				elem: <name: "bgp" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(bgpFeatureExpected),
			valTest:     true,
			testInit: func() {
			},
		},
		{
			desc:       "query SHOW feature autorestart non-existent feature",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "feature" >
				elem: <name: "autorestart" >
				elem: <name: "non_existent_feature" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() {
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

func TestGetShowFeatureAutoRestartErrorCases(t *testing.T) {
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

	featureAutoRestartDbDataNoFeatureFilename := "../testdata/EMPTY_JSON.txt"

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
			desc:       "query SHOW feature autorestart with missing FEATURE table",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "feature" >
				elem: <name: "autorestart" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, featureAutoRestartDbDataNoFeatureFilename)
			},
		},
		{
			desc:       "query SHOW feature autorestart with no CONFIG_DB",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "feature" >
				elem: <name: "autorestart" >
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
