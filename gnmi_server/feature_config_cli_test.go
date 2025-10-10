package gnmi

// feature_config_cli_test.go

// Tests SHOW feature config and SHOW feature config <feature_name>

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

func TestGetShowFeatureConfig(t *testing.T) {
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
	allFeaturesExpected := `{"features":[{"name":"bgp","data":{"state":"enabled","auto_restart":"enabled","owner":"local","fallback":"false"}},{"name":"database","data":{"state":"enabled","auto_restart":"disabled","owner":"local","fallback":"false"}},{"name":"lldp","data":{"state":"enabled","auto_restart":"enabled","owner":"local","fallback":"false"}},{"name":"snmp","data":{"state":"enabled","auto_restart":"enabled","owner":"local","fallback":"false"}},{"name":"swss","data":{"state":"enabled","auto_restart":"enabled","owner":"local","fallback":"false"}},{"name":"syncd","data":{"state":"enabled","auto_restart":"enabled","owner":"local","fallback":"false"}},{"name":"teamd","data":{"state":"enabled","auto_restart":"enabled","owner":"local","fallback":"false"}}]}`

	// Expected output for single feature (bgp)
	bgpFeatureExpected := `{"features":[{"name":"bgp","data":{"state":"enabled","auto_restart":"enabled","owner":"local","fallback":"false"}}]}`

	featureConfigDbDataFilename := "../testdata/FEATURE_CONFIG_DB_DATA.txt"
	featureConfigDbDataEmptyFilename := "../testdata/EMPTY_JSON.txt"

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
			desc:       "query SHOW feature config with no data",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "feature" >
				elem: <name: "config" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, featureConfigDbDataEmptyFilename)
			},
		},
		{
			desc:       "query SHOW feature config all features",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "feature" >
				elem: <name: "config" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(allFeaturesExpected),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, featureConfigDbDataFilename)
			},
		},
		{
			desc:       "query SHOW feature config bgp",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "feature" >
				elem: <name: "config" >
				elem: <name: "bgp" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(bgpFeatureExpected),
			valTest:     true,
			testInit: func() {
			},
		},
		{
			desc:       "query SHOW feature config non-existent feature",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "feature" >
				elem: <name: "config" >
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

func TestGetShowFeatureConfigErrorCases(t *testing.T) {
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

	featureConfigDbDataNoFeatureFilename := "../testdata/EMPTY_JSON.txt"

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
			desc:       "query SHOW feature config with missing FEATURE table",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "feature" >
				elem: <name: "config" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, featureConfigDbDataNoFeatureFilename)
			},
		},
		{
			desc:       "query SHOW feature config with no CONFIG_DB",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "feature" >
				elem: <name: "config" >
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
