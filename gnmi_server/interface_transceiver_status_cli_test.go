package gnmi

// interface_transceiver_status_cli_test.go
// Tests SHOW interfaces transceiver status

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

// helper to marshal expected map
func mustJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal expected json: %v", err)
	}
	return b
}

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
	configDbFile := "../testdata/INTERFACE_TRANSCEIVER_STATUS_CONFIG_DB.txt"

	notApplicable := "Transceiver status info not applicable\n"

	cmisOut := "\n" +
		"        CMIS State (SW): READY\n" +
		"        Current module state: ModuleReady\n" +
		"        Temperature high alarm flag: False\n" +
		"        Temperature high warning flag: False\n"

	ccmisOut := "\n" +
		"        Current module state: ModuleReady\n" +
		"        Tuning in progress status: True\n"

	tests := []struct {
		desc        string
		path        string
		init        func()
		wantCode    codes.Code
		wantJSONMap map[string]string
	}{
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
			wantJSONMap: map[string]string{
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
			wantJSONMap: map[string]string{
				"Ethernet0": cmisOut,
				"Ethernet4": "\n" +
					"        Disabled TX channels: 0\n" +
					"        Current module state: ModuleReady\n" +
					"        Temperature high alarm flag: False\n" +
					"        Temperature high warning flag: False\n",
				"Ethernet12": notApplicable,
				"Ethernet16": ccmisOut,
			},
		},
		{
			desc: "single interface Ethernet0 (CMIS)",
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
			},
			wantCode: codes.OK,
			wantJSONMap: map[string]string{
				"Ethernet0": cmisOut,
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
			wantJSONMap: map[string]string{
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
			wantJSONMap: map[string]string{
				"Ethernet16": ccmisOut,
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
			wantJSONMap: map[string]string{
				"Ethernet0": cmisOut,
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
	}

	for _, tc := range tests {
		tc := tc
		if tc.init != nil {
			tc.init()
		}
		t.Run(tc.desc, func(t *testing.T) {
			var want interface{}
			expectBody := tc.wantJSONMap != nil
			if expectBody {
				want = mustJSON(t, tc.wantJSONMap)
			}
			runTestGet(t, ctx, gClient, "SHOW", tc.path, tc.wantCode, want, expectBody)
		})
	}
}
