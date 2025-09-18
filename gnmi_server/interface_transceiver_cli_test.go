package gnmi

// interface_transceiver_cli_test.go

// Tests SHOW interface transceiver commands

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

func TestGetTransceiverErrorStatus(t *testing.T) {
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

	transceiverErrorStatusFileName := "../testdata/TRANSCEIVER_STATUS_SW.txt"
	transceiverErrorStatus := `{"Ethernet0":{"cmis_state": "READY","error": "N/A","status": "1"},"Ethernet40": {"cmis_state": "READY","error": "N/A","status": "1"},"Ethernet80": {"cmis_state": "READY","error": "N/A","status": "1"},"Ethernet120": {"cmis_state": "READY","error": "N/A","status": "1"},"Ethernet160": {"cmis_state": "READY","error": "N/A","status": "1"}}`
	transceiverErrorStatusPort := `{"cmis_state": "READY","error": "N/A","status": "1"}`
	ResetDataSetsAndMappings(t)

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
			desc:       "query SHOW interfaces transceiver error-status read error",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "" >
				elem: <name: "transceiver" >
				elem: <name: "error-status" >
			`,
			wantRetCode: codes.NotFound,
		},
		{
			desc:       "query SHOW interfaces transceiver error-status NO interface dataset",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "transceiver" >
				elem: <name: "error-status" >
			`,
			wantRetCode: codes.OK,
		},
		{
			desc:       "query SHOW interfaces transceiver error-status",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "transceiver" >
				elem: <name: "error-status" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(transceiverErrorStatus),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, StateDbNum)
				AddDataSet(t, StateDbNum, transceiverErrorStatusFileName)
			},
		},
		{
			desc:       "query SHOW interfaces transceiver error-status port option",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "transceiver" >
				elem: <name: "error-status" >
				elem: <name: "Ethernet80" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(transceiverErrorStatusPort),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, StateDbNum)
				AddDataSet(t, StateDbNum, transceiverErrorStatusFileName)
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

func TestGetTransceiverEEPROM(t *testing.T) {
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

	portTableFileName := "../testdata/TRANSCEIVER_PORT_TABLE.txt"
	portsFileName := "../testdata/TRANSCEIVER_PORTS.txt"
	transceiverInfoFileName := "../testdata/TRANSCEIVER_INFO.txt"
	transceiverFirmwareInfoFileName := "../testdata/TRANSCEIVER_FIRMWARE_INFO.txt"
	transceiverDomSensorFileName := "../testdata/TRANSCEIVER_DOM_SENSOR.txt"
	transceiverDomThresholdFileName := "../testdata/TRANSCEIVER_DOM_THRESHOLD.txt"
	transceiverErrorStatusFileName := "../testdata/TRANSCEIVER_STATUS_SW.txt"
	transceiverEEPROM := `{"Ethernet0":"{\"Active Firmware\":\"N/A\",\"Active application selected code assigned to host lane 1\":\"N/A\",\"Active application selected code assigned to host lane 2\":\"N/A\",\"Active application selected code assigned to host lane 3\":\"N/A\",\"Active application selected code assigned to host lane 4\":\"N/A\",\"Active application selected code assigned to host lane 5\":\"N/A\",\"Active application selected code assigned to host lane 6\":\"N/A\",\"Active application selected code assigned to host lane 7\":\"N/A\",\"Active application selected code assigned to host lane 8\":\"N/A\",\"Application Advertisement\":\"N/A\",\"CMIS Rev\":\"5.0\",\"Connector\":\"No separable connector\",\"Encoding\":\"N/A\",\"Extended Identifier\":\"Power Class 1 (0.25W Max)\",\"Extended RateSelect Compliance\":\"N/A\",\"Host Lane Count\":\"8\",\"Identifier\":\"OSFP 8X Pluggable Transceiver\",\"Inactive Firmware\":\"N/A\",\"Length Cable Assembly(m)\":\"1.0\",\"Media Interface Technology\":\"Copper cable unequalized\",\"Media Lane Count\":\"0\",\"Module Hardware Rev\":\"0.0\",\"Nominal Bit Rate(100Mbs)\":\"N/A\",\"Specification compliance\":\"passive_copper_media_interface\",\"Vendor Date Code(YYYY-MM-DD Lot)\":\"2024-11-25\",\"Vendor Name\":\"Amphenol\",\"Vendor OUI\":\"78-a7-14\",\"Vendor PN\":\"NJMMER-M201\",\"Vendor Rev\":\"A\",\"Vendor SN\":\"APF2447201229A\"}","Ethernet120":"{\"Active Firmware\":\"N/A\",\"Active application selected code assigned to host lane 1\":\"N/A\",\"Active application selected code assigned to host lane 2\":\"N/A\",\"Active application selected code assigned to host lane 3\":\"N/A\",\"Active application selected code assigned to host lane 4\":\"N/A\",\"Active application selected code assigned to host lane 5\":\"N/A\",\"Active application selected code assigned to host lane 6\":\"N/A\",\"Active application selected code assigned to host lane 7\":\"N/A\",\"Active application selected code assigned to host lane 8\":\"N/A\",\"Connector\":\"No separable connector\",\"Encoding\":\"N/A\",\"Extended Identifier\":\"Power Class 1 (0.25W Max)\",\"Extended RateSelect Compliance\":\"N/A\",\"Host Lane Count\":\"N/A\",\"Identifier\":\"OSFP 8X Pluggable Transceiver\",\"Inactive Firmware\":\"N/A\",\"Length Cable Assembly(m)\":\"2.0\",\"Media Interface Technology\":\"Copper cable unequalized\",\"Media Lane Count\":\"N/A\",\"Module Hardware Rev\":\"0.0\",\"Nominal Bit Rate(100Mbs)\":\"0\",\"Specification compliance\":\"N/A\",\"Supported Max Laser Frequency\":\"N/A\",\"Supported Max TX Power\":\"N/A\",\"Supported Min Laser Frequency\":\"N/A\",\"Supported Min TX Power\":\"N/A\",\"Vendor Date Code(YYYY-MM-DD Lot)\":\"2024-09-11   \",\"Vendor Name\":\"Arista Networks \",\"Vendor OUI\":\"a8-b0-ae\",\"Vendor PN\":\"CAB-D-D-400G-2M \",\"Vendor Rev\":\"00\",\"Vendor SN\":\"ZZ2409130192    \"}","Ethernet160":"{\"Active Firmware\":\"N/A\",\"Active application selected code assigned to host lane 1\":\"N/A\",\"Active application selected code assigned to host lane 2\":\"N/A\",\"Active application selected code assigned to host lane 3\":\"N/A\",\"Active application selected code assigned to host lane 4\":\"N/A\",\"Active application selected code assigned to host lane 5\":\"N/A\",\"Active application selected code assigned to host lane 6\":\"N/A\",\"Active application selected code assigned to host lane 7\":\"N/A\",\"Active application selected code assigned to host lane 8\":\"N/A\",\"CMIS Rev\":\"5.0\",\"Connector\":\"No separable connector\",\"Encoding\":\"N/A\",\"Extended Identifier\":\"Power Class 1 (0.25W Max)\",\"Extended RateSelect Compliance\":\"N/A\",\"Host Lane Count\":\"8\",\"Identifier\":\"\",\"Inactive Firmware\":\"N/A\",\"Length Cable Assembly(m)\":\"1.0\",\"Media Interface Technology\":\"Copper cable unequalized\",\"Media Lane Count\":\"0\",\"Module Hardware Rev\":\"0.0\",\"Nominal Bit Rate(100Mbs)\":\"N/A\",\"Specification compliance\":\"passive_copper_media_interface\",\"Vendor Date Code(YYYY-MM-DD Lot)\":\"2024-11-26\",\"Vendor Name\":\"Amphenol\",\"Vendor OUI\":\"78-a7-14\",\"Vendor PN\":\"NJMMER-M201\",\"Vendor Rev\":\"A\",\"Vendor SN\":\"APF244720121WT\"}","Ethernet40":"{\"Active Firmware\":\"N/A\",\"Active application selected code assigned to host lane 1\":\"N/A\",\"Active application selected code assigned to host lane 2\":\"N/A\",\"Active application selected code assigned to host lane 3\":\"N/A\",\"Active application selected code assigned to host lane 4\":\"N/A\",\"Active application selected code assigned to host lane 5\":\"N/A\",\"Active application selected code assigned to host lane 6\":\"N/A\",\"Active application selected code assigned to host lane 7\":\"N/A\",\"Active application selected code assigned to host lane 8\":\"N/A\",\"CMIS Rev\":\"5.0\",\"Connector\":\"No separable connector\",\"Encoding\":\"N/A\",\"Extended Identifier\":\"Power Class 1 (0.25W Max)\",\"Extended RateSelect Compliance\":\"N/A\",\"Host Lane Count\":\"8\",\"Identifier\":\"OSFP 8X Pluggable Transceiver\",\"Inactive Firmware\":\"N/A\",\"Length Cable Assembly(m)\":\"1.0\",\"Media Interface Technology\":\"Copper cable unequalized\",\"Media Lane Count\":\"0\",\"Module Hardware Rev\":\"0.0\",\"Nominal Bit Rate(100Mbs)\":\"N/A\",\"Specification compliance\":\"passive_copper_media_interface\",\"Vendor Date Code(YYYY-MM-DD Lot)\":\"2024-11-26\",\"Vendor Name\":\"Amphenol\",\"Vendor OUI\":\"78-a7-14\",\"Vendor PN\":\"NJMMER-M201\",\"Vendor Rev\":\"A\",\"Vendor SN\":\"APF244720121WT\"}","Ethernet80":"SFP EEPROM is not applicable for RJ45 port"}`
	transceiverEEPROMPort := `{"Ethernet40":"{\"Active Firmware\":\"N/A\",\"Active application selected code assigned to host lane 1\":\"N/A\",\"Active application selected code assigned to host lane 2\":\"N/A\",\"Active application selected code assigned to host lane 3\":\"N/A\",\"Active application selected code assigned to host lane 4\":\"N/A\",\"Active application selected code assigned to host lane 5\":\"N/A\",\"Active application selected code assigned to host lane 6\":\"N/A\",\"Active application selected code assigned to host lane 7\":\"N/A\",\"Active application selected code assigned to host lane 8\":\"N/A\",\"CMIS Rev\":\"5.0\",\"Connector\":\"No separable connector\",\"Encoding\":\"N/A\",\"Extended Identifier\":\"Power Class 1 (0.25W Max)\",\"Extended RateSelect Compliance\":\"N/A\",\"Host Lane Count\":\"8\",\"Identifier\":\"OSFP 8X Pluggable Transceiver\",\"Inactive Firmware\":\"N/A\",\"Length Cable Assembly(m)\":\"1.0\",\"Media Interface Technology\":\"Copper cable unequalized\",\"Media Lane Count\":\"0\",\"Module Hardware Rev\":\"0.0\",\"Nominal Bit Rate(100Mbs)\":\"N/A\",\"Specification compliance\":\"passive_copper_media_interface\",\"Vendor Date Code(YYYY-MM-DD Lot)\":\"2024-11-26\",\"Vendor Name\":\"Amphenol\",\"Vendor OUI\":\"78-a7-14\",\"Vendor PN\":\"NJMMER-M201\",\"Vendor Rev\":\"A\",\"Vendor SN\":\"APF244720121WT\"}"}`
	transceiverEEPROMDom := `{"Ethernet0":"{\"Active Firmware\":\"N/A\",\"Active application selected code assigned to host lane 1\":\"N/A\",\"Active application selected code assigned to host lane 2\":\"N/A\",\"Active application selected code assigned to host lane 3\":\"N/A\",\"Active application selected code assigned to host lane 4\":\"N/A\",\"Active application selected code assigned to host lane 5\":\"N/A\",\"Active application selected code assigned to host lane 6\":\"N/A\",\"Active application selected code assigned to host lane 7\":\"N/A\",\"Active application selected code assigned to host lane 8\":\"N/A\",\"Application Advertisement\":\"N/A\",\"CMIS Rev\":\"5.0\",\"ChannelMonitorValues\":{},\"ChannelThresholdValues\":{},\"Connector\":\"No separable connector\",\"Encoding\":\"N/A\",\"Extended Identifier\":\"Power Class 1 (0.25W Max)\",\"Extended RateSelect Compliance\":\"N/A\",\"Host Lane Count\":\"8\",\"Identifier\":\"OSFP 8X Pluggable Transceiver\",\"Inactive Firmware\":\"N/A\",\"Length Cable Assembly(m)\":\"1.0\",\"Media Interface Technology\":\"Copper cable unequalized\",\"Media Lane Count\":\"0\",\"Module Hardware Rev\":\"0.0\",\"ModuleMonitorValues\":{},\"ModuleThresholdValues\":{},\"Nominal Bit Rate(100Mbs)\":\"N/A\",\"Specification compliance\":\"passive_copper_media_interface\",\"Vendor Date Code(YYYY-MM-DD Lot)\":\"2024-11-25\",\"Vendor Name\":\"Amphenol\",\"Vendor OUI\":\"78-a7-14\",\"Vendor PN\":\"NJMMER-M201\",\"Vendor Rev\":\"A\",\"Vendor SN\":\"APF2447201229A\"}","Ethernet120":"{\"Active Firmware\":\"N/A\",\"Active application selected code assigned to host lane 1\":\"N/A\",\"Active application selected code assigned to host lane 2\":\"N/A\",\"Active application selected code assigned to host lane 3\":\"N/A\",\"Active application selected code assigned to host lane 4\":\"N/A\",\"Active application selected code assigned to host lane 5\":\"N/A\",\"Active application selected code assigned to host lane 6\":\"N/A\",\"Active application selected code assigned to host lane 7\":\"N/A\",\"Active application selected code assigned to host lane 8\":\"N/A\",\"ChannelMonitorValues\":{},\"ChannelThresholdValues\":{},\"Connector\":\"No separable connector\",\"Encoding\":\"N/A\",\"Extended Identifier\":\"Power Class 1 (0.25W Max)\",\"Extended RateSelect Compliance\":\"N/A\",\"Host Lane Count\":\"N/A\",\"Identifier\":\"OSFP 8X Pluggable Transceiver\",\"Inactive Firmware\":\"N/A\",\"Length Cable Assembly(m)\":\"2.0\",\"Media Interface Technology\":\"Copper cable unequalized\",\"Media Lane Count\":\"N/A\",\"Module Hardware Rev\":\"0.0\",\"ModuleMonitorValues\":{},\"ModuleThresholdValues\":{},\"Nominal Bit Rate(100Mbs)\":\"0\",\"Specification compliance\":\"N/A\",\"Supported Max Laser Frequency\":\"N/A\",\"Supported Max TX Power\":\"N/A\",\"Supported Min Laser Frequency\":\"N/A\",\"Supported Min TX Power\":\"N/A\",\"Vendor Date Code(YYYY-MM-DD Lot)\":\"2024-09-11   \",\"Vendor Name\":\"Arista Networks \",\"Vendor OUI\":\"a8-b0-ae\",\"Vendor PN\":\"CAB-D-D-400G-2M \",\"Vendor Rev\":\"00\",\"Vendor SN\":\"ZZ2409130192    \"}","Ethernet160":"{\"Active Firmware\":\"N/A\",\"Active application selected code assigned to host lane 1\":\"N/A\",\"Active application selected code assigned to host lane 2\":\"N/A\",\"Active application selected code assigned to host lane 3\":\"N/A\",\"Active application selected code assigned to host lane 4\":\"N/A\",\"Active application selected code assigned to host lane 5\":\"N/A\",\"Active application selected code assigned to host lane 6\":\"N/A\",\"Active application selected code assigned to host lane 7\":\"N/A\",\"Active application selected code assigned to host lane 8\":\"N/A\",\"CMIS Rev\":\"5.0\",\"Connector\":\"No separable connector\",\"Encoding\":\"N/A\",\"Extended Identifier\":\"Power Class 1 (0.25W Max)\",\"Extended RateSelect Compliance\":\"N/A\",\"Host Lane Count\":\"8\",\"Identifier\":\"\",\"Inactive Firmware\":\"N/A\",\"Length Cable Assembly(m)\":\"1.0\",\"Media Interface Technology\":\"Copper cable unequalized\",\"Media Lane Count\":\"0\",\"Module Hardware Rev\":\"0.0\",\"MonitorData\":{},\"Nominal Bit Rate(100Mbs)\":\"N/A\",\"Specification compliance\":\"passive_copper_media_interface\",\"ThresholdData\":{},\"Vendor Date Code(YYYY-MM-DD Lot)\":\"2024-11-26\",\"Vendor Name\":\"Amphenol\",\"Vendor OUI\":\"78-a7-14\",\"Vendor PN\":\"NJMMER-M201\",\"Vendor Rev\":\"A\",\"Vendor SN\":\"APF244720121WT\"}","Ethernet40":"{\"Active Firmware\":\"N/A\",\"Active application selected code assigned to host lane 1\":\"N/A\",\"Active application selected code assigned to host lane 2\":\"N/A\",\"Active application selected code assigned to host lane 3\":\"N/A\",\"Active application selected code assigned to host lane 4\":\"N/A\",\"Active application selected code assigned to host lane 5\":\"N/A\",\"Active application selected code assigned to host lane 6\":\"N/A\",\"Active application selected code assigned to host lane 7\":\"N/A\",\"Active application selected code assigned to host lane 8\":\"N/A\",\"CMIS Rev\":\"5.0\",\"ChannelMonitorValues\":{},\"ChannelThresholdValues\":{},\"Connector\":\"No separable connector\",\"Encoding\":\"N/A\",\"Extended Identifier\":\"Power Class 1 (0.25W Max)\",\"Extended RateSelect Compliance\":\"N/A\",\"Host Lane Count\":\"8\",\"Identifier\":\"OSFP 8X Pluggable Transceiver\",\"Inactive Firmware\":\"N/A\",\"Length Cable Assembly(m)\":\"1.0\",\"Media Interface Technology\":\"Copper cable unequalized\",\"Media Lane Count\":\"0\",\"Module Hardware Rev\":\"0.0\",\"ModuleMonitorValues\":{},\"ModuleThresholdValues\":{},\"Nominal Bit Rate(100Mbs)\":\"N/A\",\"Specification compliance\":\"passive_copper_media_interface\",\"Vendor Date Code(YYYY-MM-DD Lot)\":\"2024-11-26\",\"Vendor Name\":\"Amphenol\",\"Vendor OUI\":\"78-a7-14\",\"Vendor PN\":\"NJMMER-M201\",\"Vendor Rev\":\"A\",\"Vendor SN\":\"APF244720121WT\"}","Ethernet80":"SFP EEPROM is not applicable for RJ45 port"}`

	ResetDataSetsAndMappings(t)

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
			desc:       "query SHOW interface transceiver eeprom read error",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "" >
				elem: <name: "transceiver" >
				elem: <name: "eeprom" >
			`,
			wantRetCode: codes.NotFound,
		},
		{
			desc:       "query SHOW interface transceiver eeprom NO interface dataset",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "transceiver" >
				elem: <name: "eeprom" >
			`,
			wantRetCode: codes.OK,
		},
		{
			desc:       "query SHOW interface transceiver eeprom",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "transceiver" >
				elem: <name: "eeprom" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(transceiverEEPROM),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ApplDbNum)
				FlushDataSet(t, ConfigDbNum)
				FlushDataSet(t, StateDbNum)
				AddDataSet(t, ApplDbNum, portTableFileName)
				AddDataSet(t, ConfigDbNum, portsFileName)
				AddDataSet(t, StateDbNum, transceiverInfoFileName)
				AddDataSet(t, StateDbNum, transceiverFirmwareInfoFileName)
				AddDataSet(t, StateDbNum, transceiverDomSensorFileName)
				AddDataSet(t, StateDbNum, transceiverDomThresholdFileName)
				AddDataSet(t, StateDbNum, transceiverErrorStatusFileName)
			},
		},
		{
			desc:       "query SHOW interface transceiver eeprom port option",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "transceiver" >
				elem: <name: "eeprom" >
				elem: <name: "Ethernet40" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(transceiverEEPROMPort),
			valTest:     true,
		},
		{
			desc:       "query SHOW interface transceiver eeprom dom option",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "transceiver" >
				elem: <name: "eeprom" key: { key: "dom" value: "true" }>
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(transceiverEEPROMDom),
			valTest:     true,
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

func TestGetTransceiverInfo(t *testing.T) {
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

	portTableFileName := "../testdata/TRANSCEIVER_PORT_TABLE.txt"
	portsFileName := "../testdata/TRANSCEIVER_PORTS.txt"
	transceiverInfoFileName := "../testdata/TRANSCEIVER_INFO.txt"
	transceiverFirmwareInfoFileName := "../testdata/TRANSCEIVER_FIRMWARE_INFO.txt"
	transceiverDomSensorFileName := "../testdata/TRANSCEIVER_DOM_SENSOR.txt"
	transceiverDomThresholdFileName := "../testdata/TRANSCEIVER_DOM_THRESHOLD.txt"
	transceiverErrorStatusFileName := "../testdata/TRANSCEIVER_STATUS_SW.txt"

	transceiverInfo := `{"Ethernet0":"{\"Active Firmware\":\"N/A\",\"Active application selected code assigned to host lane 1\":\"N/A\",\"Active application selected code assigned to host lane 2\":\"N/A\",\"Active application selected code assigned to host lane 3\":\"N/A\",\"Active application selected code assigned to host lane 4\":\"N/A\",\"Active application selected code assigned to host lane 5\":\"N/A\",\"Active application selected code assigned to host lane 6\":\"N/A\",\"Active application selected code assigned to host lane 7\":\"N/A\",\"Active application selected code assigned to host lane 8\":\"N/A\",\"Application Advertisement\":\"N/A\",\"CMIS Rev\":\"5.0\",\"Connector\":\"No separable connector\",\"Encoding\":\"N/A\",\"Extended Identifier\":\"Power Class 1 (0.25W Max)\",\"Extended RateSelect Compliance\":\"N/A\",\"Host Lane Count\":\"8\",\"Identifier\":\"OSFP 8X Pluggable Transceiver\",\"Inactive Firmware\":\"N/A\",\"Length Cable Assembly(m)\":\"1.0\",\"Media Interface Technology\":\"Copper cable unequalized\",\"Media Lane Count\":\"0\",\"Module Hardware Rev\":\"0.0\",\"Nominal Bit Rate(100Mbs)\":\"N/A\",\"Specification compliance\":\"passive_copper_media_interface\",\"Vendor Date Code(YYYY-MM-DD Lot)\":\"2024-11-25\",\"Vendor Name\":\"Amphenol\",\"Vendor OUI\":\"78-a7-14\",\"Vendor PN\":\"NJMMER-M201\",\"Vendor Rev\":\"A\",\"Vendor SN\":\"APF2447201229A\"}","Ethernet120":"{\"Active Firmware\":\"N/A\",\"Active application selected code assigned to host lane 1\":\"N/A\",\"Active application selected code assigned to host lane 2\":\"N/A\",\"Active application selected code assigned to host lane 3\":\"N/A\",\"Active application selected code assigned to host lane 4\":\"N/A\",\"Active application selected code assigned to host lane 5\":\"N/A\",\"Active application selected code assigned to host lane 6\":\"N/A\",\"Active application selected code assigned to host lane 7\":\"N/A\",\"Active application selected code assigned to host lane 8\":\"N/A\",\"Connector\":\"No separable connector\",\"Encoding\":\"N/A\",\"Extended Identifier\":\"Power Class 1 (0.25W Max)\",\"Extended RateSelect Compliance\":\"N/A\",\"Host Lane Count\":\"N/A\",\"Identifier\":\"OSFP 8X Pluggable Transceiver\",\"Inactive Firmware\":\"N/A\",\"Length Cable Assembly(m)\":\"2.0\",\"Media Interface Technology\":\"Copper cable unequalized\",\"Media Lane Count\":\"N/A\",\"Module Hardware Rev\":\"0.0\",\"Nominal Bit Rate(100Mbs)\":\"0\",\"Specification compliance\":\"N/A\",\"Supported Max Laser Frequency\":\"N/A\",\"Supported Max TX Power\":\"N/A\",\"Supported Min Laser Frequency\":\"N/A\",\"Supported Min TX Power\":\"N/A\",\"Vendor Date Code(YYYY-MM-DD Lot)\":\"2024-09-11   \",\"Vendor Name\":\"Arista Networks \",\"Vendor OUI\":\"a8-b0-ae\",\"Vendor PN\":\"CAB-D-D-400G-2M \",\"Vendor Rev\":\"00\",\"Vendor SN\":\"ZZ2409130192    \"}","Ethernet160":"{\"Active Firmware\":\"N/A\",\"Active application selected code assigned to host lane 1\":\"N/A\",\"Active application selected code assigned to host lane 2\":\"N/A\",\"Active application selected code assigned to host lane 3\":\"N/A\",\"Active application selected code assigned to host lane 4\":\"N/A\",\"Active application selected code assigned to host lane 5\":\"N/A\",\"Active application selected code assigned to host lane 6\":\"N/A\",\"Active application selected code assigned to host lane 7\":\"N/A\",\"Active application selected code assigned to host lane 8\":\"N/A\",\"CMIS Rev\":\"5.0\",\"Connector\":\"No separable connector\",\"Encoding\":\"N/A\",\"Extended Identifier\":\"Power Class 1 (0.25W Max)\",\"Extended RateSelect Compliance\":\"N/A\",\"Host Lane Count\":\"8\",\"Identifier\":\"\",\"Inactive Firmware\":\"N/A\",\"Length Cable Assembly(m)\":\"1.0\",\"Media Interface Technology\":\"Copper cable unequalized\",\"Media Lane Count\":\"0\",\"Module Hardware Rev\":\"0.0\",\"Nominal Bit Rate(100Mbs)\":\"N/A\",\"Specification compliance\":\"passive_copper_media_interface\",\"Vendor Date Code(YYYY-MM-DD Lot)\":\"2024-11-26\",\"Vendor Name\":\"Amphenol\",\"Vendor OUI\":\"78-a7-14\",\"Vendor PN\":\"NJMMER-M201\",\"Vendor Rev\":\"A\",\"Vendor SN\":\"APF244720121WT\"}","Ethernet40":"{\"Active Firmware\":\"N/A\",\"Active application selected code assigned to host lane 1\":\"N/A\",\"Active application selected code assigned to host lane 2\":\"N/A\",\"Active application selected code assigned to host lane 3\":\"N/A\",\"Active application selected code assigned to host lane 4\":\"N/A\",\"Active application selected code assigned to host lane 5\":\"N/A\",\"Active application selected code assigned to host lane 6\":\"N/A\",\"Active application selected code assigned to host lane 7\":\"N/A\",\"Active application selected code assigned to host lane 8\":\"N/A\",\"CMIS Rev\":\"5.0\",\"Connector\":\"No separable connector\",\"Encoding\":\"N/A\",\"Extended Identifier\":\"Power Class 1 (0.25W Max)\",\"Extended RateSelect Compliance\":\"N/A\",\"Host Lane Count\":\"8\",\"Identifier\":\"OSFP 8X Pluggable Transceiver\",\"Inactive Firmware\":\"N/A\",\"Length Cable Assembly(m)\":\"1.0\",\"Media Interface Technology\":\"Copper cable unequalized\",\"Media Lane Count\":\"0\",\"Module Hardware Rev\":\"0.0\",\"Nominal Bit Rate(100Mbs)\":\"N/A\",\"Specification compliance\":\"passive_copper_media_interface\",\"Vendor Date Code(YYYY-MM-DD Lot)\":\"2024-11-26\",\"Vendor Name\":\"Amphenol\",\"Vendor OUI\":\"78-a7-14\",\"Vendor PN\":\"NJMMER-M201\",\"Vendor Rev\":\"A\",\"Vendor SN\":\"APF244720121WT\"}","Ethernet80":"SFP EEPROM is not applicable for RJ45 port"}`
	transceiverInfoPort := `{"Ethernet40":"{\"Active Firmware\":\"N/A\",\"Active application selected code assigned to host lane 1\":\"N/A\",\"Active application selected code assigned to host lane 2\":\"N/A\",\"Active application selected code assigned to host lane 3\":\"N/A\",\"Active application selected code assigned to host lane 4\":\"N/A\",\"Active application selected code assigned to host lane 5\":\"N/A\",\"Active application selected code assigned to host lane 6\":\"N/A\",\"Active application selected code assigned to host lane 7\":\"N/A\",\"Active application selected code assigned to host lane 8\":\"N/A\",\"CMIS Rev\":\"5.0\",\"Connector\":\"No separable connector\",\"Encoding\":\"N/A\",\"Extended Identifier\":\"Power Class 1 (0.25W Max)\",\"Extended RateSelect Compliance\":\"N/A\",\"Host Lane Count\":\"8\",\"Identifier\":\"OSFP 8X Pluggable Transceiver\",\"Inactive Firmware\":\"N/A\",\"Length Cable Assembly(m)\":\"1.0\",\"Media Interface Technology\":\"Copper cable unequalized\",\"Media Lane Count\":\"0\",\"Module Hardware Rev\":\"0.0\",\"Nominal Bit Rate(100Mbs)\":\"N/A\",\"Specification compliance\":\"passive_copper_media_interface\",\"Vendor Date Code(YYYY-MM-DD Lot)\":\"2024-11-26\",\"Vendor Name\":\"Amphenol\",\"Vendor OUI\":\"78-a7-14\",\"Vendor PN\":\"NJMMER-M201\",\"Vendor Rev\":\"A\",\"Vendor SN\":\"APF244720121WT\"}"}`
	ResetDataSetsAndMappings(t)

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
			desc:       "query SHOW interface transceiver info read error",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "" >
				elem: <name: "transceiver" >
				elem: <name: "info" >
			`,
			wantRetCode: codes.NotFound,
		},
		{
			desc:       "query SHOW interface transceiver info NO interface dataset",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "transceiver" >
				elem: <name: "info" >
			`,
			wantRetCode: codes.OK,
		},
		{
			desc:       "query SHOW interface transceiver info",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "transceiver" >
				elem: <name: "info" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(transceiverInfo),
			valTest:     true,
			testInit: func() {
				FlushDataSet(t, ApplDbNum)
				FlushDataSet(t, ConfigDbNum)
				FlushDataSet(t, StateDbNum)
				AddDataSet(t, ApplDbNum, portTableFileName)
				AddDataSet(t, ConfigDbNum, portsFileName)
				AddDataSet(t, StateDbNum, transceiverInfoFileName)
				AddDataSet(t, StateDbNum, transceiverFirmwareInfoFileName)
				AddDataSet(t, StateDbNum, transceiverDomSensorFileName)
				AddDataSet(t, StateDbNum, transceiverDomThresholdFileName)
				AddDataSet(t, StateDbNum, transceiverErrorStatusFileName)
			},
		},
		{
			desc:       "query SHOW interface transceiver info port option",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "transceiver" >
				elem: <name: "eeprom" >
				elem: <name: "Ethernet40" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(transceiverInfoPort),
			valTest:     true,
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
