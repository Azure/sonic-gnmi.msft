package gnmi

import (
	"crypto/tls"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestGetShowProcessesCPU(t *testing.T) {
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

	expectedRetValue := `
    {
                "uptime": "05:54:44 up 1 day, 20:50,  1 user,  load average: 0.78, 1.25, 1.45",
                "tasks": "394 total,   2 running, 386 sleeping,   0 stopped,   6 zombie",
                "cpu_usage": "5.9 us,  5.9 sy,  0.0 ni, 88.2 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st",
                "memory_usage": "31905.5 total,  21836.2 free,   7214.5 used,   3777.4 buff/cache",
                "swap_usage": "0.0 total,      0.0 free,      0.0 used.  24691.0 avail Mem",
                "processes": [
                {
                        "pid": "5010",
                        "user": "root",
                        "pr": "20",
                        "ni": "0",
                        "virt": "4688440",
                        "res": "1.5g",
                        "shr": "632704",
                        "s": "S",
                        "cpu": "106.2",
                        "mem": "4.8",
                        "time": "37,31",
                        "command": "syncd"
                },
                {
                        "pid": "18922",
                        "user": "300",
                        "pr": "20",
                        "ni": "0",
                        "virt": "970244",
                        "res": "756808",
                        "shr": "8752",
                        "s": "S",
                        "cpu": "6.2",
                        "mem": "2.3",
                        "time": "34:49.50",
                        "command": "bgpd"
                }
                ]
        }
`
	tests := []struct {
		desc           string
		pathTarget     string
		textPbPath     string
		wantRetCode    codes.Code
		wantRespVal    interface{}
		valTest        bool
		mockOutputFile map[string]string
		testInit       func()
	}{
		{
			desc:       "query SHOW processes cpu",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "processes" >
                elem: <name: "cpu" >
            `,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedRetValue),
			valTest:     false,
			mockOutputFile: map[string]string{
				"top": "../testdata/PROCESSES_CPU.txt",
			},
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
			},
		},
	}

	for _, test := range tests {
		if test.testInit != nil {
			test.testInit()
		}

		var patches *gomonkey.Patches
		if len(test.mockOutputFile) > 0 {
			patches = MockExecCmds(t, test.mockOutputFile)
		}

		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})
		if patches != nil {
			patches.Reset()
		}
	}
}
