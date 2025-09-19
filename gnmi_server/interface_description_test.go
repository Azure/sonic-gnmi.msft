package gnmi

// Tests SHOW interface/description

import (
	"crypto/tls"
	"testing"
	"time"

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
	appDbFileName := "../testdata/INTF_DESC_APPL_DB.json"
	configDbFileName := "../testdata/INTF_DESC_CONFIG_DB.json"

	expectedRetValue := `
{
    "Ethernet0": {
        "Admin":"up","Alias":"etp0","Description":"ARISTA01T1:Ethernet1","Oper":"up"
        },
    "Ethernet40": {
       "Admin":"up","Alias":"etp10","Description":"Servers4:eth0","Oper":"up"
        }
}
`
	expectedSingleRetValue := `
{
    "Ethernet0": {
        "Admin":"up","Alias":"etp0","Description":"ARISTA01T1:Ethernet1","Oper":"up"
        }
}
`
	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal interface{}
		valTest     bool
		testInit    func()
	}{
		{
			desc:       "query SHOW interfaces description NO DATA",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "description" >
            `,
			wantRetCode: codes.OK,
			valTest:     false,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
			},
		},
		{
			desc:       "query SHOW interface description with no interface option",
			pathTarget: "SHOW",
			textPbPath: `
                elem: <name: "interfaces" >
                elem: <name: "description" >
            `,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedRetValue),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				FlushDataSet(t, ApplDbNum)
				AddDataSet(t, ConfigDbNum, configDbFileName)
				AddDataSet(t, ApplDbNum, appDbFileName)
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
			wantRespVal: []byte(expectedSingleRetValue),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				FlushDataSet(t, ApplDbNum)
				AddDataSet(t, ConfigDbNum, configDbFileName)
				AddDataSet(t, ApplDbNum, appDbFileName)
			},
		},
	}

	for _, test := range tests {
		if test.testInit != nil {
			test.testInit()
		}

		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})
	}
}
