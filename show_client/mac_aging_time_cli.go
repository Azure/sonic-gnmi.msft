package show_client

import (
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
		{"APPL_DB", "SWITCH_TABLE:switch", "fdb_aging_time"},
	}
	data, err := GetDataFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to get mac aging time data from queries %v, got err: %v", queries, err)
		return nil, err
	}
	log.Infof("GetDataFromQueries result: %s", string(data))
	return data, nil
}
