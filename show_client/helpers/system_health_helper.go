package helpers

import (
	"encoding/json"
	"net"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

func serviceHealthCheck(configs map[string]interface) map[string]interface {
}

func hardwareHealthCheck(configs map[string]interface) map[string]interface {
}

func ServiceAndHardwareHealthCheck(configs map[string]interface) map[string]interface {
    servicStats := serviceHealthCheck(configs)
    hardwareStats := hardwareHealthCheck(configs)
    
    stats := common.MergeMaps(serviceStats, hardwareStats)
    return stats
}
