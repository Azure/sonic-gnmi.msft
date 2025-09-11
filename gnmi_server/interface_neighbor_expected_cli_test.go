package gnmi

// Enriched tests for SHOW/interfaces/neighbor/expected

import (
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"

	show_client "github.com/sonic-net/sonic-gnmi/show_client"
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
	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}
	conn, err := grpc.Dial(TargetAddr, opts...)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	gClient := pb.NewGNMIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()

	basePath := `
      elem: <name:"interfaces">
      elem: <name:"neighbor">
      elem: <name:"expected">
    `

	fullDataFile := "../testdata/NEIGHBOR_EXPECTED_FULL.txt"
	minDataFile := "../testdata/NEIGHBOR_EXPECTED_MIN.txt"

	const expectedEmpty = `{}`

	type tc struct {
		desc     string
		envMode  string
		init     func()
		addArg   string // optional single interface argument (alias or canonical)
		wantCode codes.Code
		wantVal  []byte
		valTest  bool
	}

	tests := []tc{
		{
			desc:    "empty tables default mode -> {}",
			envMode: "",
			init: func() {
				// no data loaded
			},
			wantCode: codes.OK,
			wantVal:  []byte(`{}`),
			valTest:  true,
		},
		{
			desc:    "all neighbors default mode (canonical keys)",
			envMode: "",
			init: func() {
				AddDataSet(t, ConfigDbNum, fullDataFile)
			},
			wantCode: codes.OK,
			// Keys are canonical (Ethernet0, Ethernet2)
			wantVal: []byte(`{"Ethernet0":{"Neighbor":"DeviceA","NeighborPort":"Ethernet10","NeighborLoopback":"10.0.0.1","NeighborMgmt":"192.168.0.1","NeighborType":"Leaf"},"Ethernet2":{"Neighbor":"DeviceB","NeighborPort":"Ethernet11","NeighborLoopback":"10.0.0.2","NeighborMgmt":"192.168.0.2","NeighborType":"Spine"}}`),
			valTest: true,
		},
		{
			desc:    "all neighbors alias mode (alias keys)",
			envMode: "alias",
			init: func() {
				AddDataSet(t, ConfigDbNum, fullDataFile)
			},
			wantCode: codes.OK,
			wantVal:  []byte(`{"etp1":{"Neighbor":"DeviceA","NeighborPort":"Ethernet10","NeighborLoopback":"10.0.0.1","NeighborMgmt":"192.168.0.1","NeighborType":"Leaf"},"etp2":{"Neighbor":"DeviceB","NeighborPort":"Ethernet11","NeighborLoopback":"10.0.0.2","NeighborMgmt":"192.168.0.2","NeighborType":"Spine"}}`),
			valTest:  true,
		},
		{
			desc:    "single interface alias mode valid alias",
			envMode: "alias",
			init: func() {
				AddDataSet(t, ConfigDbNum, fullDataFile)
			},
			addArg:   "etp1",
			wantCode: codes.OK,
			wantVal:  []byte(`{"etp1":{"Neighbor":"DeviceA","NeighborPort":"Ethernet10","NeighborLoopback":"10.0.0.1","NeighborMgmt":"192.168.0.1","NeighborType":"Leaf"}}`),
			valTest:  true,
		},
		{
			desc:    "single interface default mode canonical",
			envMode: "",
			init: func() {
				AddDataSet(t, ConfigDbNum, fullDataFile)
			},
			addArg:   "Ethernet2",
			wantCode: codes.OK,
			wantVal:  []byte(`{"Ethernet2":{"Neighbor":"DeviceB","NeighborPort":"Ethernet11","NeighborLoopback":"10.0.0.2","NeighborMgmt":"192.168.0.2","NeighborType":"Spine"}}`),
			valTest:  true,
		},
		{
			desc:    "alias mode invalid canonical should error",
			envMode: "alias",
			init: func() {
				AddDataSet(t, ConfigDbNum, fullDataFile)
			},
			addArg:   "Ethernet0",
			wantCode: codes.InvalidArgument,
			valTest:  false,
		},
		{
			desc:    "alias mode invalid alias",
			envMode: "alias",
			init: func() {
				AddDataSet(t, ConfigDbNum, fullDataFile)
			},
			addArg:   "etp9",
			wantCode: codes.InvalidArgument,
			valTest:  false,
		},
		{
			desc:    "missing metadata fields -> None defaults",
			envMode: "alias",
			init: func() {
				AddDataSet(t, ConfigDbNum, minDataFile)
			},
			wantCode: codes.OK,
			wantVal:  []byte(`{}`),
			valTest:  true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			ResetDataSetsAndMappings(t)
			if tc.envMode != "" {
				t.Setenv(show_client.SonicCliIfaceMode, tc.envMode)
			} else {
				t.Setenv(show_client.SonicCliIfaceMode, "")
			}
			if tc.init != nil {
				tc.init()
			}

			// Build path (append interface arg element if needed)
			path := basePath
			if tc.addArg != "" {
				path += fmt.Sprintf(`elem: <name:"%s">`, tc.addArg)
			}

			runTestGet(t, ctx, gClient, "SHOW", path, tc.wantCode, tc.wantVal, tc.valTest)
		})
	}
}
