package show_client

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

const KDUMP_CONFIG_KEY = "config"
const KDUMP_CONFIG_STATUS_CMD = "/usr/sbin/kdump-config status"
const FIND_KDUMP_CMD = "find /var/crash -name 'kdump.*'"
const FIND_DMESG_CMD = "find /var/crash -name 'dmesg.*'"

func getKdumpConfigData(fieldName string) string {
	queries := [][]string{{"CONFIG_DB", "KDUMP"}}
	rawData, err := common.GetMapFromQueries(queries)

	if err != nil {
		log.Errorf("Unable to get kdump config from queries %v, got err: %v", queries, err)
		return "Unknown"
	}

	return common.GetFieldValueString(rawData, KDUMP_CONFIG_KEY, "Unknown", fieldName)
}

func getKdumpOperMode() string {
	operMode := "Not Ready"
	
	output, err := common.GetDataFromHostCommand(KDUMP_CONFIG_STATUS_CMD)
	if err != nil {
		log.V(2).Infof("Failed to get kdump operational mode: %v", err)
		return operMode
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, ": ready to kdump") {
			operMode = "Ready"
			break
		}
	}

	return operMode
}

// show kdump config
func getKdumpConfig(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {

	adminMode := "Disabled"
	adminEnabled := getKdumpConfigData("enabled")
	if adminEnabled == "true" {
		adminMode = "Enabled"
	}

	operMode := getKdumpOperMode()
	operModeDisplay := operMode
	if adminMode == "Enabled" && operMode == "Not Ready" {
		operModeDisplay = "Ready after reboot"
	}

	configData := map[string]interface{}{
		"administrative_mode": adminMode,
		"operational_mode":    operModeDisplay,
		"memory_reservation":  getKdumpConfigData("memory"),
		"max_dump_files":      getKdumpConfigData("num_dumps"),
	}

	// adding remote SSH config if enabled
	if getKdumpConfigData("remote") == "true" {
		configData["ssh_connection_string"] = getKdumpConfigData("ssh_string")
		configData["ssh_private_key_path"] = getKdumpConfigData("ssh_path")
	} else {
		configData["ssh_connection_string"] = "connection_string not found"
		configData["ssh_private_key_path"] = "ssh_key not found"
	}

	return json.Marshal(configData)
}

func getKdumpCoreFiles() (string, []string) {
	cmdMessage := ""
	dumpFileList := []string{}

	output, err := common.GetDataFromHostCommand(FIND_KDUMP_CMD)
	if err != nil {
		log.Errorf("Failed to get kdump core files: %v", err)
		cmdMessage = "No kernel core dump file available!"
		return cmdMessage, dumpFileList
	}

	if strings.TrimSpace(output) != "" {
		dumpFileList = strings.Split(strings.TrimSpace(output), "\n")
	}

	if len(dumpFileList) == 0 {
		cmdMessage = "No kernel core dump file available!"
	}

	return cmdMessage, dumpFileList
}

func getKdumpDmesgFiles() (string, []string) {
	cmdMessage := ""
	dmesgFileList := []string{}

	output, err := common.GetDataFromHostCommand(FIND_DMESG_CMD)
	if err != nil {
		log.Errorf("Failed to get kdump dmesg files: %v", err)
		cmdMessage = "No kernel dmesg file available!"
		return cmdMessage, dmesgFileList
	}

	if strings.TrimSpace(output) != "" {
		dmesgFileList = strings.Split(strings.TrimSpace(output), "\n")
	}

	if len(dmesgFileList) == 0 {
		cmdMessage = "No kernel dmesg file available!"
	}

	return cmdMessage, dmesgFileList
}

// show kdump files
func getKdumpFiles(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	coreFileResult := []string{}
	dmesgFileResult := []string{}

	cmdMessage, coreFiles := getKdumpCoreFiles()
	if len(coreFiles) == 0 {
		coreFileResult = append(coreFileResult, cmdMessage)
	} else {
		coreFileResult = coreFiles
	}

	cmdMessage, dmesgFiles := getKdumpDmesgFiles()
	if len(dmesgFiles) == 0 {
		dmesgFileResult = append(dmesgFileResult, cmdMessage)
	} else {
		dmesgFileResult = dmesgFiles
	}

	// sorts files as newest first
	sort.Slice(coreFileResult, func(i, j int) bool {
		return coreFileResult[i] > coreFileResult[j]
	})
	sort.Slice(dmesgFileResult, func(i, j int) bool {
		return dmesgFileResult[i] > dmesgFileResult[j]
	})

	fileData := map[string][]string{
		"kernel_core_dump_files": coreFileResult,
		"kernel_dmesg_files":     dmesgFileResult,
	}

	return json.Marshal(fileData)
}

// show kdump logging
func getKdumpLogging(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	var filename string
	lines := "10" // default

	if len(args) > 0 {
		filename = args[0]
	}

	// if lines option is given
	if linesOpt, exists := options["lines"]; exists {
		linesValue, _ := linesOpt.Int()
		lines = strconv.Itoa(linesValue)
	}

	var filePath string
	var cmd string

	if filename != "" {
		// extracting timestamp from filename and construct path
		parts := strings.Split(filename, ".")
		if len(parts) < 2 {
			return nil, fmt.Errorf("Invalid filename: '%s'!", filename)
		}
		timestamp := parts[len(parts)-1]
		filePath = "/var/crash/" + timestamp + "/" + filename
		
		// checking if file exists
		checkCmd := "test -f " + filePath
		_, err := common.GetDataFromHostCommand(checkCmd)
		if err != nil {
			return nil, fmt.Errorf("Invalid filename: '%s'!", filename)
		}
		cmd = "sudo tail -" + lines + " " + filePath
	} else {
		// latest dmesg file
		cmdMessage, dmesgFiles := getKdumpDmesgFiles()
		if len(dmesgFiles) == 0 {
			//no dmesg file available
			logData := map[string]interface{}{
				"logs": []string{cmdMessage},
			}
			return json.Marshal(logData)
		}

		// latest sorting
		sort.Slice(dmesgFiles, func(i, j int) bool {
			return dmesgFiles[i] > dmesgFiles[j]
		})
		filePath = dmesgFiles[0]
		cmd = "sudo tail -" + lines + " " + filePath
	}

	output, err := common.GetDataFromHostCommand(cmd)
	if err != nil {
		log.Errorf("Failed to get kdump logging: %v", err)
		return nil, fmt.Errorf("Failed to read log file: %v", err)
	}

	logLines := strings.Split(strings.TrimSpace(output), "\n")
	if len(logLines) == 1 && logLines[0] == "" {
		logLines = []string{}
	}

	logData := map[string]interface{}{
		"logs":  logLines,
	}

	return json.Marshal(logData)
}
