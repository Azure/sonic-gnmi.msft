package gnmi

import (
	"crypto/tls"
	"os"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestGetQueuePersistentWatermarks(t *testing.T) {
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

	portsFileName := "../testdata/PORTS.txt"
	portTableFileName := "../testdata/PORT_TABLE.txt"
	queueOidMappingFileName := "../testdata/QUEUE_OID_MAPPING.txt"
	queueTypeMappingFileName := "../testdata/QUEUE_TYPE_MAPPING.txt"
	queuePersistentWatermarksFileName := "../testdata/QUEUE_PERSISTENT_WATERMARKS.txt"
	rootQueuePersistentWatermarksHelp := []byte(`{"subcommands": {"all": "show/queue/persistent-watermark/all", "unicast": "show/queue/persistent-watermark/unicast", "multicast": "show/queue/persistent-watermark/multicast"}}`)
	allQueuePersistentWatermarksAllPorts, err := os.ReadFile("../testdata/QUEUE_PERSISTENT_WATERMARKS_RESULTS_ALL.txt")
	if err != nil {
		t.Fatalf("Failed to read expected query results for persistent queue watermarks of all interfaces: %v", err)
	}
	unicastQueuePersistentWatermarksAllPorts, err := os.ReadFile("../testdata/QUEUE_PERSISTENT_WATERMARKS_RESULTS_UNICAST.txt")
	if err != nil {
		t.Fatalf("Failed to read expected query results for unicast persistent queue watermarks of all interfaces: %v", err)
	}
	multicastQueuePersistentWatermarksAllPorts, err := os.ReadFile("../testdata/QUEUE_PERSISTENT_WATERMARKS_RESULTS_MULTICAST.txt")
	if err != nil {
		t.Fatalf("Failed to read expected query results for multicast persistent queue watermarks of all interfaces: %v", err)
	}
	allQueuePersistentWatermarksEth0 := []byte(`{"Ethernet0": {"UC0": "100", "UC1": "0", "MC2": "200"}}`)
	unicastQueuePersistentWatermarksEth40 := []byte(`{"Ethernet40": {"UC0": "300", "UC1": "400"}}`)
	multicastQueuePersistentWatermarksEth80 := []byte(`{"Ethernet80": {"MC2": "0"}}`)
	allQueuePersistentWatermarksEth0And40 := []byte(`{"Ethernet0": {"UC0": "100", "UC1": "0", "MC2": "200"}, "Ethernet40": {"UC0": "300", "UC1": "400", "MC2": "N/A"}}`)

	ResetDataSetsAndMappings(t)

	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal interface{}
		valTest     bool
		testInit    func()
	}{
		{
			desc:       "query SHOW queue persistent-watermark (root help)",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "queue" >
				elem: <name: "persistent-watermark" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: rootQueuePersistentWatermarksHelp,
			valTest:     true,
		},
		{
			desc:       "query SHOW queue persistent-watermark all NO DATA",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "queue" >
				elem: <name: "persistent-watermark" >
				elem: <name: "all" >
			`,
			wantRetCode: codes.OK,
		},
		{
			desc:       "query SHOW queue persistent-watermark unicast NO DATA",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "queue" >
				elem: <name: "persistent-watermark" >
				elem: <name: "unicast" >
			`,
			wantRetCode: codes.OK,
		},
		{
			desc:       "query SHOW queue persistent-watermark multicast NO DATA",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "queue" >
				elem: <name: "persistent-watermark" >
				elem: <name: "multicast" >
			`,
			wantRetCode: codes.OK,
		},
		{
			desc:       "query SHOW queue persistent-watermark all queue types for all interfaces",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "queue" >
				elem: <name: "persistent-watermark" >
				elem: <name: "all" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: allQueuePersistentWatermarksAllPorts,
			valTest:     true,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, portsFileName)
				AddDataSet(t, ApplDbNum, portTableFileName)
				AddDataSet(t, CountersDbNum, queueOidMappingFileName)
				AddDataSet(t, CountersDbNum, queueTypeMappingFileName)
				AddDataSet(t, CountersDbNum, queuePersistentWatermarksFileName)
			},
		},
		{
			desc:       "query SHOW queue persistent-watermark unicast for all interfaces",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "queue" >
				elem: <name: "persistent-watermark" >
				elem: <name: "unicast" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: unicastQueuePersistentWatermarksAllPorts,
			valTest:     true,
		},
		{
			desc:       "query SHOW queue persistent-watermark multicast for all interfaces",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "queue" >
				elem: <name: "persistent-watermark" >
				elem: <name: "multicast" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: multicastQueuePersistentWatermarksAllPorts,
			valTest:     true,
		},
		{
			desc:       "query SHOW queue persistent-watermark all for Ethernet0",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "queue" >
				elem: <name: "persistent-watermark" >
				elem: <name: "all" key: { key: "interfaces" value: "Ethernet0" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: allQueuePersistentWatermarksEth0,
			valTest:     true,
		},
		{
			desc:       "query SHOW queue persistent-watermark unicast for Ethernet40",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "queue" >
				elem: <name: "persistent-watermark" >
				elem: <name: "unicast" key: { key: "interfaces" value: "Ethernet40" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: unicastQueuePersistentWatermarksEth40,
			valTest:     true,
		},
		{
			desc:       "query SHOW queue persistent-watermark multicast for Ethernet80",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "queue" >
				elem: <name: "persistent-watermark" >
				elem: <name: "multicast" key: { key: "interfaces" value: "Ethernet80" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: multicastQueuePersistentWatermarksEth80,
			valTest:     true,
		},
		{
			desc:       "query SHOW queue persistent-watermark all for Ethernet0 and Ethernet40",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "queue" >
				elem: <name: "persistent-watermark" >
				elem: <name: "all" key: { key: "interfaces" value: "Ethernet0,Ethernet40" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: allQueuePersistentWatermarksEth0And40,
			valTest:     true,
		},
		// Test cases for invalid requests
		{
			desc:       "query SHOW queue persistent-watermark all for an invalid interface",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "queue" >
				elem: <name: "persistent-watermark" >
				elem: <name: "all" key: { key: "interfaces" value: "Ethernet7" } >
			`,
			wantRetCode: codes.NotFound,
		},
	}

	for _, test := range tests {
		if test.testInit != nil {
			test.testInit()
		}
		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})
	}
}
