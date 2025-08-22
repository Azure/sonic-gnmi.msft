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

func TestGetShowVlanBrief(t *testing.T) {
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

	vlanBriefFullDataFileName := "../testdata/VLAN_BRIEF_DB_DATA.txt"
	vlanBriefNoVlanDataFileName := "../testdata/VLAN_BRIEF_DB_DATA_NO_VLAN.txt"
	vlanBriefNoVlanIntDataFileName := "../testdata/VLAN_BRIEF_DB_DATA_NO_VLANINT.txt"
	vlanBriefNoVlanMemDataFileName := "../testdata/VLAN_BRIEF_DB_DATA_NO_VLANMEM.txt"
	vlanBriefWrongIpDataFileName := "../testdata/VLAN_BRIEF_DB_DATA_WRONGIP.txt"
	vlanBriefResp := `{"Vlan1":{"IP Address":["192.168.0.1/21"],"Port Tagging":["Ethernet120;untagged"],"Ports":["Ethernet120"],"Proxy ARP":["disabled"], "VLAN ID":["1"]}}`

	vlanBriefRespEmpty := `{}`
	vlanBriefRespEmptyIp := `{"Vlan1":{"IP Address":null,"Port Tagging":["Ethernet120;untagged"],"Ports":["Ethernet120"],"Proxy ARP":["disabled"], "VLAN ID":["1"]}}`
	vlanBriefRespEmptyMem := `{"Vlan1":{"IP Address":["192.168.0.1/21"],"Port Tagging":null,"Ports":null,"Proxy ARP":["disabled"], "VLAN ID":["1"]}}`

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
			desc:       "query SHOW vlan brief dataset status check",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "vlan" >
				elem: <name: "brief" >
			`,
			wantRetCode: codes.OK,
		},
		{
			desc:       "query SHOW vlan brief dataset",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "vlan" >
				elem: <name: "brief" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(vlanBriefResp),
			valTest:     true,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, vlanBriefFullDataFileName)
			},
		},
		{
			desc:       "query SHOW vlan brief no vlan dataset",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "vlan" >
				elem: <name: "brief" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(vlanBriefRespEmpty),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, vlanBriefNoVlanDataFileName)
			},
		},
		{
			desc:       "query SHOW vlan brief no vlan interface dataset",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "vlan" >
				elem: <name: "brief" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(vlanBriefRespEmptyIp),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, vlanBriefNoVlanIntDataFileName)
			},
		},
		{
			desc:       "query SHOW vlan brief no vlan mem dataset",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "vlan" >
				elem: <name: "brief" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(vlanBriefRespEmptyMem),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, vlanBriefNoVlanMemDataFileName)
			},
		},
		{
			desc:       "query SHOW vlan brief no dataset",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "vlan" >
				elem: <name: "brief" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(vlanBriefRespEmpty),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
			},
		},
		{
			desc:       "query SHOW vlan brief wrong vlan interface dataset",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "vlan" >
				elem: <name: "brief" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(vlanBriefRespEmptyIp),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, vlanBriefWrongIpDataFileName)
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
