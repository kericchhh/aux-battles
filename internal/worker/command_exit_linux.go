package worker

import "syscall"

func waitCommandExit(pid int) error {
	for {
		// Linux permits a nil siginfo pointer. WNOWAIT leaves the child
		// waitable so os/exec can collect its status after group cleanup.
		_, _, err := syscall.Syscall6(syscall.SYS_WAITID, 1, uintptr(pid), 0, syscall.WEXITED|syscall.WNOWAIT, 0, 0)
		if err == syscall.EINTR {
			continue
		}
		if err != 0 {
			return err
		}
		return nil
	}
}
