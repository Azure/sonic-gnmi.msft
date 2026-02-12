package gnmi

// platform_cli_test.go

// Tests SHOW platform summary, syseeprom, and psustatus

import (
	"crypto/tls"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"

	show_client "github.com/sonic-net/sonic-gnmi/show_client"
)

func TestGetShowPlatformSummary(t *testing.T) {
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

	versionInfo := `
build_version: test_branch.1-a8fbac59d
asic_type: mellanox
`
	deviceMetadataFilename := "../testdata/PLATFORM_METADATA.txt"
	chassisDataFilename := "../testdata/PLATFORM_CHASSIS.txt"

	expectedOutput := `{"platform":"x86_64-mlnx_msn2700-r0","hwsku":"Mellanox-SN2700","asic_type":"mellanox","asic_count":"1","serial_number":"MT1234X56789","model_number":"MSN2700-CS2FO","hardware_revision":"A1"}`

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
			desc:       "query SHOW platform summary error",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "summary" >
			`,
			wantRetCode: codes.NotFound,
		},
		{
			desc:       "query SHOW platform summary success",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "summary" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedOutput),
			valTest:     true,
			testInit: func() {
				MockReadFile(show_client.SonicVersionYamlPath, versionInfo, nil)
				MockReadFile(deviceMetadataFilename, "onie_platform=x86_64-mlnx_msn2700-r0", nil)
				MockReadFile(chassisDataFilename, `{"chassis 1": {"serial": "MT1234X56789", "model": "MSN2700-CS2FO", "revision": "A1"}}`, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if tt.testInit != nil {
				tt.testInit()
				defer ClearMockReadFile()
			}

			runTestGet(t, ctx, gClient, tt.pathTarget, tt.textPbPath, tt.wantRetCode, tt.wantRespVal, tt.valTest)
		})
	}
}

func TestGetShowPlatformSyseepromTlvFormat(t *testing.T) {
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

	// TLV format EEPROM data (Mellanox/ONIE)
	eepromDataTlv := `{
		"State": {"Initialized": "1"},
		"TlvHeader": {"Id String": "TlvInfo", "Version": "1", "Total Length": "256"},
		"0x21": {"Name": "Product Name", "Len": "10", "Value": "MSN2700"},
		"0x23": {"Name": "Serial Number", "Len": "16", "Value": "MT1234X56789"},
		"0x24": {"Name": "Base MAC Address", "Len": "12", "Value": "00:02:03:04:05:06"},
		"0xfe": {"Name": "CRC-32", "Len": "4", "Value": "0x12345678"},
		"Checksum": {"Valid": "1"}
	}`

	expectedOutputTlv := `{"tlvHeader":{"id":"TlvInfo","version":"1","length":"256"},"tlv_list":[{"code":"0x21","name":"Product Name","length":"10","value":"MSN2700"},{"code":"0x23","name":"Serial Number","length":"16","value":"MT1234X56789"},{"code":"0x24","name":"Base MAC Address","length":"12","value":"00:02:03:04:05:06"},{"code":"0xfe","name":"CRC-32","length":"4","value":"0x12345678"}],"checksum_valid":true}`

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
			desc:       "query SHOW platform syseeprom not initialized",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "syseeprom" >
			`,
			wantRetCode: codes.NotFound,
			testInit: func() {
				MockGetDataFromQueries([]byte(`{"State": {"Initialized": "0"}}`), nil)
			},
		},
		{
			desc:       "query SHOW platform syseeprom TLV format",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "syseeprom" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedOutputTlv),
			valTest:     true,
			testInit: func() {
				MockGetDataFromQueries([]byte(eepromDataTlv), nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if tt.testInit != nil {
				tt.testInit()
				defer ClearMockGetDataFromQueries()
			}

			runTestGet(t, ctx, gClient, tt.pathTarget, tt.textPbPath, tt.wantRetCode, tt.wantRespVal, tt.valTest)
		})
	}
}

func TestGetShowPlatformSyseepromSimpleFormat(t *testing.T) {
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

	// Simple format EEPROM data (Broadcom)
	eepromDataSimple := `{
		"State": {"Initialized": "1"},
		"ASY": {"Value": "dummy"},
		"HwApi": {"Value": "02.00"},
		"HwRev": {"Value": "11.00"},
		"MAC": {"Value": "00:11:22:33:44:55"},
		"SerialNumber": {"Value": "ABC123456789"},
		"SKU": {"Value": "BCM56850"}
	}`

	expectedOutputSimple := `{"ASY":"dummy","HwApi":"02.00","HwRev":"11.00","MAC":"00:11:22:33:44:55","SKU":"BCM56850","SerialNumber":"ABC123456789"}`

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
			desc:       "query SHOW platform syseeprom simple format (Broadcom)",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "syseeprom" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedOutputSimple),
			valTest:     true,
			testInit: func() {
				MockGetDataFromQueries([]byte(eepromDataSimple), nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if tt.testInit != nil {
				tt.testInit()
				defer ClearMockGetDataFromQueries()
			}

			runTestGet(t, ctx, gClient, tt.pathTarget, tt.textPbPath, tt.wantRetCode, tt.wantRespVal, tt.valTest)
		})
	}
}

func TestGetShowPlatformSyseepromVendorExtension(t *testing.T) {
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

	// TLV format with vendor extensions
	eepromDataVendorExt := `{
		"State": {"Initialized": "1"},
		"TlvHeader": {"Id String": "TlvInfo", "Version": "1", "Total Length": "256"},
		"0x21": {"Name": "Product Name", "Len": "10", "Value": "MSN2700"},
		"0xfd": {
			"Num_vendor_ext": "2",
			"Name_0": "Vendor Extension", "Len_0": "10", "Value_0": "CustomData1",
			"Name_1": "Vendor Extension", "Len_1": "8", "Value_1": "CustomData2"
		},
		"Checksum": {"Valid": "1"}
	}`

	expectedOutputVendorExt := `{"tlvHeader":{"id":"TlvInfo","version":"1","length":"256"},"tlv_list":[{"code":"0x21","name":"Product Name","length":"10","value":"MSN2700"},{"code":"0xfd","name":"Vendor Extension","length":"10","value":"CustomData1"},{"code":"0xfd","name":"Vendor Extension","length":"8","value":"CustomData2"}],"checksum_valid":true}`

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
			desc:       "query SHOW platform syseeprom with vendor extensions",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "syseeprom" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedOutputVendorExt),
			valTest:     true,
			testInit: func() {
				MockGetDataFromQueries([]byte(eepromDataVendorExt), nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if tt.testInit != nil {
				tt.testInit()
				defer ClearMockGetDataFromQueries()
			}

			runTestGet(t, ctx, gClient, tt.pathTarget, tt.textPbPath, tt.wantRetCode, tt.wantRespVal, tt.valTest)
		})
	}
}

func TestGetShowPlatformPsustatus(t *testing.T) {
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

	// PSU data with 2 PSUs (one present, one not present)
	psuData := `{
		"chassis 1": {"psu_num": "2"},
		"PSU 1": {
			"presence": "true",
			"status": "true",
			"power_overload": "false",
			"model": "PWR-500AC-F",
			"serial": "ABC12345678",
			"revision": "A1",
			"voltage": "12.00",
			"current": "5.50",
			"power": "66.00",
			"led_status": "green"
		},
		"PSU 2": {
			"presence": "false",
			"led_status": "off"
		}
	}`

	expectedOutput := `[{"index":"1","name":"PSU 1","presence":"true","model":"PWR-500AC-F","serial":"ABC12345678","revision":"A1","voltage":"12.00","current":"5.50","power":"66.00","status":"OK","led_status":"green"},{"index":"2","name":"PSU 2","presence":"false","model":"N/A","serial":"N/A","revision":"N/A","voltage":"N/A","current":"N/A","power":"N/A","status":"NOT PRESENT","led_status":"off"}]`

	// PSU data with power overload warning (note: 'True' not 'true')
	psuDataWarning := `{
		"chassis 1": {"psu_num": "1"},
		"PSU 1": {
			"presence": "true",
			"status": "true",
			"power_overload": "True",
			"model": "PWR-500AC-F",
			"serial": "XYZ98765432",
			"revision": "B2",
			"voltage": "12.50",
			"current": "8.00",
			"power": "100.00",
			"led_status": "amber"
		}
	}`

	expectedOutputWarning := `[{"index":"1","name":"PSU 1","presence":"true","model":"PWR-500AC-F","serial":"XYZ98765432","revision":"B2","voltage":"12.50","current":"8.00","power":"100.00","status":"WARNING","led_status":"amber"}]`

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
			desc:       "query SHOW platform psustatus no PSUs",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "psustatus" >
			`,
			wantRetCode: codes.NotFound,
			testInit: func() {
				MockGetDataFromQueries([]byte(`{"chassis 1": {}}`), nil)
			},
		},
		{
			desc:       "query SHOW platform psustatus with mixed PSU states",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "psustatus" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedOutput),
			valTest:     true,
			testInit: func() {
				MockGetDataFromQueries([]byte(psuData), nil)
			},
		},
		{
			desc:       "query SHOW platform psustatus with power overload warning",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "platform" >
				elem: <name: "psustatus" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedOutputWarning),
			valTest:     true,
			testInit: func() {
				MockGetDataFromQueries([]byte(psuDataWarning), nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if tt.testInit != nil {
				tt.testInit()
				defer ClearMockGetDataFromQueries()
			}

			runTestGet(t, ctx, gClient, tt.pathTarget, tt.textPbPath, tt.wantRetCode, tt.wantRespVal, tt.valTest)
		})
	}
}
