package show_client

import (
	"encoding/json"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

const PreviousRebootCauseFilePath = "/host/reboot-cause/previous-reboot-cause.json"
const RedactedUserString = "$USER$"
const RedactField = "user"

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

		return json.Marshal(common.RedactSensitiveData(msi, []string{RedactField}, RedactedUserString))
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

	if redact {
		msi, err := common.GetMapFromQueries(queries)
		if err != nil {
			log.Errorf("Unable to get data from queries %v, got err: %v", queries, err)
			return nil, err
		}
		// Redact user in each reboot cause entry
		for key, value := range msi {
			if nestedMap, ok := value.(map[string]interface{}); ok {
				msi[key] = common.RedactSensitiveData(nestedMap, []string{RedactField}, RedactedUserString)
			}
		}
		return json.Marshal(msi)
	}
	return common.GetDataFromQueries(queries)
}
