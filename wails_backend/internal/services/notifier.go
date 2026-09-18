package services

import (
	"strconv"
	"strings"
)

// Notification setting keys and kinds.
//
// Settings are opt-out: a missing key means enabled, "0" disables it. Keep
// these in sync with the bun backend
// (bun_backend/server/services/notifier.ts) and the frontend defaults
// (frontend/lib/notifications.ts).
const (
	NotifyKeyEnabled  = "notifications.enabled"
	NotifyKeyReady    = "notifications.on_ready"
	NotifyKeyFinished = "notifications.on_finished"
	NotifyKeyError    = "notifications.on_error"
	NotifyKeyIdle     = "notifications.on_idle"
	NotifyKeyIdleSecs = "notifications.idle_seconds"

	DefaultIdleSeconds = 15
)

const (
	NotifyReady    = "ready"
	NotifyFinished = "finished"
	NotifyError    = "error"
	NotifyIdle     = "idle"
)

func notifyKey(kind string) string {
	switch kind {
	case NotifyReady:
		return NotifyKeyReady
	case NotifyFinished:
		return NotifyKeyFinished
	case NotifyError:
		return NotifyKeyError
	case NotifyIdle:
		return NotifyKeyIdle
	}
	return ""
}

func notifyEnabled(settings map[string]string, kind string) bool {
	if strings.TrimSpace(settings[NotifyKeyEnabled]) == "0" {
		return false
	}
	key := notifyKey(kind)
	return key != "" && strings.TrimSpace(settings[key]) != "0"
}

// idleSeconds is the delay before assuming a URL-less app is running.
func idleSeconds(settings map[string]string) int {
	n, err := strconv.Atoi(strings.TrimSpace(settings[NotifyKeyIdleSecs]))
	if err != nil || n < 1 {
		return DefaultIdleSeconds
	}
	if n > 600 {
		return 600
	}
	return n
}
