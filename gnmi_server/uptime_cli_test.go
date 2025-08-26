package gnmi

import (
	"crypto/tls"
	"errors"
	"os"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

func TestGetUptime(t *testing.T) {
	cleanup := saveEnv("PLATFORM")
	t.Cleanup(cleanup)

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

	expectedUptime := `
    {
        "uptime": "up 3 weeks, 4 days, 10 hours, 15 minutes"
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
		mockOutputFile map[string]string
		testInit       func()
	}{
		{
			desc:       "query show uptime with success case",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "version" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedUptime),
			valTest:     true,
			mockOutputFile: map[string]string{
				"uptime": "../testdata/UPTIME.txt",
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
