package services

import (
	"os/exec"
	"syscall"
)

func hideProviderWindow(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CmdLine: `"` + c.Path + `" /d /s /c "` + c.Args[len(c.Args)-1] + `"`}
}
