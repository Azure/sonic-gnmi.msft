package show_client

import (
	"encoding/json"
	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
	"strings"
)

type uptimeResponse struct {
	Uptime string `json:"uptime"`
}

func getUptime(options sdc.OptionMap) ([]byte, error) {
	uptimeCommandWithParam := "uptime -p"
	uptimeData, err := GetDataFromHostCommand(uptimeCommandWithParam)
	if err != nil {
		return nil, err
	}

	var uptimeResp uptimeResponse
	uptimeResp.Uptime = strings.TrimSuffix(uptimeData, "\n")
	return json.Marshal(uptimeResp)
}
