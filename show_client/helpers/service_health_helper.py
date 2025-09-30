package helpers

import (
	"encoding/json"
	"net"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"

	"bufio"
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)


const (
	SYSLOG_IDENTIFIER         = "service_checker"
	EVENTS_PUBLISHER_SOURCE   = "sonic-events-host"
	EVENTS_PUBLISHER_TAG      = "process-not-running"
	CRITICAL_PROCESS_CACHE    = "/tmp/critical_process_cache"
	CRITICAL_PROCESSES_PATH   = "etc/supervisor/critical_processes"
	GET_CONTAINER_FOLDER_CMD  = `docker inspect %s --format '{{.GraphDriver.Data.MergedDir}}'`
	CHECK_MONIT_SERVICE_CMD   = "systemctl is-active monit.service"
	CHECK_CMD                 = "monit summary -B"
	MIN_CHECK_CMD_LINES       = 3
    STATUS_NOT_OK = "Not OK"
    STATUS_OK = "OK"
)



func ServiceHealthCheck(configs map[string]interface, stats map[string]interface) map[string]interface {
	reset()
	checkByMonit(config, stats)
	checkServices(config, stats)
}

func SetStat(stats map[string]interface, objectType string, objectName string, message string, status string) {
    value, ok := stats[objectName]; !ok {
        stats[objectName] = make(map[string] interface)
    }

    stats[objectName]["type"] = objectType
    stats[objectName]["message"] = messaage
    stats[objectName]["status"] = status
}

func ignoreService(configs map[string]interface, serviceName string) {
    value, ok := configs["services_to_ignore"]; !ok {
        return false
    }

    ignoredServices, isSlice := val.([]string)
    if !isSlice {
        return false
    }

    for _, service := range ignoredServices {
		if service == serviceName {
			return true
		}
	}

    return false
}

func reset() {
    //Cleanup if needed.	
}

func checkByMonit(config map[string]interface, stats map[string]interface) {
	output, err := common.GetDataFromHostCommand(CHECK_MONIT_SERVICE_CMD)

    if err != nil {
        fmt.Errorf("Unable to execute service check by monit command: %v", err)
        return
    }

	if strings.TrimSpace(output) != "active" {
		SetStat("Service", "monit", "monit service is not running", STATUS_NOT_OK)
		return
	}
	output, err = common.GetDataFromHostCommand(CHECK_CMD)
    if err != nil {
        fmt.Errorf("Unable to execute service check by monit command: %v", err)
        return
    }
	lines := strings.Split(output, "\n")
	if len(lines) < MIN_CHECK_CMD_LINES {
		SetStat("Service", "monit", "monit service is not ready", STATUS_NOT_OK)
		return
	}
	statusBegin := strings.Index(lines[1], "Status")
	typeBegin := strings.Index(lines[1], "Type")
	if statusBegin < 0 || typeBegin < 0 {
		SetStat("Service", "monit", "output of monit summary -B is invalid or incompatible", STATUS_NOT_OK)
		return
	}
	for _, line := range lines[2:] {
		serviceName := strings.TrimSpace(line[:statusBegin])
		if  ignoreService(config, serviceName) {
			continue
		}
		status := strings.TrimSpace(line[statusBegin:typeBegin])
		serviceType := strings.TrimSpace(line[typeBegin:])
		expectStatus, ok := EXPECT_STATUS_DICT[serviceType]
		if !ok {
			continue
		}
		if expectStatus != status {
			SetStat(serviceType, serviceName, serviceName+" is not "+expectStatus, STATUS_NOT_OK)
		} else {
			SetStat(serviceType, serviceName, "", STATUS)
		}
	}
}

func getExpectedRunningContainers(featureTable map[string]interface) (map[string]struct{}, map[string]string) {
	expectedRunningContainers := make(map[string]struct{})
	containerFeatureDict := make(map[string]string)

	runAllInstanceList := map[string]struct{}{
		"database": {},
		"bgp":      {},
	}

	containerList := []string{}
	for containerName := range featureTable {
		if containerName == "frr_bmp" {
			continue
		}
		// slim image does not have telemetry container and corresponding docker image
		if containerName == "telemetry" {
			if !common.CheckDockerImageExist("docker-sonic-telemetry") {
				if !common.CheckDockerImageExist("docker-sonic-gnmi") {
					log.Errorf("Ignoring telemetry container check on image which has no corresponding docker image")
				} else {
					containerList = append(containerList, "gnmi")
				}
				continue
			}
		}
		containerList = append(containerList, containerName)
	}

	for _, containerName := range containerList {
        //TODO: convert below obj
		featureEntry := featureTable[containerName]
		state := featureEntry["state"]
		if state != "disabled" && state != "always_disabled" {
			if common.IsMultiAsic() {
                log.Errorf("Currently multi ASIC not supported."
			} else {
				expectedRunningContainers[containerName] = struct{}{}
				containerFeatureDict[containerName] = containerName
			}
		}
	}

	if device_info.IsSupervisor() || device_info.IsDisaggregatedChassis() {
		expectedRunningContainers["database-chassis"] = struct{}{}
		containerFeatureDict["database-chassis"] = "database"
	}

	return expectedRunningContainers, containerFeatureDict
}

func checkServices(config map[string]interface) {

    queries : = {}
    featureData, err := //GetFeatureList

    expectedRunningContainer, containerFeature := getExpectedRunningContainer(featureData)

	// Minimal stub for demo: just mark system as ok if any critical process exists
	if len(sc.containerCriticalProcesses) == 0 {
		SetStat("Service", "system", "no critical process found", STATUS_NOT_OK)
		return
	}
	SetStat("Service", "system", "", STATUS_OK)
}

var EXPECT_STATUS_DICT = map[string]string{
	"System":     "Running",
	"Process":    "Running",
	"Filesystem": "Accessible",
	"Program":    "Status ok",
}

func hasKey(m map[string]struct{}, key string) bool {
	_, ok := m[key]
	return ok
}
