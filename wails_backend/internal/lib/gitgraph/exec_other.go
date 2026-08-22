//go:build !windows

package gitgraph

import "os/exec"

func hideWindow(cmd *exec.Cmd) {}
