package gnmi

import (
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"context"
	"github.com/agiledragon/gomonkey/v2"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestGetManagementInterfaceAddress(t *testing.T) {
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

	expectedManagementInterface := `[{"Management IP address":"10.0.0.5/8","Management Network Default Gateway":"10.0.0.1"},{"Management IP address":"192.168.1.100/24","Management Network Default Gateway":"192.168.1.1"}]`

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
			desc:       "query SHOW management-interface address success",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "management-interface" >
				elem: <name: "address" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedManagementInterface),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(common.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					return map[string]interface{}{
						"eth0|192.168.1.100/24": map[string]interface{}{
							"gwaddr": "192.168.1.1",
						},
						"eth0|10.0.0.5/8": map[string]interface{}{
							"gwaddr": "10.0.0.1",
						},
					}, nil
				})
				return patches
			},
		},
		{
			desc:       "query SHOW management-interface address with empty data",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "management-interface" >
				elem: <name: "address" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`[]`),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(common.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					return map[string]interface{}{}, nil
				})
				return patches
			},
		},
		{
			desc:       "query SHOW management-interface address without gateway",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "management-interface" >
				elem: <name: "address" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`[{"Management IP address":"192.168.1.100/24","Management Network Default Gateway":""}]`),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(common.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					return map[string]interface{}{
						"eth0|192.168.1.100/24": map[string]interface{}{},
					}, nil
				})
				return patches
			},
		},
		{
			desc:       "query SHOW management-interface address with invalid key format",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "management-interface" >
				elem: <name: "address" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`[]`),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(common.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					return map[string]interface{}{
						"eth0_invalid_key": map[string]interface{}{
							"gwaddr": "192.168.1.1",
						},
					}, nil
				})
				return patches
			},
		},
		{
			desc:       "query SHOW management-interface address with database error",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "management-interface" >
				elem: <name: "address" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(common.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					return nil, fmt.Errorf("simulated database error")
				})
				return patches
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
