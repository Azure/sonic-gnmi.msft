package common

import (
	"encoding/json"
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

func GetMapFromFile(filePath string) (map[string]interface{}, error) {
	jsonBytes, err := GetDataFromFile(filePath)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON from %s: %w", filePath, err)
	}

	return result, nil
}

func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func ReadJsonToMap(filePath string) (map[string]interface{}, error) {
	data, err := GetDataFromFile(filePath)
	if err != nil {
		log.Errorf("Error reading file: %v", err)
		return nil, err
	}

	var parsedData map[string]interface{}
	err = json.Unmarshal(data, &parsedData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON from %s: %w", filePath, err)
	}

	return parsedData, nil
}

func WriteFile(filePath string, content string) bool {
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		log.Errorf("Failed to write %s to file %s - %v", content, filePath, err)
		return false
	}
	return true
}

func ReadStringFromFile(filePath string, defaultVal string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return defaultVal
	}
	return strings.TrimSpace(string(data))
}
