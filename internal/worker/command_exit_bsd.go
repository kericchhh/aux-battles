//go:build darwin || freebsd

package worker

import (
	"errors"
	"syscall"
)

func waitCommandExit(pid int) error {
	queue, err := syscall.Kqueue()
	if err != nil {
		return err
	}
	defer syscall.Close(queue)
	syscall.CloseOnExec(queue)
	var change syscall.Kevent_t
	syscall.SetKevent(&change, pid, syscall.EVFILT_PROC, syscall.EV_ADD|syscall.EV_ONESHOT)
	change.Fflags = syscall.NOTE_EXIT
	for {
		_, err = syscall.Kevent(queue, []syscall.Kevent_t{change}, nil, nil)
		if !errors.Is(err, syscall.EINTR) {
			break
		}
	}
	if errors.Is(err, syscall.ESRCH) {
		// An already exited child remains unreaped until command.Wait.
		return nil
	}
	if err != nil {
		return err
	}
	var events [1]syscall.Kevent_t
	for {
		_, err = syscall.Kevent(queue, nil, events[:], nil)
		if !errors.Is(err, syscall.EINTR) {
			return err
		}
	}
}
