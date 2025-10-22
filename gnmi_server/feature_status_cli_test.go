package gnmi

// feature_status_cli_test.go

// Tests SHOW feature status and SHOW feature status <feature_name>

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

func TestGetShowFeatureStatus(t *testing.T) {
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
	allFeaturesExpected := `{"features":[{"name":"bgp","data":{"state":"enabled","auto_restart":"enabled","update_time":"2024-10-15 10:30:15","container_id":"bgp_container_123","container_version":"1.2.3","set_owner":"local","current_owner":"local","remote_state":"enabled"}},{"name":"database","data":{"state":"enabled","auto_restart":"disabled","update_time":"2024-10-15 10:25:10","container_id":"database_container_456","container_version":"2.1.0","set_owner":"local","current_owner":"local","remote_state":"enabled"}},{"name":"lldp","data":{"state":"enabled","auto_restart":"enabled","update_time":"2024-10-15 10:28:45","container_id":"lldp_container_789","container_version":"1.5.2","set_owner":"local","current_owner":"local","remote_state":"enabled"}},{"name":"snmp","data":{"state":"enabled","auto_restart":"enabled","update_time":"2024-10-15 10:32:20","container_id":"snmp_container_101","container_version":"3.0.1","set_owner":"local","current_owner":"local","remote_state":"enabled"}},{"name":"swss","data":{"state":"enabled","auto_restart":"enabled","update_time":"2024-10-15 10:20:30","container_id":"swss_container_202","container_version":"4.1.5","set_owner":"local","current_owner":"local","remote_state":"enabled"}},{"name":"syncd","data":{"state":"enabled","auto_restart":"enabled","update_time":"2024-10-15 10:35:12","container_id":"syncd_container_303","container_version":"2.3.1","set_owner":"local","current_owner":"local","remote_state":"enabled"}},{"name":"teamd","data":{"state":"enabled","auto_restart":"enabled","update_time":"2024-10-15 10:27:55","container_id":"teamd_container_404","container_version":"1.8.0","set_owner":"local","current_owner":"local","remote_state":"enabled"}}]}`

	// expected output for single feature (bgp)
	bgpFeatureExpected := `{"features":[{"name":"bgp","data":{"state":"enabled","auto_restart":"enabled","update_time":"2024-10-15 10:30:15","container_id":"bgp_container_123","container_version":"1.2.3","set_owner":"local","current_owner":"local","remote_state":"enabled"}}]}`

	featureStatusDbDataFilename := "../testdata/FEATURE_DB_DATA.txt"
	featureStatusDbDataEmptyFilename := "../testdata/EMPTY_JSON.txt"

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
			desc:       "query SHOW feature status with no data",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "feature" >
				elem: <name: "status" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, featureStatusDbDataEmptyFilename)
			},
		},
		{
			desc:       "query SHOW feature status all features",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "feature" >
				elem: <name: "status" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(allFeaturesExpected),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, featureStatusDbDataFilename)
			},
		},
		{
			desc:       "query SHOW feature status bgp",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "feature" >
				elem: <name: "status" >
				elem: <name: "bgp" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(bgpFeatureExpected),
			valTest:     true,
			testInit: func() {
			},
		},
		{
			desc:       "query SHOW feature status non-existent feature",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "feature" >
				elem: <name: "status" >
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

func TestGetShowFeatureStatusErrorCases(t *testing.T) {
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

	featureStatusDbDataNoFeatureFilename := "../testdata/EMPTY_JSON.txt"

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
			desc:       "query SHOW feature status with missing FEATURE table",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "feature" >
				elem: <name: "status" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, featureStatusDbDataNoFeatureFilename)
			},
		},
		{
			desc:       "query SHOW feature status with no CONFIG_DB",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "feature" >
				elem: <name: "status" >
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
