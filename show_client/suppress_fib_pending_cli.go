package show_client

import (
	"encoding/json"
	"strings"

	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// show suppress-fib-pending
func getSuppressFibPending(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	state := common.GetLocalhostInfo("suppress-fib-pending")
	if state == "" {
		state = "enabled"
	}

	result := map[string]string{
		"status": strings.Title(state),
	}

	return json.Marshal(result)
}
