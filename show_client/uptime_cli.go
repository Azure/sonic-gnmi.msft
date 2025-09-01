package show_client

import (
	"encoding/json"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
	"strings"
)

type uptimeResponse struct {
	Uptime string `json:"uptime"`
}

func getUptime(options sdc.OptionMap) ([]byte, error) {
	uptimeParam := []string{"-p"}
	uptimeData, err := GetUptime(uptimeParam)
	if err != nil {
		return nil, err
	}

	var uptimeResp uptimeResponse
	uptimeResp.Uptime = strings.TrimSuffix(uptimeData, "\n")
	return json.Marshal(uptimeResp)
}
