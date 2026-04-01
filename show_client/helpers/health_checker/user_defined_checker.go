package helpers

import (
	"fmt"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
)

/*
UserDefinedChecker allows user to implement a script or program to perform
customize check for particular system. In order to enable a user defined checker:
  1. Add an element to "user_defined_checkers" in the configuration file.
     The element must be a command string that can be executed by shell.
     For example: "python my_checker.py".
  2. The command output must match the following pattern:

	${UserDefinedCategory}
	${Object1}:${ObjectStatusMessage1}
	${Object2}:${ObjectStatusMessage2}

An example of the command output:

	MyCategory
	Device1:OK
	Device2:OK
	Device3:Out of power
*/
type UserDefinedChecker struct {
	HealthChecker
	cmd      string
	category string
}

func NewUserDefinedChecker(cmd string) *UserDefinedChecker {
	/* NewUserDefinedChecker creates a new UserDefinedChecker.
	:param cmd: Command string of the user defined checker.*/
	return &UserDefinedChecker{
		HealthChecker: NewHealthChecker(),
		cmd:           cmd,
	}
}

func (udc *UserDefinedChecker) GetCategory() string {
	/* GetCategory returns the category determined from command output. */
	return udc.category
}

func (udc *UserDefinedChecker) String() string {
	/* String returns "UserDefinedChecker - <cmd>". */
	return fmt.Sprintf("UserDefinedChecker - %s", udc.cmd)
}

func (udc *UserDefinedChecker) Check(config *Config) {
	/* Check executes the user defined command and parses the output.
	:param config: Health checker configuration.
	:return:*/
	udc.Reset()
	udc.category = "UserDefine"

	checkerName := fmt.Sprintf("UserDefinedChecker - %s", udc.cmd)

	output, err := common.GetDataFromHostCommand(udc.cmd)
	if err != nil {
		log.Errorf("Failed to run user defined checker command '%s': %v", udc.cmd, err)
		udc.SetObjectNotOK("UserDefine", checkerName,
			fmt.Sprintf("Failed to get output of command \"%s\"", udc.cmd))
		return
	}

	output = strings.TrimSpace(output)
	if output == "" {
		udc.SetObjectNotOK("UserDefine", checkerName,
			fmt.Sprintf("Failed to get output of command \"%s\"", udc.cmd))
		return
	}

	rawLines := strings.Split(output, "\n")
	if len(rawLines) == 0 {
		udc.SetObjectNotOK("UserDefine", checkerName,
			fmt.Sprintf("Invalid output of command \"%s\"", udc.cmd))
		return
	}

	// Filter empty lines
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}

	if len(lines) == 0 {
		udc.SetObjectNotOK("UserDefine", checkerName,
			fmt.Sprintf("Invalid output of command \"%s\"", udc.cmd))
		return
	}

	// First line is the category, remaining lines are object:status pairs
	udc.category = lines[0]

	if len(lines) > 1 {
		for _, line := range lines[1:] {
			pos := strings.Index(line, ":")
			if pos == -1 {
				continue
			}
			objectName := strings.TrimSpace(line[:pos])
			message := strings.TrimSpace(line[pos+1:])

			if message != "OK" {
				udc.SetObjectNotOK("UserDefine", objectName, message)
			} else {
				udc.SetObjectOK("UserDefine", objectName)
			}
		}
	}
}
