package gnmi

// interface_naming_mode_cli_test.go
// Tests SHOW interface/naming_mode

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

func TestGetShowInterfaceNamingMode(t *testing.T) {
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

	expectedDefault := `{"naming_mode":"default"}`
	expectedAlias := `{"naming_mode":"alias"}`

	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal interface{}
		valTest     bool
	}{
		{
			desc:       "query SHOW interfaces naming_mode (default)",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "naming_mode" >
            `,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedDefault),
			valTest:     true,
		},
		{
			desc:       "query SHOW interfaces naming_mode with SONIC_CLI_IFACE_MODE=alias option",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "naming_mode" key: { key: "SONIC_CLI_IFACE_MODE" value: "alias" } >
            `,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedAlias),
			valTest:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})
	}
}
