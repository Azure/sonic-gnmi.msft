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
		testTime       time.Time
		testInit       func()
	}{
		{
			desc:       "query SHOW version without evnironment variable",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "version" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedOutputWithEmptyPlat),
			valTest:     true,
			mockOutputFile: map[string]string{
				"docker": "../testdata/VERSION_DOCKER_IMAGEDATA.txt",
				"uptime": "../testdata/VERSION_UPTIME.txt",
			},
			testTime: time.Date(2025, 7, 18, 18, 0, 0, 0, time.UTC),
			testInit: func() {
				MockReadFile(show_client.SonicVersionYamlPath, versionInfo, nil)
				MockEnvironmentVariable(t, "PLATFORM", "")
				AddDataSet(t, ConfigDbNum, deviceMetadataFilename)
				AddDataSet(t, chassisStateDbNum, chassisDataFilename)
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

		testTime := test.testTime
		timepatch := gomonkey.ApplyFunc(time.Now, func() time.Time {
			return testTime
		})

		show_client.MachineConfPath = "../testdata/VERSION_MACHINE_CONF.conf"
		show_client.HostDevicePath = "../testdata/"

		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})

		if patches != nil {
			patches.Reset()
		}
		if timepatch != nil {
			timepatch.Reset()
		}
	}

}
