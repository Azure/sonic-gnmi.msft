//go:build linux

package common

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/google/shlex"
)

const (
	hostCommandTimeout   = 360 * time.Second
	hostCommandWaitDelay = 2 * time.Second
)

// GetDataFromHostCommand runs a SHOW command through nsenter with a server-side limit.
// On timeout it kills the launcher's group, not work delegated to another
// service (such as docker exec) or processes that create their own group.
// The limit applies to each command, not the entire SHOW RPC.
func GetDataFromHostCommand(command string) (string, error) {
	commandParts, err := shlex.Split(command)
	if err != nil {
		return "", err
	}
	if len(commandParts) == 0 {
		return "", errors.New("empty host command")
	}
	args := []string{
		"--target", "1",
		"--pid", "--mount", "--uts", "--ipc", "--net",
		"--",
	}
	return hostCommandOutput(hostCommandTimeout, hostCommandWaitDelay, "nsenter", append(args, commandParts...)...)
}

func hostCommandOutput(timeout, waitDelay time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	// A new session also creates a process group and drops the inherited
	// controlling terminal. It does not change cgroup membership.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	// CommandContext normally kills only the launcher; cancel its group instead.
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	// This bounds output waiting, not descendant lifetime after early launcher exit.
	cmd.WaitDelay = waitDelay
	output, err := cmd.CombinedOutput()
	return string(output), err
}
