package show_client

import (
	"encoding/json"
	"fmt"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	"github.com/sonic-net/sonic-gnmi/show_client/helpers"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

const BGP_DEVICE_GLOBAL_KEY = "STATE"
const TSA_KEY = "tsa_enabled"
const WCMP_KEY = "wcmp_enabled"

func buildBgpDeviceGlobalEntry(data map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"tsa":    helpers.StateBoolToStr(common.GetValueOrDefault(data, TSA_KEY, "N/A")),
		"w-ecmp": helpers.StateBoolToStr(common.GetValueOrDefault(data, WCMP_KEY, "N/A")),
	}
}

func getBGPDeviceGlobal(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	// BGP_DEVICE_GLOBAL table from CONFIG_DB
	queries := [][]string{{"CONFIG_DB", "BGP_DEVICE_GLOBAL"}}

	rawData, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to get BGP device global data from queries %v, got err: %v", queries, err)
		return nil, err
	}

	bgpDeviceGlobalTable := rawData

	if len(bgpDeviceGlobalTable) == 0 {
		log.Errorf("No configuration is present in CONFIG_DB")
		return nil, fmt.Errorf("No configuration is present")
	}

	deviceGlobalData, ok := bgpDeviceGlobalTable[BGP_DEVICE_GLOBAL_KEY].(map[string]interface{})
	if !ok {
		log.Errorf("No valid configuration is present in CONFIG_DB")
		return nil, fmt.Errorf("No valid configuration is present")
	}

	response := buildBgpDeviceGlobalEntry(deviceGlobalData)

	return json.Marshal(response)
}
