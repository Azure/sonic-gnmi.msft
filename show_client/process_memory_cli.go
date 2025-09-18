package show_client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// Struct to hold individual process details
type TopProcessResponse struct {
	PID     string `json:"pid"`
	User    string `json:"user"`
	PR      string `json:"pr"`
	NI      string `json:"ni"`
	VIRT    string `json:"virt"`
	RES     string `json:"res"`
	SHR     string `json:"shr"`
	S       string `json:"s"`
	CPU     string `json:"cpu"`
	MEM     string `json:"mem"`
	TIME    string `json:"time"`
	Command string `json:"command"`
}

// Struct to hold the full snapshot
type TopProcessMemoryResponse struct {
	Uptime      string               `json:"uptime"`
	Tasks       string               `json:"tasks"`
	CPUUsage    string               `json:"cpu_usage"`
	MemoryUsage string               `json:"memory_usage"`
	SwapUsage   string               `json:"swap_usage"`
	Processes   []TopProcessResponse `json:"processes"`
}

const (
	topMemoryCommand     = "top -bn 1 -o %MEM"
	countOfProcessFields = 12
)

func cleanPrefix(line, prefix string) string {
	return strings.TrimSpace(strings.TrimPrefix(line, prefix))
}

func parseProcessLine(line string) (*TopProcessResponse, error) {
	fields := strings.Fields(line)
	if len(fields) < countOfProcessFields {
		return nil, fmt.Errorf("invalid process line: %q", line)
	}
	return &TopProcessResponse{
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

func getTopMemoryUsage(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	output, err := common.GetDataFromHostCommand(topMemoryCommand)

	if err != nil {
		log.Errorf("Unable to execute top command: %v", err)
		return nil, fmt.Errorf("GetDataFromHostCommand errored out")
	}
	if strings.TrimSpace(output) == "" {
		log.Errorf("Got empty output for top command")
		return nil, fmt.Errorf("Got empty output for top command")
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var (
		uptime, tasks, cpuUsage, memoryUsage, swapUsage string
		processes                                       []TopProcessResponse
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

	response := TopProcessMemoryResponse{
		Uptime:      uptime,
		Tasks:       tasks,
		CPUUsage:    cpuUsage,
		MemoryUsage: memoryUsage,
		SwapUsage:   swapUsage,
		Processes:   processes,
	}

	return json.MarshalIndent(response, "", "  ")
}
