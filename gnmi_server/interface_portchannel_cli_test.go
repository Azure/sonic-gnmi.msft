package gnmi

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

func TestShowInterfacePortchannel(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}
	conn, err := grpc.Dial(TargetAddr, opts...)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	gClient := pb.NewGNMIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()

	portchannelFile := "../testdata/PORTCHANNEL_EXPECTED.txt"
	lagTableStateFile := "../testdata/LAG_TABLE_STATE_EXPECTED.txt"
	lagTableApplFile := "../testdata/LAG_TABLE_APPL_EXPECTED.txt"
	lagMemberTableStateFile := "../testdata/LAG_MEMBER_TABLE_STATE_EXPECTED.txt"
	lagMemberTableApplFile := "../testdata/LAG_MEMBER_TABLE_APPL_EXPECTED.txt"
	portAliasFile := "../testdata/PORT_ALIAS_EXPECTED.txt"

	tests := []struct {
		desc       string
		init       func()
		textPbPath string
		wantCode   codes.Code
		wantVal    string
		valTest    bool
	}{
		{
			desc: "multiple portchannels: active/up vs active/down; selected vs deselected",
			init: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, portchannelFile)
				AddDataSet(t, StateDbNum, lagTableStateFile)
				AddDataSet(t, StateDbNum, lagMemberTableStateFile)
				AddDataSet(t, ApplDbNum, lagTableApplFile)
				AddDataSet(t, ApplDbNum, lagMemberTableApplFile)
			},
			textPbPath: `
				elem: <name: "interfaces">
				elem: <name: "portchannel">
			`,
			wantCode: codes.OK,
			wantVal:  `{"101":{"Team Dev":"PortChannel101","Protocol":{"name":"LACP","active":true,"operational_status":"up"},"Ports":[{"name":"Ethernet0","selected":true,"status":"enabled","in_sync":true}]},"102":{"Team Dev":"PortChannel102","Protocol":{"name":"LACP","active":true,"operational_status":"down"},"Ports":[{"name":"Ethernet0","selected":false,"status":"disabled","in_sync":true}]},"103":{"Team Dev":"PortChannel103","Protocol":{"name":"LACP","active":false,"operational_status":"up"},"Ports":[{"name":"Ethernet0","selected":true,"status":"enabled","in_sync":true},{"name":"Ethernet8","selected":false,"status":"disabled","in_sync":true}]}}`,
			valTest:  true,
		},
		{
			desc: "alias mode via path key: member ports rendered with aliases",
			init: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, portchannelFile)
				AddDataSet(t, StateDbNum, lagTableStateFile)
				AddDataSet(t, StateDbNum, lagMemberTableStateFile)
				AddDataSet(t, ApplDbNum, lagTableApplFile)
				AddDataSet(t, ApplDbNum, lagMemberTableApplFile)
				AddDataSet(t, ConfigDbNum, portAliasFile)
			},
			textPbPath: `
				elem: <name: "interfaces">
				elem: <name: "portchannel" key: { key: "SONIC_CLI_IFACE_MODE" value: "alias" } >
			`,
			wantCode: codes.OK,
			wantVal:  `{"101":{"Team Dev":"PortChannel101","Protocol":{"name":"LACP","active":true,"operational_status":"up"},"Ports":[{"name":"etp1","selected":true,"status":"enabled","in_sync":true}]},"102":{"Team Dev":"PortChannel102","Protocol":{"name":"LACP","active":true,"operational_status":"down"},"Ports":[{"name":"etp1","selected":false,"status":"disabled","in_sync":true}]},"103":{"Team Dev":"PortChannel103","Protocol":{"name":"LACP","active":false,"operational_status":"up"},"Ports":[{"name":"etp1","selected":true,"status":"enabled","in_sync":true},{"name":"etp2","selected":false,"status":"disabled","in_sync":true}]}}`,
			valTest:  true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			if tc.init != nil {
				tc.init()
			}
			runTestGet(t, ctx, gClient, "SHOW", tc.textPbPath, tc.wantCode, tc.wantVal, tc.valTest)
		})
	}
}
