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

func TestShowBufferPoolWatermarks(t *testing.T) {
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
	userWatermarksFile := "../testdata/USER_WATERMARKS:BUFFER_POOL.txt"
	persistentWatermarksFile := "../testdata/PERSISTENT_WATERMARKS:BUFFER_POOL.txt"

	// populate COUNTERS_DB
	t.Logf("Loading datasets into COUNTERS_DB: %s, %s, %s", poolsMapFile, userWatermarksFile, persistentWatermarksFile)
	AddDataSet(t, CountersDbNum, poolsMapFile)
	AddDataSet(t, CountersDbNum, userWatermarksFile)
	AddDataSet(t, CountersDbNum, persistentWatermarksFile)

	// SHOW buffer_pool watermark
	t.Run("query SHOW buffer_pool watermark", func(t *testing.T) {
		textPbPath := `
			elem: <name: "buffer_pool" >
			elem: <name: "watermark" >
		`
		t.Log("Sending GET for SHOW/buffer_pool/watermark (user watermarks)")
		// Expected JSON (order not guaranteed, DeepEqual done on map) matching USER_WATERMARKS testdata
		expectedJSON := `{"egress_lossless_pool":{"Bytes":"12345"},"egress_lossy_pool":{"Bytes":"67890"},"ingress_lossless_pool":{"Bytes":"24680"}}`
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, []byte(expectedJSON), true)
	})

	// SHOW buffer_pool persistent-watermark
	t.Run("query SHOW buffer_pool persistent-watermark", func(t *testing.T) {
		textPbPath := `
			elem: <name: "buffer_pool" >
			elem: <name: "persistent-watermark" >
		`
		t.Log("Sending GET for SHOW/buffer_pool/persistent-watermark (persistent watermarks)")
		// Expected JSON matches PERSISTENT_WATERMARKS testdata values
		expectedJSON := `{"egress_lossless_pool":{"Bytes":"9216"},"egress_lossy_pool":{"Bytes":"6656"},"ingress_lossless_pool":{"Bytes":"40192"}}`
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, []byte(expectedJSON), true)
	})
}

// Covers: name-map invalid (no oid: values) -> loadBufferPoolNameMap returns error -> gRPC NotFound
func TestShowBufferPoolWatermarks_NameMapInvalid(t *testing.T) {
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

	// Only load an invalid COUNTERS_BUFFER_POOL_NAME_MAP (no oid: prefixes)
	badPoolsMapFile := "../testdata/COUNTERS_BUFFER_POOL_NAME_MAP_BAD.txt"
	t.Logf("Loading dataset into COUNTERS_DB: %s", badPoolsMapFile)
	AddDataSet(t, CountersDbNum, badPoolsMapFile)

	textPbPath := `
		elem: <name: "buffer_pool" >
		elem: <name: "watermark" >
	`
	t.Log("Sending GET for SHOW/buffer_pool/watermark (invalid name map -> NotFound)")
	runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
}

// Covers: per-pool empty hash (len(data)==0) -> Bytes = N/A
func TestShowBufferPoolWatermarks_EmptyHash(t *testing.T) {
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

	// Load valid name map and a USER_WATERMARKS file with one empty hash
	poolsMapFile := "../testdata/COUNTERS_BUFFER_POOL_NAME_MAP.txt"
	userWatermarksEmptyFile := "../testdata/USER_WATERMARKS_EMPTY:BUFFER_POOL.txt"
	t.Logf("Loading datasets into COUNTERS_DB: %s, %s", poolsMapFile, userWatermarksEmptyFile)
	AddDataSet(t, CountersDbNum, poolsMapFile)
	AddDataSet(t, CountersDbNum, userWatermarksEmptyFile)

	textPbPath := `
		elem: <name: "buffer_pool" >
		elem: <name: "watermark" >
	`
	t.Log("Sending GET for SHOW/buffer_pool/watermark (one pool empty hash)")
	// egress_lossless_pool -> empty {}; others have values
	expectedJSON := `{"egress_lossless_pool":{"Bytes":"N/A"},"egress_lossy_pool":{"Bytes":"67890"},"ingress_lossless_pool":{"Bytes":"24680"}}`
	runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, []byte(expectedJSON), true)
}

// Covers error branch where the expected watermark field is missing in a hash.
func TestShowBufferPoolWatermarks_MissingField(t *testing.T) {
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

	// Test data files: valid pool map + USER watermark where one pool misses the expected field
	poolsMapFile := "../testdata/COUNTERS_BUFFER_POOL_NAME_MAP.txt"
	userWatermarksMissingFieldFile := "../testdata/USER_WATERMARKS_MISSING_FIELD:BUFFER_POOL.txt"

	// populate COUNTERS_DB
	t.Logf("Loading datasets into COUNTERS_DB: %s, %s", poolsMapFile, userWatermarksMissingFieldFile)
	AddDataSet(t, CountersDbNum, poolsMapFile)
	AddDataSet(t, CountersDbNum, userWatermarksMissingFieldFile)

	// SHOW buffer_pool watermark should return N/A for the pool missing the field
	t.Run("query SHOW buffer_pool watermark with missing field", func(t *testing.T) {
		textPbPath := `
			elem: <name: "buffer_pool" >
			elem: <name: "watermark" >
		`
		t.Log("Sending GET for SHOW/buffer_pool/watermark (user watermarks, missing field)")
		expectedJSON := `{"egress_lossless_pool":{"Bytes":"N/A"},"egress_lossy_pool":{"Bytes":"67890"},"ingress_lossless_pool":{"Bytes":"24680"}}`
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, []byte(expectedJSON), true)
	})
}
