package health_checker

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
)

// Default boot up timeout. When reboot system, system health will wait a few seconds
// before starting to work.
const DefaultBootupTimeout = 300

// System health configuration file name.
const ConfigFile = "system_health_monitoring_config.json"

// Monit service configuration file path.
const MonitConfigFile = "/etc/monit/monitrc"

// Monit service start delay configuration entry.
const MonitStartDelayConfig = "with start delay"

// Device path where platform-specific config files are stored.
const DevicePath = "/usr/share/sonic/device/"

// DefaultLEDConfig is the default LED configuration. Different platform has different LED capability.
// This configuration allows vendor to override the default behavior.
var DefaultLEDConfig = map[string]string{
	"fault":   "red",
	"normal":  "green",
	"booting": "red",
}

// Config manages configuration of system health.
type Config struct {
	platformName        string
	configFile          string
	ConfigData          map[string]interface{}
	IgnoreServices      map[string]struct{}
	IgnoreDevices       map[string]struct{}
	IncludeDevices      map[string]struct{}
	UserDefinedCheckers map[string]struct{}
}

func NewConfig() *Config {
	/* NewConfig creates a new Config instance.
	Initialize all configuration entry to default value in case there is no
	configuration file.*/
	platformName := common.GetPlatform()
	return &Config{
		platformName:        platformName,
		configFile:          filepath.Join(DevicePath, platformName, ConfigFile),
		ConfigData:          nil,
		IgnoreServices:      nil,
		IgnoreDevices:       nil,
		IncludeDevices:      nil,
		UserDefinedCheckers: nil,
	}
}

func (c *Config) ConfigFileExists() bool {
	/* ConfigFileExists checks if the configuration file exists on disk.
	:return: True if configuration file exists.*/
	return common.FileExists(c.configFile)
}

func (c *Config) LoadConfig() {
	/* LoadConfig loads the configuration file from disk.
	1. If there is no configuration file, current config entries will reset to default value.
	2. If there is any format issues in configuration file, current config entries will
	   reset to default value.
	:return:
	Note: Go implementation does not track last_mtime for incremental reload - used for daemon logic.
	The file is re-read on every call.*/
	// the config data, catch the panic and reset to defaults.
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Failed to parse health config from %s: %v", c.configFile, r)
			c.reset()
		}
	}()

	configData, err := common.ReadJsonToMap(c.configFile)
	if err != nil {
		log.Errorf("Failed to read health config from %s: %v", c.configFile, err)
		c.reset()
		return
	}

	c.ConfigData = configData
	c.IgnoreServices = c.getListData(configData, "services_to_ignore")
	c.IgnoreDevices = c.getListData(configData, "devices_to_ignore")
	c.IncludeDevices = c.getListData(configData, "include_devices")
	c.UserDefinedCheckers = c.getListData(configData, "user_defined_checkers")
}

func (c *Config) reset() {
	/* reset resets current configuration entry to default value.
	:return:*/
	c.ConfigData = nil
	c.IgnoreServices = nil
	c.IgnoreDevices = nil
	c.IncludeDevices = nil
	c.UserDefinedCheckers = nil
}

func (c *Config) GetLEDColor(status string) string {
	/* GetLEDColor gets desired LED color according to the input status.
	:param status: System health status.
	:return: String LED color.*/
	if c.ConfigData != nil {
		if ledColorRaw, ok := c.ConfigData["led_color"]; ok {
			if ledColorMap, ok := ledColorRaw.(map[string]interface{}); ok {
				if color, ok := ledColorMap[status].(string); ok {
					return color
				}
			}
		}
	}

	return DefaultLEDConfig[status]
}

func (c *Config) getListData(configData map[string]interface{}, key string) map[string]struct{} {
	/* getListData gets list type configuration data by key and removes duplicate element.
	:param key: Key of the configuration entry.
	:return: A set of configuration data if key exists, nil otherwise.
	Returns nil when the key is missing or the value is not a list,
	matching Python's return of None.*/
	if configData == nil {
		return nil
	}
	val, ok := configData[key]
	if !ok {
		return nil
	}
	rawList, ok := val.([]interface{})
	if !ok {
		return nil
	}
	result := make(map[string]struct{})
	for _, item := range rawList {
		if s, ok := item.(string); ok {
			result[s] = struct{}{}
		}
	}
	return result
}

func (c *Config) GetBootupTimeout() (result int) {
	/* GetBootupTimeout gets boot up timeout from monit configuration file.
	1. If monit configuration file does not exist, return default value.
	2. If there is any exception while parsing monit config, return default value.
	:return: Integer timeout value.*/
	if !common.FileExists(MonitConfigFile) {
		return DefaultBootupTimeout
	}

	defer func() {
		if r := recover(); r != nil {
			result = DefaultBootupTimeout
		}
	}()

	data, err := os.ReadFile(MonitConfigFile)
	if err != nil {
		return DefaultBootupTimeout
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		pos := strings.Index(line, "#")
		if pos == 0 {
			continue
		}
		if pos != -1 {
			line = line[:pos]
		}

		pos = strings.Index(line, MonitStartDelayConfig)
		if pos != -1 {
			valStr := strings.TrimSpace(line[pos+len(MonitStartDelayConfig):])
			if val, err := strconv.Atoi(valStr); err == nil {
				return val
			}
		}
	}

	return DefaultBootupTimeout
}
