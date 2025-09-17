package show_client

import (
        "encoding/json"
        "fmt"
        sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

const (
        cfgSwitchTrimming = "SWITCH_TRIMMING"
        cfgTrimKey        = "GLOBAL"
)

// SwitchTrimmingResponse defines the structured output
type SwitchTrimmingResponse struct {
        Size       string `json:"size"`
        DSCPValue  string `json:"dscp_value"`
        TCValue    string `json:"tc_value"`
        QueueIndex string `json:"queue_index"`
}

// getOrNA returns the value for a key or "N/A" if missing or empty
func getOrNA(entry map[string]interface{}, key string) string {
        if val, ok := entry[key]; ok {
                if str, ok := val.(string); ok && str != "" {
                        return str
                }
        }
        return "N/A"
}

// getSwitchTrimmingGlobalConfig queries CONFIG_DB and returns JSON response
func getSwitchTrimmingGlobalConfig(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
        row, err := GetMapFromQueries([][]string{{"CONFIG_DB", cfgSwitchTrimming, cfgTrimKey}})
        if err != nil {
                return nil, fmt.Errorf("failed to query CONFIG_DB: %w", err)
        }

        if len(row) == 0 {
                return json.MarshalIndent(map[string]string{
                        "response": "No configuration is present in CONFIG DB",
                }, "", "  ")
        }

        response := SwitchTrimmingResponse{
                Size:       getOrNA(row, "size"),
                DSCPValue:  getOrNA(row, "dscp_value"),
                TCValue:    getOrNA(row, "tc_value"),
                QueueIndex: getOrNA(row, "queue_index"),
        }

        return json.MarshalIndent(response, "", "  ")
}
