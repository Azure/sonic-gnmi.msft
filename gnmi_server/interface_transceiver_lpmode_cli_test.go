package gnmi

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

func TestShowInterfaceTransceiverLpMode(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	// Ensure cache clearing paths active
	MockEnvironmentVariable(t, "UNIT_TEST", "1")

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

	expectedAll := `[{"Port":"Ethernet0","Low-power Mode":"Off"},{"Port":"Ethernet8","Low-power Mode":"Off"},{"Port":"Ethernet16","Low-power Mode":"On"}]`
	expectedOne := `[{"Port":"Ethernet8","Low-power Mode":"Off"}]`

	tests := []struct {
		desc        string
		path        string
		mockFile    string
		wantRetCode codes.Code
		wantRespVal []byte
		valTest     bool
	}{
		{
			desc:     "all ports",
			path: `
				elem: <name: "interfaces" >
				elem: <name: "transceiver" >
				elem: <name: "lpmode" >
			`,
			mockFile:    "../testdata/SFPUTIL_SHOW_LPMODE_ALL.txt",
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedAll),
			valTest:     true,
		},
		{
			desc:     "single existing port",
			path: `
				elem: <name: "interfaces" >
				elem: <name: "transceiver" >
				elem: <name: "lpmode" >
				elem: <name: "Ethernet8" >
			`,
			mockFile:    "../testdata/SFPUTIL_SHOW_LPMODE_Ethernet8.txt",
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedOne),
			valTest:     true,
		},
	}

	for _, tc := range tests {
		patch := MockNSEnterOutput(t, tc.mockFile)
		t.Run(tc.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, "SHOW", tc.path, tc.wantRetCode, tc.wantRespVal, tc.valTest)
		})
		patch.Reset()
	}
}
