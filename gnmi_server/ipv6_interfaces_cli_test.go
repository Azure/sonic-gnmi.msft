package gnmi

// Tests SHOW ipv6 interfaces

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

func TestGetIPv6InterfacesCLI(t *testing.T) {
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

	// We rely on show_client.GetMapFromQueries to read fixture files via the test helpers,
	// so no extra dataset is required here. The ipinterfaces path reads DEVICE_METADATA only
	// when resolving namespaces, which is backed by our in-memory data during tests.

	tests := []struct {
		desc        string
		textPbPath  string
		wantRetCode codes.Code
	}{
		{
			desc: "show ipv6 interfaces default",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "interfaces" >
			`,
			wantRetCode: codes.OK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, "SHOW", tc.textPbPath, tc.wantRetCode, nil, false)
		})
	}
}
