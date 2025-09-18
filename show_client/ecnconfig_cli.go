package show_client

import (
	"encoding/json"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// getEcnProfiles fetches all WRED_PROFILE entries from CONFIG_DB.
// Redis keys: "WRED_PROFILE|<profile_name>"
func getEcnProfiles(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	queries := [][]string{
		{"CONFIG_DB", "WRED_PROFILE", "*"},
	}
	data, err := common.GetMapFromQueries(queries)

	if err != nil {
		log.Errorf("Unable to get ECN WRED profile data from queries %v, err: %v", queries, err)
		return nil, err
	}
	return json.Marshal(data)
}
