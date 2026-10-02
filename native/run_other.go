//go:build !windows

package native

import "os/exec"

func hideWindow(cmd *exec.Cmd) {}
