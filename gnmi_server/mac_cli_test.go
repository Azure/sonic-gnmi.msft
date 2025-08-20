package gnmi

// mac_cli_test.go

// Tests SHOW mac

import (
	"crypto/tls"
	"time"
	show_client "github.com/sonic-net/sonic-gnmi/show_client"
	"io/ioutil"
	"path/filepath"
	"testing"
	 pb "github.com/openconfig/gnmi/proto/gnmi"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)


func TestShowMac(t *testing.T) {
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

	showMacOutputFileName := "../testdata/showMacOutput.txt"
	patches := MockNSEnterBGPSummary(t, showMacOutputFileName)

	t.Logf("Start to test 'SHOW mac'")
	t.Run("query SHOW mac", func(t *testing.T){
		textPbPath := `
			elem: <name: "mac" >
		`
		wantRespVal := []byte(`[
														{"macAddress":"04:27:28:1B:9A:A1","port":"Ethernet44","type":"Dynamic","vlan":"19"},
														{"macAddress":"04:27:28:1B:98:A8","port":"Ethernet104","type":"Dynamic","vlan":"19"},
														{"macAddress":"00:22:48:A8:6F:66","port":"Ethernet60","type":"Dynamic","vlan":"18"}
													]`)
		t.Log("Sending GET for SHOW/mac")
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	t.Logf("Start to test 'SHOW mac -c'")
	t.Run("query SHOW mac -c", func(t *testing.T){
			textPbPath := `
				elem: <name: "mac"  key: { key: "count" value: "True" } >
			`
			wantRespVal := []byte(`{"count": 3}`)
			t.Log("Sending GET for SHOW/mac -c")
			runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
		})

	patches.Reset()
}


func TestParseFdbshowTable(t *testing.T) {
	t.Logf("Start to test TestParseFdbshowTable")
	testDataFilePath := filepath.Join("testdata/showMacOutput.txt")
	content, _ := ioutil.ReadFile(testDataFilePath)
	entries, err := show_client.ParseFdbshowTable(string(content))
	if err != nil {
		t.Fatalf("Failed to parse FDB show table: %v", err)
	}
	t.Logf("Parsed entries: %d", len(entries))

	expectedLen := 3
	if len(entries) != expectedLen {
		t.Errorf("Expected number of entries: %d, got: %d", expectedLen, len(entries))
	}

	if entries[0].Vlan != "19" || entries[0].Port != "Ethernet44" || entries[0].MacAddress != "04:27:28:1B:9A:A1" || entries[0].Type != "Dynamic" {
		t.Errorf("Expected entry: {Vlan: %q, Port: %q, MacAddress: %q, Type: %q}, got: {Vlan: %q, Port: %q, MacAddress: %q, Type: %q}",
			"19", "Ethernet44", "04:27:28:1B:9A:A1", "Dynamic",
			entries[0].Vlan, entries[0].Port, entries[0].MacAddress, entries[0].Type)
	}

	if entries[2].Vlan != "18" || entries[2].Port != "Ethernet60" || entries[2].MacAddress != "00:22:48:A8:6F:66" || entries[2].Type != "Dynamic" {
		t.Errorf("Expected entry: {Vlan: %q, Port: %q, MacAddress: %q, Type: %q}, got: {Vlan: %q, Port: %q, MacAddress: %q, Type: %q}",
			"18", "Ethernet60", "00:22:48:A8:6F:66", "Dynamic",
			entries[2].Vlan, entries[2].Port, entries[2].MacAddress, entries[2].Type)
	}
}

func TestParseFdbshowCount(t *testing.T) {
	t.Logf("Start to test TestParseFdbshowCount")
	content := "Total number of entries 120"
	count, err := show_client.ParseFdbshowCount(content)
	if err != nil {
		t.Fatalf("Failed to parse FDB show count: %v", err)
	}
	t.Logf("Parsed count: %d", count)
	expectedLen := 3
	if count != expectedLen {
		t.Errorf("Expected count: %d, got: %d", expectedLen, count)
	}
}