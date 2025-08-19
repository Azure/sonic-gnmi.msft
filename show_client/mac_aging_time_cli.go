package show_client

import (
	"encoding/json"
	"fmt"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

/*
admin@sonic:~$ show mac aging-time
Aging time for switch is 600 seconds
admin@sonic:~$ redis-cli -n 0 hget "SWITCH_TABLE:switch" "fdb_aging_time"
"600"
*/

func getMacAgingTime(options sdc.OptionMap) ([]byte, error) {
	queries := [][]string{
		{"APPL_DB", "SWITCH_TABLE", "switch"},
	}
	data, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to get mac aging time data from queries %v, got err: %v", queries, err)
		return nil, err
	}
	log.V(6).Infof("GetMapFromQueries result: %+v", data)

	// Default value if not found
	agingTime := "N/A"

	if val, ok := data["fdb_aging_time"]; ok && val != nil {
		strVal := fmt.Sprintf("%v", val)
		if strVal != "" {
			agingTime = strVal + "s"
		} else {
			log.Warningf("Key 'fdb_aging_time' found but empty in data")
		}
	} else {
		log.Warningf("Key 'fdb_aging_time' not found or empty in data")
	}

	// Build response, append "s" for seconds
	result := map[string]string{
		"fdb_aging_time": agingTime,
	}
	return json.Marshal(result)
}
