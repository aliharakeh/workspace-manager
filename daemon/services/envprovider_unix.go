//go:build !windows

package services

import "os/exec"

func hideProviderWindow(c *exec.Cmd) {}
