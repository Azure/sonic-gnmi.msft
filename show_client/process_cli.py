package show_client

import (
	"encoding/json"
	"fmt"
	"show_client/common"
	"show_client/helpers"
	"sort"
	"strconv"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

const (
	topCommand = "top -bn 1"
	orderByCPU = " -o %CPU"
)

func getProcesesByCPU(options sdc.OptionMap) ([]byte, error) {
	cmdForProcessByCPU := topCommand + orderByCPU

	processDetails, err := GetDataFromHostCommand(cmdForProcessByCPU)
	if err != nil {
		return []byte(""), err
	}

	processesOrdered := helpers.LoadProcessesDataFromCmdOutput(cmdForProcessByCPU)

	return json.Marshal(processesOrdered)
}
