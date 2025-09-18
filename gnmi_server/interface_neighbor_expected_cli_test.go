package gnmi

// Tests for SHOW/interfaces/neighbor/expected

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

// getInterfaceNeighborExpected returns JSON like:
//
//	{
//	  "Ethernet2": {
//	    "neighbor":"DEVICE01T1",
//	    "neighbor_port":"Ethernet1",
//	    "neighbor_loopback":"10.1.1.1",
//	    "neighbor_mgmt":"192.0.2.10",
//	    "neighbor_type":"BackEndLeafRouter"
//	  }
//	}
func TestShowInterfaceNeighborExpected(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	conn, err := grpc.Dial(TargetAddr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()
	gClient := pb.NewGNMIClient(conn)

	fullDataFile := "../testdata/NEIGHBOR_EXPECTED_FULL.txt"
	minDataFile := "../testdata/NEIGHBOR_EXPECTED_MIN.txt"

	type tc struct {
		desc       string
		init       func()
		pathTarget string
		textPbPath string
		wantCode   codes.Code
		wantVal    []byte
		valTest    bool
	}

	tests := []tc{
		{
			desc:       "empty tables default mode -> {}",
			init:       func() {},
			pathTarget: "SHOW",
			textPbPath: `
              elem: <name:"interfaces">
              elem: <name:"neighbor">
              elem: <name:"expected">
            `,
			wantCode: codes.OK,
			wantVal:  []byte(`{}`),
			valTest:  true,
		},
		{
			desc: "all neighbors default mode (canonical keys)",
			init: func() {
				AddDataSet(t, ConfigDbNum, fullDataFile)
			},
			pathTarget: "SHOW",
			textPbPath: `
              elem: <name:"interfaces">
              elem: <name:"neighbor">
              elem: <name:"expected">
            `,
			wantCode: codes.OK,
			wantVal:  []byte(`{"Ethernet0":{"Neighbor":"DeviceA","NeighborPort":"Ethernet10","NeighborLoopback":"10.0.0.1","NeighborMgmt":"192.168.0.1","NeighborType":"Leaf"},"Ethernet2":{"Neighbor":"DeviceB","NeighborPort":"Ethernet11","NeighborLoopback":"10.0.0.2","NeighborMgmt":"192.168.0.2","NeighborType":"Spine"}}`),
			valTest:  true,
		},
		{
			desc: "all neighbors alias mode (alias keys)",
			init: func() {
				AddDataSet(t, ConfigDbNum, fullDataFile)
			},
			pathTarget: "SHOW",
			textPbPath: `
              elem: <name:"interfaces">
              elem: <name:"neighbor">
              elem: <name:"expected" key:{ key:"SONIC_CLI_IFACE_MODE" value:"alias" } >
            `,
			wantCode: codes.OK,
			wantVal:  []byte(`{"etp1":{"Neighbor":"DeviceA","NeighborPort":"Ethernet10","NeighborLoopback":"10.0.0.1","NeighborMgmt":"192.168.0.1","NeighborType":"Leaf"},"etp2":{"Neighbor":"DeviceB","NeighborPort":"Ethernet11","NeighborLoopback":"10.0.0.2","NeighborMgmt":"192.168.0.2","NeighborType":"Spine"}}`),
			valTest:  true,
		},
		{
			desc: "single interface alias mode valid alias",
			init: func() {
				AddDataSet(t, ConfigDbNum, fullDataFile)
			},
			pathTarget: "SHOW",
			textPbPath: `
              elem: <name:"interfaces">
              elem: <name:"neighbor">
              elem: <name:"expected" key:{ key:"SONIC_CLI_IFACE_MODE" value:"alias" } >
              elem: <name:"etp1">
            `,
			wantCode: codes.OK,
			wantVal:  []byte(`{"etp1":{"Neighbor":"DeviceA","NeighborPort":"Ethernet10","NeighborLoopback":"10.0.0.1","NeighborMgmt":"192.168.0.1","NeighborType":"Leaf"}}`),
			valTest:  true,
		},
		{
			desc: "single interface default mode canonical",
			init: func() {
				AddDataSet(t, ConfigDbNum, fullDataFile)
			},
			pathTarget: "SHOW",
			textPbPath: `
              elem: <name:"interfaces">
              elem: <name:"neighbor">
              elem: <name:"expected">
              elem: <name:"Ethernet2">
            `,
			wantCode: codes.OK,
			wantVal:  []byte(`{"Ethernet2":{"Neighbor":"DeviceB","NeighborPort":"Ethernet11","NeighborLoopback":"10.0.0.2","NeighborMgmt":"192.168.0.2","NeighborType":"Spine"}}`),
			valTest:  true,
		},
		{
			desc: "alias mode invalid canonical should error",
			init: func() {
				AddDataSet(t, ConfigDbNum, fullDataFile)
			},
			pathTarget: "SHOW",
			textPbPath: `
              elem: <name:"interfaces">
              elem: <name:"neighbor">
              elem: <name:"expected" key:{ key:"SONIC_CLI_IFACE_MODE" value:"alias" } >
              elem: <name:"Ethernet0">
            `,
			wantCode: codes.InvalidArgument,
			valTest:  false,
		},
		{
			desc: "alias mode invalid alias",
			init: func() {
				AddDataSet(t, ConfigDbNum, fullDataFile)
			},
			pathTarget: "SHOW",
			textPbPath: `
              elem: <name:"interfaces">
              elem: <name:"neighbor">
              elem: <name:"expected" key:{ key:"SONIC_CLI_IFACE_MODE" value:"alias" } >
              elem: <name:"etp9">
            `,
			wantCode: codes.InvalidArgument,
			valTest:  false,
		},
		{
			desc: "missing metadata fields -> None defaults",
			init: func() {
				AddDataSet(t, ConfigDbNum, minDataFile)
			},
			pathTarget: "SHOW",
			textPbPath: `
              elem: <name:"interfaces">
              elem: <name:"neighbor">
              elem: <name:"expected" key:{ key:"SONIC_CLI_IFACE_MODE" value:"alias" } >
            `,
			wantCode: codes.OK,
			wantVal:  []byte(`{}`),
			valTest:  true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			ResetDataSetsAndMappings(t)
			if tc.init != nil {
				tc.init()
			}
			ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
			defer cancel()
			runTestGet(t, ctx, gClient, tc.pathTarget, tc.textPbPath, tc.wantCode, tc.wantVal, tc.valTest)
		})
	}
}
