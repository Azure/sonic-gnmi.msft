package gnmi

// kdump_files_cli_test.go

// Tests SHOW kdump files

import (
	"crypto/tls"
	"fmt"
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

func mockKdumpFilesCommands(kdumpOutput, dmesgOutput string) *gomonkey.Patches {
	return gomonkey.ApplyFunc(common.GetDataFromHostCommand, func(cmd string) (string, error) {
		if cmd == "find /var/crash -name 'kdump.*'" {
			return kdumpOutput, nil
		} else if cmd == "find /var/crash -name 'dmesg.*'" {
			return dmesgOutput, nil
		}
		return "", fmt.Errorf("unknown command")
	})
}

func TestGetShowKdumpFiles(t *testing.T) {
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
	kdumpFilesWithDataExpected := `{"kernel_core_dump_files":["/var/crash/202411101200/kdump.202411101200","/var/crash/202411101100/kdump.202411101100"],"kernel_dmesg_files":["/var/crash/202411101200/dmesg.202411101200","/var/crash/202411101100/dmesg.202411101100"]}`
	kdumpFilesNoFilesExpected := `{"kernel_core_dump_files":["No kernel core dump file available!"],"kernel_dmesg_files":["No kernel dmesg file available!"]}`
	kdumpFilesOnlyKdumpExpected := `{"kernel_core_dump_files":["/var/crash/202411101200/kdump.202411101200"],"kernel_dmesg_files":["No kernel dmesg file available!"]}`
	kdumpFilesOnlyDmesgExpected := `{"kernel_core_dump_files":["No kernel core dump file available!"],"kernel_dmesg_files":["/var/crash/202411101200/dmesg.202411101200"]}`

	// command outputs
	kdumpOutputWithData := "/var/crash/202411101100/kdump.202411101100\n/var/crash/202411101200/kdump.202411101200"
	dmesgOutputWithData := "/var/crash/202411101100/dmesg.202411101100\n/var/crash/202411101200/dmesg.202411101200"
	kdumpOutputSingle := "/var/crash/202411101200/kdump.202411101200"
	dmesgOutputSingle := "/var/crash/202411101200/dmesg.202411101200"
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
			desc:       "query SHOW kdump files with data",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "kdump" >
				elem: <name: "files" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(kdumpFilesWithDataExpected),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				return mockKdumpFilesCommands(kdumpOutputWithData, dmesgOutputWithData)
			},
		},
		{
			desc:       "query SHOW kdump files no files",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "kdump" >
				elem: <name: "files" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(kdumpFilesNoFilesExpected),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				return mockKdumpFilesCommands(emptyOutput, emptyOutput)
			},
		},
		{
			desc:       "query SHOW kdump files only kdump files",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "kdump" >
				elem: <name: "files" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(kdumpFilesOnlyKdumpExpected),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				return mockKdumpFilesCommands(kdumpOutputSingle, emptyOutput)
			},
		},
		{
			desc:       "query SHOW kdump files only dmesg files",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "kdump" >
				elem: <name: "files" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(kdumpFilesOnlyDmesgExpected),
			valTest:     true,
			testInit: func() *gomonkey.Patches {
				return mockKdumpFilesCommands(emptyOutput, dmesgOutputSingle)
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
