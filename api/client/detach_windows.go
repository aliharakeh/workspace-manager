//go:build windows

package client

import (
	"os/exec"
	"syscall"
)

const exeSuffixValue = ".exe"

func exeSuffix() string { return exeSuffixValue }

// detach starts the daemon in its own process group with no console window, so
// it keeps running after the client exits.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x08000000, // DETACHED_PROCESS
		HideWindow:    true,
	}
}
