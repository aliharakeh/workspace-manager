//go:build !windows

package native

import (
	"os"
	"syscall"
)

// PidAlive reports whether a process with this id is still running.
func PidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = p.Signal(syscall.Signal(0))
	return err == nil || err == syscall.EPERM
}
