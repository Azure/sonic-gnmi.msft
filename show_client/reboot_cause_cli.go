package show_client

import (
	"encoding/json"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

const PreviousRebootCauseFilePath = "/host/reboot-cause/previous-reboot-cause.json"

func getPreviousRebootCause(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	redact := true
	if redactOption, ok := options["redact"].Bool(); ok {
		redact = redactOption
	}

	if redact {
		msi, err := common.GetMapFromFile(PreviousRebootCauseFilePath)
		if err != nil {
			log.Errorf("Unable to read JSON from file %v, got err: %v", PreviousRebootCauseFilePath, err)
			return nil, err
		}

		redactedMSI, err := common.RedactSensitiveData(msi, []string{"user"})
		if err != nil {
			log.Errorf("Unable to redact data, got err: %v", err)
			return nil, err
		}

		return json.Marshal(redactedMSI)
	}
	return common.GetDataFromFile(PreviousRebootCauseFilePath)
}

func getRebootCauseHistory(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	redact := true
	if redactOption, ok := options["redact"].Bool(); ok {
		redact = redactOption
	}

	queries := [][]string{
		{"STATE_DB", "REBOOT_CAUSE"},
	}

	data, err := common.GetDataFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to get data from queries %v, got err: %v", queries, err)
		return nil, err
	}

	if redact {
		var msi map[string]interface{}
		err = json.Unmarshal(data, &msi)
		if err != nil {
			log.Errorf("Unable to parse JSON, got err: %v", err)
			return nil, err
		}

		// Iterate through each timestamp entry and redact the nested map
		for key, value := range msi {
			if nestedMap, ok := value.(map[string]interface{}); ok {
				redactedNested, err := common.RedactSensitiveData(nestedMap, []string{"user"})
				if err != nil {
					log.Errorf("Unable to redact nested data for key %v, got err: %v", key, err)
					return nil, err
				}
				msi[key] = redactedNested
			}
		}

		return json.Marshal(msi)
	}
	return data, nil
}
