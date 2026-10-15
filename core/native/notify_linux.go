//go:build linux

package native

import (
	"os/exec"
	"strings"
)

// Notify sends a best-effort OS notification through notify-send. It never
// returns an error: notification failures must not interrupt the runner.
func Notify(title, body string) {
	title = strings.TrimSpace(title)
	if title == "" {
		return
	}
	_ = exec.Command("notify-send", title, body).Start()
}
