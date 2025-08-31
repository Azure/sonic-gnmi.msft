package show_client

import (
	"encoding/json"
	"strings"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

var showSystemMemoryCommand = "free -m"

func getSystemMemory(options sdc.OptionMap) ([]byte, error) {
	// Get data from host command
	output, err := GetDataFromHostCommand(showSystemMemoryCommand)
	if err != nil {
		log.Errorf("Unable to succesfully execute command %v, get err %v", showSystemMemoryCommand, err)
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	header := strings.Fields(lines[0])
	systemMemoryResponse := make([]map[string]string, len(lines)-1)
	for i, line := range lines[1:] {
		entry := make(map[string]string)
		fields := strings.Fields(line)

		entry["type"] = strings.ReplaceAll(fields[0], ":", "")
		for j, field := range fields[1:] {
			entry[header[j]] = field
		}
		systemMemoryResponse[i] = entry
	}
	return json.Marshal(systemMemoryResponse), nil
}
