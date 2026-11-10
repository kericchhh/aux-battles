package worker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCommandHelperProcess(t *testing.T) {
	if os.Getenv("AUX_WORKER_COMMAND_HELPER") != "1" {
		return
	}
	index := 0
	for i, arg := range os.Args {
		if arg == "--" {
			index = i + 1
			break
		}
	}
	if index == 0 || index >= len(os.Args) {
		os.Exit(3)
	}
	mode := os.Args[index]
	args := os.Args[index+1:]
	switch mode {
	case "noisy":
		fmt.Fprint(os.Stdout, "HEAD-MARKER")
		os.Stdout.Write(bytes.Repeat([]byte("x"), 1<<20))
		fmt.Fprint(os.Stdout, "TAIL-MARKER")
		os.Exit(7)
	case "block":
		if err := os.WriteFile(args[0], []byte("ready"), 0600); err != nil {
			os.Exit(4)
		}
		time.Sleep(time.Hour)
	case "child":
		time.Sleep(time.Hour)
	case "parent", "leave-child":
		child := exec.Command(os.Args[0], "-test.run=^TestCommandHelperProcess$", "--", "child")
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		if err := child.Start(); err != nil {
			os.Exit(5)
		}
		if err := os.WriteFile(args[0], []byte(strconv.Itoa(child.Process.Pid)), 0600); err != nil {
			child.Process.Kill()
			os.Exit(6)
		}
		if mode == "leave-child" {
			os.Exit(0)
		}
		child.Wait()
	case "success":
		os.Exit(0)
	default:
		os.Exit(2)
	}
	os.Exit(0)
}

func helperArguments(mode string, args ...string) []string {
	return append([]string{"-test.run=^TestCommandHelperProcess$", "--", mode}, args...)
}

func waitForCommandFile(t *testing.T, path string) []byte {
	t.Helper()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
			return data
		}
		select {
		case <-deadline.C:
			t.Fatal("helper process did not become ready")
			return nil
		case <-tick.C:
		}
	}
}

func TestCommandRetainsOnlyOutputTail(t *testing.T) {
	t.Setenv("AUX_WORKER_COMMAND_HELPER", "1")
	err := runCommand(context.Background(), os.Args[0], helperArguments("noisy")...)
	if err == nil {
		t.Fatal("failing helper returned success")
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 7 {
		t.Fatalf("command error=%v", err)
	}
	if strings.Contains(err.Error(), "HEAD-MARKER") || !strings.HasSuffix(err.Error(), "TAIL-MARKER") || len(err.Error()) > commandOutputLimit+200 {
		t.Fatalf("unexpected bounded output: length=%d suffix=%q", len(err.Error()), err.Error()[max(0, len(err.Error())-40):])
	}
	if err := runCommand(context.Background(), os.Args[0], helperArguments("success")...); err != nil {
		t.Fatal(err)
	}
}

func TestCommandCancellationReturnsPromptly(t *testing.T) {
	t.Setenv("AUX_WORKER_COMMAND_HELPER", "1")
	ready := filepath.Join(t.TempDir(), "ready")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- runCommand(ctx, os.Args[0], helperArguments("block", ready)...) }()
	waitForCommandFile(t, ready)
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation error=%v", err)
		}
	case <-time.After(commandWaitDelay + 3*time.Second):
		t.Fatal("command did not stop after cancellation")
	}
}

func TestOutputTailHandlesChunkBoundaries(t *testing.T) {
	for _, chunks := range [][]string{{"hello", " world"}, {strings.Repeat("a", 1999), "b", "c"}, {strings.Repeat("a", 4000), "end"}, {"", strings.Repeat("b", 2000), strings.Repeat("c", 1999)}} {
		var tail outputTail
		var all strings.Builder
		for _, chunk := range chunks {
			n, err := tail.Write([]byte(chunk))
			if n != len(chunk) || err != nil {
				t.Fatalf("write=%d,%v", n, err)
			}
			all.WriteString(chunk)
		}
		want := all.String()
		want = want[max(0, len(want)-commandOutputLimit):]
		if got := string(tail.data[:tail.size]); got != want {
			t.Fatalf("tail length=%d, want %d", len(got), len(want))
		}
	}
}
