package gnmi

// kdump_config_cli_test.go

// Tests SHOW kdump config

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

func TestGetShowKdumpConfig(t *testing.T) {
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

	kdumpConfigExpectedEnabled := `{"administrative_mode":"Enabled","max_dump_files":"3","memory_reservation":"64M","operational_mode":"Ready after reboot","ssh_connection_string":"connection_string not found","ssh_private_key_path":"ssh_key not found"}`
	kdumpConfigExpectedDisabled := `{"administrative_mode":"Disabled","max_dump_files":"3","memory_reservation":"64M","operational_mode":"Not Ready","ssh_connection_string":"connection_string not found","ssh_private_key_path":"ssh_key not found"}`
	kdumpConfigExpectedWithSSH := `{"administrative_mode":"Enabled","max_dump_files":"3","memory_reservation":"64M","operational_mode":"Ready after reboot","ssh_connection_string":"user@remote.host:/path","ssh_private_key_path":"/etc/kdump/ssh_key"}`

	kdumpConfigDbDataEnabledFilename := "../testdata/KDUMP_CONFIG_DB_DATA_ENABLED.txt"
	kdumpConfigDbDataDisabledFilename := "../testdata/KDUMP_CONFIG_DB_DATA_DISABLED.txt"
	kdumpConfigDbDataWithSSHFilename := "../testdata/KDUMP_CONFIG_DB_DATA_SSH.txt"
	kdumpConfigDbDataEmptyFilename := "../testdata/EMPTY_JSON.txt"

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
			desc:       "Test kdump config enabled",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "kdump" >
				elem: <name: "config" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(kdumpConfigExpectedEnabled),
			valTest:     true,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, kdumpConfigDbDataEnabledFilename)
			},
		},
		{
			desc:       "Test kdump config disabled",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "kdump" >
				elem: <name: "config" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(kdumpConfigExpectedDisabled),
			valTest:     true,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, kdumpConfigDbDataDisabledFilename)
			},
		},
		{
			desc:       "Test kdump config with SSH",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "kdump" >
				elem: <name: "config" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(kdumpConfigExpectedWithSSH),
			valTest:     true,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, kdumpConfigDbDataWithSSHFilename)
			},
		},
		{
			desc:       "Test kdump config empty data",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "kdump" >
				elem: <name: "config" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{"administrative_mode":"Disabled","max_dump_files":"Unknown","memory_reservation":"Unknown","operational_mode":"Not Ready","ssh_connection_string":"connection_string not found","ssh_private_key_path":"ssh_key not found"}`),
			valTest:     true,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, kdumpConfigDbDataEmptyFilename)
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
