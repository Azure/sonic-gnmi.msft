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
				elem: <name: "interface">
				elem: <name: "portchannel">
			`,
			wantCode: codes.OK,
			wantVal:  `{"101":{"Team Dev":"PortChannel101","Protocol":"LACP(A)(Up)","Ports":"Ethernet0(S)"},"102":{"Team Dev":"PortChannel102","Protocol":"LACP(A)(Dw)","Ports":"Ethernet0(D)"},"103":{"Team Dev":"PortChannel103","Protocol":"LACP(I)(Up)","Ports":"Ethernet0(S) Ethernet8(D)"}}`,
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
