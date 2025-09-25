package gnmi

// interface_transceiver_status_cli_test.go
// Tests SHOW interfaces transceiver status

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

func TestShowInterfaceTransceiverStatus(t *testing.T) {
	// Single server reused for all cases
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	conn, err := grpc.Dial(TargetAddr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	gClient := pb.NewGNMIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()

	// Testdata files
	applDbFile := "../testdata/INTERFACE_TRANSCEIVER_STATUS_APPL_PORT_TABLE.txt"
	stateDbFile := "../testdata/INTERFACE_TRANSCEIVER_STATUS_STATE_DB.txt"
	vdmFlagStateDbFile := "../testdata/INTERFACE_TRANSCEIVER_STATUS_VDM_FLAG_STATE_DB.txt"
	configDbFile := "../testdata/INTERFACE_TRANSCEIVER_STATUS_CONFIG_DB.txt"

	notApplicable := "Transceiver status info not applicable\n"

	cmisExpected := map[string]string{
		"CMIS State (SW)":               "READY",
		"Current module state":          "ModuleReady",
		"Temperature high alarm flag":   "False",
		"Temperature high warning flag": "False",
	}
	eth4Expected := map[string]string{
		"Disabled TX channels":          "0",
		"Current module state":          "ModuleReady",
		"Temperature high alarm flag":   "False",
		"Temperature high warning flag": "False",
	}
	ccmisExpected := map[string]string{
		"Current module state":      "ModuleReady",
		"Tuning in progress status": "True",
	}

	type testCase struct {
		desc     string
		path     string
		init     func()
		wantCode codes.Code
		want     map[string]interface{}
	}

	tests := []testCase{
		{
			desc: "all ports no STATE_DB loaded -> all Not Applicable",
			path: `
              elem: <name: "interfaces">
              elem: <name: "transceiver">
              elem: <name: "status">
            `,
			init: func() {
				FlushDataSet(t, ApplDbNum)
				FlushDataSet(t, StateDbNum)
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ApplDbNum, applDbFile)
				AddDataSet(t, ConfigDbNum, configDbFile)
			},
			wantCode: codes.OK,
			want: map[string]interface{}{
				"Ethernet0":  notApplicable,
				"Ethernet4":  notApplicable,
				"Ethernet12": notApplicable,
				"Ethernet16": notApplicable,
			},
		},
		{
			desc: "all ports with STATE_DB -> CMIS, C-CMIS, minimal, unknown",
			path: `
              elem: <name: "interfaces">
              elem: <name: "transceiver">
              elem: <name: "status">
            `,
			init: func() {
				FlushDataSet(t, ApplDbNum)
				FlushDataSet(t, StateDbNum)
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ApplDbNum, applDbFile)
				AddDataSet(t, StateDbNum, stateDbFile)
				AddDataSet(t, ConfigDbNum, configDbFile)
			},
			wantCode: codes.OK,
			want: map[string]interface{}{
				"Ethernet0":  cmisExpected,
				"Ethernet4":  eth4Expected,
				"Ethernet12": notApplicable,
				"Ethernet16": ccmisExpected,
			},
		},
		{
			desc: "single interface Ethernet0 (CMIS with VDM flags -> legacy prefecber/postfecber *_flag fields)",
			path: `
              elem: <name: "interfaces">
              elem: <name: "transceiver">
              elem: <name: "status">
              elem: <name: "Ethernet0">
            `,
			init: func() {
				FlushDataSet(t, ApplDbNum)
				FlushDataSet(t, StateDbNum)
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ApplDbNum, applDbFile)
				AddDataSet(t, StateDbNum, stateDbFile)
				AddDataSet(t, ConfigDbNum, configDbFile)
				AddDataSet(t, StateDbNum, vdmFlagStateDbFile)
			},
			wantCode: codes.OK,
			want: map[string]interface{}{
				"Ethernet0": map[string]string{
					"CMIS State (SW)":               "READY",
					"Current module state":          "ModuleReady",
					"Temperature high alarm flag":   "False",
					"Temperature high warning flag": "False",
					"Prefec ber high alarm flag":    "1e-6",
					"Prefec ber high warning flag":  "2e-6",
					"Prefec ber low alarm flag":     "3e-6",
					"Prefec ber low warning flag":   "4e-6",
					"Postfec ber high alarm flag":   "5",
					"Postfec ber high warning flag": "6",
					"Postfec ber low alarm flag":    "7",
					"Postfec ber low warning flag":  "8",
				},
			},
		},
		{
			desc: "single interface Ethernet12 (unknown keys -> Not Applicable)",
			path: `
              elem: <name: "interfaces">
              elem: <name: "transceiver">
              elem: <name: "status">
              elem: <name: "Ethernet12">
            `,
			init: func() {
				FlushDataSet(t, ApplDbNum)
				FlushDataSet(t, StateDbNum)
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ApplDbNum, applDbFile)
				AddDataSet(t, StateDbNum, stateDbFile)
				AddDataSet(t, ConfigDbNum, configDbFile)
			},
			wantCode: codes.OK,
			want: map[string]interface{}{
				"Ethernet12": notApplicable,
			},
		},
		{
			desc: "single interface Ethernet16 (C-CMIS tuning)",
			path: `
              elem: <name: "interfaces">
              elem: <name: "transceiver">
              elem: <name: "status">
              elem: <name: "Ethernet16">
            `,
			init: func() {
				FlushDataSet(t, ApplDbNum)
				FlushDataSet(t, StateDbNum)
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ApplDbNum, applDbFile)
				AddDataSet(t, StateDbNum, stateDbFile)
				AddDataSet(t, ConfigDbNum, configDbFile)
			},
			wantCode: codes.OK,
			want: map[string]interface{}{
				"Ethernet16": ccmisExpected,
			},
		},
		{
			desc: "alias mode query (fortyGigE0/0) -> Ethernet0 CMIS",
			path: `
              elem: <name: "interfaces">
              elem: <name: "transceiver">
              elem: <name: "status" key: { key: "SONIC_CLI_IFACE_MODE" value: "alias" } >
              elem: <name: "fortyGigE0/0">
            `,
			init: func() {
				FlushDataSet(t, ApplDbNum)
				FlushDataSet(t, StateDbNum)
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ApplDbNum, applDbFile)
				AddDataSet(t, StateDbNum, stateDbFile)
				AddDataSet(t, ConfigDbNum, configDbFile)
			},
			wantCode: codes.OK,
			want: map[string]interface{}{
				"Ethernet0": cmisExpected,
			},
		},
		{
			desc: "alias mode unknown alias -> NotFound",
			path: `
              elem: <name: "interfaces">
              elem: <name: "transceiver">
              elem: <name: "status" key: { key: "SONIC_CLI_IFACE_MODE" value: "alias" } >
              elem: <name: "etp999">
            `,
			init: func() {
				FlushDataSet(t, ApplDbNum)
				AddDataSet(t, ApplDbNum, applDbFile)
			},
			wantCode: codes.NotFound,
		},
		{
			desc: "invalid interface (not in PORT_TABLE) -> NotFound",
			path: `
              elem: <name: "interfaces">
              elem: <name: "transceiver">
              elem: <name: "status">
              elem: <name: "Ethernet999">
            `,
			init: func() {
				FlushDataSet(t, ApplDbNum)
				AddDataSet(t, ApplDbNum, applDbFile)
			},
			wantCode: codes.NotFound,
		},
		{
			desc: "invalid subinterface Ethernet0.10 (not in PORT_TABLE) -> NotFound",
			path: `
              elem: <name: "interfaces">
              elem: <name: "transceiver">
              elem: <name: "status">
              elem: <name: "Ethernet0.10">
            `,
			init: func() {
				FlushDataSet(t, ApplDbNum)
				AddDataSet(t, ApplDbNum, applDbFile)
			},
			wantCode: codes.NotFound,
			want:     nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		if tc.init != nil {
			tc.init()
		}
		t.Run(tc.desc, func(t *testing.T) {
			doGetAndCompare(t, ctx, gClient, tc.path, tc.wantCode, tc.want, notApplicable)
		})
	}
}

func doGetAndCompare(t *testing.T, ctx context.Context, client pb.GNMIClient, textPbPath string,
	wantCode codes.Code, want map[string]interface{}, naSentinel string) {

	var pbPath pb.Path
	if err := proto.UnmarshalText(textPbPath, &pbPath); err != nil {
		t.Fatalf("unmarshal path: %v", err)
	}
	prefix := pb.Path{Target: "SHOW"}
	req := &pb.GetRequest{Prefix: &prefix, Path: []*pb.Path{&pbPath}, Encoding: pb.Encoding_JSON_IETF}

	resp, err := client.Get(ctx, req)
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("non-grpc error: %v", err)
	}
	if st.Code() != wantCode {
		t.Fatalf("got code %v want %v: %v", st.Code(), wantCode, st.Message())
	}

	if want == nil {
		// For NotFound cases we expect no body comparison.
		return
	}

	if wantCode != codes.OK {
		t.Fatalf("unexpected want body with non-OK code %v", wantCode)
	}

	notifs := resp.GetNotification()
	if len(notifs) != 1 {
		t.Fatalf("got %d notifications want 1", len(notifs))
	}
	updates := notifs[0].GetUpdate()
	if len(updates) != 1 {
		t.Fatalf("got %d updates want 1", len(updates))
	}

	val := updates[0].GetVal()
	if val.GetJsonIetfVal() == nil {
		t.Fatalf("expected JsonIetfVal body")
	}

	var outer map[string]interface{}
	if err := json.Unmarshal(val.GetJsonIetfVal(), &outer); err != nil {
		t.Fatalf("unmarshal outer json: %v", err)
	}

	// Check each expected port
	for port, exp := range want {
		gotRaw, present := outer[port]
		if !present {
			t.Fatalf("missing port %s in response", port)
		}

		switch expTyped := exp.(type) {
		case string:
			// Not applicable sentinel
			gotStr, ok := gotRaw.(string)
			if !ok {
				t.Fatalf("port %s expected NA string, got %T", port, gotRaw)
			}
			if gotStr != expTyped {
				t.Errorf("port %s: got %q want %q", port, gotStr, expTyped)
			}
		case map[string]string:
			gotStr, ok := gotRaw.(string)
			if !ok {
				t.Fatalf("port %s expected inner JSON string, got %T", port, gotRaw)
			}
			var inner map[string]string
			if err := json.Unmarshal([]byte(gotStr), &inner); err != nil {
				t.Fatalf("port %s: unmarshal inner json: %v (payload=%s)", port, err, gotStr)
			}

			// Compare keys/values (order independent)
			if len(inner) != len(expTyped) {
				t.Errorf("port %s: inner field count mismatch got=%d want=%d inner=%v", port, len(inner), len(expTyped), inner)
			}
			for k, v := range expTyped {
				gv, ok := inner[k]
				if !ok {
					t.Errorf("port %s: missing field %s", port, k)
					continue
				}
				if gv != v {
					t.Errorf("port %s field %s: got %q want %q", port, k, gv, v)
				}
			}
		default:
			t.Fatalf("unsupported expected type for port %s: %T", port, exp)
		}
	}

	// Ensure no unexpected extra ports
	if len(outer) != len(want) {
		t.Errorf("unexpected port count: got=%d want=%d (outer=%v)", len(outer), len(want), outer)
	}
}
