//go:build windows

package native

import (
	"os/exec"
	"strings"
	"syscall"
)

// Notify sends a best-effort OS notification. It never returns an error:
// notification failures must not interrupt the runner.
func Notify(title, body string) {
	title = strings.TrimSpace(title)
	if title == "" {
		return
	}
	script := "[reflection.assembly]::LoadWithPartialName('System.Windows.Forms') > $null; " +
		"[reflection.assembly]::LoadWithPartialName('System.Drawing') > $null; " +
		"$n = New-Object System.Windows.Forms.NotifyIcon; $n.Icon = [System.Drawing.SystemIcons]::Information; " +
		"$n.Visible = $true; $n.ShowBalloonTip(10000, $env:WM_NOTIFY_TITLE, $env:WM_NOTIFY_BODY, 'Info'); " +
		"Start-Sleep -Seconds 10; $n.Dispose()"
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", script)
	cmd.Env = append(cmd.Environ(), "WM_NOTIFY_TITLE="+title, "WM_NOTIFY_BODY="+body)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Start()
	go func() { _ = cmd.Wait() }()
}
