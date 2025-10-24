package show_client

import (
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

func getBufferInfo(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	verbose := false
	if v, ok := options[common.OptionKeyVerbose].Bool(); ok {
		verbose = v
	}

	return common.GetMmuConfig(common.StateDb, verbose)
}
