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
	ndpExpectedOutput := `{"total_entries": 59,"entries": [{"address":"2a01:111:e210:b000::a40:f66f","mac_address":"dc:f4:01:e6:54:a9","iface":"eth0","vlan":"-","status":"REACHABLE"},{"address":"2a01:111:e210:b000::a40:f779","mac_address":"86:cd:79:2c:8e:0f","iface":"eth0","vlan":"-","status":"REACHABLE"},{"address":"fc00::7a","mac_address":"1e:d1:69:80:90:95","iface":"PortChannel106","vlan":"-","status":"REACHABLE"},{"address":"fc00::7e","mac_address":"4e:91:80:58:1f:c9","iface":"PortChannel108","vlan":"-","status":"REACHABLE"},{"address":"fc00::8a","mac_address":"0a:30:af:ca:e2:73","iface":"PortChannel1011","vlan":"-","status":"REACHABLE"},{"address":"fc00::8e","mac_address":"0a:a7:34:ab:d6:36","iface":"PortChannel1012","vlan":"-","status":"REACHABLE"},{"address":"fc00::17a","mac_address":"f2:c0:27:b9:a5:9e","iface":"Ethernet64","vlan":"-","status":"REACHABLE"},{"address":"fc00::17e","mac_address":"f2:e4:b5:13:2b:49","iface":"Ethernet68","vlan":"-","status":"REACHABLE"},{"address":"fc00::72","mac_address":"1e:ad:cf:d3:5e:ea","iface":"PortChannel102","vlan":"-","status":"REACHABLE"},{"address":"fc00::76","mac_address":"0a:c7:f3:af:e0:b5","iface":"PortChannel104","vlan":"-","status":"REACHABLE"},{"address":"fc00::82","mac_address":"22:c8:2e:4a:74:f1","iface":"PortChannel109","vlan":"-","status":"REACHABLE"},{"address":"fc00::86","mac_address":"1a:f0:6a:af:2f:10","iface":"PortChannel1010","vlan":"-","status":"REACHABLE"},{"address":"fc00::172","mac_address":"2a:04:4b:71:b3:08","iface":"Ethernet56","vlan":"-","status":"REACHABLE"},{"address":"fc00::176","mac_address":"aa:f9:f4:cb:d2:3f","iface":"Ethernet60","vlan":"-","status":"REACHABLE"},{"address":"fe80::1c02:63ff:fe1e:5019","mac_address":"1e:02:63:1e:50:19","iface":"PortChannel1012","vlan":"-","status":"STALE"},{"address":"fe80::1cad:cfff:fed3:5eea","mac_address":"1e:ad:cf:d3:5e:ea","iface":"PortChannel102","vlan":"-","status":"STALE"},{"address":"fe80::1cd1:69ff:fe80:9095","mac_address":"1e:d1:69:80:90:95","iface":"PortChannel106","vlan":"-","status":"STALE"},{"address":"fe80::2e0:ecff:fe83:b80f","mac_address":"00:e0:ec:83:b8:0f","iface":"eth0","vlan":"-","status":"REACHABLE"},{"address":"fe80::4c91:80ff:fe58:1fc9","mac_address":"4e:91:80:58:1f:c9","iface":"PortChannel108","vlan":"-","status":"STALE"},{"address":"fe80::6a8b:f4ff:fe87:9ddc","mac_address":"68:8b:f4:87:9d:dc","iface":"-","vlan":"1000","status":"STALE"},{"address":"fe80::8a7:34ff:feab:d636","mac_address":"0a:a7:34:ab:d6:36","iface":"PortChannel1012","vlan":"-","status":"STALE"},{"address":"fe80::8c7:f3ff:feaf:e0b5","mac_address":"0a:c7:f3:af:e0:b5","iface":"PortChannel104","vlan":"-","status":"STALE"},{"address":"fe80::18f0:6aff:feaf:2f10","mac_address":"1a:f0:6a:af:2f:10","iface":"PortChannel1010","vlan":"-","status":"STALE"},{"address":"fe80::20c8:2eff:fe4a:74f1","mac_address":"22:c8:2e:4a:74:f1","iface":"PortChannel109","vlan":"-","status":"STALE"},{"address":"fe80::34df:24ff:fedc:6018","mac_address":"36:df:24:dc:60:18","iface":"PortChannel1011","vlan":"-","status":"REACHABLE"},{"address":"fe80::106f:a5ff:fe37:2007","mac_address":"12:6f:a5:37:20:07","iface":"PortChannel104","vlan":"-","status":"REACHABLE"},{"address":"fe80::547b:d0ff:fe0d:6","mac_address":"56:7b:d0:0d:00:06","iface":"PortChannel102","vlan":"-","status":"REACHABLE"},{"address":"fe80::549b:d3ff:fe02:7017","mac_address":"56:9b:d3:02:70:17","iface":"PortChannel1010","vlan":"-","status":"REACHABLE"},{"address":"fe80::830:afff:feca:e273","mac_address":"0a:30:af:ca:e2:73","iface":"PortChannel1011","vlan":"-","status":"STALE"},{"address":"fe80::2804:4bff:fe71:b308","mac_address":"2a:04:4b:71:b3:08","iface":"Ethernet56","vlan":"-","status":"STALE"},{"address":"fe80::3476:c2ff:fe09:b011","mac_address":"36:76:c2:09:b0:11","iface":"Ethernet68","vlan":"-","status":"STALE"},{"address":"fe80::a8f9:f4ff:fecb:d23f","mac_address":"aa:f9:f4:cb:d2:3f","iface":"Ethernet60","vlan":"-","status":"STALE"},{"address":"fe80::b5:a2ff:feb1:5010","mac_address":"02:b5:a2:b1:50:10","iface":"Ethernet64","vlan":"-","status":"REACHABLE"},{"address":"fe80::bace:f6ff:fee5:500a","mac_address":"b8:ce:f6:e5:50:0a","iface":"Ethernet40","vlan":"1000","status":"REACHABLE"},{"address":"fe80::bace:f6ff:fee5:500b","mac_address":"b8:ce:f6:e5:50:0b","iface":"Ethernet44","vlan":"1000","status":"REACHABLE"},{"address":"fe80::bace:f6ff:fee5:500c","mac_address":"b8:ce:f6:e5:50:0c","iface":"Ethernet48","vlan":"1000","status":"REACHABLE"},{"address":"fe80::bace:f6ff:fee5:500d","mac_address":"b8:ce:f6:e5:50:0d","iface":"Ethernet52","vlan":"1000","status":"REACHABLE"},{"address":"fe80::bace:f6ff:fee5:501a","mac_address":"b8:ce:f6:e5:50:1a","iface":"Ethernet104","vlan":"1000","status":"REACHABLE"},{"address":"fe80::bace:f6ff:fee5:501b","mac_address":"b8:ce:f6:e5:50:1b","iface":"Ethernet108","vlan":"1000","status":"REACHABLE"},{"address":"fe80::bace:f6ff:fee5:501c","mac_address":"b8:ce:f6:e5:50:1c","iface":"Ethernet112","vlan":"1000","status":"REACHABLE"},{"address":"fe80::bace:f6ff:fee5:501d","mac_address":"b8:ce:f6:e5:50:1d","iface":"Ethernet116","vlan":"1000","status":"REACHABLE"},{"address":"fe80::bace:f6ff:fee5:5000","mac_address":"b8:ce:f6:e5:50:00","iface":"Ethernet0","vlan":"1000","status":"REACHABLE"},{"address":"fe80::bace:f6ff:fee5:5001","mac_address":"b8:ce:f6:e5:50:01","iface":"Ethernet4","vlan":"1000","status":"REACHABLE"},{"address":"fe80::bace:f6ff:fee5:5002","mac_address":"b8:ce:f6:e5:50:02","iface":"Ethernet8","vlan":"1000","status":"REACHABLE"},{"address":"fe80::bace:f6ff:fee5:5003","mac_address":"b8:ce:f6:e5:50:03","iface":"Ethernet12","vlan":"1000","status":"REACHABLE"},{"address":"fe80::bace:f6ff:fee5:5004","mac_address":"b8:ce:f6:e5:50:04","iface":"Ethernet16","vlan":"1000","status":"REACHABLE"},{"address":"fe80::bace:f6ff:fee5:5005","mac_address":"b8:ce:f6:e5:50:05","iface":"Ethernet20","vlan":"1000","status":"REACHABLE"},{"address":"fe80::bace:f6ff:fee5:5012","mac_address":"b8:ce:f6:e5:50:12","iface":"Ethernet72","vlan":"1000","status":"REACHABLE"},{"address":"fe80::bace:f6ff:fee5:5013","mac_address":"b8:ce:f6:e5:50:13","iface":"Ethernet76","vlan":"1000","status":"REACHABLE"},{"address":"fe80::bace:f6ff:fee5:5014","mac_address":"b8:ce:f6:e5:50:14","iface":"Ethernet80","vlan":"1000","status":"REACHABLE"},{"address":"fe80::bace:f6ff:fee5:5015","mac_address":"b8:ce:f6:e5:50:15","iface":"Ethernet84","vlan":"1000","status":"REACHABLE"},{"address":"fe80::c0b0:6eff:fea4:c00e","mac_address":"c2:b0:6e:a4:c0:0e","iface":"Ethernet56","vlan":"-","status":"REACHABLE"},{"address":"fe80::c05f:8bff:fe44:8008","mac_address":"c2:5f:8b:44:80:08","iface":"PortChannel106","vlan":"-","status":"REACHABLE"},{"address":"fe80::cc75:6fff:fe59:7c85","mac_address":"dc:f4:01:e6:54:a9","iface":"eth0","vlan":"-","status":"REACHABLE"},{"address":"fe80::d45d:10ff:fef9:8016","mac_address":"d6:5d:10:f9:80:16","iface":"PortChannel109","vlan":"-","status":"REACHABLE"},{"address":"fe80::ecf0:64ff:fea8:600f","mac_address":"ee:f0:64:a8:60:0f","iface":"Ethernet60","vlan":"-","status":"REACHABLE"},{"address":"fe80::f0c0:27ff:feb9:a59e","mac_address":"f2:c0:27:b9:a5:9e","iface":"Ethernet64","vlan":"-","status":"STALE"},{"address":"fe80::f0e4:b5ff:fe13:2b49","mac_address":"f2:e4:b5:13:2b:49","iface":"Ethernet68","vlan":"-","status":"STALE"},{"address":"fe80::f043:f0ff:feb7:2009","mac_address":"f2:43:f0:b7:20:09","iface":"PortChannel108","vlan":"-","status":"REACHABLE"}]}`

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
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})
		if patches != nil {
			patches.Reset()
		}
	}
}
