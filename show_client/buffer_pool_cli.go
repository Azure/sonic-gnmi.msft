package show_client

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/golang/glog"
	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
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

	// Primary field used by Python implementation
	fieldPrimaryBufferPool = "SAI_BUFFER_POOL_STAT_WATERMARK_BYTES"
	// Fallback field names sometimes present in older / different schemas
	fieldFallbackShared1 = "SAI_BUFFER_POOL_STAT_SHARED_WATERMARK_BYTES"
	fieldFallbackShared2 = "SHARED_WATERMARK_BYTES"
	fieldFallbackShared3 = "WATERMARK_BYTES"
)

// Ordered candidate fields for buffer pool watermarks (primary first, then fallbacks)
var bufferPoolFieldsOrdered = []string{fieldPrimaryBufferPool, fieldFallbackShared1, fieldFallbackShared2, fieldFallbackShared3}

// loadBufferPoolNameMap fetches and normalizes the buffer pool name -> oid mapping.
// It returns a flat map[poolName]oid (e.g. egress_lossless_pool -> oid:0x...).
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
			log.Errorf(logPrefix+"debug poolToOid %s -> %s", k, v)
		}
	}
	return poolToOid, nil
}

// User watermarks: align with Python 'show buffer_pool watermark' which uses USER_WATERMARKS: prefix
func getBufferPoolWatermark(prefix, path *gnmipb.Path) ([]byte, error) {
	return getBufferPoolWatermarkByType(prefix, path, false)
}

// Persistent watermarks: align with Python 'show buffer_pool persistent-watermark'
func getBufferPoolPersistentWatermark(prefix, path *gnmipb.Path) ([]byte, error) {
	return getBufferPoolWatermarkByType(prefix, path, true)
}

func getBufferPoolWatermarkByType(prefix, path *gnmipb.Path, persistent bool) ([]byte, error) {
	// 1. Load buffer pool name -> OID map (poolName -> oid:0x...)
	poolToOid, err := loadBufferPoolNameMap()
	if err != nil {
		return nil, err
	}

	// 2. For each buffer pool: build <TABLE_PREFIX><OID> key and read watermark fields.
	tableName := userWatermarkTable
	if persistent {
		tableName = persistentWatermarkTable
	}

	result := make(map[string]BufferPoolStat, len(poolToOid))
	for pool, oid := range poolToOid {
		data, err2 := GetMapFromQueries([][]string{{"COUNTERS_DB", tableName, oid}})
		if err2 != nil {
			// Fetch failed (hash missing / Redis error)
			log.Errorf(logPrefix+"Fetch db failed, pool %s oid %s fetch error: %v -> Bytes=%s", pool, oid, err2, defaultMissingCounterValue)
			result[pool] = BufferPoolStat{Bytes: defaultMissingCounterValue}
			continue
		}
		if len(data) == 0 {
			// Hash exists but has no fields (counter not yet produced / abnormal)
			log.Errorf(logPrefix+"Empty hash, pool %s oid %s -> Bytes=%s", pool, oid, defaultMissingCounterValue)
			result[pool] = BufferPoolStat{Bytes: defaultMissingCounterValue}
			continue
		}

		// Select first available field in priority order (primary then fallbacks)
		bytes := defaultMissingCounterValue
		for _, f := range bufferPoolFieldsOrdered {
			if val, ok := data[f]; ok {
				bytes = toString(val)
				break
			}
		}

		// Record the bytes for the pool
		result[pool] = BufferPoolStat{Bytes: bytes}
	}
	return json.Marshal(result)
}

func toString(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	default:
		return fmt.Sprint(v)
	}
}
