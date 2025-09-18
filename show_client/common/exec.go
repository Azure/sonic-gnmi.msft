package common

import (
	"os/exec"

	"github.com/google/shlex"
)

const (
	hostNamespace = "1" // PID 1 is the host init process
)

func GetDataFromHostCommand(command string) (string, error) {
	baseArgs := []string{
		"--target", hostNamespace,
		"--pid", "--mount", "--uts", "--ipc", "--net",
		"--",
	}
	commandParts, err := shlex.Split(command)
	if err != nil {
		return "", err
	}
	cmdArgs := append(baseArgs, commandParts...)
	cmd := exec.Command("nsenter", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
