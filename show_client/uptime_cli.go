package show_client

import (
	"encoding/json"
	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

type uptimeResponse struct {
	uptime string `json:"uptime"`
}

func getUptime(options sdc.OptionMap) ([]byte, error) {
	uptimeCommandWithParam := "uptime -p"
	uptimeData, err := GetDataFromHostCommand(uptimeCommandWithParam)
	if err != nil {
		log.Errorf("Unable to get data uptime, got err: %v", err)
		return nil, err
	}

	var uptimeResp uptimeResponse
	uptimeResp.uptime = uptimeData
	return json.Marshal(uptimeResp)
}
