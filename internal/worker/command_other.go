//go:build !linux && !darwin && !freebsd

package worker

import "os/exec"

func runPlatformCommand(command *exec.Cmd) error { return command.Run() }
