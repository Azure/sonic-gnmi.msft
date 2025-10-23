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

	allFeaturesExpected := `{"features":[{"name":"bgp","data":{"state":"enabled","auto_restart":"enabled","system_state":"Up","update_time":"2024-10-15 10:30:15","container_id":"bgp","container_version":"1.2.3","set_owner":"local","current_owner":"local","remote_state":"enabled"}},{"name":"database","data":{"state":"enabled","auto_restart":"disabled","system_state":"Up","update_time":"2024-10-15 10:25:10","container_id":"database","container_version":"2.1.0","set_owner":"local","current_owner":"local","remote_state":"enabled"}},{"name":"lldp","data":{"state":"enabled","auto_restart":"enabled","system_state":"Up","update_time":"2024-10-15 10:28:45","container_id":"lldp","container_version":"1.5.2","set_owner":"local","current_owner":"local","remote_state":"enabled"}},{"name":"snmp","data":{"state":"enabled","auto_restart":"enabled","system_state":"Up","update_time":"2024-10-15 10:32:20","container_id":"snmp","container_version":"3.0.1","set_owner":"local","current_owner":"local","remote_state":"enabled"}},{"name":"swss","data":{"state":"enabled","auto_restart":"enabled","system_state":"Up","update_time":"2024-10-15 10:20:30","container_id":"swss","container_version":"4.1.5","set_owner":"local","current_owner":"local","remote_state":"enabled"}},{"name":"syncd","data":{"state":"enabled","auto_restart":"enabled","system_state":"Up","update_time":"2024-10-15 10:35:12","container_id":"syncd","container_version":"2.3.1","set_owner":"local","current_owner":"local","remote_state":"enabled"}},{"name":"teamd","data":{"state":"enabled","auto_restart":"enabled","system_state":"Up","update_time":"2024-10-15 10:27:55","container_id":"teamd","container_version":"1.8.0","set_owner":"local","current_owner":"local","remote_state":"enabled"}}]}`

	bgpFeatureExpected := `{"features":[{"name":"bgp","data":{"state":"enabled","auto_restart":"enabled","system_state":"Up","update_time":"2024-10-15 10:30:15","container_id":"bgp","container_version":"1.2.3","set_owner":"local","current_owner":"local","remote_state":"enabled"}}]}`

	featureStatusDbDataFilename := "../testdata/FEATURE_DB_DATA.txt"
	featureStateDbDataFilename := "../testdata/FEATURE_STATE_DB_DATA.txt"
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
				AddDataSet(t, StateDbNum, featureStatusDbDataEmptyFilename)
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
				FlushDataSet(t, StateDbNum)
				AddDataSet(t, ConfigDbNum, featureStatusDbDataFilename)
				AddDataSet(t, StateDbNum, featureStateDbDataFilename)
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

	featureStatusDbDataFilename := "../testdata/FEATURE_DB_DATA.txt"
	featureStateDbDataFilename := "../testdata/FEATURE_STATE_DB_DATA.txt"
	featureStatusDbDataEmptyFilename := "../testdata/EMPTY_JSON.txt"
	featureStateDbDataPartialFilename := "../testdata/FEATURE_STATE_DB_DATA_PARTIAL.txt"
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
				AddDataSet(t, StateDbNum, featureStatusDbDataNoFeatureFilename)
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
				FlushDataSet(t, StateDbNum)
			},
		},
		{
			desc:       "query SHOW feature status with CONFIG_DB only (no STATE_DB)",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "feature" >
				elem: <name: "status" >
				elem: <name: "bgp" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{"features":[{"name":"bgp","data":{"state":"enabled","auto_restart":"enabled","system_state":"","update_time":"","container_id":"","container_version":"","set_owner":"local","current_owner":"","remote_state":""}}]}`),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				FlushDataSet(t, StateDbNum)
				AddDataSet(t, ConfigDbNum, featureStatusDbDataFilename)
				AddDataSet(t, StateDbNum, featureStatusDbDataEmptyFilename)
			},
		},
		{
			desc:       "query SHOW feature status with STATE_DB only (no CONFIG_DB)",
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
				FlushDataSet(t, StateDbNum)
				AddDataSet(t, ConfigDbNum, featureStatusDbDataEmptyFilename)
				AddDataSet(t, StateDbNum, featureStateDbDataFilename)
			},
		},
		{
			desc:       "query SHOW feature status with partial STATE_DB data",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "feature" >
				elem: <name: "status" >
				elem: <name: "database" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{"features":[{"name":"database","data":{"state":"enabled","auto_restart":"disabled","system_state":"","update_time":"","container_id":"","container_version":"","set_owner":"local","current_owner":"","remote_state":""}}]}`),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				FlushDataSet(t, StateDbNum)
				AddDataSet(t, ConfigDbNum, featureStatusDbDataFilename)
				// Add STATE_DB with only bgp feature, but query database feature
				AddDataSet(t, StateDbNum, featureStateDbDataPartialFilename)
			},
		},
		{
			desc:       "query SHOW feature status all features with mixed STATE_DB data",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "feature" >
				elem: <name: "status" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{"features":[{"name":"bgp","data":{"state":"enabled","auto_restart":"enabled","system_state":"Up","update_time":"2024-10-15 10:30:15","container_id":"bgp","container_version":"1.2.3","set_owner":"local","current_owner":"local","remote_state":"enabled"}},{"name":"database","data":{"state":"enabled","auto_restart":"disabled","system_state":"","update_time":"","container_id":"","container_version":"","set_owner":"local","current_owner":"","remote_state":""}},{"name":"lldp","data":{"state":"enabled","auto_restart":"enabled","system_state":"","update_time":"","container_id":"","container_version":"","set_owner":"local","current_owner":"","remote_state":""}},{"name":"snmp","data":{"state":"enabled","auto_restart":"enabled","system_state":"","update_time":"","container_id":"","container_version":"","set_owner":"local","current_owner":"","remote_state":""}},{"name":"swss","data":{"state":"enabled","auto_restart":"enabled","system_state":"","update_time":"","container_id":"","container_version":"","set_owner":"local","current_owner":"","remote_state":""}},{"name":"syncd","data":{"state":"enabled","auto_restart":"enabled","system_state":"","update_time":"","container_id":"","container_version":"","set_owner":"local","current_owner":"","remote_state":""}},{"name":"teamd","data":{"state":"enabled","auto_restart":"enabled","system_state":"","update_time":"","container_id":"","container_version":"","set_owner":"local","current_owner":"","remote_state":""}}]}`),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				FlushDataSet(t, StateDbNum)
				AddDataSet(t, ConfigDbNum, featureStatusDbDataFilename)
				// Only BGP has STATE_DB data, others will have empty runtime fields
				AddDataSet(t, StateDbNum, featureStateDbDataPartialFilename)
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
