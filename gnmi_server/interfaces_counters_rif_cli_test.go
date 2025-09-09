package gnmi

// interfaces_counters_rif_cli.go

// Tests SHOW interfaces counters rif
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

func TestGetInterfaceRifCounters(t *testing.T) {
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

	FlushDataSet(t, CountersDbNum)
	interfacesCountersRifTestData := "../testdata/InterfacesCountersRifTestData.txt"
	AddDataSet(t, CountersDbNum, interfacesCountersRifTestData)

	t.Run("query SHOW interfaces counters rif", func(t *testing.T) {
		textPbPath := `
			elem: <name: "interfaces" >
			elem: <name: "counters" >
			elem: <name: "rif" >
		`
		wantRespVal := []byte(`{
  "PortChannel101": {
    "RxBps": "19.431878785629241",
    "RxErrBits": "0",
    "RxErrPackets": "100",
    "RxOk": "5940",
    "RxOkBits": "1048449",
    "RxPps": "0.21111450848388191",
    "TxBps": "0.5",
    "TxErrBits": "0",
    "TxErrPackets": "10",
    "TxOkBits": "0",
    "TxOkPackets": "650",
    "TxPps": "12.21"
  },
  "PortChannel102": {
    "RxBps": "1.2202977000824049",
    "RxErrBits": "0",
    "RxErrPackets": "0",
    "RxOk": "5937",
    "RxOkBits": "1048207",
    "RxPps": "0.013699805079217392",
    "TxBps": "0",
    "TxErrBits": "0",
    "TxErrPackets": "0",
    "TxOkBits": "0",
    "TxOkPackets": "0",
    "TxPps": "0"
  },
  "PortChannel103": {
    "RxBps": "6.0568048649819142",
    "RxErrBits": "0",
    "RxErrPackets": "0",
    "RxOk": "5943",
    "RxOkBits": "1048821",
    "RxPps": "0.058547265917178126",
    "TxBps": "0",
    "TxErrBits": "0",
    "TxErrPackets": "0",
    "TxOkBits": "0",
    "TxOkPackets": "0",
    "TxPps": "0"
  },
  "PortChannel104": {
    "RxBps": "20.260496891870496",
    "RxErrBits": "0",
    "RxErrPackets": "0",
    "RxOk": "5950",
    "RxOkBits": "1049477",
    "RxPps": "0.24715843207997978",
    "TxBps": "0",
    "TxErrBits": "N/A",
    "TxErrPackets": "0",
    "TxOkBits": "0",
    "TxOkPackets": "0",
    "TxPps": "0"
  },
  "Vlan1000": {
    "RxBps": "0.0003231896674387374",
    "RxErrBits": "0",
    "RxErrPackets": "0",
    "RxOk": "17856",
    "RxOkBits": "1865088",
    "RxPps": "3.2330838487270913e-06",
    "TxBps": "0",
    "TxErrBits": "0",
    "TxErrPackets": "0",
    "TxOkBits": "0",
    "TxOkPackets": "0",
    "TxPps": "0"
  }
}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	t.Run("query SHOW interfaces counters rif --i PortChannel101", func(t *testing.T) {
		textPbPath := `
			elem: <name: "interfaces" >
			elem: <name: "counters" >
			elem: <name: "rif"  key: {key: "interface" value: "PortChannel101"} >
		`
		wantRespVal := []byte(`{
			"PortChannel101": {
				"RxBps": "19.431878785629241",
				"RxErrBits": "0",
				"RxErrPackets": "100",
				"RxOk": "5940",
				"RxOkBits": "1048449",
				"RxPps": "0.21111450848388191",
				"TxBps": "0.5",
				"TxErrBits": "0",
				"TxErrPackets": "10",
				"TxOkBits": "0",
				"TxOkPackets": "650",
				"TxPps": "12.21"
			}
	  }`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	t.Run("query SHOW interfaces counters rif --i PortChannel101 -p 1", func(t *testing.T) {
		textPbPath := `
			elem: <name: "interfaces" >
			elem: <name: "counters" >
			elem: <name: "rif"  key: {key: "interface" value: "PortChannel101"}
													key: {key: "period" value: "1"} >
		`
		wantRespVal := []byte(`{
			"PortChannel101": {
				"RxBps": "19.431878785629241",
				"RxErrBits": "0",
				"RxErrPackets": "0",
				"RxOk": "0",
				"RxOkBits": "0",
				"RxPps": "0.21111450848388191",
				"TxBps": "0.5",
				"TxErrBits": "0",
				"TxErrPackets": "0",
				"TxOkBits": "0",
				"TxOkPackets": "0",
				"TxPps": "12.21"
			}
	  }`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})
}
