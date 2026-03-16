package gnmi

// system_health_cli_test.go

// Tests SHOW system-health summary

import (
	"crypto/tls"
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
			"summary": "OK",
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
			"summary": "Not OK",
			"system_status_led": "",
			"services": {
				"status": "Not OK",
				"not_running": ["container_checker"]
			},
			"hardware": {"status": "OK"}
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})

	// Test 4: Filesystem not accessible → services Not OK with not_accessible
	t.Run("query SHOW system-health summary filesystem not accessible", func(t *testing.T) {
		patches := mockSystemWithFailedFilesystem(t)
		defer patches.Reset()

		wantRespVal := []byte(`{
			"summary": "Not OK",
			"system_status_led": "",
			"services": {
				"status": "Not OK",
				"not_accessible": ["root-overlay"]
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
			"summary": "Not OK",
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
			"summary": "Not OK",
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
			"summary": "Not OK",
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
			"summary": "Not OK",
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
			"summary": "Not OK",
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
			"summary": "OK",
			"system_status_led": "",
			"services": {"status": "OK"},
			"hardware": {"status": "OK"}
		}`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true, true)
	})
}

// mockHealthySystem sets up gomonkey patches for a fully healthy system.
// Monit reports all services running, no hardware issues, config file exists.
func mockHealthySystem(t *testing.T) *gomonkey.Patches {
	t.Helper()
	patches := gomonkey.ApplyFunc(sccommon.GetPlatform, func() string {
		return "x86_64-test_platform"
	})

	// Config file exists and returns valid JSON
	patches.ApplyFunc(sccommon.FileExists, func(path string) bool {
		return true
	})
	patches.ApplyFunc(sccommon.ReadJsonToMap, func(filePath string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"services_to_ignore": []interface{}{},
			"devices_to_ignore":  []interface{}{},
		}, nil
	})

	// Monit active, summary all OK
	patches.ApplyFunc(sccommon.GetDataFromHostCommand, func(command string) (string, error) {
		switch {
		case strings.Contains(command, "systemctl is-active monit"):
			return "active\n", nil
		case strings.Contains(command, "monit summary"):
			return "Monit 5.26.0 uptime: 10d 5h 20m\n" +
				" Name                            Status                   Type\n" +
				" container_checker               Running                  Process\n" +
				" sonic                           Running                  System\n" +
				" root-overlay                    Accessible               Filesystem\n" +
				" var-log                         Accessible               Filesystem\n", nil
		case strings.Contains(command, "docker ps"):
			return "swss\nbgp\nteamd\n", nil
		case strings.Contains(command, "docker inspect"):
			return "/var/lib/docker/overlay2/merged\n", nil
		case strings.Contains(command, "docker images"):
			return "docker-sonic-telemetry\n", nil
		case strings.Contains(command, "supervisorctl status"):
			return "orchagent RUNNING pid 100, uptime 1:00:00\n", nil
		}
		return "", nil
	})

	// DB queries: FEATURE table for service checker, STATE_DB for hardware
	patches.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
		if len(queries) > 0 && len(queries[0]) > 1 {
			switch queries[0][1] {
			case "FEATURE":
				return map[string]interface{}{
					"swss": map[string]interface{}{"state": "enabled", "has_global_scope": "True"},
					"bgp":  map[string]interface{}{"state": "enabled", "has_global_scope": "True"},
				}, nil
			case "TEMPERATURE_INFO":
				return map[string]interface{}{
					"ASIC0": map[string]interface{}{
						"temperature":    "50",
						"high_threshold": "100",
					},
				}, nil
			case "FAN_INFO":
				return map[string]interface{}{
					"Fan1": map[string]interface{}{
						"presence":       "True",
						"status":         "True",
						"speed":          "6000",
						"speed_target":   "6000",
						"is_under_speed": "False",
						"is_over_speed":  "False",
						"direction":      "intake",
					},
				}, nil
			case "PSU_INFO":
				return map[string]interface{}{
					"PSU 1": map[string]interface{}{
						"presence":              "True",
						"status":                "True",
						"temp":                  "25",
						"temp_threshold":         "60",
						"voltage":               "12.0",
						"voltage_min_threshold":  "11.0",
						"voltage_max_threshold":  "13.0",
					},
				}, nil
			case "LIQUID_COOLING_INFO":
				return map[string]interface{}{}, nil
			}
		}
		return map[string]interface{}{}, nil
	})

	// No multi-asic, not supervisor
	patches.ApplyFunc(sccommon.IsMultiAsic, func() bool { return false })
	patches.ApplyFunc(sccommon.IsSupervisor, func() bool { return false })

	// Docker info for image check
	patches.ApplyFunc(sccommon.GetDockerInfo, func() string {
		return "docker-sonic-telemetry"
	})

	// File/dir helpers for critical process checks
	patches.ApplyFunc(sccommon.DirExists, func(path string) bool { return true })
	patches.ApplyFunc(sccommon.GetDataFromFile, func(fileName string) ([]byte, error) {
		return []byte("program:orchagent\n"), nil
	})

	return patches
}

// mockSystemWithFailedService patches for a system where monit reports a failed process.
func mockSystemWithFailedService(t *testing.T) *gomonkey.Patches {
	t.Helper()
	patches := mockHealthySystem(t)

	// Override monit to return a failed service
	patches.ApplyFunc(sccommon.GetDataFromHostCommand, func(command string) (string, error) {
		switch {
		case strings.Contains(command, "systemctl is-active monit"):
			return "active\n", nil
		case strings.Contains(command, "monit summary"):
			return "Monit 5.26.0 uptime: 10d 5h 20m\n" +
				" Name                            Status                   Type\n" +
				" container_checker               Not Running              Process\n" +
				" sonic                           Running                  System\n" +
				" root-overlay                    Accessible               Filesystem\n" +
				" var-log                         Accessible               Filesystem\n", nil
		case strings.Contains(command, "docker ps"):
			return "swss\nbgp\nteamd\n", nil
		case strings.Contains(command, "docker inspect"):
			return "/var/lib/docker/overlay2/merged\n", nil
		case strings.Contains(command, "docker images"):
			return "docker-sonic-telemetry\n", nil
		case strings.Contains(command, "supervisorctl status"):
			return "orchagent RUNNING pid 100, uptime 1:00:00\n", nil
		}
		return "", nil
	})

	return patches
}

// mockSystemWithFailedFilesystem patches for a system where monit reports an inaccessible filesystem.
func mockSystemWithFailedFilesystem(t *testing.T) *gomonkey.Patches {
	t.Helper()
	patches := mockHealthySystem(t)

	// Override monit to return a failed filesystem
	patches.ApplyFunc(sccommon.GetDataFromHostCommand, func(command string) (string, error) {
		switch {
		case strings.Contains(command, "systemctl is-active monit"):
			return "active\n", nil
		case strings.Contains(command, "monit summary"):
			return "Monit 5.26.0 uptime: 10d 5h 20m\n" +
				" Name                            Status                   Type\n" +
				" container_checker               Running                  Process\n" +
				" sonic                           Running                  System\n" +
				" root-overlay                    Not Accessible           Filesystem\n" +
				" var-log                         Accessible               Filesystem\n", nil
		case strings.Contains(command, "docker ps"):
			return "swss\nbgp\nteamd\n", nil
		case strings.Contains(command, "docker inspect"):
			return "/var/lib/docker/overlay2/merged\n", nil
		case strings.Contains(command, "docker images"):
			return "docker-sonic-telemetry\n", nil
		case strings.Contains(command, "supervisorctl status"):
			return "orchagent RUNNING pid 100, uptime 1:00:00\n", nil
		}
		return "", nil
	})

	return patches
}

// mockSystemWithHardwareFailure patches for a system where ASIC temperature exceeds threshold.
func mockSystemWithHardwareFailure(t *testing.T) *gomonkey.Patches {
	t.Helper()
	patches := mockHealthySystem(t)

	// Override DB queries to return high ASIC temperature
	patches.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
		if len(queries) > 0 && len(queries[0]) > 1 {
			switch queries[0][1] {
			case "FEATURE":
				return map[string]interface{}{
					"swss": map[string]interface{}{"state": "enabled", "has_global_scope": "True"},
					"bgp":  map[string]interface{}{"state": "enabled", "has_global_scope": "True"},
				}, nil
			case "TEMPERATURE_INFO":
				return map[string]interface{}{
					"ASIC0": map[string]interface{}{
						"temperature":    "120",
						"high_threshold": "100",
					},
				}, nil
			case "FAN_INFO":
				return map[string]interface{}{
					"Fan1": map[string]interface{}{
						"presence":       "True",
						"status":         "True",
						"speed":          "6000",
						"speed_target":   "6000",
						"is_under_speed": "False",
						"is_over_speed":  "False",
						"direction":      "intake",
					},
				}, nil
			case "PSU_INFO":
				return map[string]interface{}{
					"PSU 1": map[string]interface{}{
						"presence":              "True",
						"status":                "True",
						"temp":                  "25",
						"temp_threshold":         "60",
						"voltage":               "12.0",
						"voltage_min_threshold":  "11.0",
						"voltage_max_threshold":  "13.0",
					},
				}, nil
			case "LIQUID_COOLING_INFO":
				return map[string]interface{}{}, nil
			}
		}
		return map[string]interface{}{}, nil
	})

	return patches
}

// mockMonitInactive patches for a system where the monit service itself is inactive.
// checkByMonit sets monit as Not OK and returns early; checkServices still runs.
func mockMonitInactive(t *testing.T) *gomonkey.Patches {
	t.Helper()
	patches := mockHealthySystem(t)

	// Override: monit service is inactive
	patches.ApplyFunc(sccommon.GetDataFromHostCommand, func(command string) (string, error) {
		switch {
		case strings.Contains(command, "systemctl is-active monit"):
			return "inactive\n", nil
		case strings.Contains(command, "docker ps"):
			return "swss\nbgp\nteamd\n", nil
		case strings.Contains(command, "docker inspect"):
			return "/var/lib/docker/overlay2/merged\n", nil
		case strings.Contains(command, "docker images"):
			return "docker-sonic-telemetry\n", nil
		case strings.Contains(command, "supervisorctl status"):
			return "orchagent RUNNING pid 100, uptime 1:00:00\n", nil
		}
		return "", nil
	})

	return patches
}

// mockMultipleFailures patches for a system with both a failed service and failed hardware.
func mockMultipleFailures(t *testing.T) *gomonkey.Patches {
	t.Helper()
	patches := mockHealthySystem(t)

	// Override monit: container_checker not running
	patches.ApplyFunc(sccommon.GetDataFromHostCommand, func(command string) (string, error) {
		switch {
		case strings.Contains(command, "systemctl is-active monit"):
			return "active\n", nil
		case strings.Contains(command, "monit summary"):
			return "Monit 5.26.0 uptime: 10d 5h 20m\n" +
				" Name                            Status                   Type\n" +
				" container_checker               Not Running              Process\n" +
				" sonic                           Running                  System\n" +
				" root-overlay                    Accessible               Filesystem\n" +
				" var-log                         Accessible               Filesystem\n", nil
		case strings.Contains(command, "docker ps"):
			return "swss\nbgp\nteamd\n", nil
		case strings.Contains(command, "docker inspect"):
			return "/var/lib/docker/overlay2/merged\n", nil
		case strings.Contains(command, "docker images"):
			return "docker-sonic-telemetry\n", nil
		case strings.Contains(command, "supervisorctl status"):
			return "orchagent RUNNING pid 100, uptime 1:00:00\n", nil
		}
		return "", nil
	})

	// Override DB: ASIC0 temperature too hot
	patches.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
		if len(queries) > 0 && len(queries[0]) > 1 {
			switch queries[0][1] {
			case "FEATURE":
				return map[string]interface{}{
					"swss": map[string]interface{}{"state": "enabled", "has_global_scope": "True"},
					"bgp":  map[string]interface{}{"state": "enabled", "has_global_scope": "True"},
				}, nil
			case "TEMPERATURE_INFO":
				return map[string]interface{}{
					"ASIC0": map[string]interface{}{
						"temperature":    "120",
						"high_threshold": "100",
					},
				}, nil
			case "FAN_INFO":
				return map[string]interface{}{
					"Fan1": map[string]interface{}{
						"presence":       "True",
						"status":         "True",
						"speed":          "6000",
						"speed_target":   "6000",
						"is_under_speed": "False",
						"is_over_speed":  "False",
						"direction":      "intake",
					},
				}, nil
			case "PSU_INFO":
				return map[string]interface{}{
					"PSU 1": map[string]interface{}{
						"presence":              "True",
						"status":                "True",
						"temp":                  "25",
						"temp_threshold":         "60",
						"voltage":               "12.0",
						"voltage_min_threshold":  "11.0",
						"voltage_max_threshold":  "13.0",
					},
				}, nil
			case "LIQUID_COOLING_INFO":
				return map[string]interface{}{}, nil
			}
		}
		return map[string]interface{}{}, nil
	})

	return patches
}

// mockMonitNotReady patches for a system where monit is active but its summary
// output is too short (fewer than 3 lines), indicating monit is not ready.
func mockMonitNotReady(t *testing.T) *gomonkey.Patches {
	t.Helper()
	patches := mockHealthySystem(t)

	// Override: monit active but summary output only has header lines (< 3 lines of services)
	patches.ApplyFunc(sccommon.GetDataFromHostCommand, func(command string) (string, error) {
		switch {
		case strings.Contains(command, "systemctl is-active monit"):
			return "active\n", nil
		case strings.Contains(command, "monit summary"):
			return "Monit 5.26.0 uptime: 10d 5h 20m\n" +
				" Name                            Status                   Type\n", nil
		case strings.Contains(command, "docker ps"):
			return "swss\nbgp\nteamd\n", nil
		case strings.Contains(command, "docker inspect"):
			return "/var/lib/docker/overlay2/merged\n", nil
		case strings.Contains(command, "docker images"):
			return "docker-sonic-telemetry\n", nil
		case strings.Contains(command, "supervisorctl status"):
			return "orchagent RUNNING pid 100, uptime 1:00:00\n", nil
		}
		return "", nil
	})

	return patches
}

// mockUserDefinedCheckerFailure patches for a system where a user-defined checker
// command reports a hardware failure. The failure appears under the hardware section
// because its category ("CustomHardware") is not "Services".
func mockUserDefinedCheckerFailure(t *testing.T) *gomonkey.Patches {
	t.Helper()
	patches := mockHealthySystem(t)

	// Override config to include a user-defined checker command
	patches.ApplyFunc(sccommon.ReadJsonToMap, func(filePath string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"services_to_ignore":    []interface{}{},
			"devices_to_ignore":     []interface{}{},
			"user_defined_checkers": []interface{}{"check_my_hardware.py"},
		}, nil
	})

	// Override commands to also handle the user-defined checker command
	patches.ApplyFunc(sccommon.GetDataFromHostCommand, func(command string) (string, error) {
		switch {
		case strings.Contains(command, "systemctl is-active monit"):
			return "active\n", nil
		case strings.Contains(command, "monit summary"):
			return "Monit 5.26.0 uptime: 10d 5h 20m\n" +
				" Name                            Status                   Type\n" +
				" container_checker               Running                  Process\n" +
				" sonic                           Running                  System\n" +
				" root-overlay                    Accessible               Filesystem\n" +
				" var-log                         Accessible               Filesystem\n", nil
		case strings.Contains(command, "check_my_hardware.py"):
			// User-defined checker output: first line = category, remaining = object:status
			return "CustomHardware\nSensor1:Temperature too high\n", nil
		case strings.Contains(command, "docker ps"):
			return "swss\nbgp\nteamd\n", nil
		case strings.Contains(command, "docker inspect"):
			return "/var/lib/docker/overlay2/merged\n", nil
		case strings.Contains(command, "docker images"):
			return "docker-sonic-telemetry\n", nil
		case strings.Contains(command, "supervisorctl status"):
			return "orchagent RUNNING pid 100, uptime 1:00:00\n", nil
		}
		return "", nil
	})

	return patches
}

// mockSystemWithIgnoreConfig patches for a system that has failures but they are
// suppressed by services_to_ignore and devices_to_ignore in the config.
// container_checker is Not Running but ignored; ASIC temperature is too hot but "asic" is ignored.
func mockSystemWithIgnoreConfig(t *testing.T) *gomonkey.Patches {
	t.Helper()
	patches := mockHealthySystem(t)

	// Override config: ignore the failing service and device
	patches.ApplyFunc(sccommon.ReadJsonToMap, func(filePath string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"services_to_ignore": []interface{}{"container_checker"},
			"devices_to_ignore":  []interface{}{"asic"},
		}, nil
	})

	// Override monit: container_checker Not Running (will be ignored)
	patches.ApplyFunc(sccommon.GetDataFromHostCommand, func(command string) (string, error) {
		switch {
		case strings.Contains(command, "systemctl is-active monit"):
			return "active\n", nil
		case strings.Contains(command, "monit summary"):
			return "Monit 5.26.0 uptime: 10d 5h 20m\n" +
				" Name                            Status                   Type\n" +
				" container_checker               Not Running              Process\n" +
				" sonic                           Running                  System\n" +
				" root-overlay                    Accessible               Filesystem\n" +
				" var-log                         Accessible               Filesystem\n", nil
		case strings.Contains(command, "docker ps"):
			return "swss\nbgp\nteamd\n", nil
		case strings.Contains(command, "docker inspect"):
			return "/var/lib/docker/overlay2/merged\n", nil
		case strings.Contains(command, "docker images"):
			return "docker-sonic-telemetry\n", nil
		case strings.Contains(command, "supervisorctl status"):
			return "orchagent RUNNING pid 100, uptime 1:00:00\n", nil
		}
		return "", nil
	})

	// Override DB: ASIC0 temperature too hot (will be ignored because "asic" is in devices_to_ignore)
	patches.ApplyFunc(sccommon.GetMapFromQueries, func(queries [][]string) (map[string]interface{}, error) {
		if len(queries) > 0 && len(queries[0]) > 1 {
			switch queries[0][1] {
			case "FEATURE":
				return map[string]interface{}{
					"swss": map[string]interface{}{"state": "enabled", "has_global_scope": "True"},
					"bgp":  map[string]interface{}{"state": "enabled", "has_global_scope": "True"},
				}, nil
			case "TEMPERATURE_INFO":
				return map[string]interface{}{
					"ASIC0": map[string]interface{}{
						"temperature":    "120",
						"high_threshold": "100",
					},
				}, nil
			case "FAN_INFO":
				return map[string]interface{}{
					"Fan1": map[string]interface{}{
						"presence":       "True",
						"status":         "True",
						"speed":          "6000",
						"speed_target":   "6000",
						"is_under_speed": "False",
						"is_over_speed":  "False",
						"direction":      "intake",
					},
				}, nil
			case "PSU_INFO":
				return map[string]interface{}{
					"PSU 1": map[string]interface{}{
						"presence":              "True",
						"status":                "True",
						"temp":                  "25",
						"temp_threshold":         "60",
						"voltage":               "12.0",
						"voltage_min_threshold":  "11.0",
						"voltage_max_threshold":  "13.0",
					},
				}, nil
			case "LIQUID_COOLING_INFO":
				return map[string]interface{}{}, nil
			}
		}
		return map[string]interface{}{}, nil
	})

	return patches
}
