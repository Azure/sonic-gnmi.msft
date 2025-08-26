package show_client

import (
	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

func getUptime(options sdc.OptionMap) ([]byte, error) {
    uptimeCommandWithParam = "uptime -p"
	data, err := GetDataFromHostCommand(uptimeCommandWithParam)
	if err != nil {
		log.Errorf("Unable to get data uptime, got err: %v", err)
		return nil, err
	}
	return data, nil
}
