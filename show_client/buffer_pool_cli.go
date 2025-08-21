package show_client

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// BufferPoolWatermarkResponse mirrors the CLI table: Pool -> Bytes
// Example output: {"egress_lossless_pool":{"Bytes":"9216"}, ...}

type BufferPoolStat struct {
	Bytes string `json:"Bytes"`
}

// Constants / schema strings
// Runtime schema: COUNTERS_BUFFER_POOL_NAME_MAP is a single hash mapping pool_name -> oid:0x...
// Watermark tables: USER_WATERMARKS / PERSISTENT_WATERMARKS (keys: <table>:oid:<hex>)
const (
	userWatermarkTable       = "USER_WATERMARKS"
	persistentWatermarkTable = "PERSISTENT_WATERMARKS"
	bufferPoolNameMapKey     = "COUNTERS_BUFFER_POOL_NAME_MAP"
	logPrefix                = "[buffer_pool] "
	fieldPrimaryBufferPool   = "SAI_BUFFER_POOL_STAT_WATERMARK_BYTES"
)

// loadBufferPoolNameMap fetches and normalizes the buffer pool name -> oid mapping.
// See UT data testdata/COUNTERS_BUFFER_POOL_NAME_MAP.txt
func loadBufferPoolNameMap() (map[string]string, error) {
	nameMapQueries := [][]string{{"COUNTERS_DB", bufferPoolNameMapKey}}
	nameMap, err := GetMapFromQueries(nameMapQueries)
	if err != nil {
		return nil, fmt.Errorf(logPrefix+"get buffer pool name map %s failed: %w", bufferPoolNameMapKey, err)
	}

	poolToOid := make(map[string]string, len(nameMap))
	for pool, val := range nameMap {
		if s, ok := val.(string); ok && strings.HasPrefix(s, "oid:") {
			poolToOid[pool] = s
			continue
		}
	}
	if len(poolToOid) == 0 {
		return nil, fmt.Errorf(logPrefix+"no buffer pool OIDs extracted from %s", bufferPoolNameMapKey)
	}
	if log.V(4) {
		for k, v := range poolToOid {
			log.Infof(logPrefix+"debug poolToOid %s -> %s", k, v)
		}
	}
	return poolToOid, nil
}

// User watermarks: align with Python 'show buffer_pool watermark' which uses USER_WATERMARKS: prefix
func getBufferPoolWatermark(options sdc.OptionMap) ([]byte, error) {
	return getBufferPoolWatermarkByType(false)
}

// Persistent watermarks: align with Python 'show buffer_pool persistent-watermark'
func getBufferPoolPersistentWatermark(options sdc.OptionMap) ([]byte, error) {
	return getBufferPoolWatermarkByType(true)
}

func getBufferPoolWatermarkByType(persistent bool) ([]byte, error) {
	// 1. Load buffer pool name -> OID map (poolName -> oid:0x...)
	poolToOid, err := loadBufferPoolNameMap()
	if err != nil {
		return nil, err
	}

	// 2. For each buffer pool: build <TABLE_PREFIX><OID> key and get watermark fields value.
	// See UT data testdata/USER_WATERMARKS.txt and testdata/PERSISTENT_WATERMARKS.txt
	tableName := userWatermarkTable
	if persistent {
		tableName = persistentWatermarkTable
	}

	result := make(map[string]BufferPoolStat, len(poolToOid))
	for pool, oid := range poolToOid {
		data, err := GetMapFromQueries([][]string{{"COUNTERS_DB", tableName, oid}})
		if err != nil {
			// Fetch failed (hash missing / Redis error)
			log.Errorf(logPrefix+"Fetch db failed, pool %s oid %s fetch error: %v -> Bytes=%s", pool, oid, err, defaultMissingCounterValue)
			result[pool] = BufferPoolStat{Bytes: defaultMissingCounterValue}
			continue
		}
		if len(data) == 0 {
			// Hash exists but has no fields (counter not yet produced / abnormal)
			log.Errorf(logPrefix+"Empty hash, pool %s oid %s -> Bytes=%s", pool, oid, defaultMissingCounterValue)
			result[pool] = BufferPoolStat{Bytes: defaultMissingCounterValue}
			continue
		}

		bytes := defaultMissingCounterValue
		if val, ok := data[fieldPrimaryBufferPool]; ok {
			bytes = toString(val)
		} else {
			// Expected field missing in COUNTERS_DB watermark table
			log.Errorf(logPrefix+"Missing field %s in %s for pool %s oid %s -> Bytes=%s", fieldPrimaryBufferPool, tableName, pool, oid, bytes)
		}

		// Record the bytes for the pool
		result[pool] = BufferPoolStat{Bytes: bytes}
		log.V(4).Infof(logPrefix+"Gets Pool %s oid %s -> Bytes=%s", pool, oid, bytes)
	}
	return json.Marshal(result)
}
