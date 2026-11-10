//go:build linux || darwin || freebsd

package worker

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

func runPlatformCommand(command *exec.Cmd) error {
	return runUnixCommand(command, waitCommandExit)
}

func runUnixCommand(command *exec.Cmd, observeExit func(int) error) error {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var mu sync.Mutex
	stopped := false
	command.Cancel = func() error {
		mu.Lock()
		defer mu.Unlock()
		if stopped {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	if err := command.Start(); err != nil {
		return err
	}
	// Observe exit without reaping: the leader pins its PID until group cleanup
	// finishes. No group signal may run after Wait releases that identity.
	exitErr := observeExit(command.Process.Pid)
	mu.Lock()
	var stopErr error
	if !errors.Is(exitErr, syscall.ECHILD) {
		stopErr = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		if errors.Is(stopErr, syscall.ESRCH) {
			stopErr = nil
		}
	} else {
		// ECHILD means ownership was lost; a numeric group signal is unsafe.
		stopErr = command.Process.Kill()
		if errors.Is(stopErr, os.ErrProcessDone) {
			stopErr = nil
		}
	}
	stopped = true
	mu.Unlock()
	return errors.Join(command.Wait(), exitErr, stopErr)
}
