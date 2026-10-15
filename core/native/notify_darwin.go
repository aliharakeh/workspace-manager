//go:build darwin

package native

import (
	"os/exec"
	"strings"
)

// Notify sends a best-effort OS notification. It never returns an error:
// notification failures must not interrupt the runner.
func Notify(title, body string) {
	title = strings.TrimSpace(title)
	if title == "" {
		return
	}
	script := `display notification ` + quoteAppleScript(body) + ` with title ` + quoteAppleScript(title)
	_ = exec.Command("osascript", "-e", script).Start()
}

func quoteAppleScript(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
