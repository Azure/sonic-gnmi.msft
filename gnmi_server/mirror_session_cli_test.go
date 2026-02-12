package gnmi

import (
	"crypto/tls"
	"encoding/json"
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

func TestGetMirrorSession(t *testing.T) {
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

	// Mock database data separated into CONFIG_DB and STATE_DB
	// CONFIG_DB data (first query)
	configResult := map[string]interface{}{
		"session1": map[string]interface{}{
			"type":      "ERSPAN",
			"src_ip":    "1.1.1.1",
			"dst_ip":    "2.2.2.2",
			"gre_type":  "0x88be",
			"dscp":      "8",
			"ttl":       "255",
			"queue":     "7",
			"src_port":  "Ethernet0",
			"direction": "both",
		},
		"session2": map[string]interface{}{
			"type":      "SPAN",
			"dst_port":  "Ethernet4",
			"src_port":  "Ethernet8",
			"direction": "ingress",
			"queue":     "6",
		},
	}

	// STATE_DB data (per-session queries) - returned directly without session name wrapper
	// This matches the actual behavior of GetMapFromQueries when querying a specific key
	stateResult1Direct := map[string]interface{}{
		"status":       "active",
		"monitor_port": "Ethernet12",
	}

	stateResult2Direct := map[string]interface{}{
		"status": "active",
	}

	emptyConfigResult := map[string]interface{}{}

	// Empty state results to simulate STATE_DB query errors
	emptyStateResult := map[string]interface{}{}

	expectedAllSessions := `{
		"erspan_sessions": [
			{
				"name": "session1",
				"status": "active",
				"src_ip": "1.1.1.1",
				"dst_ip": "2.2.2.2",
				"gre": "0x88be",
				"dscp": "8",
				"ttl": "255",
				"queue": "7",
				"policer": "",
				"monitor_port": "Ethernet12",
				"src_port": "Ethernet0",
				"direction": "both"
			}
		],
		"span_sessions": [
			{
				"name": "session2",
				"status": "active",
				"dst_port": "Ethernet4",
				"src_port": "Ethernet8",
				"direction": "ingress",
				"queue": "6",
				"policer": ""
			}
		]
	}`

	expectedFilteredSession := `{
		"erspan_sessions": [
			{
				"name": "session1",
				"status": "active",
				"src_ip": "1.1.1.1",
				"dst_ip": "2.2.2.2",
				"gre": "0x88be",
				"dscp": "8",
				"ttl": "255",
				"queue": "7",
				"policer": "",
				"monitor_port": "Ethernet12",
				"src_port": "Ethernet0",
				"direction": "both"
			}
		],
		"span_sessions": []
	}`

	expectedErrorState := `{
		"erspan_sessions": [
			{
				"name": "session1",
				"status": "error",
				"src_ip": "1.1.1.1",
				"dst_ip": "2.2.2.2",
				"gre": "0x88be",
				"dscp": "8",
				"ttl": "255",
				"queue": "7",
				"policer": "",
				"monitor_port": "",
				"src_port": "Ethernet0",
				"direction": "both"
			}
		],
		"span_sessions": [
			{
				"name": "session2",
				"status": "error",
				"dst_port": "Ethernet4",
				"src_port": "Ethernet8",
				"direction": "ingress",
				"queue": "6",
				"policer": ""
			}
		]
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
			desc:       "query show mirror_session - all sessions",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "mirror_session" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedAllSessions),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				callCount := 0
				patches := gomonkey.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					callCount++
					// First call: CONFIG_DB
					if callCount == 1 {
						return configResult, nil
					}
					// Subsequent calls: STATE_DB for individual sessions (returned directly)
					if len(queries) > 0 && len(queries[0]) > 2 {
						sessionName := queries[0][2]
						if sessionName == "session1" {
							return stateResult1Direct, nil
						} else if sessionName == "session2" {
							return stateResult2Direct, nil
						}
					}
					return map[string]interface{}{}, nil
				})
				return patches
			},
		},
		{
			desc:       "query show mirror_session - specific session",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "mirror_session" >
				elem: <name: "session1" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedFilteredSession),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				callCount := 0
				patches := gomonkey.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					callCount++
					if callCount == 1 {
						return configResult, nil
					}
					if len(queries) > 0 && len(queries[0]) > 2 {
						sessionName := queries[0][2]
						if sessionName == "session1" {
							return stateResult1Direct, nil
						} else if sessionName == "session2" {
							return stateResult2Direct, nil
						}
					}
					return map[string]interface{}{}, nil
				})
				return patches
			},
		},
		{
			desc:       "query show mirror_session - error state",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "mirror_session" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedErrorState),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				callCount := 0
				patches := gomonkey.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					callCount++
					// Return CONFIG_DB data on first call
					if callCount == 1 {
						return configResult, nil
					}
					// Return errors for STATE_DB queries to simulate missing state
					return emptyStateResult, fmt.Errorf("state data not found")
				})
				return patches
			},
		},
		{
			desc:       "query show mirror_session - empty response",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "mirror_session" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{"erspan_sessions":[],"span_sessions":[]}`),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				patches := gomonkey.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					return emptyConfigResult, nil
				})
				return patches
			},
		},
		{
			desc:       "query show mirror_session - readSessionsInfo error",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "mirror_session" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() *gomonkey.Patches {
				patches := gomonkey.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					return nil, fmt.Errorf("database connection failed")
				})
				return patches
			},
		},
		{
			desc:       "query show mirror_session - nonexistent session",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "mirror_session" >
				elem: <name: "nonexistent" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{"erspan_sessions":[],"span_sessions":[]}`),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				callCount := 0
				patches := gomonkey.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
					callCount++
					if callCount == 1 {
						return configResult, nil
					}
					if len(queries) > 0 && len(queries[0]) > 2 {
						sessionName := queries[0][2]
						if sessionName == "session1" {
							return stateResult1Direct, nil
						} else if sessionName == "session2" {
							return stateResult2Direct, nil
						}
					}
					return map[string]interface{}{}, nil
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
			runTestGetWithJSONValidation(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})

		if patch != nil {
			patch.Reset()
		}
	}
}

// runTestGetWithJSONValidation validates JSON structure in addition to exact byte comparison
func runTestGetWithJSONValidation(t *testing.T, ctx context.Context, gClient pb.GNMIClient, pathTarget string, textPbPath string, wantRetCode codes.Code, wantRespVal interface{}, valTest bool) {
	if valTest {
		// First run the standard test
		runTestGet(t, ctx, gClient, pathTarget, textPbPath, wantRetCode, wantRespVal, valTest)

		// Then validate JSON structure if response is expected to be JSON
		if wantRetCode == codes.OK && wantRespVal != nil {
			if expectedBytes, ok := wantRespVal.([]byte); ok {
				// Parse the expected JSON to validate structure
				var expectedJSON map[string]interface{}
				if err := json.Unmarshal(expectedBytes, &expectedJSON); err != nil {
					t.Fatalf("Expected response is not valid JSON: %v", err)
				}

				// Validate the expected structure has required fields
				if _, hasERSPAN := expectedJSON["erspan_sessions"]; !hasERSPAN {
					t.Errorf("Expected JSON missing 'erspan_sessions' field")
				}
				if _, hasSPAN := expectedJSON["span_sessions"]; !hasSPAN {
					t.Errorf("Expected JSON missing 'span_sessions' field")
				}
			}
		}
	} else {
		runTestGet(t, ctx, gClient, pathTarget, textPbPath, wantRetCode, wantRespVal, valTest)
	}
}
