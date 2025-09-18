package common

import (
	"fmt"
	"os"
	"strings"

	log "github.com/golang/glog"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
	"gopkg.in/yaml.v2"
)

func GetDataFromFile(fileName string) ([]byte, error) {
	fileContent, err := sdc.ImplIoutilReadFile(fileName)
	if err != nil {
		log.Errorf("Failed to read'%v', %v", fileName, err)
		return nil, err
	}
	log.V(4).Infof("getDataFromFile, output: %v", string(fileContent))
	return fileContent, nil
}

func ReadYamlToMap(filePath string) (map[string]interface{}, error) {
	yamlFile, err := sdc.ImplIoutilReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file: %w", err)
	}
	var data map[string]interface{}
	err = yaml.Unmarshal(yamlFile, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}
	return data, nil
}

func ReadConfToMap(filePath string) (map[string]interface{}, error) {
	dataBytes, err := sdc.ImplIoutilReadFile(filePath)

	if err != nil {
		return nil, fmt.Errorf("failed to read CONF: %w", err)
	}

	confData := make(map[string]interface{})

	content := string(dataBytes)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			confData[key] = value
		}
	}

	return confData, nil
}

func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
