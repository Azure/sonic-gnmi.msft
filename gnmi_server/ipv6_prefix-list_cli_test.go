package gnmi

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

// Tests SHOW ipv6 prefix-list
func TestGetIPv6PrefixList(t *testing.T) {
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

	// Expected results
	var expectedFileName = "../testdata/Expected_show_ipv6_prefix-list_single_response.txt"
	singleExpectedResponse, err := ioutil.ReadFile(expectedFileName)
	if err != nil {
		t.Fatalf("Failed to read expected result file %q: %v", expectedFileName, err)
	}

	var expectedFileNameMulti = "../testdata/Expected_show_ipv6_prefix-list_multi_response.txt"
	multiExpectedResponse, err := ioutil.ReadFile(expectedFileNameMulti)
	if err != nil {
		t.Fatalf("Failed to read expected result file %q: %v", expectedFileNameMulti, err)
	}

	var expectedFileNameFilteredMulti = "../testdata/Expected_show_ipv6_prefix-list_filtered_multi_response.txt"
	filteredMultiExpectedResponse, err := ioutil.ReadFile(expectedFileNameFilteredMulti)
	if err != nil {
		t.Fatalf("Failed to read expected result file %q: %v", expectedFileNameFilteredMulti, err)
	}

	// expected empty response
	emptyExpected, err := json.Marshal(map[string]interface {}{})
	if err != nil {
		t.Fatalf("Failed to marshal expected empty result, error: %v", err)
	}

	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal interface{}
		valTest     bool
		mockFile    string
	}{
		{
			desc:       "single prefix list output",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "prefix-list" >
			`,
			wantRetCode: codes.OK,
			wantRespVal:    []byte(singleExpectedResponse),
			valTest:     true,
			mockFile:    "../testdata/VTYSH_SHOW_IPV6_PREFIX-LIST_SINGLE.txt",
		},
		{
			desc:       "multi Prefix list output in one valid JSON block",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "prefix-list" >
			`,
			wantRetCode: codes.OK,
			wantRespVal:    []byte(multiExpectedResponse),
			valTest:     true,
			mockFile:    "../testdata/VTYSH_SHOW_IPV6_PREFIX-LIST_MULTI_IN_ONE.txt",
		},
		// There is a bug in vtysh that multiple JSON blocks are printed before FRR version 10.1
		{
			desc:       "multi Prefix list output in multiple valid JSON blocks",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "prefix-list" >
			`,
			wantRetCode: codes.OK,
			wantRespVal:    []byte(multiExpectedResponse),
			valTest:     true,
			mockFile:    "../testdata/VTYSH_SHOW_IPV6_PREFIX-LIST_MULTI.txt",
		},
		{
			desc:       "multi Prefix list output-filter by prefix_list_name",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "prefix-list" >
				elem: <name: "DEFAULT_IPV6" >
			`,
			wantRetCode: codes.OK,
			wantRespVal:    []byte(filteredMultiExpectedResponse),
			valTest:     true,
			mockFile:    "../testdata/VTYSH_SHOW_IPV6_PREFIX-LIST_MULTI.txt",
		},
		{
			desc:       "invalid json output from vtysh command",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "prefix-list">
			`,
			wantRetCode: codes.NotFound,
			valTest:     false,
			mockFile:    "../testdata/VTYSH_SHOW_IPV6_PREFIX-LIST_INVALID_JSON.txt",
		},
		{
			desc:       "multi Prefix list output-filter by not exist prefix_list_name",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "prefix-list">
				elem: <name: "NON_EXISTENT" >
			`,
			wantRetCode: codes.OK,
			wantRespVal:    []byte(emptyExpected),
			valTest:     true,
			mockFile:    "../testdata/VTYSH_SHOW_IPV6_PREFIX-LIST_MULTI.txt",
		},
		{
			desc:       "empty output",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "ipv6" >
				elem: <name: "prefix-list" >
			`,
			wantRetCode: codes.OK,
			wantRespVal:    []byte(emptyExpected),
			valTest:     true,
			mockFile:    "../testdata/VTYSH_SHOW_EMPTY.txt",
		},
	}

	for _, test := range tests {
		var patches interface{ Reset() }
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
