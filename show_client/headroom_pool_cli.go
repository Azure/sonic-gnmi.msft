package show_client

import (
	"encoding/json"
	"strings"

	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// https://github.com/Azure/sonic-utilities.msft/blob/3cb0eb2402a8da806b7c858eaa7e6be950c92fe3/scripts/watermarkstat#L212C44-L212C90
const (
	fieldHeadroomPool       = "SAI_BUFFER_POOL_STAT_XOFF_ROOM_WATERMARK_BYTES"
	ingressLosslessPoolName = "ingress_lossless"
)

func getHeadroomPoolWatermark(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	return getHeadroomPoolWatermarkByType(false)
}

func getHeadroomPoolPersistentWatermark(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	return getHeadroomPoolWatermarkByType(true)
}

// https://github.com/Azure/sonic-utilities.msft/blob/3cb0eb2402a8da806b7c858eaa7e6be950c92fe3/scripts/watermarkstat#L290
func getHeadroomPoolWatermarkByType(persistent bool) ([]byte, error) {
	// 1. Load buffer pool name -> OID map (poolName -> oid:0x...)
	poolToOid, err := loadBufferPoolNameMap()
	if err != nil {
		return nil, err
	}

	// 2. Filter to ALL ingress lossless pools
	// https://github.com/Azure/sonic-utilities.msft/blob/3cb0eb2402a8da806b7c858eaa7e6be950c92fe3/scripts/watermarkstat#L293-L302
	filtered := make(map[string]string)
	for pool, oid := range poolToOid {
		if strings.Contains(pool, ingressLosslessPoolName) {
			filtered[pool] = oid
		}
	}

	tableName := userWatermarkTable
	if persistent {
		tableName = persistentWatermarkTable
	}

	result := collectBufferPoolWatermarks(filtered, tableName, fieldHeadroomPool)
	return json.Marshal(result)
}
