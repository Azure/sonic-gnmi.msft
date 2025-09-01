package gnmi

// Tests SHOW system-memory

import (
	"crypto/tls"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"

	"github.com/agiledragon/gomonkey/v2"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestGetSystemMemory(t *testing.T) {
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

	systemMemoryDefault := `[{"type":"Mem","total":"64252","used":"15017","free":"7526","shared":"563","buff/cache":"41708","available":"46228"},{"type":"Swap","total":"0","used":"0","free":"0"}]`

	ResetDataSetsAndMappings(t)

	tests := []struct {
		desc           string
		pathTarget     string
		textPbPath     string
		wantRetCode    codes.Code
		wantRespVal    interface{}
		valTest        bool
		mockOutputFile string
		testInit       func()
	}{
		{
			desc:       "query SHOW system-memory read error",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "system-memory" >
			`,
			wantRetCode: codes.NotFound,
		},
		{
			desc:       "query SHOW system-memory malformed cmd output",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "system-memory" >
			`,
			wantRetCode:    codes.NotFound,
			mockOutputFile: "../testdata/SYSTEM_MEMORY_MALFORMED.txt",
		},
		{
			desc:       "query SHOW system-memory",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "system-memory" >
			`,
			wantRetCode:    codes.OK,
			wantRespVal:    []byte(systemMemoryDefault),
			valTest:        true,
			mockOutputFile: "../testdata/SYSTEM_MEMORY.txt",
			testInit: func() {
			},
		},
	}

	for _, test := range tests {
		if test.testInit != nil {
			test.testInit()
		}
		var patches *gomonkey.Patches
		if test.mockOutputFile != "" {
			patches = MockNSEnterOutput(t, test.mockOutputFile)
		}

		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})
		if patches != nil {
			patches.Reset()
		}
	}
}
