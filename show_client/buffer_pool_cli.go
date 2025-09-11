package show_client

import (
	"encoding/json"

	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// https://github.com/Azure/sonic-utilities.msft/blob/3cb0eb2402a8da806b7c858eaa7e6be950c92fe3/scripts/watermarkstat#L209C1-L210C1
const fieldBufferPool = "SAI_BUFFER_POOL_STAT_WATERMARK_BYTES"

// User watermarks: align with Python 'show buffer_pool watermark' which uses USER_WATERMARKS: prefix
func getBufferPoolWatermark(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	return getBufferPoolWatermarkByType(false)
}

// Persistent watermarks: align with Python 'show buffer_pool persistent-watermark'
func getBufferPoolPersistentWatermark(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	return getBufferPoolWatermarkByType(true)
}

// https://github.com/Azure/sonic-utilities.msft/blob/3cb0eb2402a8da806b7c858eaa7e6be950c92fe3/scripts/watermarkstat#L290
func getBufferPoolWatermarkByType(persistent bool) ([]byte, error) {
	// 1. Load buffer pool name -> OID map (poolName -> oid:0x...)
	poolToOid, err := loadBufferPoolNameMap()
	if err != nil {
		return nil, err
	}

	tableName := userWatermarkTable
	if persistent {
		tableName = persistentWatermarkTable
	}

	// 2. Collect buffer pool watermarks
	result := collectBufferPoolWatermarks(poolToOid, tableName, fieldBufferPool)
	return json.Marshal(result)
}
