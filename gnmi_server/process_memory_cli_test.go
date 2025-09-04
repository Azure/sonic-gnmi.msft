package gnmi

import (
	"crypto/tls"
	"testing"
	"time"
	"fmt"

	pb "github.com/openconfig/gnmi/proto/gnmi"
	show_client "github.com/sonic-net/sonic-gnmi/show_client"

	"github.com/agiledragon/gomonkey/v2"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestGetTopMemoryUsage(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()

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

	expectedTopMemory := `
	top - 15:02:01 up 3 days,  4:12,  1 user,  load average: 0.00, 0.01, 0.05
	Tasks: 123 total,   1 running, 122 sleeping,   0 stopped,   0 zombie
	%Cpu(s): 1.0 us,  0.5 sy,  0.0 ni, 98.0 id,  0.5 wa,  0.0 hi,  0.0 si,  0.0 st
	MiB Mem :  7989.3 total,   1234.5 free,   2345.6 used,   3409.2 buff/cache
	MiB Swap:  2048.0 total,   2048.0 free,      0.0 used.   4567.8 avail Mem
	PID USER      PR  NI    VIRT    RES    SHR S  %CPU %MEM     TIME+ COMMAND
	1234 root      20   0  123456   65432   1234 S   0.3  5.2   0:01.23 myapp
	5678 daemon    20   0  234567   54321   2345 S   0.1  4.8   0:00.98 anotherapp
	`

	ResetDataSetsAndMappings(t)

	tests := []struct {
		desc           string
		pathTarget     string
		textPbPath     string
		wantRetCode    codes.Code
		wantRespVal    interface{}
		valTest        bool
		mockOutputFile string
		testInit       func() *gomonkey.Patches
	}{
		{
			desc:       "query show memory-usage with success case",
			pathTarget: "SHOW",
			textPbPath: `
			elem: <name: "memory-usage" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: expectedTopMemory,
			valTest:     true,
			mockOutputFile: "../testdata/TOP_MEMORY.txt",
		},
		{
			desc:       "query show memory-usage with blank output",
			pathTarget: "SHOW",
			textPbPath: `
			elem: <name: "memory-usage" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: "",
			valTest:     false,
			testInit: func() *gomonkey.Patches {
				return gomonkey.ApplyFunc(show_client.GetDataFromHostCommand, func(cmd string) (string, error) {
					return "", nil
				})
			},
		},
		{
			desc:       "query show memory-usage with error from command",
			pathTarget: "SHOW",
			textPbPath: `
			elem: <name: "memory-usage" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: "",
			valTest:     false,
			testInit: func() *gomonkey.Patches {
				return gomonkey.ApplyFunc(show_client.GetDataFromHostCommand, func(cmd string) (string, error) {
					return "", fmt.Errorf("simulated command failure")
				})
			},
		},
	}

	for _, test := range tests {
		var patch1, patch2 *gomonkey.Patches
		if test.testInit != nil {
			patch1 = test.testInit()
		}

		if len(test.mockOutputFile) > 0 {
			patch2 = MockNSEnterOutput(t, test.mockOutputFile)
		}

		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})

		if patch1 != nil {
			patch1.Reset()
		}
		if patch2 != nil {
			patch2.Reset()
		}
	}
}
