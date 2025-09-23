package show_client

import (
	"encoding/json"
	"strings"

	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

type uptimeResponse struct {
	Uptime string `json:"uptime"`
}

func getUptime(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	uptimeParam := []string{"-p"}
	uptimeData := common.GetUptime(uptimeParam)

	var uptimeResp uptimeResponse
	uptimeResp.Uptime = strings.TrimSuffix(uptimeData, "\n")
	return json.Marshal(uptimeResp)
}
