package common

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestHostCommandOutputArguments(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "nsenter"), []byte("#!/bin/sh\nprintf '%s\\n' \"$@\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	output, err := GetDataFromHostCommand(`printf "%s" "two words"`)
	if err != nil {
		t.Fatal(err)
	}
	want := "--target\n1\n--pid\n--mount\n--uts\n--ipc\n--net\n--\nprintf\n%s\ntwo words\n"
	if output != want {
		t.Fatalf("output = %q, want %q", output, want)
	}
	for _, command := range []string{"", " ", `printf "unterminated`} {
		if _, err := GetDataFromHostCommand(command); err == nil {
			t.Errorf("GetDataFromHostCommand(%q) succeeded", command)
		}
	}
}

func TestHostCommandOutputResult(t *testing.T) {
	for _, code := range []int{0, 7} {
		t.Run(strconv.Itoa(code), func(t *testing.T) {
			output, err := hostCommandOutput(5*time.Second, 100*time.Millisecond,
				"/bin/sh", "-c", fmt.Sprintf("printf stdout; printf stderr >&2; exit %d", code))
			if output != "stdoutstderr" {
				t.Fatalf("output = %q", output)
			}
			if code == 0 && err != nil {
				t.Fatal(err)
			}
			if code != 0 {
				exitErr, ok := err.(*exec.ExitError)
				if !ok || exitErr.ExitCode() != code {
					t.Fatalf("error = %v, want exit code %d", err, code)
				}
			}
		})
	}
	if _, err := hostCommandOutput(time.Second, time.Millisecond, filepath.Join(t.TempDir(), "missing-command")); err == nil {
		t.Fatal("missing executable succeeded")
	}
}

func TestHostCommandSession(t *testing.T) {
	t.Setenv("SONIC_HOST_COMMAND_HELPER", "1")
	cmd := hostHelperCommand("session", t.TempDir())
	output, err := hostCommandOutput(5*time.Second, time.Millisecond, cmd.Path, cmd.Args[1:]...)
	if err != nil || strings.TrimSpace(output) != "new session; no controlling terminal" {
		t.Fatalf("output = %q, error = %v", output, err)
	}
}

func TestHostCommandTimeoutAndPipes(t *testing.T) {
	t.Setenv("SONIC_HOST_COMMAND_HELPER", "1")
	// Adopt only these test workers so the test can reap them, rather than
	// leaving zombies for the test environment's PID 1.
	if err := unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	defer unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 0, 0, 0, 0)

	for _, tc := range []struct {
		mode     string
		deadline bool
		nsenter  bool
	}{
		{mode: "early-exit"},
		{mode: "timeout", deadline: true},
		{mode: "timeout", deadline: true, nsenter: true},
	} {
		name := tc.mode
		if tc.nsenter {
			name = "nsenter-" + name
		}
		t.Run(name, func(t *testing.T) {
			if tc.nsenter && os.Getenv("SONIC_TEST_NSENTER") != "1" {
				t.Skip("set SONIC_TEST_NSENTER=1 to test real namespace entry (requires CAP_SYS_ADMIN)")
			}
			dir := t.TempDir()
			workerPID := 0
			t.Cleanup(func() {
				if workerPID == 0 {
					data, _ := os.ReadFile(filepath.Join(dir, "worker.pid"))
					workerPID, _ = strconv.Atoi(string(data))
				}
				if workerPID > 0 {
					// The adopted worker is not reaped until this cleanup.
					_ = syscall.Kill(workerPID, syscall.SIGKILL)
					var status syscall.WaitStatus
					_, _ = syscall.Wait4(workerPID, &status, 0, nil)
				}
			})
			timeout := 5 * time.Second
			if tc.deadline {
				timeout = 2 * time.Second
			}
			const pipeWait = 100 * time.Millisecond
			cmd := hostHelperCommand(tc.mode, dir)
			if tc.nsenter {
				args := []string{
					"--target", strconv.Itoa(os.Getpid()),
					"--pid", "--mount", "--uts", "--ipc", "--net", "--",
				}
				namespaced := exec.Command("nsenter", append(args, cmd.Args...)...)
				namespaced.Env = cmd.Env
				cmd = namespaced
			}
			start := time.Now()
			_, err := hostCommandOutput(timeout, pipeWait, cmd.Path, cmd.Args[1:]...)
			elapsed := time.Since(start)
			data, readErr := os.ReadFile(filepath.Join(dir, "worker.pid"))
			if readErr != nil {
				t.Fatalf("worker not started: %v (command error: %v)", readErr, err)
			}
			workerPID, readErr = strconv.Atoi(string(data))
			if readErr != nil || workerPID <= 0 {
				t.Fatalf("invalid worker PID: %q", data)
			}
			if !tc.deadline {
				if !errors.Is(err, exec.ErrWaitDelay) || elapsed < pipeWait {
					t.Fatalf("error = %v, elapsed = %s; want bounded pipe wait", err, elapsed)
				}
				if elapsed >= timeout {
					t.Fatalf("output waiting reached the execution timeout: %s", elapsed)
				}
				// The sample intentionally has no early-exit group cleanup.
				var info unix.Siginfo
				if err := unix.Waitid(unix.P_PID, workerPID, &info, unix.WEXITED|unix.WNOHANG|unix.WNOWAIT, nil); err != nil || info.Signo != 0 {
					t.Fatalf("worker should remain alive until test cleanup: %v", err)
				}
				return
			}
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.Sys().(syscall.WaitStatus).Signal() != syscall.SIGKILL {
				t.Fatalf("error = %v, want SIGKILL on timeout", err)
			}
			if elapsed < timeout || elapsed >= timeout+5*time.Second {
				t.Fatalf("unexpected timeout duration: %s", elapsed)
			}
			// WNOWAIT verifies death while retaining ownership for t.Cleanup.
			deadline := time.Now().Add(5 * time.Second)
			for {
				var info unix.Siginfo
				err := unix.Waitid(unix.P_PID, workerPID, &info, unix.WEXITED|unix.WNOHANG|unix.WNOWAIT, nil)
				// nsenter may be reaped before its dying descendants are adopted.
				if err != nil && err != unix.ECHILD {
					t.Fatalf("observe worker: %v", err)
				}
				if info.Signo != 0 {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("worker survived group cleanup")
				}
				time.Sleep(10 * time.Millisecond)
			}
		})
	}
}

func TestHostCommandTimeoutIsolated(t *testing.T) {
	t.Setenv("SONIC_HOST_COMMAND_HELPER", "1")
	dir := t.TempDir()
	type result struct {
		output string
		err    error
	}
	done := make(chan result, 1)
	go func() {
		cmd := hostHelperCommand("peer", dir)
		output, err := hostCommandOutput(10*time.Second, time.Millisecond, cmd.Path, cmd.Args[1:]...)
		done <- result{output, err}
	}()
	received := false
	t.Cleanup(func() {
		_ = os.WriteFile(filepath.Join(dir, "finish"), nil, 0600)
		if !received {
			<-done
		}
	})
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(dir, "ready")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("peer command did not start")
		}
		time.Sleep(time.Millisecond)
	}
	cmd := hostHelperCommand("block", dir)
	_, err := hostCommandOutput(100*time.Millisecond, time.Millisecond, cmd.Path, cmd.Args[1:]...)
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.Sys().(syscall.WaitStatus).Signal() != syscall.SIGKILL {
		t.Fatalf("error = %v, want SIGKILL on timeout", err)
	}
	select {
	case result := <-done:
		received = true
		t.Fatalf("cleanup stopped the other invocation: %+v", result)
	default:
	}
	if err := os.WriteFile(filepath.Join(dir, "finish"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	got := <-done
	received = true
	if got.err != nil || got.output != "peer survived\n" {
		t.Fatalf("peer result = %+v", got)
	}
}

func hostHelperCommand(mode, dir string) *exec.Cmd {
	cmd := exec.Command(os.Args[0], "-test.run=^TestHostCommandHelperProcess$", "--", mode, dir)
	cmd.Env = append(os.Environ(), "SONIC_HOST_COMMAND_HELPER=1")
	return cmd
}

func TestHostCommandHelperProcess(t *testing.T) {
	if os.Getenv("SONIC_HOST_COMMAND_HELPER") != "1" {
		return
	}
	mode, dir := os.Args[len(os.Args)-2], os.Args[len(os.Args)-1]
	if mode == "block" {
		signal.Ignore(syscall.SIGTERM)
		time.Sleep(time.Hour)
		os.Exit(0)
	}
	if mode == "peer" {
		if err := os.WriteFile(filepath.Join(dir, "ready"), nil, 0600); err != nil {
			os.Exit(10)
		}
		for {
			if _, err := os.Stat(filepath.Join(dir, "finish")); err == nil {
				fmt.Println("peer survived")
				os.Exit(0)
			}
			time.Sleep(time.Millisecond)
		}
	}
	if mode == "session" {
		sid, err := unix.Getsid(0)
		if err != nil || sid != os.Getpid() || syscall.Getpgrp() != os.Getpid() {
			os.Exit(2)
		}
		if terminal, err := os.Open("/dev/tty"); err == nil {
			terminal.Close()
			os.Exit(3)
		}
		fmt.Println("new session; no controlling terminal")
		os.Exit(0)
	}
	if mode == "worker" {
		signal.Ignore(syscall.SIGTERM)
		if err := os.WriteFile(filepath.Join(dir, "ready"), nil, 0600); err != nil {
			os.Exit(4)
		}
		time.Sleep(time.Hour)
		os.Exit(0)
	}
	signal.Ignore(syscall.SIGTERM)
	worker := hostHelperCommand("worker", dir)
	worker.Stdout, worker.Stderr = os.Stdout, os.Stderr
	if err := worker.Start(); err != nil {
		os.Exit(5)
	}
	if err := os.WriteFile(filepath.Join(dir, "worker.pid"), []byte(strconv.Itoa(worker.Process.Pid)), 0600); err != nil {
		os.Exit(6)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(dir, "ready")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			os.Exit(8)
		}
		time.Sleep(time.Millisecond)
	}
	if mode == "timeout" {
		time.Sleep(time.Hour)
	}
	os.Exit(0)
}
