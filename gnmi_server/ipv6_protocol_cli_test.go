package gnmi

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

// Tests SHOW ipv6 protocol
func TestGetIPv6Protocol(t *testing.T) {
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

	singleStruct := []map[string]interface{}{
		{"VRF": "default", "Protocols": []interface{}{
			map[string]interface{}{"Protocol": "system", "route-map": "none"},
			map[string]interface{}{"Protocol": "kernel", "route-map": "none"},
			map[string]interface{}{"Protocol": "connected", "route-map": "none"},
			map[string]interface{}{"Protocol": "local", "route-map": "none"},
			map[string]interface{}{"Protocol": "static", "route-map": "none"},
			map[string]interface{}{"Protocol": "bgp", "route-map": "RM_SET_SRC6"},
		}},
	}
	multiStruct := []map[string]interface{}{
		{"VRF": "default", "Protocols": []interface{}{
			map[string]interface{}{"Protocol": "system", "route-map": "none"},
			map[string]interface{}{"Protocol": "kernel", "route-map": "none"},
			map[string]interface{}{"Protocol": "bgp", "route-map": "RM_SET_SRC6"},
		}},
		{"VRF": "vrf1", "Protocols": []interface{}{
			map[string]interface{}{"Protocol": "system", "route-map": "none"},
			map[string]interface{}{"Protocol": "kernel", "route-map": "none"},
			map[string]interface{}{"Protocol": "bgp", "route-map": "RM_CUSTOM_VRF1"},
		}},
		{"VRF": "vrf2", "Protocols": []interface{}{
			map[string]interface{}{"Protocol": "system", "route-map": "none"},
			map[string]interface{}{"Protocol": "kernel", "route-map": "none"},
			map[string]interface{}{"Protocol": "bgp", "route-map": "none"},
		}},
	}

	singleExpected, _ := json.Marshal(singleStruct)
	multiExpected, _ := json.Marshal(multiStruct)
	emptyExpected, _ := json.Marshal([]interface{}{})

	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal interface{}
		valTest     bool
		mockFile    string
	}{
		{
			desc:       "single VRF output",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "protocol" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: singleExpected,
			valTest:     true,
			mockFile:    "../testdata/VTYSH_SHOW_IPV6_PROTOCOL_SINGLE.txt",
		},
		{
			desc:       "multi VRF output",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "protocol" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: multiExpected,
			valTest:     true,
			mockFile:    "../testdata/VTYSH_SHOW_IPV6_PROTOCOL_MULTI.txt",
		},
		{
			desc:       "empty output",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "protocol" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: emptyExpected,
			valTest:     true,
			mockFile:    "../testdata/VTYSH_SHOW_IPV6_PROTOCOL_EMPTY.txt",
		},
	}

	for _, test := range tests {
		var patches interface{ Reset() }
		if test.mockFile != "" {
			patches = MockNSEnterOutput(t, test.mockFile)
		}
		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})
		if patches != nil {
			patches.Reset()
		}
	}
}
