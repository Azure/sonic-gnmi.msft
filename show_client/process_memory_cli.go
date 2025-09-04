package show_client

import (
	"fmt"
	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
	"strings"
)

func getTopMemoryUsage(options sdc.OptionMap) (string, error) {
	output, err := GetDataFromHostCommand(topMemoryCommand)
	if err != nil {
		log.Errorf("Unable to execute top command: %v", err)
		return "", fmt.Errorf("failed to execute top command: %v", err)
	}
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		log.Errorf("Got empty output for top command")
		return "", fmt.Errorf("top command returned empty output")
	}

	processTopMemoryUsage := strings.TrimRight(trimmed, "\n")

	return processTopMemoryUsage, nil
}
