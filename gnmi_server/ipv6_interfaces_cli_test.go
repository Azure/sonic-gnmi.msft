package gnmi

// Tests SHOW ipv6 interfaces

import (
	"crypto/tls"
	"encoding/json"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
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

// TestGetIPv6InterfacesCLIShape validates that the response is now a JSON object (map)
// whose values contain the 'ipv6_addresses' field (and not the old 'ip_addresses').
func TestGetIPv6InterfacesCLIShape(t *testing.T) {
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

	var pbPath pb.Path
	if err := proto.UnmarshalText(`
		elem: <name: "ipv6" >
		elem: <name: "interfaces" >
	`, &pbPath); err != nil {
		t.Fatalf("error unmarshaling path: %v", err)
	}
	prefix := pb.Path{Target: "SHOW"}
	req := &pb.GetRequest{Prefix: &prefix, Path: []*pb.Path{&pbPath}, Encoding: pb.Encoding_JSON_IETF}
	resp, err := gClient.Get(ctx, req)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	notifs := resp.GetNotification()
	if len(notifs) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifs))
	}
	updates := notifs[0].GetUpdate()
	if len(updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(updates))
	}
	val := updates[0].GetVal()
	if val.GetJsonIetfVal() == nil {
		t.Fatalf("expected JSON_IETF value, got scalar")
	}
	raw := val.GetJsonIetfVal()
	var decoded interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("failed to unmarshal json: %v", err)
	}
	obj, ok := decoded.(map[string]interface{})
	if !ok {
		t.Fatalf("expected top-level JSON object (map), got %T", decoded)
	}
	// Iterate (if empty map that's acceptable) and ensure each value has ipv6_addresses and not ip_addresses.
	for name, v := range obj {
		entry, ok := v.(map[string]interface{})
		if !ok {
			t.Fatalf("entry %s is not an object: %T", name, v)
		}
		if _, hasOld := entry["ip_addresses"]; hasOld {
			t.Fatalf("entry %s unexpectedly has 'ip_addresses' field", name)
		}
		if _, hasNew := entry["ipv6_addresses"]; !hasNew {
			t.Fatalf("entry %s missing required 'ipv6_addresses' field", name)
		}
	}
}
