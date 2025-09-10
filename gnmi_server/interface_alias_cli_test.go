package gnmi

// interface_alias_cli_test.go

// Tests SHOW interface/alias

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

func TestGetShowInterfaceAlias(t *testing.T) {
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

	portsFileName := "../testdata/PORTS.txt"

	aliasSingleEthernet0 := `{"Ethernet0":{"alias":"etp0"}}`
	aliasAllInterface := `{"Ethernet0":{"alias":"etp0"},"Ethernet40":{"alias":"etp10"},"Ethernet80":{"alias":"etp20"}}`

	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal interface{}
		valTest     bool
		mockSleep   bool
		testInit    func()
	}{
		{
			desc:       "query SHOW interfaces alias NO DATA",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "alias" >
            `,
			wantRetCode: codes.OK,
			valTest:     false,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
			},
		},
		{
			desc:       "query SHOW interfaces alias (load base ports)",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "alias" >
            `,
			wantRetCode: codes.OK,
			wantRespVal: []byte(aliasAllInterface),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, portsFileName)
			},
		},
		{
			desc:       "query SHOW interfaces alias with interface option (by name)",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "alias" key: { key: "interface" value: "Ethernet0" } >
            `,
			wantRetCode: codes.OK,
			wantRespVal: []byte(aliasSingleEthernet0),
			valTest:     true,
		},
	}

	for _, test := range tests {
		if test.testInit != nil {
			test.testInit()
		}
		var patches *gomonkey.Patches
		if test.mockSleep {
			patches = gomonkey.ApplyFunc(time.Sleep, func(d time.Duration) {})
		}

		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})
		if patches != nil {
			patches.Reset()
		}
	}
}
