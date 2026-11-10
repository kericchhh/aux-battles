package worker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func assertCommandChildStopped(t *testing.T, pid int) {
	t.Helper()
	// A killed orphan can remain a zombie until the host's init process reaps it.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		_, state, _ := strings.Cut(string(data), ") ")
		if strings.HasPrefix(state, "Z ") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("child process %d is still running", pid)
}

func TestCommandStopsDescendantsAndBoundsInheritedPipes(t *testing.T) {
	for _, mode := range []string{"parent", "leave-child"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("AUX_WORKER_COMMAND_HELPER", "1")
			pidFile := filepath.Join(t.TempDir(), "pid")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			result := make(chan error, 1)
			go func() { result <- runCommand(ctx, os.Args[0], helperArguments(mode, pidFile)...) }()
			raw := waitForCommandFile(t, pidFile)
			pid, err := strconv.Atoi(string(raw))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
			if mode == "parent" {
				cancel()
			}
			select {
			case err := <-result:
				if mode == "parent" && !errors.Is(err, context.Canceled) {
					t.Fatalf("cancellation=%v", err)
				}
				if mode == "leave-child" && err != nil {
					t.Fatalf("inherited pipe error=%v", err)
				}
			case <-time.After(commandWaitDelay + 4*time.Second):
				t.Fatal("subprocess pipes blocked command completion")
			}
			assertCommandChildStopped(t, pid)
		})
	}
}

func TestCommandExitObservationLeavesStatusWaitable(t *testing.T) {
	t.Setenv("AUX_WORKER_COMMAND_HELPER", "1")
	command := exec.Command(os.Args[0], helperArguments("success")...)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = command.Process.Kill(); _ = command.Wait() })
	if err := waitCommandExit(command.Process.Pid); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err != nil {
		t.Fatalf("exit observation consumed the child's status: %v", err)
	}
}

func TestCommandCancellationAfterWaitDoesNotSignal(t *testing.T) {
	t.Setenv("AUX_WORKER_COMMAND_HELPER", "1")
	for range 5 {
		ctx, cancel := context.WithCancel(context.Background())
		command := exec.CommandContext(ctx, os.Args[0], helperArguments("success")...)
		if err := runPlatformCommand(command); err != nil {
			cancel()
			t.Fatal(err)
		}
		cancel()
		if err := command.Cancel(); !errors.Is(err, os.ErrProcessDone) {
			t.Fatalf("late cancellation=%v, want process done", err)
		}
	}
}

func TestCommandObserverFailureStopsDescendants(t *testing.T) {
	t.Setenv("AUX_WORKER_COMMAND_HELPER", "1")
	pidFile := filepath.Join(t.TempDir(), "pid")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], helperArguments("parent", pidFile)...)
	command.WaitDelay = commandWaitDelay
	var childPID int
	err := runUnixCommand(command, func(int) error {
		raw := waitForCommandFile(t, pidFile)
		var err error
		childPID, err = strconv.Atoi(string(raw))
		if err != nil {
			t.Fatal(err)
		}
		child, err := os.FindProcess(childPID)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = child.Kill(); _ = child.Release() })
		return syscall.EMFILE
	})
	if !errors.Is(err, syscall.EMFILE) {
		t.Fatalf("observer failure=%v, want too many open files", err)
	}
	assertCommandChildStopped(t, childPID)
	if err := command.Cancel(); !errors.Is(err, os.ErrProcessDone) {
		t.Fatalf("late cancellation=%v, want process done", err)
	}
}

func TestCommandObserverLostChildOwnership(t *testing.T) {
	t.Setenv("AUX_WORKER_COMMAND_HELPER", "1")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], helperArguments("success")...)
	err := runUnixCommand(command, func(int) error {
		if _, err := command.Process.Wait(); err != nil {
			t.Fatal(err)
		}
		return syscall.ECHILD
	})
	if !errors.Is(err, syscall.ECHILD) {
		t.Fatalf("observer failure=%v, want lost child ownership", err)
	}
	if err := command.Cancel(); !errors.Is(err, os.ErrProcessDone) {
		t.Fatalf("late cancellation=%v, want process done", err)
	}
}
