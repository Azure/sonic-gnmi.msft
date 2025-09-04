package gnmi

// Tests SHOW dropcounters capabilities

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

func TestShowDropcountersCapabilities(t *testing.T) {
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

	stateCapsFile := "../testdata/DEBUG_COUNTER_CAPABILITIES.txt"
	jsonBytes := `{"PORT_INGRESS_DROPS":{"count":"10","reasons":"[MPLS_MISS,FDB_AND_BLACKHOLE_DISCARDS,IP_HEADER_ERROR,L3_EGRESS_LINK_DOWN,EXCEEDS_L3_MTU,DIP_LINK_LOCAL,SIP_LINK_LOCAL,ACL_ANY,SMAC_EQUALS_DMAC]"}}`

	tests := []struct {
		desc       string
		init       func()
		textPbPath string
		wantCode   codes.Code
		wantVal    []byte
		valTest    bool
	}{
		{
			desc: "capabilities no data",
			textPbPath: `
              elem: <name: "dropcounters">
              elem: <name: "capabilities">
            `,
			wantCode: codes.OK,
		},
		{
			desc: "capabilities json",
			init: func() {
				AddDataSet(t, StateDbNum, stateCapsFile)
			},
			textPbPath: `
              elem: <name: "dropcounters">
              elem: <name: "capabilities">
            `,
			wantCode: codes.OK,
			wantVal:  []byte(jsonBytes),
			valTest:  true,
		},
	}

	for _, tc := range tests {
		if tc.init != nil {
			tc.init()
		}
		t.Run(tc.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, "SHOW", tc.textPbPath, tc.wantCode, tc.wantVal, tc.valTest)
		})
	}
}
