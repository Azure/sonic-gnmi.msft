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
    "RxBps": "4214812716.943851",
    "RxErrBits": "17866494",
    "RxErrPackets": "172078",
    "RxOkPackets": "43864767060035",
    "RxOkBits": "4561966927266923",
    "RxPps": "40527122.163856164",
    "TxBps": "4214792810.2678127",
    "TxErrBits": "52942226547142352",
    "TxErrPackets": "509056042421691",
    "TxOkBits": "4561964553298733",
    "TxOkPackets": "43864743789853",
    "TxPps": "40526901.803920366"
  },
  "PortChannel102": {
    "RxBps": "1.2202977000824049",
    "RxErrBits": "0",
    "RxErrPackets": "0",
    "RxOkPackets": "5937",
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
    "RxOkPackets": "5943",
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
    "RxOkPackets": "5950",
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
    "RxOkPackets": "17856",
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
				"RxBps": "4214812716.943851",
				"RxErrBits": "17866494",
				"RxErrPackets": "172078",
				"RxOkPackets": "43864767060035",
				"RxOkBits": "4561966927266923",
				"RxPps": "40527122.163856164",
				"TxBps": "4214792810.2678127",
				"TxErrBits": "52942226547142352",
				"TxErrPackets": "509056042421691",
				"TxOkBits": "4561964553298733",
				"TxOkPackets": "43864743789853",
				"TxPps": "40526901.803920366"
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
				"RxBps": "4214812716.943851",
				"RxErrBits": "0",
				"RxErrPackets": "0",
				"RxOkPackets": "0",
				"RxOkBits": "0",
				"RxPps": "40527122.163856164",
				"TxBps": "4214792810.2678127",
				"TxErrBits": "0",
				"TxErrPackets": "0",
				"TxOkBits": "0",
				"TxOkPackets": "0",
				"TxPps": "40526901.803920366"
			}
	  }`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})
}
