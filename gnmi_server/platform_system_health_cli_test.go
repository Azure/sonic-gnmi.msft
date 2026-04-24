package gnmi

// system_health_cli_test.go

// Tests SHOW system-health summary, detail, and monitor-list

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"strings"
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

// Testdata file paths — config JSON
const (
	systemHealthConfigFile            = "../testdata/SYSTEM_HEALTH_CONFIG.json"
	systemHealthConfigIgnoreFile      = "../testdata/SYSTEM_HEALTH_CONFIG_IGNORE.json"
	systemHealthConfigUserDefinedFile = "../testdata/SYSTEM_HEALTH_CONFIG_USER_DEFINED.json"
)

// Inline mock data — monit status and summary outputs.
const (
	mockMonitStatusActive   = "active\n"
	mockMonitStatusInactive = "inactive\n"

	mockMonitSummaryOK = "Monit 5.34.3 uptime: 10d 5h 20m\n" +
		" Name                            Status                   Type\n" +
		" container_checker               OK                       Process\n" +
		" sonic                           OK                       System\n" +
		" root-overlay                    OK                       Filesystem\n" +
		" var-log                         OK                       Filesystem\n"

	mockMonitSummarySvcFail = "Monit 5.34.3 uptime: 10d 5h 20m\n" +
		" Name                            Status                   Type\n" +
		" container_checker               Does not exist           Process\n" +
		" sonic                           OK                       System\n" +
		" root-overlay                    OK                       Filesystem\n" +
		" var-log                         OK                       Filesystem\n"

	mockMonitSummaryFsFail = "Monit 5.34.3 uptime: 10d 5h 20m\n" +
		" Name                            Status                   Type\n" +
		" container_checker               OK                       Process\n" +
		" sonic                           OK                       System\n" +
		" root-overlay                    Does not exist           Filesystem\n" +
		" var-log                         OK                       Filesystem\n"

	mockMonitSummaryNotReady = "Monit 5.34.3 uptime: 10d 5h 20m\n" +
		" Name                            Status                   Type\n"
)

// Testdata file paths — DB query responses
const (
	systemHealthFeatureDBFile       = "../testdata/SYSTEM_HEALTH_FEATURE_DB.json"
	systemHealthTemperatureOKFile   = "../testdata/SYSTEM_HEALTH_TEMPERATURE_OK.json"
	systemHealthTemperatureFailFile = "../testdata/SYSTEM_HEALTH_TEMPERATURE_FAIL.json"
	systemHealthFanOKFile           = "../testdata/SYSTEM_HEALTH_FAN_OK.json"
	systemHealthPsuOKFile           = "../testdata/SYSTEM_HEALTH_PSU_OK.json"
)

// Inline mock data — short, fixed host command outputs that never vary across scenarios.
const (
	mockDockerPS                 = "swss\t\nbgp\t\nteamd\t\n"
	mockDockerImages             = "docker-sonic-telemetry\n"
	mockSupervisorctlStatus      = "orchagent RUNNING pid 100, uptime 1:00:00\n"
	mockCriticalProcesses        = "program:orchagent\n"
	mockUserDefinedCheckerOutput = "CustomHardware\nSensor1:Temperature too high\n"
)

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

	// Test 1: Config file not found → error
	t.Run("query SHOW system-health summary config not found", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(sccommon.GetPlatform, func() string {
			return "x86_64-test_platform"
		})
		defer patches.Reset()

		patches.ApplyFunc(sccommon.FileExists, func(path string) bool {
			return false
		})

		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
	})

	// Test 2: All OK — services and hardware healthy
	t.Run("query SHOW system-health summary all OK", func(t *testing.T) {
		patches := mockHealthySystem(t)
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "",
			"services": {"status": "OK"},
			"hardware": {"status": "OK"}
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 3: Service not running → services Not OK
	t.Run("query SHOW system-health summary service not running", func(t *testing.T) {
		patches := mockSystemWithFailedService(t)
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "",
			"services": {
				"status": "Not OK",
				"not_running": ["container_checker"]
			},
			"hardware": {"status": "OK"}
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 4: Filesystem not accessible → with Monit 5.34+ the message format is
	// "root-overlay status is Does not exist, expected OK" which does not contain
	// "Accessible", so it falls into not_running (matching Python master behavior).
	t.Run("query SHOW system-health summary filesystem not accessible", func(t *testing.T) {
		patches := mockSystemWithFailedFilesystem(t)
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "",
			"services": {
				"status": "Not OK",
				"not_running": ["root-overlay"]
			},
			"hardware": {"status": "OK"}
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 5: Hardware failure (ASIC temperature too hot)
	t.Run("query SHOW system-health summary hardware failure", func(t *testing.T) {
		patches := mockSystemWithHardwareFailure(t)
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "",
			"services": {"status": "OK"},
			"hardware": {
				"status": "Not OK",
				"reasons": ["ASIC0 temperature is too hot, temperature=120, threshold=100"]
			}
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 6: Monit inactive → monit reported as not running
	t.Run("query SHOW system-health summary monit inactive", func(t *testing.T) {
		patches := mockMonitInactive(t)
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "",
			"services": {
				"status": "Not OK",
				"not_running": ["monit"]
			},
			"hardware": {"status": "OK"}
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 7: Multiple simultaneous failures — service not running + ASIC too hot
	t.Run("query SHOW system-health summary multiple failures", func(t *testing.T) {
		patches := mockMultipleFailures(t)
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "",
			"services": {
				"status": "Not OK",
				"not_running": ["container_checker"]
			},
			"hardware": {
				"status": "Not OK",
				"reasons": ["ASIC0 temperature is too hot, temperature=120, threshold=100"]
			}
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 8: Monit active but output too short (not ready)
	t.Run("query SHOW system-health summary monit not ready", func(t *testing.T) {
		patches := mockMonitNotReady(t)
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "",
			"services": {
				"status": "Not OK",
				"not_running": ["monit"]
			},
			"hardware": {"status": "OK"}
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 9: User-defined checker reports failure → appears in hardware reasons
	t.Run("query SHOW system-health summary user defined checker failure", func(t *testing.T) {
		patches := mockUserDefinedCheckerFailure(t)
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "",
			"services": {"status": "OK"},
			"hardware": {
				"status": "Not OK",
				"reasons": ["Temperature too high"]
			}
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 10: Config with ignore lists — ignored service/device not reported
	t.Run("query SHOW system-health summary with ignore config", func(t *testing.T) {
		patches := mockSystemWithIgnoreConfig(t)
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

	// Test 1: Config file not found → error
	t.Run("query SHOW system-health detail config not found", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(sccommon.GetPlatform, func() string {
			return "x86_64-test_platform"
		})
		defer patches.Reset()

		patches.ApplyFunc(sccommon.FileExists, func(path string) bool {
			return false
		})

		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
	})

	// Test 2: All OK — services and hardware healthy, all entries in monitor list, empty ignore list
	t.Run("query SHOW system-health detail all OK", func(t *testing.T) {
		patches := mockHealthySystem(t)
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "",
			"services": {"status": "OK"},
			"hardware": {"status": "OK"},
			"monitor_list": [
				{"name": "sonic", "status": "OK", "type": "System"},
				{"name": "container_checker", "status": "OK", "type": "Process"},
				{"name": "root-overlay", "status": "OK", "type": "Filesystem"},
				{"name": "var-log", "status": "OK", "type": "Filesystem"},
				{"name": "bgp:orchagent", "status": "OK", "type": "Process"},
				{"name": "swss:orchagent", "status": "OK", "type": "Process"},
				{"name": "teamd:orchagent", "status": "OK", "type": "Process"},
				{"name": "ASIC0", "status": "OK", "type": "ASIC"},
				{"name": "Fan1", "status": "OK", "type": "Fan"},
				{"name": "PSU 1", "status": "OK", "type": "PSU"}
			],
			"ignore_list": []
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 3: Service not running — appears in both services.not_running and monitor_list as Not OK
	t.Run("query SHOW system-health detail service not running", func(t *testing.T) {
		patches := mockSystemWithFailedService(t)
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "",
			"services": {
				"status": "Not OK",
				"not_running": ["container_checker"]
			},
			"hardware": {"status": "OK"},
			"monitor_list": [
				{"name": "container_checker", "status": "Not OK", "type": "Process"},
				{"name": "sonic", "status": "OK", "type": "System"},
				{"name": "root-overlay", "status": "OK", "type": "Filesystem"},
				{"name": "var-log", "status": "OK", "type": "Filesystem"},
				{"name": "bgp:orchagent", "status": "OK", "type": "Process"},
				{"name": "swss:orchagent", "status": "OK", "type": "Process"},
				{"name": "teamd:orchagent", "status": "OK", "type": "Process"},
				{"name": "ASIC0", "status": "OK", "type": "ASIC"},
				{"name": "Fan1", "status": "OK", "type": "Fan"},
				{"name": "PSU 1", "status": "OK", "type": "PSU"}
			],
			"ignore_list": []
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 4: Hardware failure — ASIC0 appears in both hardware.reasons and monitor_list as Not OK
	t.Run("query SHOW system-health detail hardware failure", func(t *testing.T) {
		patches := mockSystemWithHardwareFailure(t)
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "",
			"services": {"status": "OK"},
			"hardware": {
				"status": "Not OK",
				"reasons": ["ASIC0 temperature is too hot, temperature=120, threshold=100"]
			},
			"monitor_list": [
				{"name": "ASIC0", "status": "Not OK", "type": "ASIC"},
				{"name": "sonic", "status": "OK", "type": "System"},
				{"name": "container_checker", "status": "OK", "type": "Process"},
				{"name": "root-overlay", "status": "OK", "type": "Filesystem"},
				{"name": "var-log", "status": "OK", "type": "Filesystem"},
				{"name": "bgp:orchagent", "status": "OK", "type": "Process"},
				{"name": "swss:orchagent", "status": "OK", "type": "Process"},
				{"name": "teamd:orchagent", "status": "OK", "type": "Process"},
				{"name": "Fan1", "status": "OK", "type": "Fan"},
				{"name": "PSU 1", "status": "OK", "type": "PSU"}
			],
			"ignore_list": []
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 5: Ignore config — ignored items absent from monitor_list, present in ignore_list
	t.Run("query SHOW system-health detail with ignore config", func(t *testing.T) {
		patches := mockSystemWithIgnoreConfig(t)
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "",
			"services": {"status": "OK"},
			"hardware": {"status": "OK"},
			"monitor_list": [
				{"name": "sonic", "status": "OK", "type": "System"},
				{"name": "root-overlay", "status": "OK", "type": "Filesystem"},
				{"name": "var-log", "status": "OK", "type": "Filesystem"},
				{"name": "bgp:orchagent", "status": "OK", "type": "Process"},
				{"name": "swss:orchagent", "status": "OK", "type": "Process"},
				{"name": "teamd:orchagent", "status": "OK", "type": "Process"},
				{"name": "Fan1", "status": "OK", "type": "Fan"},
				{"name": "PSU 1", "status": "OK", "type": "PSU"}
			],
			"ignore_list": [
				{"name": "asic", "status": "Ignored", "type": "Device"},
				{"name": "container_checker", "status": "Ignored", "type": "Service"}
			]
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 6: User-defined checker failure — extra category appears in monitor_list
	t.Run("query SHOW system-health detail user defined checker failure", func(t *testing.T) {
		patches := mockUserDefinedCheckerFailure(t)
		defer patches.Reset()

		wantRespVal := []byte(`{
			"system_status_led": "",
			"services": {"status": "OK"},
			"hardware": {
				"status": "Not OK",
				"reasons": ["Temperature too high"]
			},
			"monitor_list": [
				{"name": "Sensor1", "status": "Not OK", "type": "UserDefine"},
				{"name": "sonic", "status": "OK", "type": "System"},
				{"name": "container_checker", "status": "OK", "type": "Process"},
				{"name": "root-overlay", "status": "OK", "type": "Filesystem"},
				{"name": "var-log", "status": "OK", "type": "Filesystem"},
				{"name": "bgp:orchagent", "status": "OK", "type": "Process"},
				{"name": "swss:orchagent", "status": "OK", "type": "Process"},
				{"name": "teamd:orchagent", "status": "OK", "type": "Process"},
				{"name": "ASIC0", "status": "OK", "type": "ASIC"},
				{"name": "Fan1", "status": "OK", "type": "Fan"},
				{"name": "PSU 1", "status": "OK", "type": "PSU"}
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

	// Test 1: Config file not found → error
	t.Run("query SHOW system-health monitor-list config not found", func(t *testing.T) {
		patches := gomonkey.ApplyFunc(sccommon.GetPlatform, func() string {
			return "x86_64-test_platform"
		})
		defer patches.Reset()

		patches.ApplyFunc(sccommon.FileExists, func(path string) bool {
			return false
		})

		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.NotFound, nil, false)
	})

	// Test 2: All OK — all monitored entries with OK status
	t.Run("query SHOW system-health monitor-list all OK", func(t *testing.T) {
		patches := mockHealthySystem(t)
		defer patches.Reset()

		wantRespVal := []byte(`{
			"monitor_list": [
				{"name": "sonic", "status": "OK", "type": "System"},
				{"name": "container_checker", "status": "OK", "type": "Process"},
				{"name": "root-overlay", "status": "OK", "type": "Filesystem"},
				{"name": "var-log", "status": "OK", "type": "Filesystem"},
				{"name": "bgp:orchagent", "status": "OK", "type": "Process"},
				{"name": "swss:orchagent", "status": "OK", "type": "Process"},
				{"name": "teamd:orchagent", "status": "OK", "type": "Process"},
				{"name": "ASIC0", "status": "OK", "type": "ASIC"},
				{"name": "Fan1", "status": "OK", "type": "Fan"},
				{"name": "PSU 1", "status": "OK", "type": "PSU"}
			]
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 3: Service not running — container_checker shows Not OK in monitor list
	t.Run("query SHOW system-health monitor-list service not running", func(t *testing.T) {
		patches := mockSystemWithFailedService(t)
		defer patches.Reset()

		wantRespVal := []byte(`{
			"monitor_list": [
				{"name": "container_checker", "status": "Not OK", "type": "Process"},
				{"name": "sonic", "status": "OK", "type": "System"},
				{"name": "root-overlay", "status": "OK", "type": "Filesystem"},
				{"name": "var-log", "status": "OK", "type": "Filesystem"},
				{"name": "bgp:orchagent", "status": "OK", "type": "Process"},
				{"name": "swss:orchagent", "status": "OK", "type": "Process"},
				{"name": "teamd:orchagent", "status": "OK", "type": "Process"},
				{"name": "ASIC0", "status": "OK", "type": "ASIC"},
				{"name": "Fan1", "status": "OK", "type": "Fan"},
				{"name": "PSU 1", "status": "OK", "type": "PSU"}
			]
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 4: Multiple failures — both service and hardware Not OK in monitor list
	t.Run("query SHOW system-health monitor-list multiple failures", func(t *testing.T) {
		patches := mockMultipleFailures(t)
		defer patches.Reset()

		wantRespVal := []byte(`{
			"monitor_list": [
				{"name": "container_checker", "status": "Not OK", "type": "Process"},
				{"name": "ASIC0", "status": "Not OK", "type": "ASIC"},
				{"name": "sonic", "status": "OK", "type": "System"},
				{"name": "root-overlay", "status": "OK", "type": "Filesystem"},
				{"name": "var-log", "status": "OK", "type": "Filesystem"},
				{"name": "bgp:orchagent", "status": "OK", "type": "Process"},
				{"name": "swss:orchagent", "status": "OK", "type": "Process"},
				{"name": "teamd:orchagent", "status": "OK", "type": "Process"},
				{"name": "Fan1", "status": "OK", "type": "Fan"},
				{"name": "PSU 1", "status": "OK", "type": "PSU"}
			]
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 5: Monit inactive — fewer entries since monit never enumerated services, monit itself Not OK
	t.Run("query SHOW system-health monitor-list monit inactive", func(t *testing.T) {
		patches := mockMonitInactive(t)
		defer patches.Reset()

		wantRespVal := []byte(`{
			"monitor_list": [
				{"name": "monit", "status": "Not OK", "type": "Service"},
				{"name": "bgp:orchagent", "status": "OK", "type": "Process"},
				{"name": "swss:orchagent", "status": "OK", "type": "Process"},
				{"name": "teamd:orchagent", "status": "OK", "type": "Process"},
				{"name": "ASIC0", "status": "OK", "type": "ASIC"},
				{"name": "Fan1", "status": "OK", "type": "Fan"},
				{"name": "PSU 1", "status": "OK", "type": "PSU"}
			]
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})
}

// readTestFile reads a testdata file and returns its content as a string, failing the test on error.
// Follows the same pattern as AddDataSet in cli_helpers_test.go.
func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read test file %s: %v", path, err)
	}
	return string(data)
}

// readTestFileJSON reads a testdata JSON file and unmarshals it into a map.
func readTestFileJSON(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read test file %s: %v", path, err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse JSON from %s: %v", path, err)
	}
	return result
}

// mockHostCommands patches sccommon.GetDataFromHostCommand.
// monitStatus and monitSummary are inline strings that vary per scenario.
// Docker, supervisorctl, and critical-process outputs use inline consts (fixed across all scenarios).
// extraCmds maps command substrings to inline response strings for one-off commands (e.g. user-defined checkers).
func mockHostCommands(t *testing.T, patches *gomonkey.Patches, monitStatus, monitSummary string, extraCmds map[string]string) {
	t.Helper()

	patches.ApplyFunc(sccommon.GetDataFromHostCommand, func(command string) (string, error) {
		for cmdSubstr, data := range extraCmds {
			if strings.Contains(command, cmdSubstr) {
				return data, nil
			}
		}
		switch {
		case strings.Contains(command, "systemctl is-active monit"):
			return monitStatus, nil
		case strings.Contains(command, "monit summary"):
			return monitSummary, nil
		case strings.Contains(command, "docker ps"):
			return mockDockerPS, nil
		case strings.Contains(command, "cat /etc/supervisor/critical_processes"):
			return mockCriticalProcesses, nil
		case strings.Contains(command, "docker images"):
			return mockDockerImages, nil
		case strings.Contains(command, "supervisorctl status"):
			return mockSupervisorctlStatus, nil
		}
		return "", nil
	})
}

// mockDBQueries patches sccommon.GetMapFromQueries using testdata JSON files.
// temperatureFile is the varying part per scenario (OK vs failure).
func mockDBQueries(t *testing.T, patches *gomonkey.Patches, temperatureFile string) {
	t.Helper()
	featureData := readTestFileJSON(t, systemHealthFeatureDBFile)
	tempData := readTestFileJSON(t, temperatureFile)
	fanData := readTestFileJSON(t, systemHealthFanOKFile)
	psuData := readTestFileJSON(t, systemHealthPsuOKFile)

	patches.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
		if len(queries) > 0 && len(queries[0]) > 1 {
			switch queries[0][1] {
			case "FEATURE":
				return featureData, nil
			case "TEMPERATURE_INFO":
				return tempData, nil
			case "FAN_INFO":
				return fanData, nil
			case "PSU_INFO":
				return psuData, nil
			case "LIQUID_COOLING_INFO":
				return map[string]interface{}{}, nil
			}
		}
		return map[string]interface{}{}, nil
	})
}

// mockBaseSystem sets up gomonkey patches for common system mocks
// (platform, config, multi-asic, docker, etc.) WITHOUT patching host commands
// or DB queries. Callers add those separately to avoid double-patching.
// configFile specifies which health config JSON to use for GetMapFromFile.
func mockBaseSystem(t *testing.T, configFile string) *gomonkey.Patches {
	t.Helper()
	patches := gomonkey.ApplyFunc(sccommon.GetPlatform, func() string {
		return "x86_64-test_platform"
	})

	patches.ApplyFunc(sccommon.FileExists, func(path string) bool {
		return true
	})

	configData := readTestFileJSON(t, configFile)
	patches.ApplyFunc(sccommon.GetMapFromFile, func(filePath string) (map[string]interface{}, error) {
		return configData, nil
	})

	patches.ApplyFunc(sccommon.IsMultiAsic, func() bool { return false })
	patches.ApplyFunc(sccommon.IsSupervisor, func() bool { return false })

	patches.ApplyFunc(sccommon.GetDockerInfo, func() string {
		return "docker-sonic-telemetry"
	})

	return patches
}

// mockHealthySystem sets up gomonkey patches for a fully healthy system.
// All data is loaded from testdata files. Monit reports all services running,
// no hardware issues, config file exists with empty ignore lists.
func mockHealthySystem(t *testing.T) *gomonkey.Patches {
	t.Helper()
	patches := mockBaseSystem(t, systemHealthConfigFile)
	mockHostCommands(t, patches, mockMonitStatusActive, mockMonitSummaryOK, nil)
	mockDBQueries(t, patches, systemHealthTemperatureOKFile)
	return patches
}

// mockSystemWithFailedService patches for a system where monit reports a failed process.
func mockSystemWithFailedService(t *testing.T) *gomonkey.Patches {
	t.Helper()
	patches := mockBaseSystem(t, systemHealthConfigFile)
	mockHostCommands(t, patches, mockMonitStatusActive, mockMonitSummarySvcFail, nil)
	mockDBQueries(t, patches, systemHealthTemperatureOKFile)
	return patches
}

// mockSystemWithFailedFilesystem patches for a system where monit reports an inaccessible filesystem.
func mockSystemWithFailedFilesystem(t *testing.T) *gomonkey.Patches {
	t.Helper()
	patches := mockBaseSystem(t, systemHealthConfigFile)
	mockHostCommands(t, patches, mockMonitStatusActive, mockMonitSummaryFsFail, nil)
	mockDBQueries(t, patches, systemHealthTemperatureOKFile)
	return patches
}

// mockSystemWithHardwareFailure patches for a system where ASIC temperature exceeds threshold.
func mockSystemWithHardwareFailure(t *testing.T) *gomonkey.Patches {
	t.Helper()
	patches := mockBaseSystem(t, systemHealthConfigFile)
	mockHostCommands(t, patches, mockMonitStatusActive, mockMonitSummaryOK, nil)
	mockDBQueries(t, patches, systemHealthTemperatureFailFile)
	return patches
}

// mockMonitInactive patches for a system where the monit service itself is inactive.
// checkByMonit sets monit as Not OK and returns early; checkServices still runs.
func mockMonitInactive(t *testing.T) *gomonkey.Patches {
	t.Helper()
	patches := mockBaseSystem(t, systemHealthConfigFile)
	mockHostCommands(t, patches, mockMonitStatusInactive, "", nil)
	mockDBQueries(t, patches, systemHealthTemperatureOKFile)
	return patches
}

// mockMultipleFailures patches for a system with both a failed service and failed hardware.
func mockMultipleFailures(t *testing.T) *gomonkey.Patches {
	t.Helper()
	patches := mockBaseSystem(t, systemHealthConfigFile)
	mockHostCommands(t, patches, mockMonitStatusActive, mockMonitSummarySvcFail, nil)
	mockDBQueries(t, patches, systemHealthTemperatureFailFile)
	return patches
}

// mockMonitNotReady patches for a system where monit is active but its summary
// output is too short (fewer than 3 lines), indicating monit is not ready.
func mockMonitNotReady(t *testing.T) *gomonkey.Patches {
	t.Helper()
	patches := mockBaseSystem(t, systemHealthConfigFile)
	mockHostCommands(t, patches, mockMonitStatusActive, mockMonitSummaryNotReady, nil)
	mockDBQueries(t, patches, systemHealthTemperatureOKFile)
	return patches
}

// mockUserDefinedCheckerFailure patches for a system where a user-defined checker
// command reports a hardware failure. The failure appears under the hardware section
// because its category ("CustomHardware") is not "Services".
func mockUserDefinedCheckerFailure(t *testing.T) *gomonkey.Patches {
	t.Helper()
	patches := mockBaseSystem(t, systemHealthConfigUserDefinedFile)
	mockHostCommands(t, patches, mockMonitStatusActive, mockMonitSummaryOK, map[string]string{
		"check_my_hardware.py": mockUserDefinedCheckerOutput,
	})
	mockDBQueries(t, patches, systemHealthTemperatureOKFile)

	return patches
}

// mockSystemWithIgnoreConfig patches for a system that has failures but they are
// suppressed by services_to_ignore and devices_to_ignore in the config.
// container_checker is Not Running but ignored; ASIC temperature is too hot but "asic" is ignored.
func mockSystemWithIgnoreConfig(t *testing.T) *gomonkey.Patches {
	t.Helper()
	patches := mockBaseSystem(t, systemHealthConfigIgnoreFile)
	mockHostCommands(t, patches, mockMonitStatusActive, mockMonitSummarySvcFail, nil)
	mockDBQueries(t, patches, systemHealthTemperatureFailFile)

	return patches
}
