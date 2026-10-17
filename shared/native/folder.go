package native

import (
	"os/exec"
	"runtime"
)

// OpenFolder shows path in the system file manager.
func OpenFolder(path string) error {
	switch runtime.GOOS {
	case "windows":
		// No hideWindow: it would hide the Explorer window too.
		return exec.Command("explorer", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}
