//go:build !windows

package client

import (
	"os/exec"
	"syscall"
)

func exeSuffix() string { return "" }

// detach starts the daemon in its own session, so it keeps running after the
// client exits.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
