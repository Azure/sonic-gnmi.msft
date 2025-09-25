package gnmi

// interface_cli_test.go

// Tests SHOW interface/counters

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

func TestGetInterfaceCounters(t *testing.T) {
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

	MockReadFile("/tmp/cache/portstat/1000/portstat", "", nil)

	portsFileName := "../testdata/PORTS.txt"
	portOidMappingFileName := "../testdata/PORT_COUNTERS_MAPPING.txt"
	portCountersFileName := "../testdata/PORT_COUNTERS.txt"
	portCountersTwoFileName := "../testdata/PORT_COUNTERS_TWO.txt"
	portRatesFileName := "../testdata/PORT_RATES.txt"
	portRatesTwoFileName := "../testdata/PORT_RATES_TWO.txt"
	portRatesThreeFileName := "../testdata/PORT_RATES_THREE.txt"
	portTableFileName := "../testdata/PORT_TABLE.txt"
	stateDBPortTableFileName := "../testdata/INTERFACE_COUNTERS_STATE_PORT_TABLE.txt"
	interfaceCountersAll := `{"Ethernet0":{"State":"U","RxOk":"149903","RxBps":"25.12 B/s","RxUtil":"0.00%","RxErr":"0","RxDrp":"957","RxOvr":"0","TxOk":"144782","TxBps":"773.23 KB/s","TxUtil":"0.01%","TxErr":"0","TxDrp":"2","TxOvr":"0"},"Ethernet40":{"State":"U","RxOk":"7295","RxBps":"0.00 B/s","RxUtil":"0.00%","RxErr":"0","RxDrp":"0","RxOvr":"0","TxOk":"50184","TxBps":"633.66 KB/s","TxUtil":"0.01%","TxErr":"0","TxDrp":"1","TxOvr":"0"},"Ethernet80":{"State":"U","RxOk":"76555","RxBps":"0.37 B/s","RxUtil":"0.00%","RxErr":"0","RxDrp":"0","RxOvr":"0","TxOk":"144767","TxBps":"631.94 KB/s","TxUtil":"0.01%","TxErr":"0","TxDrp":"1","TxOvr":"0"}}`
	interfaceCountersSelectPorts := `{"Ethernet0":{"State":"U","RxOk":"149903","RxBps":"25.12 B/s","RxUtil":"0.00%","RxErr":"0","RxDrp":"957","RxOvr":"0","TxOk":"144782","TxBps":"773.23 KB/s","TxUtil":"0.01%","TxErr":"0","TxDrp":"2","TxOvr":"0"}}`
	interfaceCountersDiff := `{"Ethernet0":{"State":"U","RxOk":"11658","RxBps":"21.39 B/s","RxUtil":"0.00%","RxErr":"0","RxDrp":"76","RxOvr":"0","TxOk":"11270","TxBps":"634.00 KB/s","TxUtil":"0.01%","TxErr":"0","TxDrp":"0","TxOvr":"0"}}`
	interfaceCountersAllPrintall := `{"Ethernet0":{"State":"U","RxOk":"149903","RxBps":"25.12 B/s","RxPps":"0.18/s","RxUtil":"0.00%","RxErr":"0","RxDrp":"957","RxOvr":"0","TxOk":"144782","TxBps":"773.23 KB/s","TxPps":"0.27/s","TxUtil":"0.01%","TxErr":"0","TxDrp":"2","TxOvr":"0","TrimPkts":"7","TrimSent":"1","TrimDrp":"0"},"Ethernet40":{"State":"U","RxOk":"7295","RxBps":"0.00 B/s","RxPps":"0.00/s","RxUtil":"0.00%","RxErr":"0","RxDrp":"0","RxOvr":"0","TxOk":"50184","TxBps":"633.66 KB/s","TxPps":"0.10/s","TxUtil":"0.01%","TxErr":"0","TxDrp":"1","TxOvr":"0","TrimPkts":"N/A","TrimSent":"N/A","TrimDrp":"N/A"},"Ethernet80":{"State":"U","RxOk":"76555","RxBps":"0.37 B/s","RxPps":"0.00/s","RxUtil":"0.00%","RxErr":"0","RxDrp":"0","RxOvr":"0","TxOk":"144767","TxBps":"631.94 KB/s","TxPps":"0.04/s","TxUtil":"0.01%","TxErr":"0","TxDrp":"1","TxOvr":"0","TrimPkts":"N/A","TrimSent":"N/A","TrimDrp":"N/A"}}`
	interfaceCountersPrintallEth0 := `{"Ethernet0":{"State":"U","RxOk":"149903","RxBps":"25.12 B/s","RxPps":"0.18/s","RxUtil":"0.00%","RxErr":"0","RxDrp":"957","RxOvr":"0","TxOk":"144782","TxBps":"773.23 KB/s","TxPps":"0.27/s","TxUtil":"0.01%","TxErr":"0","TxDrp":"2","TxOvr":"0","TrimPkts":"7","TrimSent":"1","TrimDrp":"0"}}`
	interfaceCountersPrintallEth0Period := `{"Ethernet0":{"State":"U","RxOk":"11658","RxBps":"21.39 B/s","RxPps":"0.09/s","RxUtil":"0.00%","RxErr":"0","RxDrp":"76","RxOvr":"0","TxOk":"11270","TxBps":"634.00 KB/s","TxPps":"0.11/s","TxUtil":"0.01%","TxErr":"0","TxDrp":"0","TxOvr":"0","TrimPkts":"2","TrimSent":"2","TrimDrp":"1"}}`
	interfaceCountersErrorsAll := `{"Ethernet0":{"State":"U","RxErr":"0","RxDrp":"957","RxOvr":"0","TxErr":"0","TxDrp":"2","TxOvr":"0"},"Ethernet40":{"State":"U","RxErr":"0","RxDrp":"0","RxOvr":"0","TxErr":"0","TxDrp":"1","TxOvr":"0"},"Ethernet80":{"State":"U","RxErr":"0","RxDrp":"0","RxOvr":"0","TxErr":"0","TxDrp":"1","TxOvr":"0"}}`
	interfaceCountersErrorsAllPeriod := `{"Ethernet0":{"State":"U","RxErr":"0","RxDrp":"76","RxOvr":"0","TxErr":"0","TxDrp":"0","TxOvr":"0"},"Ethernet40":{"State":"U","RxErr":"0","RxDrp":"0","RxOvr":"0","TxErr":"0","TxDrp":"0","TxOvr":"0"},"Ethernet80":{"State":"U","RxErr":"0","RxDrp":"0","RxOvr":"0","TxErr":"0","TxDrp":"0","TxOvr":"0"}}`
	interfaceCountersTrimAll := `{"Ethernet0":{"State":"U","TrimPkts":"7","TrimSent":"1","TrimDrp":"0"},"Ethernet40":{"State":"U","TrimPkts":"N/A","TrimSent":"N/A","TrimDrp":"N/A"},"Ethernet80":{"State":"U","TrimPkts":"N/A","TrimSent":"N/A","TrimDrp":"N/A"}}`
	interfaceCountersTrimEth0 := `{"Ethernet0":{"State":"U","TrimPkts":"7","TrimSent":"1","TrimDrp":"0"}}`
	interfaceCountersTrimEth0Period := `{"Ethernet0":{"State":"U","TrimPkts":"2","TrimSent":"2","TrimDrp":"1"}}`
	interfaceCountersRatesAll := `{"Ethernet0":{"State":"U","RxOk":"149903","RxBps":"25.12 B/s","RxPps":"0.18/s","RxUtil":"0.00%","TxOk":"144782","TxBps":"773.23 KB/s","TxPps":"0.27/s","TxUtil":"0.01%"},"Ethernet40":{"State":"U","RxOk":"7295","RxBps":"0.00 B/s","RxPps":"0.00/s","RxUtil":"0.00%","TxOk":"50184","TxBps":"633.66 KB/s","TxPps":"0.10/s","TxUtil":"0.01%"},"Ethernet80":{"State":"U","RxOk":"76555","RxBps":"0.37 B/s","RxPps":"0.00/s","RxUtil":"0.00%","TxOk":"144767","TxBps":"631.94 KB/s","TxPps":"0.04/s","TxUtil":"0.01%"}}`
	interfaceCountersRatesAllPeriod := `{"Ethernet0":{"State":"U","RxOk":"11658","RxBps":"21.39 B/s","RxPps":"0.09/s","RxUtil":"0.00%","TxOk":"11270","TxBps":"634.00 KB/s","TxPps":"0.11/s","TxUtil":"0.01%"},"Ethernet40":{"State":"U","RxOk":"568","RxBps":"0.00 B/s","RxPps":"0.00/s","RxUtil":"0.00%","TxOk":"3893","TxBps":"631.47 KB/s","TxPps":"0.00/s","TxUtil":"0.01%"},"Ethernet80":{"State":"U","RxOk":"5980","RxBps":"4.60 B/s","RxPps":"0.03/s","RxUtil":"0.00%","TxOk":"11313","TxBps":"634.75 KB/s","TxPps":"0.03/s","TxUtil":"0.01%"}}`
	interfaceCountersFecStatsAll := `{"Ethernet0":{"State":"U","FecCorr":"0","FecUncorr":"0","FecSymbolErr":"0","FecPreBer":"0.00e+00","FecPostBer":"0.00e+00"},"Ethernet40":{"State":"U","FecCorr":"0","FecUncorr":"0","FecSymbolErr":"0","FecPreBer":"0.00e+00","FecPostBer":"0.00e+00"},"Ethernet80":{"State":"U","FecCorr":"0","FecUncorr":"0","FecSymbolErr":"0","FecPreBer":"0.00e+00","FecPostBer":"0.00e+00"}}`
	interfaceCountersFecStatsAllPeriod := `{"Ethernet0":{"State":"U","FecCorr":"0","FecUncorr":"0","FecSymbolErr":"0","FecPreBer":"4.70e+07","FecPostBer":"0.00e+00"},"Ethernet40":{"State":"U","FecCorr":"0","FecUncorr":"0","FecSymbolErr":"0","FecPreBer":"4.70e+07","FecPostBer":"0.00e+00"},"Ethernet80":{"State":"U","FecCorr":"0","FecUncorr":"0","FecSymbolErr":"0","FecPreBer":"4.70e+07","FecPostBer":"0.00e+00"}}`
	interfaceCountersFecHistogramEth0 := `[{"BinIndex":"BIN0","Codewords":"20113191987857"},{"BinIndex":"BIN1","Codewords":"0"},{"BinIndex":"BIN2","Codewords":"0"},{"BinIndex":"BIN3","Codewords":"0"},{"BinIndex":"BIN4","Codewords":"0"},{"BinIndex":"BIN5","Codewords":"0"},{"BinIndex":"BIN6","Codewords":"0"},{"BinIndex":"BIN7","Codewords":"0"},{"BinIndex":"BIN8","Codewords":"0"},{"BinIndex":"BIN9","Codewords":"0"},{"BinIndex":"BIN10","Codewords":"0"},{"BinIndex":"BIN11","Codewords":"0"},{"BinIndex":"BIN12","Codewords":"0"},{"BinIndex":"BIN13","Codewords":"0"},{"BinIndex":"BIN14","Codewords":"0"},{"BinIndex":"BIN15","Codewords":"0"}]`
	interfaceCountersDetailedEth0NoCache := `{"Ethernet0":{"TrimPkts":"7","TrimSent":"1","TrimDrp":"0","Rx64":"2","Rx65_127":"81146","Rx128_255":"68687","Rx256_511":"0","Rx512_1023":"3","Rx1024_1518":"2","Rx1519_2047":"0","Rx2048_4095":"N/A","Rx4096_9216":"N/A","Rx9217_16383":"N/A","Tx64":"1","Tx65_127":"76016","Tx128_255":"34346","Tx256_511":"34331","Tx512_1023":"0","Tx1024_1518":"0","Tx1519_2047":"13","Tx2048_4095":"N/A","Tx4096_9216":"N/A","Tx9217_16383":"N/A","RxAll":"149903","RxUnicast":"80654","RxMulticast":"69249","RxBroadcast":"0","TxAll":"144782","TxUnicast":"144782","TxMulticast":"0","TxBroadcast":"0","RxJabbers":"N/A","RxFragments":"N/A","RxUndersize":"0","RxOverruns":"N/A","TimestampClearedCounters":"None"}}`
	interfaceCountersDetailedEth0Cache := `{"Ethernet0":{"TrimPkts":"7","TrimSent":"1","TrimDrp":"0","Rx64":"0","Rx65_127":"0","Rx128_255":"0","Rx256_511":"0","Rx512_1023":"0","Rx1024_1518":"0","Rx1519_2047":"0","Rx2048_4095":"N/A","Rx4096_9216":"N/A","Rx9217_16383":"N/A","Tx64":"0","Tx65_127":"0","Tx128_255":"0","Tx256_511":"34330","Tx512_1023":"0","Tx1024_1518":"0","Tx1519_2047":"11","Tx2048_4095":"N/A","Tx4096_9216":"N/A","Tx9217_16383":"N/A","RxAll":"0","RxUnicast":"0","RxMulticast":"0","RxBroadcast":"0","TxAll":"0","TxUnicast":"0","TxMulticast":"0","TxBroadcast":"0","RxJabbers":"N/A","RxFragments":"N/A","RxUndersize":"0","RxOverruns":"N/A","TimestampClearedCounters":"2025-09-21T18:45:04.083017"}}`
	interfaceCountersDetailedEth0CachePeriod := `{"Ethernet0":{"TrimPkts":"2","TrimSent":"2","TrimDrp":"1","Rx64":"0","Rx65_127":"0","Rx128_255":"0","Rx256_511":"0","Rx512_1023":"0","Rx1024_1518":"0","Rx1519_2047":"0","Rx2048_4095":"N/A","Rx4096_9216":"N/A","Rx9217_16383":"N/A","Tx64":"0","Tx65_127":"0","Tx128_255":"0","Tx256_511":"2672","Tx512_1023":"0","Tx1024_1518":"0","Tx1519_2047":"0","Tx2048_4095":"N/A","Tx4096_9216":"N/A","Tx9217_16383":"N/A","RxAll":"0","RxUnicast":"0","RxMulticast":"0","RxBroadcast":"0","TxAll":"0","TxUnicast":"0","TxMulticast":"0","TxBroadcast":"0","RxJabbers":"N/A","RxFragments":"N/A","RxUndersize":"0","RxOverruns":"N/A","TimestampClearedCounters":"2025-09-21T18:45:04.083017"}}`
	portStatCacheJSON := `{"time":"2025-09-21T18:45:04.083017","Ethernet0":{"rx_ok":"163588","rx_err":"0","rx_drop":"1042","rx_ovr":"0","tx_ok":"158164","tx_err":"0","tx_drop":"1","tx_ovr":"0","rx_byt":"21752387","tx_byt":"790069589049","rx_64":"6","rx_65_127":"88534","rx_128_255":"74911","rx_256_511":"1","rx_512_1023":"3","rx_1024_1518":"2","rx_1519_2047":"5","rx_2048_4095":"N/A","rx_4096_9216":"N/A","rx_9217_16383":"N/A","rx_uca":"88066","rx_mca":"75521","rx_bca":"1","rx_all":"163588","tx_64":"5","tx_65_127":"83139","tx_128_255":"74928","tx_256_511":"1","tx_512_1023":"0","tx_1024_1518":"1","tx_1519_2047":"2","tx_2048_4095":"N/A","tx_4096_9216":"N/A","tx_9217_16383":"N/A","tx_uca":"158164","tx_mca":"0","tx_bca":"0","tx_all":"158164","rx_jbr":"N/A","rx_frag":"N/A","rx_usize":"0","rx_ovrrun":"N/A","fec_corr":"0","fec_uncorr":"0","fec_symbol_err":"0","wred_grn_drp_pkt":"N/A","wred_ylw_drp_pkt":"N/A","wred_red_drp_pkt":"N/A","wred_tot_drp_pkt":"N/A","trim":"N/A"},"Ethernet40":{"rx_ok":"7963","rx_err":"0","rx_drop":"0","rx_ovr":"0","tx_ok":"54826","tx_err":"0","tx_drop":"1","tx_ovr":"0","rx_byt":"832246","tx_byt":"790047128811","rx_64":"0","rx_65_127":"7963","rx_128_255":"0","rx_256_511":"0","rx_512_1023":"0","rx_1024_1518":"0","rx_1519_2047":"0","rx_2048_4095":"N/A","rx_4096_9216":"N/A","rx_9217_16383":"N/A","rx_uca":"7963","rx_mca":"0","rx_bca":"0","rx_all":"7963","tx_64":"1","tx_65_127":"17348","tx_128_255":"28","tx_256_511":"37449","tx_512_1023":"0","tx_1024_1518":"0","tx_1519_2047":"0","tx_2048_4095":"N/A","tx_4096_9216":"N/A","tx_9217_16383":"N/A","tx_uca":"54826","tx_mca":"0","tx_bca":"0","tx_all":"54826","rx_jbr":"N/A","rx_frag":"N/A","rx_usize":"0","rx_ovrrun":"N/A","fec_corr":"0","fec_uncorr":"0","fec_symbol_err":"0","wred_grn_drp_pkt":"N/A","wred_ylw_drp_pkt":"N/A","wred_red_drp_pkt":"N/A","wred_tot_drp_pkt":"N/A","trim":"N/A"},"Ethernet80":{"rx_ok":"83480","rx_err":"0","rx_drop":"0","rx_ovr":"0","tx_ok":"158320","tx_err":"0","tx_drop":"1","tx_ovr":"0","rx_byt":"13348940","tx_byt":"790055614701","rx_64":"0","rx_65_127":"8577","rx_128_255":"74903","rx_256_511":"0","rx_512_1023":"0","rx_1024_1518":"0","rx_1519_2047":"0","rx_2048_4095":"N/A","rx_4096_9216":"N/A","rx_9217_16383":"N/A","rx_uca":"7966","rx_mca":"75514","rx_bca":"0","rx_all":"83480","tx_64":"2","tx_65_127":"83161","tx_128_255":"74955","tx_256_511":"1","tx_512_1023":"1","tx_1024_1518":"0","tx_1519_2047":"14","tx_2048_4095":"N/A","tx_4096_9216":"N/A","tx_9217_16383":"N/A","tx_uca":"158320","tx_mca":"0","tx_bca":"0","tx_all":"158320","rx_jbr":"N/A","rx_frag":"N/A","rx_usize":"0","rx_ovrrun":"N/A","fec_corr":"0","fec_uncorr":"0","fec_symbol_err":"0","wred_grn_drp_pkt":"N/A","wred_ylw_drp_pkt":"N/A","wred_red_drp_pkt":"N/A","wred_tot_drp_pkt":"N/A","trim":"N/A"}}`

	ResetDataSetsAndMappings(t)

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
			desc:       "query SHOW interfaces counters NO DATA",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" >
			`,
			wantRetCode: codes.OK,
		},
		{
			desc:       "query SHOW interfaces counters",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersAll),
			valTest:     true,
			testInit: func() {
				AddDataSet(t, ConfigDbNum, portsFileName)
				AddDataSet(t, CountersDbNum, portOidMappingFileName)
				AddDataSet(t, CountersDbNum, portCountersFileName)
				AddDataSet(t, CountersDbNum, portRatesFileName)
				AddDataSet(t, ApplDbNum, portTableFileName)
			},
		},
		{
			desc:       "query SHOW interfaces counters interfaces option",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" key: { key: "interface" value: "Ethernet0" }>
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersSelectPorts),
			valTest:     true,
		},
		{
			desc:       "query SHOW interfaces counters interfaces option via STATE_DB",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" key: { key: "interface" value: "Ethernet0" }>
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersSelectPorts),
			testInit: func() {
				AddDataSet(t, StateDbNum, stateDBPortTableFileName)
			},
			valTest: true,
		},
		{
			desc:       "query SHOW interfaces counters period option",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters"
				      key: { key: "interface" value: "Ethernet0" }
				      key: { key: "period" value: "5" }>
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersDiff),
			valTest:     true,
			mockSleep:   true,
		},
		{
			desc:       "SHOW interfaces counters printall all ports",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters"  key: <key: "printall" value: "true"> >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersAllPrintall),
			valTest:     true,
		},
		{
			desc:       "SHOW interfaces counters printall single port",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters"  key: <key: "interface" value: "Ethernet0">  key: <key: "printall" value: "true"> >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersPrintallEth0),
			valTest:     true,
		},
		{
			desc:       "SHOW interfaces counters printall single port with period",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters"  key: <key: "interface" value: "Ethernet0">  key: <key: "printall" value: "true">  key: <key: "period" value: "5"> >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersPrintallEth0Period),
			valTest:     true,
			mockSleep:   true,
		},
		{
			desc:       "SHOW interfaces counters errors all ports",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" >
				elem: <name: "errors" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersErrorsAll),
			valTest:     true,
		},
		{
			desc:       "SHOW interfaces counters errors all ports with period",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" >
				elem: <name: "errors"  key: <key: "period" value: "5"> >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersErrorsAllPeriod),
			valTest:     true,
			mockSleep:   true,
		},
		{
			desc:       "SHOW interfaces counters trim all ports",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" >
				elem: <name: "trim" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersTrimAll),
			valTest:     true,
		},
		{
			desc:       "SHOW interfaces counters trim Ethernet0",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" >
				elem: <name: "trim" >
				elem: <name: "Ethernet0" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersTrimEth0),
			valTest:     true,
		},
		{
			desc:       "SHOW interfaces counters trim Ethernet0 with period",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" >
				elem: <name: "trim" >
				elem: <name: "Ethernet0"  key: <key: "period" value: "5"> >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersTrimEth0Period),
			valTest:     true,
			mockSleep:   true,
		},
		{
			desc:       "SHOW interfaces counters rates all ports",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" >
				elem: <name: "rates" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersRatesAll),
			valTest:     true,
		},
		{
			desc:       "SHOW interfaces counters rates with RXUTIL/TXUTIL in data",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" >
				elem: <name: "rates" >
			`,
			testInit: func() {
				AddDataSet(t, CountersDbNum, portCountersFileName)
				AddDataSet(t, CountersDbNum, portRatesThreeFileName)
			},
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersRatesAll),
			valTest:     true,
		},
		{
			desc:       "SHOW interfaces counters rates all ports with period",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" >
				elem: <name: "rates"  key: <key: "period" value: "5"> >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersRatesAllPeriod),
			valTest:     true,
			mockSleep:   true,
		},
		{
			desc:       "SHOW interfaces counters fec-stats all ports",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" >
				elem: <name: "fec-stats" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersFecStatsAll),
			valTest:     true,
		},
		{
			desc:       "SHOW interfaces counters fec-stats all ports with period",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" >
				elem: <name: "fec-stats"  key: <key: "period" value: "5"> >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersFecStatsAllPeriod),
			valTest:     true,
			mockSleep:   true,
		},
		{
			desc:       "SHOW interfaces counters fec-histogram missing interface -> InvalidArgument",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" >
				elem: <name: "fec-histogram" >
			`,
			wantRetCode: codes.InvalidArgument,
		},
		{
			desc:       "SHOW interfaces counters fec-histogram Ethernet0",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" >
				elem: <name: "fec-histogram" >
				elem: <name: "Ethernet0" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersFecHistogramEth0),
			valTest:     true,
		},
		{
			desc:       "SHOW interfaces counters detailed missing interface -> InvalidArgument",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" >
				elem: <name: "detailed" >
			`,
			wantRetCode: codes.InvalidArgument,
		},
		{
			desc:       "SHOW interfaces counters detailed Ethernet0 (no cache, timestamp None)",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" >
				elem: <name: "detailed" >
				elem: <name: "Ethernet0" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersDetailedEth0NoCache),
			valTest:     true,
		},
		{
			desc:       "SHOW interfaces counters detailed Ethernet0 (with portstat cache diff + timestamp)",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" >
				elem: <name: "detailed" >
				elem: <name: "Ethernet0" >
			`,
			testInit: func() {
				MockReadFile("/tmp/cache/portstat/1000/portstat", portStatCacheJSON, nil)
			},
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersDetailedEth0Cache),
			valTest:     true,
		},
		{
			desc:       "SHOW interfaces counters detailed Ethernet0 (with portstat cache diff) + period",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "interfaces" >
				elem: <name: "counters" >
				elem: <name: "detailed" >
				elem: <name: "Ethernet0"  key: <key: "period" value: "5"> >
			`,
			testInit: func() {
				MockReadFile("/tmp/cache/portstat/1000/portstat", portStatCacheJSON, nil)
			},
			wantRetCode: codes.OK,
			wantRespVal: []byte(interfaceCountersDetailedEth0CachePeriod),
			valTest:     true,
			mockSleep:   true,
		},
	}

	for _, test := range tests {
		if test.testInit != nil {
			test.testInit()
		}
		var patches *gomonkey.Patches
		if test.mockSleep {
			patches = gomonkey.ApplyFunc(time.Sleep, func(d time.Duration) {
				AddDataSet(t, CountersDbNum, portCountersTwoFileName)
				AddDataSet(t, CountersDbNum, portRatesTwoFileName)
			})
		}

		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})
		if patches != nil {
			patches.Reset()
			AddDataSet(t, CountersDbNum, portCountersFileName)
			AddDataSet(t, CountersDbNum, portRatesFileName)
		}
	}
}

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
    "RxOkBits": "N/A",
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

	t.Run("query SHOW interfaces counters rif PortChannel101", func(t *testing.T) {
		textPbPath := `
			elem: <name: "interfaces" >
			elem: <name: "counters" >
			elem: <name: "rif" >
			elem: <name: "PortChannel101" >
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

	t.Run("query SHOW interfaces counters rif PortChannel104 -p 2", func(t *testing.T) {
		textPbPath := `
			elem: <name: "interfaces" >
			elem: <name: "counters" >
			elem: <name: "rif" >
			elem: <name: "PortChannel104" key: {key: "period" value: "1"} >
		`
		wantRespVal := []byte(`{
			"PortChannel104": {
				"RxBps": "20.260496891870496",
				"RxErrBits": "0",
				"RxErrPackets": "0",
				"RxOkPackets": "0",
				"RxOkBits": "0",
				"RxPps": "0.24715843207997978",
				"TxBps": "0",
				"TxErrBits": "N/A",
				"TxErrPackets": "0",
				"TxOkBits": "0",
				"TxOkPackets": "0",
				"TxPps": "0"
			}
	  }`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	t.Run("query SHOW interfaces counters rif PortChannel101 -p 1", func(t *testing.T) {
		textPbPath := `
			elem: <name: "interfaces" >
			elem: <name: "counters" >
			elem: <name: "rif" >
			elem: <name: "PortChannel101"  key: {key: "period" value: "1"} >
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

	t.Run("query SHOW interfaces counters rif PortChannel102 -p 1", func(t *testing.T) {
		textPbPath := `
			elem: <name: "interfaces" >
			elem: <name: "counters" >
			elem: <name: "rif" >
			elem: <name: "PortChannel102"  key: {key: "period" value: "1"} >
		`
		wantRespVal := []byte(`{
			"PortChannel102": {
				"RxBps": "1.2202977000824049",
				"RxErrBits": "0",
				"RxErrPackets": "0",
				"RxOkPackets": "0",
				"RxOkBits": "N/A",
				"RxPps": "0.013699805079217392",
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

	// invalid interface name
	t.Run("query SHOW interfaces counters rif PortChannel11 -p 1", func(t *testing.T) {
		textPbPath := `
			elem: <name: "interfaces" >
			elem: <name: "counters" >
			elem: <name: "rif" >
			elem: <name: "PortChannel11"  key: {key: "period" value: "1"} >
		`
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.InvalidArgument, nil, false)
	})
}
