package gnmi

// reboot_cause_cli_test.go

// Tests SHOW reboot-cause and SHOW reboot-cause history

import (
	"crypto/tls"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"

	show_client "github.com/sonic-net/sonic-gnmi/show_client"
)

func TestGetShowRebootCauseHistory(t *testing.T) {
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

	vlanBriefDataFileName := "../testdata/VLAN_BRIEF_DB_DATA.txt"
	rebootCauseHistoryWatchdog := `{"2025_07_10_20_06_34":{"cause":"reboot","comment":"N/A","time":"Thu Jul 10 08:05:33 PM UTC 2025","user":"admin"},"2025_07_10_20_12_49":{"cause":"fast-reboot","comment":"N/A","time":"Thu Jul 10 08:10:49 PM UTC 2025","user":"admin"},"2025_07_10_20_19_34":{"cause":"warm-reboot","comment":"N/A","time":"Thu Jul 10 08:17:34 PM UTC 2025","user":"admin"},"2025_07_10_20_31_14":{"cause":"Watchdog (watchdog, description: Watchdog fired, time: 2025-07-10 20:30:26)","comment":"Unknown","time":"N/A","user":"N/A"},"2025_07_10_20_36_35":{"cause":"reboot","comment":"N/A","time":"Thu Jul 10 08:35:34 PM UTC 2025","user":"admin"},"2025_07_10_20_41_54":{"cause":"reboot","comment":"N/A","time":"Thu Jul 10 08:40:52 PM UTC 2025","user":"admin"},"2025_07_10_20_47_15":{"cause":"reboot","comment":"N/A","time":"Thu Jul 10 08:46:13 PM UTC 2025","user":"admin"},"2025_07_11_01_49_30":{"cause":"reboot","comment":"N/A","time":"Fri Jul 11 01:48:29 AM UTC 2025","user":"admin"},"2025_07_11_02_00_24":{"cause":"reboot","comment":"N/A","time":"Fri Jul 11 01:59:22 AM UTC 2025","user":"admin"},"2025_07_11_02_35_51":{"cause":"Unknown","comment":"N/A","time":"N/A","user":"N/A"}}`
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
				AddDataSet(t, ConfigDbNum, vlanBriefDataFileName)
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
