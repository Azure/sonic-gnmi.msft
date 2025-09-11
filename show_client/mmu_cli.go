package show_client

import (
	"encoding/json"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

type MmuTotals struct {
	Pools    int `json:"pools"`
	Profiles int `json:"profiles"`
}

type MmuConfigResponse struct {
	// renamed JSON keys and fields
	LosslessTrafficPatterns map[string]map[string]interface{} `json:"losslessTrafficPatterns,omitempty"`
	Pools                   map[string]map[string]interface{} `json:"pools,omitempty"`
	Profiles                map[string]map[string]interface{} `json:"profiles,omitempty"`
	Totals                  *MmuTotals                        `json:"totals,omitempty"`
}

const (
	// CONFIG_DB(cfg) table names
	cfgTableDefaultLossless = "DEFAULT_LOSSLESS_BUFFER_PARAMETER"
	cfgTableBufferPool      = "BUFFER_POOL"
	cfgTableBufferProfile   = "BUFFER_PROFILE"
)

// getMmuConfig implements Python `mmuconfig -l` in Go by reading CONFIG_DB tables.
// https://github.com/Azure/sonic-utilities.msft/blob/3cb0eb2402a8da806b7c858eaa7e6be950c92fe3/scripts/mmuconfig#L90
// Example output:
// $ show mmu -vv
// Pool: egress_lossless_pool
// ----  ---------
// mode  static
// size  164075364
// type  egress
// ----  ---------

// Pool: ingress_lossless_pool
// ----  ---------
// mode  dynamic
// size  164075364
// type  ingress
// xoff  20181824
// ----  ---------

// Total pools: 2

// The following is the data in redis
// $ redis-cli -n 4 KEYS 'BUFFER_POOL*'
//  1. "BUFFER_POOL|ingress_lossless_pool"
//  2. "BUFFER_POOL|egress_lossless_pool"
//
// $ redis-cli -n 4 HGETALL 'BUFFER_POOL|ingress_lossless_pool'
//  1. "mode"
//  2. "dynamic"
//  3. "size"
//  4. "164075364"
//  5. "type"
//  6. "ingress"
//  7. "xoff"
//  8. "20181824"
//
// $ redis-cli -n 4 HGETALL 'BUFFER_POOL|egress_lossless_pool'
//  1. "mode"
//  2. "static"
//  3. "size"
//  4. "164075364"
//  5. "type"
//  6. "egress"
func getMmuConfig(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	lossless, err := getTableAsNestedMap(ConfigDb, cfgTableDefaultLossless)
	if err != nil {
		log.Errorf("[show mmu]|Failed to read %s: %v", cfgTableDefaultLossless, err)
		return nil, err
	}
	pools, err := getTableAsNestedMap(ConfigDb, cfgTableBufferPool)
	if err != nil {
		log.Errorf("[show mmu]|Failed to read %s: %v", cfgTableBufferPool, err)
		return nil, err
	}
	profiles, err := getTableAsNestedMap(ConfigDb, cfgTableBufferProfile)
	if err != nil {
		log.Errorf("[show mmu]|Failed to read %s: %v", cfgTableBufferProfile, err)
		return nil, err
	}

	resp := MmuConfigResponse{
		LosslessTrafficPatterns: lossless,
		Pools:                   pools,
		Profiles:                profiles,
	}

	if v, ok := options[OptionKeyVerbose].Bool(); ok && v {
		resp.Totals = &MmuTotals{
			Pools:    len(pools),
			Profiles: len(profiles),
		}
	}

	return json.Marshal(resp)
}

// pls read the comments on getMmuConfig
func getTableAsNestedMap(db string, table string) (map[string]map[string]interface{}, error) {
	// Get all keys and values in the table
	queries := [][]string{{db, table}}
	msi, err := GetMapFromQueries(queries)
	if err != nil {
		return nil, err
	}

	out := make(map[string]map[string]interface{}, len(msi))
	for k, v := range msi {
		// The value is also a map, e.g. key is mode, value is static
		if row, ok := v.(map[string]interface{}); ok {
			out[k] = row
		} else {
			log.Errorf("[show mmu]|Unexpected value for %s: %v", k, v)
		}
	}
	return out, nil
}
