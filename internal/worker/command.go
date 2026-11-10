package worker

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	commandOutputLimit = 2000
	commandWaitDelay   = 2 * time.Second
)

type outputTail struct {
	data [commandOutputLimit]byte
	size int
}

func (b *outputTail) Write(p []byte) (int, error) {
	n := len(p)
	if n >= len(b.data) {
		copy(b.data[:], p[n-len(b.data):])
		b.size = len(b.data)
		return n, nil
	}
	if excess := b.size + n - len(b.data); excess > 0 {
		copy(b.data[:], b.data[excess:b.size])
		b.size -= excess
	}
	copy(b.data[b.size:], p)
	b.size += n
	return n, nil
}

func runCommand(ctx context.Context, name string, args ...string) error {
	command := exec.CommandContext(ctx, name, args...)
	var output outputTail
	// A shared writer makes os/exec serialize stdout and stderr writes.
	command.Stdout = &output
	command.Stderr = &output
	command.WaitDelay = commandWaitDelay
	err := runPlatformCommand(command)
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		err = errors.Join(ctx.Err(), err)
	}
	detail := strings.TrimSpace(string(output.data[:output.size]))
	if detail == "" {
		return fmt.Errorf("%s: %w", filepath.Base(name), err)
	}
	return fmt.Errorf("%s: %w: %s", filepath.Base(name), err, detail)
}
