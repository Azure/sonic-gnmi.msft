package gnmi

// system_health_cli_test.go

// Tests for SHOW system-health summary, detail, monitor-list, and sysready-status.
// Summary, detail, and monitor-list use cgo RunPyCode to execute an embedded
// Python script; tests mock RunSystemHealthCheck to return canned JSON output.
// Sysready-status reads from STATE_DB; tests mock GetMapFromQueries.

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	show_client "github.com/sonic-net/sonic-gnmi/show_client"
	sccommon "github.com/sonic-net/sonic-gnmi/show_client/common"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

/* mockHealthCheck patches RunSystemHealthCheck to return the contents of a testdata file. */
func mockHealthCheck(t *testing.T, fileName string) *gomonkey.Patches {
	t.Helper()
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		t.Fatalf("read file %v err: %v", fileName, err)
	}
	return gomonkey.ApplyFunc(show_client.RunSystemHealthCheck, func() (string, error) {
		return string(data), nil
	})
}

func TestGetShowSystemHealthSummary(t *testing.T) {
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

	textPbPath := `
		elem: <name: "system-health" >
		elem: <name: "summary" >
	`

	// Test 1: RunPyCode / Python execution fails → gRPC error
	t.Run("query SHOW system-health summary python error", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(show_client.RunSystemHealthCheck, func() (string, error) {
			return "", fmt.Errorf("Python failure")
		})
		defer patches.Reset()

		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
	})

	// Test 2: Config file not found (Python returns error JSON)
	t.Run("query SHOW system-health summary config not found", func(t *testing.T) {
		patches := mockHealthCheck(t, "../testdata/SYSTEM_HEALTH_SUMMARY_CONFIG_MISSING.txt")
		defer patches.Reset()

		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
	})

	// Test 3: All services and hardware OK
	t.Run("query SHOW system-health summary all OK", func(t *testing.T) {
		patches := mockHealthCheck(t, "../testdata/SYSTEM_HEALTH_SUMMARY_OK.txt")
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "green",
			"services": {"status": "OK"},
			"hardware": {"status": "OK"}
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 4: Service not running → services Not OK with not_running
	t.Run("query SHOW system-health summary service not running", func(t *testing.T) {
		patches := mockHealthCheck(t, "../testdata/SYSTEM_HEALTH_SUMMARY_SVC_FAIL.txt")
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "red",
			"services": {
				"status": "Not OK",
				"not_running": ["container_checker"]
			},
			"hardware": {"status": "OK"}
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 5: Filesystem not accessible → services Not OK with not_accessible
	t.Run("query SHOW system-health summary filesystem not accessible", func(t *testing.T) {
		patches := mockHealthCheck(t, "../testdata/SYSTEM_HEALTH_SUMMARY_FS_FAIL.txt")
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "red",
			"services": {
				"status": "Not OK",
				"not_accessible": ["root-overlay"]
			},
			"hardware": {"status": "OK"}
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 6: Hardware failure (ASIC too hot)
	t.Run("query SHOW system-health summary hardware failure", func(t *testing.T) {
		patches := mockHealthCheck(t, "../testdata/SYSTEM_HEALTH_SUMMARY_HW_FAIL.txt")
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "red",
			"services": {"status": "OK"},
			"hardware": {
				"status": "Not OK",
				"reasons": ["ASIC0 temperature is too hot, temperature=120, threshold=100"]
			}
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 7: Multiple simultaneous failures — service + filesystem + hardware
	t.Run("query SHOW system-health summary multiple failures", func(t *testing.T) {
		patches := mockHealthCheck(t, "../testdata/SYSTEM_HEALTH_SUMMARY_MULTI_FAIL.txt")
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "red",
			"services": {
				"status": "Not OK",
				"not_running": ["container_checker"],
				"not_accessible": ["root-overlay"]
			},
			"hardware": {
				"status": "Not OK",
				"reasons": ["fan1 speed is abnormal", "ASIC0 temperature is too hot, temperature=120, threshold=100"]
			}
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 8: Unparseable output from Python → error
	t.Run("query SHOW system-health summary bad json", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(show_client.RunSystemHealthCheck, func() (string, error) {
			return "not valid json at all", nil
		})
		defer patches.Reset()

		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
	})

	// Test 9: Empty stat → all OK
	t.Run("query SHOW system-health summary empty stat", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(show_client.RunSystemHealthCheck, func() (string, error) {
			return `{"led": "green", "stat": {}}`, nil
		})
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "green",
			"services": {"status": "OK"},
			"hardware": {"status": "OK"}
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 10: LED missing (chassis returns empty string) → still works
	t.Run("query SHOW system-health summary no led", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(show_client.RunSystemHealthCheck, func() (string, error) {
			return `{"led": "", "stat": {"Services": {"svc1": {"status": "OK", "message": "svc1 is running", "type": "Process"}}, "Hardware": {}}}`, nil
		})
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "",
			"services": {"status": "OK"},
			"hardware": {"status": "OK"}
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})
}

func TestGetShowSystemHealthDetail(t *testing.T) {
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

	textPbPath := `
		elem: <name: "system-health" >
		elem: <name: "detail" >
	`

	// Test 1: RunPyCode / Python execution fails → gRPC error
	t.Run("query SHOW system-health detail python error", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(show_client.RunSystemHealthCheck, func() (string, error) {
			return "", fmt.Errorf("Python failure")
		})
		defer patches.Reset()

		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
	})

	// Test 2: Config file not found (Python returns error JSON)
	t.Run("query SHOW system-health detail config not found", func(t *testing.T) {
		patches := mockHealthCheck(t, "../testdata/SYSTEM_HEALTH_SUMMARY_CONFIG_MISSING.txt")
		defer patches.Reset()

		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
	})

	// Test 3: All services and hardware OK, no ignore lists
	t.Run("query SHOW system-health detail all OK", func(t *testing.T) {
		patches := mockHealthCheck(t, "../testdata/SYSTEM_HEALTH_DETAIL_OK.txt")
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "green",
			"services": {"status": "OK"},
			"hardware": {"status": "OK"},
			"monitor_list": [
				{"name": "ASIC0", "status": "OK", "type": "ASIC"},
				{"name": "PSU 1", "status": "OK", "type": "PSU"},
				{"name": "container_checker", "status": "OK", "type": "Process"},
				{"name": "fan1", "status": "OK", "type": "Fan"},
				{"name": "root-overlay", "status": "OK", "type": "Filesystem"},
				{"name": "var-log", "status": "OK", "type": "Filesystem"}
			],
			"ignore_list": []
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 4: All OK with ignore lists (services and devices ignored)
	t.Run("query SHOW system-health detail with ignore lists", func(t *testing.T) {
		patches := mockHealthCheck(t, "../testdata/SYSTEM_HEALTH_DETAIL_WITH_IGNORE.txt")
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "green",
			"services": {"status": "OK"},
			"hardware": {"status": "OK"},
			"monitor_list": [
				{"name": "ASIC0", "status": "OK", "type": "ASIC"},
				{"name": "container_checker", "status": "OK", "type": "Process"}
			],
			"ignore_list": [
				{"name": "snmp", "status": "Ignored", "type": "Service"},
				{"name": "telemetry", "status": "Ignored", "type": "Service"},
				{"name": "psu.voltage", "status": "Ignored", "type": "Device"}
			]
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 5: Multiple simultaneous failures with ignore lists
	t.Run("query SHOW system-health detail multiple failures", func(t *testing.T) {
		patches := mockHealthCheck(t, "../testdata/SYSTEM_HEALTH_DETAIL_MULTI_FAIL.txt")
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "red",
			"services": {
				"status": "Not OK",
				"not_running": ["container_checker"],
				"not_accessible": ["root-overlay"]
			},
			"hardware": {
				"status": "Not OK",
				"reasons": ["fan1 speed is abnormal", "ASIC0 temperature is too hot, temperature=120, threshold=100"]
			},
			"monitor_list": [
				{"name": "ASIC0", "status": "Not OK", "type": "ASIC"},
				{"name": "container_checker", "status": "Not OK", "type": "Process"},
				{"name": "fan1", "status": "Not OK", "type": "Fan"},
				{"name": "root-overlay", "status": "Not OK", "type": "Filesystem"}
			],
			"ignore_list": [
				{"name": "snmp", "status": "Ignored", "type": "Service"},
				{"name": "psu.voltage", "status": "Ignored", "type": "Device"}
			]
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 6: Unparseable output from Python → error
	t.Run("query SHOW system-health detail bad json", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(show_client.RunSystemHealthCheck, func() (string, error) {
			return "not valid json at all", nil
		})
		defer patches.Reset()

		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
	})

	// Test 7: Empty stat, empty ignore lists → minimal detail
	t.Run("query SHOW system-health detail empty stat", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(show_client.RunSystemHealthCheck, func() (string, error) {
			return `{"led": "green", "stat": {}, "ignore_services": [], "ignore_devices": []}`, nil
		})
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "green",
			"services": {"status": "OK"},
			"hardware": {"status": "OK"},
			"monitor_list": [],
			"ignore_list": []
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 8: No ignore fields in Python output → empty ignore list
	t.Run("query SHOW system-health detail missing ignore fields", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(show_client.RunSystemHealthCheck, func() (string, error) {
			return `{"led": "green", "stat": {"Services": {"svc1": {"status": "OK", "message": "svc1 is running", "type": "Process"}}, "Hardware": {}}}`, nil
		})
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "green",
			"services": {"status": "OK"},
			"hardware": {"status": "OK"},
			"monitor_list": [
				{"name": "svc1", "status": "OK", "type": "Process"}
			],
			"ignore_list": []
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})
}

func TestGetShowSystemHealthMonitorList(t *testing.T) {
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

	textPbPath := `
		elem: <name: "system-health" >
		elem: <name: "monitor-list" >
	`

	// Test 1: Python execution fails → gRPC error
	t.Run("query SHOW system-health monitor_list python error", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(show_client.RunSystemHealthCheck, func() (string, error) {
			return "", fmt.Errorf("Python failure")
		})
		defer patches.Reset()

		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
	})

	// Test 2: Config file not found
	t.Run("query SHOW system-health monitor_list config not found", func(t *testing.T) {
		patches := mockHealthCheck(t, "../testdata/SYSTEM_HEALTH_SUMMARY_CONFIG_MISSING.txt")
		defer patches.Reset()

		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
	})

	// Test 3: All services and hardware OK — full monitor list
	t.Run("query SHOW system-health monitor_list all OK", func(t *testing.T) {
		patches := mockHealthCheck(t, "../testdata/SYSTEM_HEALTH_DETAIL_OK.txt")
		defer patches.Reset()

		wantRespVal := []byte(`{
			"monitor_list": [
				{"name": "ASIC0", "status": "OK", "type": "ASIC"},
				{"name": "PSU 1", "status": "OK", "type": "PSU"},
				{"name": "container_checker", "status": "OK", "type": "Process"},
				{"name": "fan1", "status": "OK", "type": "Fan"},
				{"name": "root-overlay", "status": "OK", "type": "Filesystem"},
				{"name": "var-log", "status": "OK", "type": "Filesystem"}
			]
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 4: Multiple failures — sorted by status then name
	t.Run("query SHOW system-health monitor_list multiple failures", func(t *testing.T) {
		patches := mockHealthCheck(t, "../testdata/SYSTEM_HEALTH_DETAIL_MULTI_FAIL.txt")
		defer patches.Reset()

		wantRespVal := []byte(`{
			"monitor_list": [
				{"name": "ASIC0", "status": "Not OK", "type": "ASIC"},
				{"name": "container_checker", "status": "Not OK", "type": "Process"},
				{"name": "fan1", "status": "Not OK", "type": "Fan"},
				{"name": "root-overlay", "status": "Not OK", "type": "Filesystem"}
			]
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 5: Bad JSON → error
	t.Run("query SHOW system-health monitor_list bad json", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(show_client.RunSystemHealthCheck, func() (string, error) {
			return "not valid json at all", nil
		})
		defer patches.Reset()

		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
	})

	// Test 6: Empty stat → empty monitor list
	t.Run("query SHOW system-health monitor_list empty stat", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(show_client.RunSystemHealthCheck, func() (string, error) {
			return `{"led": "green", "stat": {}}`, nil
		})
		defer patches.Reset()

		wantRespVal := []byte(`{
			"monitor_list": []
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})
}

// =========================================================================
// Sysready-status tests
// =========================================================================

/*
	mockSysreadyData builds a combined mock return for GetMapFromQueries.
	It dispatches on the table being queried:
	  - SYSTEM_READY → returns sysreadyData
	  - ALL_SERVICE_STATUS → returns serviceData
*/
func mockSysreadyData(sysreadyData, serviceData map[string]interface{}) *gomonkey.Patches {
	return gomonkey.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
		if len(queries) > 0 && len(queries[0]) >= 2 {
			table := queries[0][1]
			if table == "SYSTEM_READY" {
				return sysreadyData, nil
			}
			if table == "ALL_SERVICE_STATUS" {
				return serviceData, nil
			}
		}
		return nil, fmt.Errorf("unexpected query: %v", queries)
	})
}

/* mockSysreadyError patches GetMapFromQueries to always return an error. */
func mockSysreadyError() *gomonkey.Patches {
	return gomonkey.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
		return nil, fmt.Errorf("simulated DB failure")
	})
}

func TestGetShowSysreadyStatus(t *testing.T) {
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

	sysreadyUp := map[string]interface{}{
		"SYSTEM_STATE": map[string]interface{}{
			"Status": "UP",
		},
	}
	sysreadyDown := map[string]interface{}{
		"SYSTEM_STATE": map[string]interface{}{
			"Status": "DOWN",
		},
	}
	allServicesOK := map[string]interface{}{
		"bgp": map[string]interface{}{
			"service_status":   "OK",
			"app_ready_status": "OK",
			"fail_reason":      "-",
		},
		"swss": map[string]interface{}{
			"service_status":   "OK",
			"app_ready_status": "OK",
			"fail_reason":      "-",
		},
	}
	someServicesFailing := map[string]interface{}{
		"bgp": map[string]interface{}{
			"service_status":   "OK",
			"app_ready_status": "OK",
			"fail_reason":      "-",
		},
		"swss": map[string]interface{}{
			"service_status":   "OK",
			"app_ready_status": "Down",
			"fail_reason":      "orchagent is not ready",
		},
		"teamd": map[string]interface{}{
			"service_status":   "Down",
			"app_ready_status": "Down",
			"fail_reason":      "teamd service is not up",
		},
	}

	textPbPath := `
		elem: <name: "system-health" >
		elem: <name: "sysready-status" >
	`

	// Test 1: DB error → gRPC error
	t.Run("query SHOW sysready-status db error", func(t *testing.T) {
		patches := mockSysreadyError()
		defer patches.Reset()

		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
	})

	// Test 2: System ready, all services OK
	t.Run("query SHOW sysready-status system ready", func(t *testing.T) {
		patches := mockSysreadyData(sysreadyUp, allServicesOK)
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status": "System is ready",
			"service_list": [
				{"service_name": "bgp", "service_status": "OK", "app_ready_status": "OK", "fail_reason": "-"},
				{"service_name": "swss", "service_status": "OK", "app_ready_status": "OK", "fail_reason": "-"}
			]
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 3: System not ready, some services failing
	t.Run("query SHOW sysready-status system not ready", func(t *testing.T) {
		patches := mockSysreadyData(sysreadyDown, someServicesFailing)
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status": "System is not ready - one or more services are not up",
			"service_list": [
				{"service_name": "bgp", "service_status": "OK", "app_ready_status": "OK", "fail_reason": "-"},
				{"service_name": "swss", "service_status": "OK", "app_ready_status": "Down", "fail_reason": "orchagent is not ready"},
				{"service_name": "teamd", "service_status": "Down", "app_ready_status": "Down", "fail_reason": "teamd service is not up"}
			]
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 4: Empty SYSTEM_READY table → error
	t.Run("query SHOW sysready-status empty sysready table", func(t *testing.T) {
		patches := mockSysreadyData(map[string]interface{}{}, allServicesOK)
		defer patches.Reset()

		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
	})

	// Test 5: No services → empty list
	t.Run("query SHOW sysready-status no services", func(t *testing.T) {
		patches := mockSysreadyData(sysreadyUp, map[string]interface{}{})
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status": "System is ready",
			"service_list": []
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})
}