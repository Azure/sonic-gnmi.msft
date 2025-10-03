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

var EXPECT_STATUS_DICT = map[string]string{
	"System":     "Running",
	"Process":    "Running",
	"Filesystem": "Accessible",
	"Program":    "Status ok",
}

func ServiceHealthCheck(configs map[string]interface{}, stats map[string]interface{}) map[string]interface{} {
	reset()
	checkByMonit(config, stats)
	checkServices(config, stats)
}

func reset() {
    //Cleanup if needed.	
}

func checkByMonit(config map[string]interface{}, stats map[string]interface{}) {
	output, err := common.GetDataFromHostCommand(CHECK_MONIT_SERVICE_CMD)

    if err != nil {
        fmt.Errorf("Unable to execute service check by monit command: %v", err)
        return
    }

	if strings.TrimSpace(output) != "active" {
		common.SetStat(stats, "Service", "monit", "monit service is not running", STATUS_NOT_OK)
		return
	}
	output, err = common.GetDataFromHostCommand(CHECK_CMD)
    if err != nil {
        fmt.Errorf("Unable to execute service check by monit command: %v", err)
        return
    }
	lines := strings.Split(output, "\n")
	if len(lines) < MIN_CHECK_CMD_LINES {
		common.SetStat(stats, "Service", "monit", "monit service is not ready", STATUS_NOT_OK)
		return
	}
	statusBegin := strings.Index(lines[1], "Status")
	typeBegin := strings.Index(lines[1], "Type")
	if statusBegin < 0 || typeBegin < 0 {
		common.SetStat(stats, "Service", "monit", "output of monit summary -B is invalid or incompatible", STATUS_NOT_OK)
		return
	}
	for _, line := range lines[2:] {
		serviceName := strings.TrimSpace(line[:statusBegin])
		if  common.IgnoreService(config, serviceName) {
			continue
		}
		status := strings.TrimSpace(line[statusBegin:typeBegin])
		serviceType := strings.TrimSpace(line[typeBegin:])
		expectStatus, ok := EXPECT_STATUS_DICT[serviceType]
		if !ok {
			continue
		}
		if expectStatus != status {
			common.SetStat(serviceType, serviceName, serviceName+" is not "+expectStatus, STATUS_NOT_OK)
		} else {
			common.SetStat(serviceType, serviceName, "", STATUS)
		}
	}
}

func checkServices(config map[string]interface{}, stats map[string]interface{}) {

    queries := [][]string{{"CONFIG_DB", "FEATURE"}} 
    featureData, err := common.GetMapFromQueries(queries)
	if err != nil {
		return 
	} 

    expectedRunningContainers, containerFeature := common.GetExpectedRunningContainers(featureData)
    currentRunningContainers := common.GetDockerRunningContainers()
    containerCriticalProcesses, badProcesses := common.GetContainerCriticalProcesses(currentRunningContainers)

    for expectedRunningContainer := range expectedRunningContainers {
        if _, exists := currentRunningContainers[expectedRunningContainer]; !exists {
            common.SetStat(stats, "Service", expectedRunningContainer, "Container " + expectedRunningContainer + " is not running", STATUS_NOT_OK)
        }
    }

    if len(containerCriticalProcesses) < 1 {
        common.SetStat(stats, "Service", expectedRunningContainer, "no critical process found", STATUS_NOT_OK)
    }

    for container, criticalProcesses := range containerCriticalProcesses {
        status := common.CheckProcessesStatus(container, criticalProcesses, config, containerFeature, featureData, stats)
        common.SetStat(stats, "Service", container, "no critical process found", STATUS_NOT_OK)
    }

    for badContainer, _ := range badProcesses {
        common.SetStat(stats, "Service", badContainer, "Syntax of critical_processes file is incorrect", STATUS_NOT_OK)
    }
}
