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
		log.V(2).Infof("Processing service %s", serviceName)
		cmd = fmt.Sprintf(`bash -o pipefail -c "sudo docker exec %s ps aux --no-headers | sed '$d'"`, serviceName)
		processOutput, err := GetDataFromHostCommand(cmd)
		log.V(2).Infof("Command output: %s", processOutput)
		if err != nil {
			log.Errorf("Failed to run command %q for service %s: %v", cmd, serviceName, err)
			continue
		}

		processOutput = strings.ReplaceAll(processOutput, "\r\n", "\n")
		processLines := strings.Split(strings.TrimSpace(processOutput), "\n")

		var processes []serviceProcess
		for _, line := range processLines {
			fields := strings.Fields(line)
			if len(fields) < 11 {
				log.Errorf("Invalid process line %q for service %s", line, serviceName)
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

			processes = append(processes, process)
		}

		if len(processes) > 0 {
			dockerServices[index].DockerProcessName = serviceName
			dockerServices[index].Processes = processes
		}
	}

	return json.Marshal(dockerServices)
}
