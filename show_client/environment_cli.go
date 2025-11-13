package show_client

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// Structs for sensor data
type SensorReading struct {
	Label      string            `json:"label"`
	Value      *string           `json:"value"` // nil if N/A
	Unit       string            `json:"unit"`
	Thresholds map[string]string `json:"thresholds"`
}

type SensorDevice struct {
	Device   string          `json:"device"`
	Adapter  string          `json:"adapter"`
	Readings []SensorReading `json:"readings"`
}

type SensorResponse struct {
	Devices          []SensorDevice `json:"devices"`
	TotalDeviceCount int            `json:"total_devices"`
}

const showEnvCommand = "sudo sensors"

// Main function
func getEnvironment(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	output, err := common.GetDataFromHostCommand(showEnvCommand)
	if err != nil {
		log.Errorf("Unable to successfully execute command %v, get err %v", showEnvCommand, err)
		return nil, err
	}

	output = strings.TrimSpace(output)
	if output == "" {
		log.Errorf("Got empty output for sensors command")
		return nil, fmt.Errorf("Got empty output for sensors command")
	}

	lines := strings.Split(output, "\n")
	var devices []SensorDevice
	var currentDevice *SensorDevice
	i := 0

	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		if isDeviceLine(line) {
			if currentDevice != nil {
				devices = append(devices, *currentDevice)
			}
			currentDevice = &SensorDevice{
				Device:   line,
				Adapter:  "",
				Readings: []SensorReading{},
			}
			i++
			continue
		}

		if isAdapterLine(line) {
			if currentDevice != nil {
				currentDevice.Adapter = parseAdapter(line)
			}
			i++
			continue
		}

		if isReadingLine(line) && currentDevice != nil {
			reading, nextIndex := parseReading(lines, i)
			currentDevice.Readings = append(currentDevice.Readings, reading)
			i = nextIndex
			continue
		}

		i++
	}

	if currentDevice != nil {
		devices = append(devices, *currentDevice)
	}

	response := SensorResponse{
		Devices:          devices,
		TotalDeviceCount: len(devices),
	}

	return json.Marshal(response)
}

// Helper functions

func isDeviceLine(line string) bool {
	return line != "" && !strings.HasPrefix(line, "Adapter:") && !strings.Contains(line, ":")
}

func isAdapterLine(line string) bool {
	return strings.HasPrefix(line, "Adapter:")
}

func parseAdapter(line string) string {
	return strings.TrimPrefix(line, "Adapter: ")
}

func isReadingLine(line string) bool {
	return strings.Contains(line, ":")
}

func parseThresholds(text string) map[string]string {
	thresholds := map[string]string{}
	thresholdRe := regexp.MustCompile(`(\w+\s*\w*)\s*=\s*([-+]?[0-9]*\.?[0-9]+)\s*([A-Za-z]+)`)
	for _, m := range thresholdRe.FindAllStringSubmatch(text, -1) {
		thresholds[strings.TrimSpace(m[1])] = fmt.Sprintf("%s %s", m[2], m[3])
	}
	return thresholds
}

func parseReading(lines []string, startIndex int) (SensorReading, int) {
	line := strings.TrimSpace(lines[startIndex])
	parts := strings.SplitN(line, ":", 2)
	label := strings.TrimSpace(parts[0])
	rest := strings.TrimSpace(parts[1])

	var value *string
	unit := "N/A"
	valueRe := regexp.MustCompile(`([-+]?[0-9]*\.?[0-9]+)\s*([A-Za-z]+|N/A)`)
	if match := valueRe.FindStringSubmatch(rest); match != nil {
		if match[2] != "N/A" {
			val := match[1]
			value = &val
		}
		unit = match[2]
	}

	thresholds := parseThresholds(rest)

	j := startIndex + 1
	for j < len(lines) {
		nextLine := strings.TrimSpace(lines[j])
		if nextLine == "" || (!strings.HasPrefix(nextLine, "Adapter:") && strings.Contains(nextLine, ":")) {
			break
		}
		for k, v := range parseThresholds(nextLine) {
			thresholds[k] = v
		}
		j++
	}

	return SensorReading{
		Label:      label,
		Value:      value,
		Unit:       unit,
		Thresholds: thresholds,
	}, j
}
