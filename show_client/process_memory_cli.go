package show_client

import (
        "fmt"
        "encoding/json"
        log "github.com/golang/glog"
        sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
        "strings"
)

var (
        topMemoryCommand = "top -bn 1 -o %MEM"
)

func getTopMemoryUsage(options sdc.OptionMap) ([]byte, error) {
        output, err := GetDataFromHostCommand(topMemoryCommand)
        if err != nil {
                log.Errorf("Unable to execute top command: %v", err)
                return nil, fmt.Errorf("failed to execute top command: %v", err)
        }
        trimmed := strings.TrimSpace(output)
        if trimmed == "" {
                log.Errorf("Got empty output for top command")
                return nil, fmt.Errorf("top command returned empty output")
        }
        wrapped := map[string]string{
                "process_memory": trimmed,
        }
        topMemoryJSON, err := json.MarshalIndent(wrapped, "", "  ")
        if err != nil {
                log.Errorf("Failed to marshal top output: %v", err)
                return nil, fmt.Errorf("failed to marshal top output: %v", err)
        }
        return topMemoryJSON, nil
}
