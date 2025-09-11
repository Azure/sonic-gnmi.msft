package show_client

import (
	"fmt"
	"strings"

	log "github.com/golang/glog"
)

// BufferPoolStat represents the JSON shape for buffer pool stats (e.g., {"Bytes":"1234"}).
type BufferPoolStat struct {
	Bytes string `json:"Bytes"`
}

const (

	// Redis schema tables and keys used across show_client
	userWatermarkTable       = "USER_WATERMARKS"
	persistentWatermarkTable = "PERSISTENT_WATERMARKS"
	bufferPoolNameMapKey     = "COUNTERS_BUFFER_POOL_NAME_MAP"
)

// loadBufferPoolNameMap fetches and normalizes the buffer pool name -> oid mapping.
// See UT data testdata/COUNTERS_BUFFER_POOL_NAME_MAP.txt
func loadBufferPoolNameMap() (map[string]string, error) {
	nameMapQueries := [][]string{{"COUNTERS_DB", bufferPoolNameMapKey}}
	nameMap, err := GetMapFromQueries(nameMapQueries)
	if err != nil {
		return nil, fmt.Errorf("Get buffer pool name map %s failed: %w", bufferPoolNameMapKey, err)
	}

	poolToOid := make(map[string]string, len(nameMap))
	for pool, val := range nameMap {
		if s, ok := val.(string); ok && strings.HasPrefix(s, "oid:") {
			poolToOid[pool] = s
			continue
		}
	}
	if len(poolToOid) == 0 {
		return nil, fmt.Errorf("No buffer pool OIDs extracted from %s", bufferPoolNameMapKey)
	}
	if log.V(4) {
		for k, v := range poolToOid {
			log.Infof("[buffer_pool] debug poolToOid %s -> %s", k, v)
		}
	}
	return poolToOid, nil
}

// collectBufferPoolWatermarks fetches watermark bytes for the provided buffer pools (name->oid)
func collectBufferPoolWatermarks(pools map[string]string, tableName string, fieldName string) map[string]BufferPoolStat {
	result := make(map[string]BufferPoolStat, len(pools))
	for pool, oid := range pools {
		data, err := GetMapFromQueries([][]string{{"COUNTERS_DB", tableName, oid}})
		if err != nil {
			log.Errorf("Fetch db failed, pool %s oid %s table %s fetch error: %v -> Bytes=%s", pool, oid, tableName, err, defaultMissingCounterValue)
			result[pool] = BufferPoolStat{Bytes: defaultMissingCounterValue}
			continue
		}
		if len(data) == 0 {
			log.Errorf("Empty hash, pool %s oid %s table %s -> Bytes=%s", pool, oid, tableName, defaultMissingCounterValue)
			result[pool] = BufferPoolStat{Bytes: defaultMissingCounterValue}
			continue
		}

		bytes := defaultMissingCounterValue
		if val, ok := data[fieldName]; ok {
			bytes = fmt.Sprint(val)
		} else {
			log.Errorf("Missing field %s in %s for pool %s oid %s -> Bytes=%s", fieldName, tableName, pool, oid, bytes)
		}
		result[pool] = BufferPoolStat{Bytes: bytes}
		log.V(4).Infof("Gets Pool %s oid %s table %s -> Bytes=%s", pool, oid, tableName, bytes)
	}
	return result
}
