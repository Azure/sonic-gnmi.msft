package helpers

import (
	"bufio"
	"encoding/json"
	"fmt"
	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	"reflect"
	"strings"
)

func cleanPrefix(line, prefix string) string {
	return strings.TrimSpace(strings.TrimPrefix(line, prefix))
}

func parseProcessLine(line string) (*common.TopProcessData, error) {
	fields := strings.Fields(line)
	if len(fields) < reflect.TypeOf(common.TopProcessData{}).NumField() {
		return nil, fmt.Errorf("invalid process line: %q", line)
	}
	return &common.TopProcessData{
		PID:     fields[0],
		User:    fields[1],
		PR:      fields[2],
		NI:      fields[3],
		VIRT:    fields[4],
		RES:     fields[5],
		SHR:     fields[6],
		S:       fields[7],
		CPU:     fields[8],
		MEM:     fields[9],
		TIME:    fields[10],
		Command: strings.Join(fields[11:], " "),
	}, nil
}

func LoadProcessesDataFromCmdOutput(output string) ([]byte, error) {

	if strings.TrimSpace(output) == "" {
		log.Errorf("Got empty output for top command")
		return nil, fmt.Errorf("Got empty output for top command")
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var (
		uptime, tasks, cpuUsage, memoryUsage, swapUsage string
		processes                                       []common.TopProcessData
		startParsing                                    bool
	)

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "top -"):
			uptime = cleanPrefix(line, "top -")
		case strings.HasPrefix(line, "Tasks:"):
			tasks = cleanPrefix(line, "Tasks:")
		case strings.HasPrefix(line, "%Cpu(s):"):
			cpuUsage = cleanPrefix(line, "%Cpu(s):")
		case strings.HasPrefix(line, "MiB Mem :"):
			memoryUsage = cleanPrefix(line, "MiB Mem :")
		case strings.HasPrefix(line, "MiB Swap:"):
			swapUsage = cleanPrefix(line, "MiB Swap:")
		case strings.Contains(line, "PID") && strings.Contains(line, "USER"):
			startParsing = true
		default:
			if !startParsing || strings.TrimSpace(line) == "" {
				continue
			}
			process, err := parseProcessLine(line)
			if err != nil {
				log.V(2).Infof("Skipping line: %v", err)
				continue
			}
			processes = append(processes, *process)
		}
	}

	if uptime == "" || len(processes) == 0 {
		return nil, fmt.Errorf("incomplete top output: missing uptime or processes")
	}

	response := common.TopProcessCompleteResponse{
		Uptime:      uptime,
		Tasks:       tasks,
		CPUUsage:    cpuUsage,
		MemoryUsage: memoryUsage,
		SwapUsage:   swapUsage,
		Processes:   processes,
	}

	return json.MarshalIndent(response, "", "  ")
}
