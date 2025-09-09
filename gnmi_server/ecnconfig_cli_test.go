package gnmi

// Tests SHOW ecn (WRED profiles)

import (
	"crypto/tls"
	"encoding/json"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestShowEcnProfiles(t *testing.T) {
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

	wredFile := "../testdata/WRED_PROFILE_EXPECTED.txt"

	wantMap := map[string]map[string]string{
		"AZURE_LOSSLESS": {
			"green_min_threshold":    "400000",
			"green_max_threshold":    "400000",
			"green_drop_probability": "5",
			"wred_green_enable":      "true",
		},
		"AZURE_LOSSY": {
			"yellow_max_threshold": "200000",
		},
	}
	wantJSONBytes, _ := json.Marshal(wantMap)

	tests := []struct {
		desc       string
		init       func()
		textPbPath string
		wantCode   codes.Code
		wantVal    []byte
		valTest    bool
	}{
		{
			desc: "ecn no data",
			textPbPath: `
              elem: <name: "ecn">
            `,
			wantCode: codes.OK,
		},
		{
			desc: "ecn with profiles",
			init: func() {
				AddDataSet(t, ConfigDbNum, wredFile)
			},
			textPbPath: `
              elem: <name: "ecn">
            `,
			wantCode: codes.OK,
			wantVal:  wantJSONBytes,
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
