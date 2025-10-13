package gnmi

import (
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	sccommon "github.com/sonic-net/sonic-gnmi/show_client/common"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestGetBufferConfig(t *testing.T) {
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

	tests := []struct {
		desc        string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal interface{}
		valTest     bool
		testInit    func() *gomonkey.Patches
	}{
		{
			desc: "empty CONFIG_DB",
			textPbPath: `elem: <name: "buffer" >
                                      elem: <name: "configuration" >`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{}`),
			valTest:     true,
			testInit:    nil,
		},
		{
			desc: "partial config with only pools",
			textPbPath: `elem: <name: "buffer" >
                                      elem: <name: "configuration" >`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{
                                "pools": {
                                        "egress_lossless_pool": { "mode":"static","size":"164075364","type":"egress" },
                                        "ingress_lossless_pool": { "mode":"dynamic","size":"164075364","type":"ingress","xoff":"20181824" }
                                }
                        }`),
			valTest: true,
			testInit: func() *gomonkey.Patches {
				AddDataSet(t, ConfigDbNum, "../testdata/CONFIG_DB_BUFFER_POOL.txt")
				return nil
			},
		},
		{
			desc: "happy path with full config",
			textPbPath: `elem: <name: "buffer" >
				      elem: <name: "configuration" >`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{
				"losslessTrafficPatterns": {
					"pattern": { "field1": "value1" }
				},
				"pools": {
					"egress_lossless_pool": { "mode":"static","size":"164075364","type":"egress" },
					"ingress_lossless_pool": { "mode":"dynamic","size":"164075364","type":"ingress","xoff":"20181824" }
				},
				"profiles": {
					"egress_lossless_profile": { "pool":"egress_lossless_pool","size":"0","static_th":"165364160" },
					"egress_lossy_profile": { "pool":"egress_lossless_pool","size":"1778","dynamic_th":"0" },
					"ingress_lossy_profile": { "pool":"ingress_lossless_pool","size":"0","static_th":"165364160" }
				}
			}`),
			valTest: true,
			testInit: func() *gomonkey.Patches {
				AddDataSet(t, ConfigDbNum, "../testdata/CONFIG_DB_DEFAULT_LOSSLESS_BUFFER_PARAMETER.txt")
				AddDataSet(t, ConfigDbNum, "../testdata/CONFIG_DB_BUFFER_POOL.txt")
				AddDataSet(t, ConfigDbNum, "../testdata/CONFIG_DB_BUFFER_PROFILE.txt")
				return nil
			},
		},
		{
			desc: "verbose totals enabled",
			textPbPath: `elem: <name: "buffer" >
                                      elem: <name: "configuration" key: { key: "verbose" value: "true" } >`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{
				"losslessTrafficPatterns": {
					"pattern": { "field1": "value1" }
				},
				"pools": {
					"egress_lossless_pool": { "mode":"static","size":"164075364","type":"egress" },
					"ingress_lossless_pool": { "mode":"dynamic","size":"164075364","type":"ingress","xoff":"20181824" }
				},
				"profiles": {
					"egress_lossless_profile": { "pool":"egress_lossless_pool","size":"0","static_th":"165364160" },
					"egress_lossy_profile": { "pool":"egress_lossless_pool","size":"1778","dynamic_th":"0" },
					"ingress_lossy_profile": { "pool":"ingress_lossless_pool","size":"0","static_th":"165364160" }
				},
				"totals": { "pools": 2, "profiles": 3 }
			}`),
			valTest: true,
			testInit: func() *gomonkey.Patches {
				AddDataSet(t, ConfigDbNum, "../testdata/CONFIG_DB_DEFAULT_LOSSLESS_BUFFER_PARAMETER.txt")
				AddDataSet(t, ConfigDbNum, "../testdata/CONFIG_DB_BUFFER_POOL.txt")
				AddDataSet(t, ConfigDbNum, "../testdata/CONFIG_DB_BUFFER_PROFILE.txt")
				return nil
			},
		},
		{
			desc: "error reading lossless table",
			textPbPath: `elem: <name: "buffer" >
                                      elem: <name: "configuration" >`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() *gomonkey.Patches {
				var calls int
				return gomonkey.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					calls++
					if calls == 1 {
						return nil, fmt.Errorf("error reading DEFAULT_LOSSLESS_BUFFER_PARAMETER")
					}
					return map[string]interface{}{}, nil
				})
			},
		},
		{
			desc: "error reading pools table",
			textPbPath: `elem: <name: "buffer" >
                                      elem: <name: "configuration" >`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() *gomonkey.Patches {
				var calls int
				return gomonkey.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					calls++
					if calls == 2 {
						return nil, fmt.Errorf("error reading BUFFER_POOL")
					}
					return map[string]interface{}{}, nil
				})
			},
		},
		{
			desc: "error reading profiles table",
			textPbPath: `elem: <name: "buffer" >
                                      elem: <name: "configuration" >`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() *gomonkey.Patches {
				var calls int
				return gomonkey.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					calls++
					if calls == 3 {
						return nil, fmt.Errorf("error reading BUFFER_PROFILE")
					}
					return map[string]interface{}{}, nil
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
			runTestGet(t, ctx, gClient, "SHOW", test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})

		if patch != nil {
			patch.Reset()
		}
	}
}
