/**
 * Keys and defaults for runner lifecycle notifications.
 *
 * The backends are opt-out: a missing key means enabled, "0" disables it.
 * Keep these keys in sync with `bun_backend/server/services/notifier.ts` and
 * `wails_backend/internal/services/notifier.go`.
 */

export const NOTIFY_ENABLED_KEY = "notifications.enabled"
export const NOTIFY_ON_READY_KEY = "notifications.on_ready"
export const NOTIFY_ON_FINISHED_KEY = "notifications.on_finished"
export const NOTIFY_ON_ERROR_KEY = "notifications.on_error"
export const NOTIFY_ON_IDLE_KEY = "notifications.on_idle"
export const NOTIFY_IDLE_SECONDS_KEY = "notifications.idle_seconds"

export const DEFAULT_NOTIFICATIONS: Record<string, string> = {
  [NOTIFY_ENABLED_KEY]: "1",
  [NOTIFY_ON_READY_KEY]: "1",
  [NOTIFY_ON_FINISHED_KEY]: "1",
  [NOTIFY_ON_ERROR_KEY]: "1",
  [NOTIFY_ON_IDLE_KEY]: "1",
  [NOTIFY_IDLE_SECONDS_KEY]: "15",
}

/** Settings store booleans as "1"/"0"; anything but "0" counts as on. */
export function settingOn(value: string | undefined): boolean {
  return value !== "0"
}
