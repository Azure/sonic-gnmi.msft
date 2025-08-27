package gnmi

// lldp_cli_test.go
// Tests SHOW lldp table

import (
	"crypto/tls"
	"io/ioutil"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestGetLLDPTable(t *testing.T) {
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
	
	ResetDataSetsAndMappings(t)

	lldpTableFileName := "../testdata/lldp/LLDP_ENTRY_TABLE.txt"

	// Expected output for the LLDP table
	expectedLLDPTableResponseFilName := "../testdata/lldp/Expected_show_lldp_table_response.txt"
	expectedEmptyDBResponse := `{"capability_codes_helper":"Capability codes: (R) Router, (B) Bridge, (O) Other","neighbors": [],"total": 0}`		
	expectedLLDPTableResponse, err := ioutil.ReadFile(expectedLLDPTableResponseFilName)
	if err != nil {
		t.Fatalf("Failed to read file %v err: %v", expectedLLDPTableResponseFilName, err)
	}

	tests := []struct {
		desc           string
		pathTarget     string
		textPbPath     string
		wantRetCode    codes.Code
		wantRespVal    interface{}
		valTest        bool
		ignoreValOrder bool
		testInit       func()
	}{
		{
			desc:       "query SHOW lldp table - empty DB",
			pathTarget: "SHOW",
			textPbPath:  `
				elem: <name: "lldp" >
				elem: <name: "table" >
			`,
			wantRetCode:    codes.OK,
			wantRespVal:    []byte(expectedEmptyDBResponse),
			valTest:        true,
			testInit: func() {
				FlushDataSet(t, ApplDbNum)
			},
		},
		{
			desc:       "query SHOW lldp table - with LLDP_ENTRY_TABLE",
			pathTarget: "SHOW",
			textPbPath:  `
				elem: <name: "lldp" >
				elem: <name: "table" >
			`,
			wantRetCode:    codes.OK,
			wantRespVal:    []byte(expectedLLDPTableResponse),
			valTest:        true,
			ignoreValOrder: true,
			testInit: func() {
				FlushDataSet(t, ApplDbNum)
				AddDataSet(t, ApplDbNum, lldpTableFileName)
			},
		},
	}

	for _, test := range tests {
		if test.testInit != nil {
			test.testInit()
		}
		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest, test.ignoreValOrder)
		})
	}
}