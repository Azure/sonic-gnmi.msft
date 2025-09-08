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

func TestShowHeadroomPoolWatermarks(t *testing.T) {
	s := createServer(t, ServerPort)
	t.Logf("Starting GNMI test server on port %d (target %s)", ServerPort, TargetAddr)
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

	// Test data files (USER + PERSISTENT watermarks)
	poolsMapFile := "../testdata/COUNTERS_BUFFER_POOL_NAME_MAP.txt"
	userWatermarksFile := "../testdata/USER_WATERMARKS:HEADROOM_POOL.txt"
	persistentWatermarksFile := "../testdata/PERSISTENT_WATERMARKS:HEADROOM_POOL.txt"

	// populate COUNTERS_DB
	t.Logf("Loading datasets into COUNTERS_DB: %s, %s, %s", poolsMapFile, userWatermarksFile, persistentWatermarksFile)
	AddDataSet(t, CountersDbNum, poolsMapFile)
	AddDataSet(t, CountersDbNum, userWatermarksFile)
	AddDataSet(t, CountersDbNum, persistentWatermarksFile)

	// SHOW headroom-pool watermark
	t.Run("query SHOW headroom-pool watermark", func(t *testing.T) {
		textPbPath := `
            elem: <name: "headroom-pool" >
            elem: <name: "watermark" >
        `
		t.Log("Sending GET for SHOW/headroom-pool/watermark (user watermarks)")
		// Only ingress_lossless_pool should be included
		expectedJSON := `{"ingress_lossless_pool":{"Bytes":"3333"}}`
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, []byte(expectedJSON), true)
	})

	// SHOW headroom-pool persistent-watermark
	t.Run("query SHOW headroom-pool persistent-watermark", func(t *testing.T) {
		textPbPath := `
            elem: <name: "headroom-pool" >
            elem: <name: "persistent-watermark" >
        `
		t.Log("Sending GET for SHOW/headroom-pool/persistent-watermark (persistent watermarks)")
		expectedJSON := `{"ingress_lossless_pool":{"Bytes":"9999"}}`
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, []byte(expectedJSON), true)
	})
}
