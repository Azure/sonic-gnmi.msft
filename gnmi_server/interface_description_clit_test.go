package gnmi

// Tests SHOW interface/description

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

func TestGetShowInterfaceDescription(t *testing.T) {
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
    "Ethernet0": {
        "Admin":"up","Alias":"etp0","Description":"etp0","Oper":"down"
        },
    "Ethernet1": {
       "Admin":"up","Alias":"etp1","Description":"etp1","Oper":"down"
        }
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
			desc:       "query SHOW interfaces description NO DATA",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "description" >
            `,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedRetValue),
			valTest:     false,
			mockOutputFile: map[string]string{
				"intfutil": "../testdata/interface_description.txt",
			},
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
			},
		},
		{
			desc:       "query SHOW interface description with interface option (by name)",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "description" key: { key: "interface" value: "Ethernet0" } >
            `,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedRetValue),
			valTest:     true,
			mockOutputFile: map[string]string{
				"intfutil": "../testdata/interface_description.txt",
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
