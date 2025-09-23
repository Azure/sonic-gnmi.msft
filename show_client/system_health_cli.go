package show_client

import (
	"encoding/json"
	"net"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

const (
	path           = "/usr/share/sonic/device/"
	configFileName = "system_health_monitoring_config.json"
)

func getPlatformInfo() string {
	versionInfo, err := common.ReadYamlToMap(SonicVersionYamlPath)
	if err != nil {
		log.Errorf("Failed to read version info from %s: %v", SonicVersionYamlPath, err)
		return nil, err
	}
	platformInfo, err := GetPlatformInfo(versionInfo)

	if err != nil {
		log.Errorf("Failed to get Platform. Error:%v", err)
		return ""
	}

	return platformInfo
}

func checkSystemHealthConfig() (string, bool) {
	if platform := getPlatformInfo(); platform != "" {
		fileFullPath := path + platform + configFileName
		return fileFullPath, common.FileExists(fileFullPath)
	}
	return "", false
}

func getSystemHealthSummary(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	if filePath, fileFound := checkSystemHealthConfig(); !fileFound {
		return json.Marshal(map[string]string{"Response": "System health configuration file not found."})
	}

	configs, err := common.ReadJsonToMap(filePath)
	if err != nil {
		log.Errorf("Failed to get System-Health configs:%v", err)
		return json.Marshal(map[string]string{"Response": "Invalid system health configurations."})
	}

	stats, err := helpers.ServiceAndHardwareHealthCheck(configs)

}
