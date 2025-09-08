package gnmi

// Tests SHOW interface neighbor expected (JSON output)

import (
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"

	show_client "github.com/sonic-net/sonic-gnmi/show_client"
)

// getInterfaceNeighborExpected returns JSON like:
// {
//   "Ethernet2": {
//     "neighbor":"DEVICE01T1",
//     "neighbor_port":"Ethernet1",
//     "neighbor_loopback":"10.1.1.1",
//     "neighbor_mgmt":"192.0.2.10",
//     "neighbor_type":"BackEndLeafRouter"
//   }
// }

func TestShowInterfaceNeighborExpected(t *testing.T) {
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

	neighborFile := "../testdata/DEVICE_NEIGHBOR_EXPECTED.txt"
	neighborMetaFile := "../testdata/DEVICE_NEIGHBOR_METADATA_EXPECTED.txt"
	neighborOnlyFile := "../testdata/DEVICE_NEIGHBOR_EXPECTED_NO_META.txt"

	const (
		expectedEmpty       = `{}`
		expectedSingle      = `{"Ethernet2":{"Neighbor":"DEVICE01T1","NeighborPort":"Ethernet1","NeighborLoopback":"10.1.1.1","NeighborMgmt":"192.0.2.10","NeighborType":"BackEndLeafRouter"}}`
		expectedMissingMeta = `{"Ethernet4":{"Neighbor":"DEVICE02T1","NeighborPort":"Ethernet9","NeighborLoopback":"None","NeighborMgmt":"None","NeighborType":"None"}}`
	)

	tests := []struct {
		desc       string
		init       func()
		textPbPath string
		wantCode   codes.Code
		wantVal    []byte
		valTest    bool
	}{
		{
			desc: "no data",
			init: func() {
				FlushDataSet(t, ConfigDbNum)
			},
			textPbPath: `
              elem: <name: "interface">
              elem: <name: "neighbor">
              elem: <name: "expected">
            `,
			wantCode: codes.OK,
			wantVal:  []byte(expectedEmpty),
			valTest:  true,
		},
		{
			desc: "single neighbor (datasets)",
			init: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, neighborFile)
				AddDataSet(t, ConfigDbNum, neighborMetaFile)
			},
			textPbPath: `
              elem: <name: "interface">
              elem: <name: "neighbor">
              elem: <name: "expected">
            `,
			wantCode: codes.OK,
			wantVal:  []byte(expectedSingle),
			valTest:  true,
		},
		{
			desc: "missing metadata defaults (datasets)",
			init: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, neighborOnlyFile)
			},
			textPbPath: `
              elem: <name: "interface">
              elem: <name: "neighbor">
              elem: <name: "expected">
            `,
			wantCode: codes.OK,
			wantVal:  []byte(expectedMissingMeta),
			valTest:  true,
		},
		{
			desc: "GetMapFromQueries error (neighbor)",
			init: func() {
				FlushDataSet(t, ConfigDbNum)
				patch := gomonkey.ApplyFunc(show_client.GetMapFromQueries,
					func(q [][]string) (map[string]interface{}, error) {
						return nil, fmt.Errorf("injected neighbor error")
					})
				t.Cleanup(func() { patch.Reset() })
			},
			textPbPath: `
              elem: <name: "interface">
              elem: <name: "neighbor">
              elem: <name: "expected">
            `,
			wantCode: codes.NotFound,
			valTest:  false,
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
