package show_client

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

const showSystemMemoryCommand = "free -m"

func getSystemMemory(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	// Get data from host command
	output, err := common.GetDataFromHostCommand(showSystemMemoryCommand)
	if err != nil {
		log.Errorf("Unable to successfully execute command %v, get err %v", showSystemMemoryCommand, err)
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) == 0 {
		log.Errorf("No output returned from command %v", showSystemMemoryCommand)
		return nil, fmt.Errorf("No output returned from command %v", showSystemMemoryCommand)
	}

	header := strings.Fields(lines[0])
	systemMemoryResponse := make([]map[string]string, 0)
	for _, line := range lines[1:] {
		entry := make(map[string]string)
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		entry["type"] = strings.ReplaceAll(fields[0], ":", "")
		for j, field := range fields[1:] {
			if j >= len(header) {
				log.Errorf("Malformed output from command %v", showSystemMemoryCommand)
				return nil, fmt.Errorf("Malformed output from command %v", showSystemMemoryCommand)
			}
			entry[header[j]] = field
		}
		systemMemoryResponse = append(systemMemoryResponse, entry)
	}
	return json.Marshal(systemMemoryResponse)
}
