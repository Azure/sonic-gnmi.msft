package gnmi

import (
	"testing"

	"crypto/tls"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestGetShowInterfaceStatus(t *testing.T) {
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

	emptyResp := `[]`
	dataWithoutApplDB := `[{"Interface":"Ethernet0","Lanes":"N/A","Speed":"N/A","MTU":"N/A","FEC":"N/A","Alias":"N/A","Vlan":"PortChannel1","Oper":"N/A","Admin":"N/A","Type":"N/A","Asym":"N/A"},{"Interface":"Ethernet40","Lanes":"N/A","Speed":"N/A","MTU":"N/A","FEC":"N/A","Alias":"N/A","Vlan":"PortChannel1","Oper":"N/A","Admin":"N/A","Type":"N/A","Asym":"N/A"},{"Interface":"Ethernet80","Lanes":"N/A","Speed":"N/A","MTU":"N/A","FEC":"N/A","Alias":"N/A","Vlan":"routed","Oper":"N/A","Admin":"N/A","Type":"N/A","Asym":"N/A"},{"Interface":"Ethernet120","Lanes":"N/A","Speed":"N/A","MTU":"N/A","FEC":"N/A","Alias":"N/A","Vlan":"trunk","Oper":"N/A","Admin":"N/A","Type":"N/A","Asym":"N/A"},{"Interface":"PortChannel1","Lanes":"N/A","Speed":"0M","MTU":"9100","FEC":"N/A","Alias":"N/A","Vlan":"routed","Oper":"N/A","Admin":"N/A","Type":"N/A","Asym":"N/A"}]`
	fullDataWithoutStateDB := `[{"Interface":"Ethernet0","Lanes":"2304,2305,2306,2307","Speed":"100G","MTU":"9100","FEC":"rs","Alias":"etp0","Vlan":"PortChannel1","Oper":"up","Admin":"up","Type":"N/A","Asym":"off"},{"Interface":"Ethernet40","Lanes":"2048,2049,2050,2051","Speed":"100G","MTU":"9100","FEC":"rs","Alias":"etp10","Vlan":"PortChannel1","Oper":"up","Admin":"up","Type":"N/A","Asym":"off"},{"Interface":"Ethernet80","Lanes":"2568,2569,2570,2571","Speed":"100G","MTU":"9100","FEC":"rs","Alias":"etp20","Vlan":"routed","Oper":"up","Admin":"up","Type":"N/A","Asym":"off"},{"Interface":"Ethernet120","Lanes":"2668,2669,2670,2671","Speed":"100G","MTU":"9100","FEC":"rs","Alias":"etp30","Vlan":"trunk","Oper":"up","Admin":"up","Type":"N/A","Asym":"off"},{"Interface":"PortChannel1","Lanes":"N/A","Speed":"200G","MTU":"9100","FEC":"N/A","Alias":"N/A","Vlan":"routed","Oper":"up","Admin":"up","Type":"N/A","Asym":"N/A"}]`
	fullDataWithStateDB := `[{"Interface":"Ethernet0","Lanes":"2304,2305,2306,2307","Speed":"200G","MTU":"9100","FEC":"rs","Alias":"etp0","Vlan":"PortChannel1","Oper":"up","Admin":"up","Type":"SFP","Asym":"off"},{"Interface":"Ethernet40","Lanes":"2048,2049,2050,2051","Speed":"200G","MTU":"9100","FEC":"rs","Alias":"etp10","Vlan":"PortChannel1","Oper":"up","Admin":"up","Type":"SFP","Asym":"off"},{"Interface":"Ethernet80","Lanes":"2568,2569,2570,2571","Speed":"2.5G","MTU":"9100","FEC":"rs","Alias":"etp20","Vlan":"routed","Oper":"up","Admin":"up","Type":"SFP","Asym":"off"},{"Interface":"Ethernet120","Lanes":"2668,2669,2670,2671","Speed":"1000M","MTU":"9100","FEC":"rs","Alias":"etp30","Vlan":"trunk","Oper":"up","Admin":"up","Type":"RJ45","Asym":"off"},{"Interface":"PortChannel1","Lanes":"N/A","Speed":"400G","MTU":"9100","FEC":"N/A","Alias":"N/A","Vlan":"routed","Oper":"up","Admin":"up","Type":"N/A","Asym":"N/A"}]`
	singleInterfaceDataWithStateDB := `[{"Interface":"Ethernet0","Lanes":"2304,2305,2306,2307","Speed":"200G","MTU":"9100","FEC":"rs","Alias":"etp0","Vlan":"PortChannel1","Oper":"up","Admin":"up","Type":"SFP","Asym":"off"}]`
	singlePortchannelDataWithStateDB := `[{"Interface":"PortChannel1","Lanes":"N/A","Speed":"400G","MTU":"9100","FEC":"N/A","Alias":"N/A","Vlan":"routed","Oper":"up","Admin":"up","Type":"N/A","Asym":"N/A"}]`
	errorDataWithoutStateDB := `[{"Interface":"Ethernet0","Lanes":"2304,2305,2306,2307","Speed":"100M","MTU":"9100","FEC":"rs","Alias":"etp0","Vlan":"PortChannel1","Oper":"up","Admin":"up","Type":"N/A","Asym":"off"},{"Interface":"Ethernet40","Lanes":"2048,2049,2050,2051","Speed":"100M","MTU":"9100","FEC":"rs","Alias":"etp10","Vlan":"PortChannel1","Oper":"up","Admin":"up","Type":"N/A","Asym":"off"},{"Interface":"Ethernet80","Lanes":"2568,2569,2570,2571","Speed":"100G","MTU":"9100","FEC":"rs","Alias":"etp20","Vlan":"routed","Oper":"up","Admin":"up","Type":"N/A","Asym":"off"},{"Interface":"Ethernet120","Lanes":"2668,2669,2670,2671","Speed":"N/A","MTU":"9100","FEC":"rs","Alias":"etp30","Vlan":"trunk","Oper":"down","Admin":"up","Type":"N/A","Asym":"off"},{"Interface":"PortChannel1","Lanes":"N/A","Speed":"200M","MTU":"9100","FEC":"N/A","Alias":"N/A","Vlan":"routed","Oper":"up","Admin":"up","Type":"N/A","Asym":"N/A"}]`
	subintfsData := `[{"Interface":"Ethernet0.100","Speed":"N/A","MTU":"N/A","Vlan":"N/A","Oper":"N/A","Admin":"N/A","Type":"N/A"}]`

	configDbFileName := "../testdata/CONFIG_DB.json"
	appDbFileName := "../testdata/APPL_DB.json"
	stateDbFileName := "../testdata/STATE_DB.json"
	errorAppDbFileName := "../testdata/ERROR_APPL_DB.json"
	subintfsAppDbFileName := "../testdata/SUBINTFS_APPL_DB.json"

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
			desc:       "query SHOW interfaces status - no data",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "status" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(emptyResp),
			valTest:     true,
		},
		{
			desc:       "query SHOW interfaces status - config db only",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "status" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(dataWithoutApplDB),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				AddDataSet(t, ConfigDbNum, configDbFileName)
			},
		},
		{
			desc:       "query SHOW interfaces status - appl db only",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "status" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(fullDataWithoutStateDB),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				FlushDataSet(t, ApplDbNum)
				AddDataSet(t, ConfigDbNum, configDbFileName)
				AddDataSet(t, ApplDbNum, appDbFileName)
			},
		},
		{
			desc:       "query SHOW interfaces status - appl db + state db",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "status" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(fullDataWithStateDB),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, StateDbNum)
				AddDataSet(t, ConfigDbNum, configDbFileName)
				AddDataSet(t, ApplDbNum, appDbFileName)
				AddDataSet(t, StateDbNum, stateDbFileName)
			},
		},
		{
			desc:       "query SHOW interfaces status - single interface",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "status" key: { key: "interface" value: "Ethernet0" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(singleInterfaceDataWithStateDB),
			valTest:     true,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, configDbFileName)
				AddDataSet(t, ApplDbNum, appDbFileName)
				AddDataSet(t, StateDbNum, stateDbFileName)
			},
		},
		{
			desc:       "query SHOW interface status - single portchannel",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "status" key: { key: "interface" value: "PortChannel1" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(singlePortchannelDataWithStateDB),
			valTest:     true,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, configDbFileName)
				AddDataSet(t, ApplDbNum, appDbFileName)
				AddDataSet(t, StateDbNum, stateDbFileName)
			},
		},
		{
			desc:       "query SHOW interfaces status - abnormal data",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "status" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(errorDataWithoutStateDB),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ApplDbNum)
				FlushDataSet(t, StateDbNum)
				AddDataSet(t, ConfigDbNum, configDbFileName)
				AddDataSet(t, ApplDbNum, errorAppDbFileName)
			},
		},
		{
			desc:       "query SHOW interfaces status - subinterfaces",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "status" key: { key: "interface" value: "subport" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(subintfsData),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ConfigDbNum)
				FlushDataSet(t, ApplDbNum)
				AddDataSet(t, ConfigDbNum, configDbFileName)
				AddDataSet(t, ApplDbNum, subintfsAppDbFileName)
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
