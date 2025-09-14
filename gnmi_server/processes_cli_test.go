package gnmi

// processes_cli_test.go
// Tests SHOW processes (root help) and SHOW processes summary|cpu|mem

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

func TestShowProcessesCommands(t *testing.T) {
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
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Seed PROCESS_STATS sample data
	FlushDataSet(t, StateDbNum)
	AddDataSet(t, StateDbNum, "../testdata/PROCESS_STATS_SAMPLE.txt")

	t.Run("SHOW processes (root help)", func(t *testing.T) {
		textPbPath := `
			elem: <name: "processes" >
		`
		expected := []byte(`{"subcommands":{"summary":"show/processes/summary: Show processses info","cpu":"show/processes/cpu: Show processes CPU info","mem":"show/processes/mem: Show processes memory info"}}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, expected, true)
	})

	t.Run("SHOW processes summary", func(t *testing.T) {
		textPbPath := `
			elem: <name: "processes" >
			elem: <name: "summary" >
		`
		expected := []byte(`[{"PID":"123","PPID":"1","CMD":"redis-server","%MEM":"1.2","%CPU":"0.5","STIME":"10:54","TIME":"00:00:42","TT":"?","UID":"999"},{"PID":"456","PPID":"1","CMD":"swss","%MEM":"3.4","%CPU":"15.0","STIME":"10:55","TIME":"00:12:05","TT":"pts/0","UID":"0"},{"PID":"789","PPID":"456","CMD":"orchagent","%MEM":"2.0","%CPU":"7.5","STIME":"10:56","TIME":"00:03:10","TT":"pts/1","UID":"0"}]`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, expected, true)
	})
}
