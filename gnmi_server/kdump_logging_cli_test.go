package gnmi

// kdump_logging_cli_test.go

// Tests SHOW kdump logging

import (
	"crypto/tls"
	"fmt"
	"strings"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"

	"github.com/agiledragon/gomonkey/v2"

	"github.com/sonic-net/sonic-gnmi/show_client/common"
)

func mockKdumpLoggingCommands(dmesgOutput, tailOutput string) *gomonkey.Patches {
	return gomonkey.ApplyFunc(common.GetDataFromHostCommand, func(cmd string) (string, error) {
		if cmd == "find /var/crash -name 'dmesg.*'" {
			return dmesgOutput, nil
		} else if strings.HasPrefix(cmd, "test -f") {
			return "", nil
		} else if strings.HasPrefix(cmd, "sudo tail") {
			return tailOutput, nil
		}
		return "", fmt.Errorf("unknown command")
	})
}

// for file not found cases
func mockKdumpLoggingFileNotFound(dmesgOutput string) *gomonkey.Patches {
	return gomonkey.ApplyFunc(common.GetDataFromHostCommand, func(cmd string) (string, error) {
		if cmd == "find /var/crash -name 'dmesg.*'" {
			return dmesgOutput, nil
		} else if strings.HasPrefix(cmd, "test -f") {
			return "", fmt.Errorf("file not found")
		} else if strings.HasPrefix(cmd, "sudo tail") {
			return "", fmt.Errorf("file not found")
		}
		return "", fmt.Errorf("unknown command")
	})
}

func TestGetShowKdumpLogging(t *testing.T) {
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

	// expected JSON outputs
	kdumpLoggingLatestFileExpected := `{"logs":["log line 1","log line 2","log line 3","log line 4","log line 5","log line 6","log line 7","log line 8","log line 9","log line 10"]}`
	kdumpLoggingSpecificFileExpected := `{"logs":["specific file line 1","specific file line 2","specific file line 3","specific file line 4","specific file line 5","specific file line 6","specific file line 7","specific file line 8","specific file line 9","specific file line 10"]}`
	kdumpLoggingEmptyExpected := `{"logs":[]}`
	kdumpLoggingNoDmesgExpected := `{"logs":["No kernel dmesg file available!"]}`
	kdumpLoggingWithLinesExpected := `{"logs":["log line 1","log line 2"]}`
	kdumpLoggingFileWithLinesExpected := `{"logs":["specific file line 1","specific file line 2","specific file line 3"]}`

	// command outputs
	dmesgOutputLatestFile := "/var/crash/202411101200/dmesg.202411101200"
	tailOutputLatestFile := "log line 1\nlog line 2\nlog line 3\nlog line 4\nlog line 5\nlog line 6\nlog line 7\nlog line 8\nlog line 9\nlog line 10"
	tailOutputSpecificFile := "specific file line 1\nspecific file line 2\nspecific file line 3\nspecific file line 4\nspecific file line 5\nspecific file line 6\nspecific file line 7\nspecific file line 8\nspecific file line 9\nspecific file line 10"
	tailOutputWithLines := "log line 1\nlog line 2"
	tailOutputFileWithLines := "specific file line 1\nspecific file line 2\nspecific file line 3"
	emptyOutput := ""

	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal []byte
		valTest     bool
		testInit    func() *gomonkey.Patches
	}{
		{
			desc:       "query SHOW kdump logging default latest file",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "kdump" >
				elem: <name: "logging" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(kdumpLoggingLatestFileExpected),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				return mockKdumpLoggingCommands(dmesgOutputLatestFile, tailOutputLatestFile)
			},
		},
		{
			desc:       "query SHOW kdump logging specific file",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "kdump" >
				elem: <name: "logging" >
				elem: <name: "dmesg.202411101100" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(kdumpLoggingSpecificFileExpected),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				return mockKdumpLoggingCommands(emptyOutput, tailOutputSpecificFile)
			},
		},
		{
			desc:       "query SHOW kdump logging with lines=2",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "kdump" >
				elem: <name: "logging"  key: <key: "lines" value: "2"> >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(kdumpLoggingWithLinesExpected),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				return mockKdumpLoggingCommands(dmesgOutputLatestFile, tailOutputWithLines)
			},
		},
		{
			desc:       "query SHOW kdump logging specific file with lines=3",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "kdump" >
				elem: <name: "logging"  key: <key: "lines" value: "3"> >
				elem: <name: "dmesg.202411101100" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(kdumpLoggingFileWithLinesExpected),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				return mockKdumpLoggingCommands(emptyOutput, tailOutputFileWithLines)
			},
		},
		{
			desc:       "query SHOW kdump logging no dmesg files",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "kdump" >
				elem: <name: "logging" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(kdumpLoggingNoDmesgExpected),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				return mockKdumpLoggingFileNotFound(emptyOutput)
			},
		},
		{
			desc:       "query SHOW kdump logging file not found",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "kdump" >
				elem: <name: "logging" >
				elem: <name: "invalidfile" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			testInit: func() *gomonkey.Patches {
				return mockKdumpLoggingFileNotFound(emptyOutput)
			},
		},
		{
			desc:       "query SHOW kdump logging empty output",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "kdump" >
				elem: <name: "logging" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(kdumpLoggingEmptyExpected),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				return mockKdumpLoggingCommands(dmesgOutputLatestFile, emptyOutput)
			},
		},
	}

	for _, test := range tests {
		var patch *gomonkey.Patches
		if test.testInit != nil {
			patch = test.testInit()
		}

		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})

		if patch != nil {
			patch.Reset()
		}
	}
}
