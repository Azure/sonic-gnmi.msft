package gnmi

import (
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"
	sccommon "github.com/sonic-net/sonic-gnmi/show_client/common"

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
        {
                "uptime": "15:02:01 up 3 days,  4:12,  1 user,  load average: 0.00, 0.01, 0.05",
                "tasks": "123 total,   1 running, 122 sleeping,   0 stopped,   0 zombie",
                "cpu_usage": "1.0 us,  0.5 sy,  0.0 ni, 98.0 id,  0.5 wa,  0.0 hi,  0.0 si,  0.0 st",
                "memory_usage": "7989.3 total,   1234.5 free,   2345.6 used,   3409.2 buff/cache",
                "swap_usage": "2048.0 total,   2048.0 free,      0.0 used.   4567.8 avail Mem",
                "processes": [
                {
                        "pid": "1234",
                        "user": "root",
                        "pr": "20",
                        "ni": "0",
                        "virt": "123456",
                        "res": "65432",
                        "shr": "1234",
                        "s": "S",
                        "cpu": "0.3",
                        "mem": "5.2",
                        "time": "0:01.23",
                        "command": "myapp"
                },
                {
                        "pid": "5678",
                        "user": "daemon",
                        "pr": "20",
                        "ni": "0",
                        "virt": "234567",
                        "res": "54321",
                        "shr": "2345",
                        "s": "S",
                        "cpu": "0.1",
                        "mem": "4.8",
                        "time": "0:00.98",
                        "command": "anotherapp"
                }
                ]
        }
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
                        elem: <name: "processes" >
                        elem: <name: "memory" >
                        `,
			wantRetCode:    codes.OK,
			wantRespVal:    []byte(expectedTopMemory),
			valTest:        true,
			mockOutputFile: "../testdata/PROCESS_MEMORY.txt",
		},
		{
			desc:       "query show memory-usage with blank output",
			pathTarget: "SHOW",
			textPbPath: `
                        elem: <name: "processes" >
                        elem: <name: "memory" >
                        `,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() *gomonkey.Patches {
				return gomonkey.ApplyFunc(sccommon.GetDataFromHostCommand, func(cmd string) (string, error) {
					return "", nil
				})
			},
		},
		{
			desc:       "query show memory-usage with error from command",
			pathTarget: "SHOW",
			textPbPath: `
                        elem: <name: "processes" >
                        elem: <name: "memory" >
                        `,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() *gomonkey.Patches {
				return gomonkey.ApplyFunc(sccommon.GetDataFromHostCommand, func(cmd string) (string, error) {
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
