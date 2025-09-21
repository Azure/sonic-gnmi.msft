package show_client

import (
	"encoding/json"
	"fmt"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

const (
	CfgSwitchTrimming = "SWITCH_TRIMMING"
	CfgTrimKey        = "GLOBAL"
)

// SwitchTrimmingResponse defines the structured output
type SwitchTrimmingResponse struct {
	Size       string `json:"size"`
	DSCPValue  string `json:"dscp_value"`
	TCValue    string `json:"tc_value"`
	QueueIndex string `json:"queue_index"`
}

// getSwitchTrimmingGlobalConfig queries CONFIG_DB and returns JSON response
func getSwitchTrimmingGlobalConfig(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	row, err := common.GetMapFromQueries([][]string{{"CONFIG_DB", CfgSwitchTrimming, CfgTrimKey}})
	if err != nil {
		return nil, fmt.Errorf("failed to query CONFIG_DB: %w", err)
	}

	if len(row) == 0 {
		return json.Marshal(map[string]string{
			"response": "No configuration is present in CONFIG DB",
		})
	}

	response := SwitchTrimmingResponse{
		Size:       GetValueOrDefault(row, "size", "N/A"),
		DSCPValue:  GetValueOrDefault(row, "dscp_value", "N/A"),
		TCValue:    GetValueOrDefault(row, "tc_value", "N/A"),
		QueueIndex: GetValueOrDefault(row, "queue_index", "N/A"),
	}

	return json.Marshal(response)
}
