package gnmi

import (
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	sccommon "github.com/sonic-net/sonic-gnmi/show_client/common"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestShowMmu_HappyPath(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}
	conn, err := grpc.Dial(TargetAddr, opts...)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	gClient := pb.NewGNMIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()

	// Load CONFIG_DB datasets
	AddDataSet(t, ConfigDbNum, "../testdata/CONFIG_DB_DEFAULT_LOSSLESS_BUFFER_PARAMETER.txt")
	AddDataSet(t, ConfigDbNum, "../testdata/CONFIG_DB_BUFFER_POOL.txt")
	AddDataSet(t, ConfigDbNum, "../testdata/CONFIG_DB_BUFFER_PROFILE.txt")

	textPbPath := `elem: <name: "mmu" >`

	// Expected JSON (order-insensitive map compare enabled by valTest=true)
	expected := []byte(`{
        "losslessTrafficPatterns": {
            "pattern": { "field1": "value1" }
        },
        "pools": {
            "egress_lossless_pool": { "mode":"static","size":"164075364","type":"egress" },
            "ingress_lossless_pool": { "mode":"dynamic","size":"164075364","type":"ingress","xoff":"20181824" }
        },
        "profiles": {
            "egress_lossless_profile": { "pool":"egress_lossless_pool","size":"0","static_th":"165364160" },
            "egress_lossy_profile": { "pool":"egress_lossless_pool","size":"1778","dynamic_th":"0" },
            "ingress_lossy_profile": { "pool":"ingress_lossless_pool","size":"0","static_th":"165364160" }
        }
    }`)
	runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, expected, true)
}

func TestShowMmu_EmptyConfigDb(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}
	conn, err := grpc.Dial(TargetAddr, opts...)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	gClient := pb.NewGNMIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()

	// No datasets loaded in CONFIG_DB -> response is {}
	textPbPath := `elem: <name: "mmu" >`
	expected := []byte(`{}`)
	runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, expected, true)
}

func TestShowMmu_PartialOnlyPools(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}
	conn, err := grpc.Dial(TargetAddr, opts...)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	gClient := pb.NewGNMIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()

	// Load only BUFFER_POOL table
	AddDataSet(t, ConfigDbNum, "../testdata/CONFIG_DB_BUFFER_POOL.txt")

	textPbPath := `elem: <name: "mmu" >`
	expected := []byte(`{
        "pools": {
            "egress_lossless_pool": { "mode":"static","size":"164075364","type":"egress" },
            "ingress_lossless_pool": { "mode":"dynamic","size":"164075364","type":"ingress","xoff":"20181824" }
        }
    }`)
	runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, expected, true)
}

func TestShowMmu_VerboseTotals(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}
	conn, err := grpc.Dial(TargetAddr, opts...)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	gClient := pb.NewGNMIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()

	// Load CONFIG_DB datasets for pools + profiles (+ optional default lossless params)
	AddDataSet(t, ConfigDbNum, "../testdata/CONFIG_DB_DEFAULT_LOSSLESS_BUFFER_PARAMETER.txt")
	AddDataSet(t, ConfigDbNum, "../testdata/CONFIG_DB_BUFFER_POOL.txt")
	AddDataSet(t, ConfigDbNum, "../testdata/CONFIG_DB_BUFFER_PROFILE.txt")

	// Pass verbose=true as an option on the "mmu" elem
	textPbPath := `
        elem: <name: "mmu" key: { key: "verbose" value: "true" } >
    `

	expected := []byte(`{
        "losslessTrafficPatterns": {
            "pattern": { "field1": "value1" }
        },
        "pools": {
            "egress_lossless_pool": { "mode":"static","size":"164075364","type":"egress" },
            "ingress_lossless_pool": { "mode":"dynamic","size":"164075364","type":"ingress","xoff":"20181824" }
        },
        "profiles": {
            "egress_lossless_profile": { "pool":"egress_lossless_pool","size":"0","static_th":"165364160" },
            "egress_lossy_profile": { "pool":"egress_lossless_pool","size":"1778","dynamic_th":"0" },
            "ingress_lossy_profile": { "pool":"ingress_lossless_pool","size":"0","static_th":"165364160" }
        },
        "totals": { "pools": 2, "profiles": 3 }
    }`)
	runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, expected, true)
}

func TestShowMmu_ErrorOnLossless(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	var calls int
	patches := gomonkey.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
		calls++
		if calls == 1 {
			return nil, fmt.Errorf("error when read table DEFAULT_LOSSLESS_BUFFER_PARAMETER")
		}
		return map[string]interface{}{}, nil
	})
	defer patches.Reset()

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}
	conn, err := grpc.Dial(TargetAddr, opts...)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	gClient := pb.NewGNMIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()

	textPbPath := `elem: <name: "mmu" >`
	runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
}

func TestShowMmu_ErrorOnPools(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	var calls int
	patches := gomonkey.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
		calls++
		switch calls {
		case 1:
			// first call: lossless table succeeds (empty map fine)
			return map[string]interface{}{}, nil
		case 2:
			// second call: pools table error
			return nil, fmt.Errorf("error when read table BUFFER_POOL")
		default:
			return map[string]interface{}{}, nil
		}
	})
	defer patches.Reset()

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}
	conn, err := grpc.Dial(TargetAddr, opts...)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	gClient := pb.NewGNMIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()

	textPbPath := `elem: <name: "mmu" >`
	runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
}

func TestShowMmu_ErrorOnProfiles(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	var calls int
	patches := gomonkey.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
		calls++
		switch calls {
		case 1: // lossless ok
			return map[string]interface{}{}, nil
		case 2: // pools ok
			return map[string]interface{}{}, nil
		case 3: // profiles error
			return nil, fmt.Errorf("error when read table BUFFER_PROFILE")
		default:
			return map[string]interface{}{}, nil
		}
	})
	defer patches.Reset()

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}
	conn, err := grpc.Dial(TargetAddr, opts...)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	gClient := pb.NewGNMIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()

	textPbPath := `elem: <name: "mmu" >`
	runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
}
