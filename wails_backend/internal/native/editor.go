package native

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func OpenInEditor(path string) error {
	if ed := strings.TrimSpace(os.Getenv("VISUAL")); ed != "" {
		return startEditor(ed, path)
	}
	if ed := strings.TrimSpace(os.Getenv("EDITOR")); ed != "" {
		return startEditor(ed, path)
	}
	switch runtime.GOOS {
	case "windows":
		c := exec.Command("explorer", path)
		hideWindow(c)
		return c.Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

func startEditor(editor, path string) error {
	parts := strings.Fields(editor)
	args := append(parts[1:], path)
	c := exec.Command(parts[0], args...)
	hideWindow(c)
	return c.Start()
}
