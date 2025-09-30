package gnmi

// ndp_cli_test.go
// Unit tests for show ndp command

import (
	"crypto/tls"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"

	"github.com/agiledragon/gomonkey/v2"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestGetNDP(t *testing.T) {
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

	countersDBFileName := "../testdata/ndp/COUNTERS_DB.txt"
	asicDBFileName := "../testdata/ndp/ASIC_DB.txt"
	ipNeighShowFileName := "../testdata/ndp/IP_NEIGH_OUTPUT.txt"
	specificInterfaceFileName := "../testdata/ndp/IP_NEIGH_OUTPUT_SPECIFIC_INTERFACE.txt"
	specificIPAddressFileName := "../testdata/ndp/IP_NEIGH_OUTPUT_SPECIFIC_IPADDRESS.txt"
	emptyFileName := "../testdata/ndp/IP_NEIGH_OUTPUT_EMPTY.txt"
	ndpExpectedOutput := `{"total_entries": 59,"entries": [
{"address":"2a01:111:e210:b000::a40:f66f","mac_address":"DC:F4:01:E6:54:A9","iface":"eth0","vlan":"-","status":"REACHABLE"},
{"address":"2a01:111:e210:b000::a40:f779","mac_address":"86:CD:79:2C:8E:0F","iface":"eth0","vlan":"-","status":"REACHABLE"},
{"address":"fc00::7a","mac_address":"1E:D1:69:80:90:95","iface":"PortChannel106","vlan":"-","status":"REACHABLE"},
{"address":"fc00::7e","mac_address":"4E:91:80:58:1F:C9","iface":"PortChannel108","vlan":"-","status":"REACHABLE"},
{"address":"fc00::8a","mac_address":"0A:30:AF:CA:E2:73","iface":"PortChannel1011","vlan":"-","status":"REACHABLE"},
{"address":"fc00::8e","mac_address":"0A:A7:34:AB:D6:36","iface":"PortChannel1012","vlan":"-","status":"REACHABLE"},
{"address":"fc00::17a","mac_address":"F2:C0:27:B9:A5:9E","iface":"Ethernet64","vlan":"-","status":"REACHABLE"},
{"address":"fc00::17e","mac_address":"F2:E4:B5:13:2B:49","iface":"Ethernet68","vlan":"-","status":"REACHABLE"},
{"address":"fc00::72","mac_address":"1E:AD:CF:D3:5E:EA","iface":"PortChannel102","vlan":"-","status":"REACHABLE"},
{"address":"fc00::76","mac_address":"0A:C7:F3:AF:E0:B5","iface":"PortChannel104","vlan":"-","status":"REACHABLE"},
{"address":"fc00::82","mac_address":"22:C8:2E:4A:74:F1","iface":"PortChannel109","vlan":"-","status":"REACHABLE"},
{"address":"fc00::86","mac_address":"1A:F0:6A:AF:2F:10","iface":"PortChannel1010","vlan":"-","status":"REACHABLE"},
{"address":"fc00::172","mac_address":"2A:04:4B:71:B3:08","iface":"Ethernet56","vlan":"-","status":"REACHABLE"},
{"address":"fc00::176","mac_address":"AA:F9:F4:CB:D2:3F","iface":"Ethernet60","vlan":"-","status":"REACHABLE"},
{"address":"fe80::1c02:63ff:fe1e:5019","mac_address":"1E:02:63:1E:50:19","iface":"PortChannel1012","vlan":"-","status":"REACHABLE"},
{"address":"fe80::1cad:cfff:fed3:5eea","mac_address":"1E:AD:CF:D3:5E:EA","iface":"PortChannel102","vlan":"-","status":"STALE"},
{"address":"fe80::1cd1:69ff:fe80:9095","mac_address":"1E:D1:69:80:90:95","iface":"PortChannel106","vlan":"-","status":"STALE"},
{"address":"fe80::2e0:ecff:fe83:b80f","mac_address":"00:E0:EC:83:B8:0F","iface":"eth0","vlan":"-","status":"REACHABLE"},
{"address":"fe80::4c91:80ff:fe58:1fc9","mac_address":"4E:91:80:58:1F:C9","iface":"PortChannel108","vlan":"-","status":"STALE"},
{"address":"fe80::6a8b:f4ff:fe87:9ddc","mac_address":"68:8B:F4:87:9D:DC","iface":"Vlan1000","vlan":"1000","status":"STALE"},
{"address":"fe80::8a7:34ff:feab:d636","mac_address":"0A:A7:34:AB:D6:36","iface":"PortChannel1012","vlan":"-","status":"STALE"},
{"address":"fe80::8c7:f3ff:feaf:e0b5","mac_address":"0A:C7:F3:AF:E0:B5","iface":"PortChannel104","vlan":"-","status":"STALE"},
{"address":"fe80::18f0:6aff:feaf:2f10","mac_address":"1A:F0:6A:AF:2F:10","iface":"PortChannel1010","vlan":"-","status":"STALE"},
{"address":"fe80::20c8:2eff:fe4a:74f1","mac_address":"22:C8:2E:4A:74:F1","iface":"PortChannel109","vlan":"-","status":"STALE"},
{"address":"fe80::34df:24ff:fedc:6018","mac_address":"36:DF:24:DC:60:18","iface":"PortChannel1011","vlan":"-","status":"REACHABLE"},
{"address":"fe80::106f:a5ff:fe37:2007","mac_address":"12:6F:A5:37:20:07","iface":"PortChannel104","vlan":"-","status":"STALE"},
{"address":"fe80::547b:d0ff:fe0d:6","mac_address":"56:7B:D0:0D:00:06","iface":"PortChannel102","vlan":"-","status":"REACHABLE"},
{"address":"fe80::549b:d3ff:fe02:7017","mac_address":"56:9B:D3:02:70:17","iface":"PortChannel1010","vlan":"-","status":"REACHABLE"},
{"address":"fe80::830:afff:feca:e273","mac_address":"0A:30:AF:CA:E2:73","iface":"PortChannel1011","vlan":"-","status":"STALE"},
{"address":"fe80::2804:4bff:fe71:b308","mac_address":"2A:04:4B:71:B3:08","iface":"Ethernet56","vlan":"-","status":"STALE"},
{"address":"fe80::3476:c2ff:fe09:b011","mac_address":"36:76:C2:09:B0:11","iface":"Ethernet68","vlan":"-","status":"REACHABLE"},
{"address":"fe80::a8f9:f4ff:fecb:d23f","mac_address":"AA:F9:F4:CB:D2:3F","iface":"Ethernet60","vlan":"-","status":"STALE"},
{"address":"fe80::b5:a2ff:feb1:5010","mac_address":"02:B5:A2:B1:50:10","iface":"Ethernet64","vlan":"-","status":"REACHABLE"},
{"address":"fe80::bace:f6ff:fee5:500a","mac_address":"B8:CE:F6:E5:50:0A","iface":"Ethernet40","vlan":"1000","status":"REACHABLE"},
{"address":"fe80::bace:f6ff:fee5:500b","mac_address":"B8:CE:F6:E5:50:0B","iface":"Ethernet44","vlan":"1000","status":"REACHABLE"},
{"address":"fe80::bace:f6ff:fee5:500c","mac_address":"B8:CE:F6:E5:50:0C","iface":"Ethernet48","vlan":"1000","status":"REACHABLE"},
{"address":"fe80::bace:f6ff:fee5:500d","mac_address":"B8:CE:F6:E5:50:0D","iface":"Ethernet52","vlan":"1000","status":"REACHABLE"},
{"address":"fe80::bace:f6ff:fee5:501a","mac_address":"B8:CE:F6:E5:50:1A","iface":"Ethernet104","vlan":"1000","status":"REACHABLE"},
{"address":"fe80::bace:f6ff:fee5:501b","mac_address":"B8:CE:F6:E5:50:1B","iface":"Ethernet108","vlan":"1000","status":"REACHABLE"},
{"address":"fe80::bace:f6ff:fee5:501c","mac_address":"B8:CE:F6:E5:50:1C","iface":"Ethernet112","vlan":"1000","status":"REACHABLE"},
{"address":"fe80::bace:f6ff:fee5:501d","mac_address":"B8:CE:F6:E5:50:1D","iface":"Ethernet116","vlan":"1000","status":"REACHABLE"},
{"address":"fe80::bace:f6ff:fee5:5000","mac_address":"B8:CE:F6:E5:50:00","iface":"Ethernet0","vlan":"1000","status":"REACHABLE"},
{"address":"fe80::bace:f6ff:fee5:5001","mac_address":"B8:CE:F6:E5:50:01","iface":"Ethernet4","vlan":"1000","status":"REACHABLE"},
{"address":"fe80::bace:f6ff:fee5:5002","mac_address":"B8:CE:F6:E5:50:02","iface":"Ethernet8","vlan":"1000","status":"REACHABLE"},
{"address":"fe80::bace:f6ff:fee5:5003","mac_address":"B8:CE:F6:E5:50:03","iface":"Ethernet12","vlan":"1000","status":"REACHABLE"},
{"address":"fe80::bace:f6ff:fee5:5004","mac_address":"B8:CE:F6:E5:50:04","iface":"Ethernet16","vlan":"1000","status":"REACHABLE"},
{"address":"fe80::bace:f6ff:fee5:5005","mac_address":"B8:CE:F6:E5:50:05","iface":"Ethernet20","vlan":"1000","status":"REACHABLE"},
{"address":"fe80::bace:f6ff:fee5:5012","mac_address":"B8:CE:F6:E5:50:12","iface":"Ethernet72","vlan":"1000","status":"REACHABLE"},
{"address":"fe80::bace:f6ff:fee5:5013","mac_address":"B8:CE:F6:E5:50:13","iface":"Ethernet76","vlan":"1000","status":"REACHABLE"},
{"address":"fe80::bace:f6ff:fee5:5014","mac_address":"B8:CE:F6:E5:50:14","iface":"Ethernet80","vlan":"1000","status":"REACHABLE"},
{"address":"fe80::bace:f6ff:fee5:5015","mac_address":"B8:CE:F6:E5:50:15","iface":"Ethernet84","vlan":"1000","status":"REACHABLE"},
{"address":"fe80::c0b0:6eff:fea4:c00e","mac_address":"C2:B0:6E:A4:C0:0E","iface":"Ethernet56","vlan":"-","status":"REACHABLE"},
{"address":"fe80::c05f:8bff:fe44:8008","mac_address":"C2:5F:8B:44:80:08","iface":"PortChannel106","vlan":"-","status":"REACHABLE"},
{"address":"fe80::cc75:6fff:fe59:7c85","mac_address":"DC:F4:01:E6:54:A9","iface":"eth0","vlan":"-","status":"REACHABLE"},
{"address":"fe80::d45d:10ff:fef9:8016","mac_address":"D6:5D:10:F9:80:16","iface":"PortChannel109","vlan":"-","status":"REACHABLE"},
{"address":"fe80::ecf0:64ff:fea8:600f","mac_address":"EE:F0:64:A8:60:0F","iface":"Ethernet60","vlan":"-","status":"REACHABLE"},
{"address":"fe80::f0c0:27ff:feb9:a59e","mac_address":"F2:C0:27:B9:A5:9E","iface":"Ethernet64","vlan":"-","status":"STALE"},
{"address":"fe80::f0e4:b5ff:fe13:2b49","mac_address":"F2:E4:B5:13:2B:49","iface":"Ethernet68","vlan":"-","status":"STALE"},
{"address":"fe80::f043:f0ff:feb7:2009","mac_address":"F2:43:F0:B7:20:09","iface":"PortChannel108","vlan":"-","status":"REACHABLE"}
]}`
	specificInterfaceNDPExpectedOutput := `{"total_entries": 3,"entries": [
{"address":"fc00::176","mac_address":"AA:F9:F4:CB:D2:3F","iface":"Ethernet60","vlan":"-","status":"REACHABLE"},
{"address":"fe80::4461:2ff:fe21:f","mac_address":"46:61:02:21:00:0F","iface":"Ethernet60","vlan":"-","status":"REACHABLE"},
{"address":"fe80::a8f9:f4ff:fecb:d23f","mac_address":"AA:F9:F4:CB:D2:3F","iface":"Ethernet60","vlan":"-","status":"REACHABLE"}
]}`
	specificIPAddressNDPExpectedOutput := `{"total_entries": 1,"entries": [{"address":"fc00::176","mac_address":"AA:F9:F4:CB:D2:3F","iface":"Ethernet60","vlan":"-","status":"REACHABLE"}]}`
	emptyNDPExpectedOutput := `{"total_entries": 0,"entries": []}`

	ResetDataSetsAndMappings(t)

	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal interface{}
		valTest     bool
		mockFile    string
		testInit    func()
	}{
		{
			desc:       "query SHOW ndp - read error",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ndp" >
			`,
			wantRetCode: codes.NotFound,
		},
		{
			desc:       "query SHOW ndp - invalid ipaddress",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ndp" >
				elem: <name: "999.999.999.999" >
			`,
			wantRetCode: codes.NotFound,
		},
		{
			desc:       "query SHOW ndp - ipv4 ipaddress",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ndp" >
				elem: <name: "192.168.1.1" >
			`,
			wantRetCode: codes.NotFound,
		},
		{
			desc:       "query SHOW ndp - ipv4 ipaddress prefix",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ndp" >
				elem: <name: "192.168.1.1/64" >
			`,
			wantRetCode: codes.NotFound,
		},
		{
			desc:       "query SHOW ndp",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ndp" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(ndpExpectedOutput),
			valTest:     true,
			mockFile:    ipNeighShowFileName,
			testInit: func() {
				FlushDataSet(t, CountersDbNum)
				FlushDataSet(t, AsicDbNum)
				AddDataSet(t, CountersDbNum, countersDBFileName)
				AddDataSet(t, AsicDbNum, asicDBFileName)
			},
		},
		{
			desc:       "query SHOW ndp - specific interface",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ndp"  key: { key: "iface" value: "Ethernet60" } >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(specificInterfaceNDPExpectedOutput),
			valTest:     true,
			mockFile:    specificInterfaceFileName,
			testInit: func() {
				FlushDataSet(t, CountersDbNum)
				FlushDataSet(t, AsicDbNum)
				AddDataSet(t, CountersDbNum, countersDBFileName)
				AddDataSet(t, AsicDbNum, asicDBFileName)
			},
		},
		{
			desc:       "query SHOW ndp - specific ipaddress",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ndp">
				elem: <name: "fc00::176" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(specificIPAddressNDPExpectedOutput),
			valTest:     true,
			mockFile:    specificIPAddressFileName,
			testInit: func() {
				FlushDataSet(t, CountersDbNum)
				FlushDataSet(t, AsicDbNum)
				AddDataSet(t, CountersDbNum, countersDBFileName)
				AddDataSet(t, AsicDbNum, asicDBFileName)
			},
		},
		{
			desc:       "query SHOW ndp - specific ipaddress with prefix",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ndp">
				elem: <name: "::/0" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(ndpExpectedOutput),
			valTest:     true,
			mockFile:    ipNeighShowFileName,
			testInit: func() {
				FlushDataSet(t, CountersDbNum)
				FlushDataSet(t, AsicDbNum)
				AddDataSet(t, CountersDbNum, countersDBFileName)
				AddDataSet(t, AsicDbNum, asicDBFileName)
			},
		},
		{
			desc:       "query SHOW ndp - empty output",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ndp" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(emptyNDPExpectedOutput),
			valTest:     true,
			mockFile:    emptyFileName,
			testInit: func() {
				FlushDataSet(t, CountersDbNum)
				FlushDataSet(t, AsicDbNum)
				AddDataSet(t, CountersDbNum, countersDBFileName)
				AddDataSet(t, AsicDbNum, asicDBFileName)
			},
		},
	}

	for _, test := range tests {
		if test.testInit != nil {
			test.testInit()
		}
		var patches *gomonkey.Patches
		if test.mockFile != "" {
			patches = MockNSEnterOutput(t, test.mockFile)
		}

		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest, true)
		})
		if patches != nil {
			patches.Reset()
		}
	}
}
