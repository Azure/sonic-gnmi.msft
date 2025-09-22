package gnmi

import (
	"context"
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"
	common "github.com/sonic-net/sonic-gnmi/show_client/common"

	"github.com/agiledragon/gomonkey/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestGetSwitchTrimmingGlobalConfig(t *testing.T) {
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

	expectedJSON := `{
		"size": "64",
		"dscp_value": "32",
		"tc_value": "5",
		"queue_index": "3"
	}`

	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal interface{}
		valTest     bool
		testInit    func() *gomonkey.Patches
	}{
		{
			desc:       "query show switch-trimming with success case",
			pathTarget: "SHOW",
			textPbPath: `
			elem: <name: "switch-trimming" >
			elem: <name: "global" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedJSON),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				return gomonkey.ApplyFunc(common.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					return map[string]interface{}{
						"size":        "64",
						"dscp_value":  "32",
						"tc_value":    "5",
						"queue_index": "3",
					}, nil
				})
			},
		},
		{
			desc:       "query show switch-trimming with empty config",
			pathTarget: "SHOW",
			textPbPath: `
			elem: <name: "switch-trimming" >
			elem: <name: "global" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{
				"response": "No configuration is present in CONFIG DB"
			}`),
			valTest: true,
			testInit: func() *gomonkey.Patches {
				return gomonkey.ApplyFunc(common.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					return map[string]interface{}{}, nil
				})
			},
		},
		{
			desc:       "query show switch-trimming with DB error",
			pathTarget: "SHOW",
			textPbPath: `
			elem: <name: "switch-trimming" >
			elem: <name: "global" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() *gomonkey.Patches {
				return gomonkey.ApplyFunc(common.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					return nil, fmt.Errorf("simulated DB failure")
				})
			},
		},
	}

	for _, test := range tests {
		var patch *gomonkey.Patches
		if test.testInit != nil {
			patch = test.testInit()
		}

		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})

		if patch != nil {
			patch.Reset()
		}
	}
}
