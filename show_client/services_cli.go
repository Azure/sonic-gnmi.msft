package show_client

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	log "github.com/golang/glog"

	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

type dockerService struct {
	DockerProcessName string           `json:"dockerProcessName"`
	Processes         []serviceProcess `json:"processes"`
}

type serviceProcess struct {
	User          string `json:"user"`
	Pid           string `json:"pid"`
	CPUPercentage string `json:"cpuPercentage"`
	MEMPercentage string `json:"memPercentage"`
	VSZ           string `json:"vsz"`
	RSS           string `json:"rss"`
	TTY           string `json:"tty"`
	Stat          string `json:"stat"`
	Start         string `json:"start"`
	Time          string `json:"time"`
	Command       string `json:"command"`
}

func getServices(_ sdc.OptionMap) ([]byte, error) {
	cmd := "sudo docker ps --format '{{.Names}}'"
	processesStr, err := GetDataFromHostCommand(cmd)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to run command:%s, err is:%v", cmd, err)
		log.Errorf(errorMessage)
		return nil, errors.New(errorMessage)
	}

	processesStr = strings.ReplaceAll(processesStr, "\r\n", "\n")
	serviceNames := strings.Split(strings.TrimSpace(processesStr), "\n")
	fmt.Printf("Found docker services: %s", processesStr)
	dockerServices := make([]dockerService, len(serviceNames))
	for index, serviceName := range serviceNames {
		cmd = fmt.Sprintf("sudo docker exec %s ps aux --no-headers", serviceName)
		processOutput, err := GetDataFromHostCommand(cmd)
		if err != nil {
			errorMessage := fmt.Sprintf("Failed to run command %q for service %s: %v", cmd, serviceName, err)
			log.Errorf(errorMessage)
			return nil, errors.New(errorMessage)
		}

		processOutput = strings.ReplaceAll(processOutput, "\r\n", "\n")
		processLines := strings.Split(strings.TrimSpace(processOutput), "\n")

		processes := make([]serviceProcess, len(processLines))
		for i, line := range processLines {
			fields := strings.Fields(line)
			if len(fields) < 11 {
				continue
			}
			process := serviceProcess{
				User:          fields[0],
				Pid:           fields[1],
				CPUPercentage: fields[2],
				MEMPercentage: fields[3],
				VSZ:           fields[4],
				RSS:           fields[5],
				TTY:           fields[6],
				Stat:          fields[7],
				Start:         fields[8],
				Time:          fields[9],
				Command:       strings.Join(fields[10:], " "),
			}

			processes[i] = process
		}

		dockerServices[index].DockerProcessName = serviceName
		dockerServices[index].Processes = processes
	}

	return json.Marshal(dockerServices)
}
