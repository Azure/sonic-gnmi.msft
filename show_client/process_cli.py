package show_client

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

const (
topCommand = "top -bn 1"
orderByCPU = " -o %CPU"
)

func loadProcessesDataFromCmdOutput(data string) string {
    //Store the data in process struct
    scanner := bufio.NewScanner(strings.NewReader(data))
    for scanner.Scan() {
		line := scanner.Text()
    }
}

func getProcesesByCPU(options sdc.OptionMap) ([]byte, error) {
	cmdForProcessByCPU := topCommand + orderByCPU 
	
    processDetails, err := GetDataFromHostCommand(cmdForProcessByCPU)
	if err != nil {
		return []byte(""), err
	}

	processesOrdered := loadProcessesDataFromCmdOutput(cmdForProcessByCPU)

	return json.Marshal(processesOrdered)
}
