package native

import (
	"bytes"
	"os/exec"
	"runtime"
)

type RunResult struct {
	Stdout string
	Stderr string
	Code   int
}

func Run(cmd []string) (RunResult, error) {
	if len(cmd) == 0 {
		return RunResult{}, nil
	}
	c := exec.Command(cmd[0], cmd[1:]...)
	hideWindow(c)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
			err = nil
		} else {
			return RunResult{Stdout: stdout.String(), Stderr: stderr.String(), Code: 1}, err
		}
	}
	return RunResult{Stdout: stdout.String(), Stderr: stderr.String(), Code: code}, nil
}

func isWindows() bool {
	return runtime.GOOS == "windows"
}
