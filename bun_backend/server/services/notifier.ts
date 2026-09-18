/**
 * Runner lifecycle notifications.
 *
 * Settings are stored in SQLite (Settings → Notifications). Everything is
 * opt-out: a missing key means enabled, "0" disables it. The same keys are
 * read by the Wails runner — keep them in sync with
 * `wails_backend/internal/services/notifier.go` and
 * `frontend/lib/notifications.ts`.
 */

import { settingsRepo } from "@db/settings"
import { notify } from "@native/notify"

export type NotificationKind = "ready" | "finished" | "error" | "idle"

export const NOTIFY_ENABLED_KEY = "notifications.enabled"
export const NOTIFY_IDLE_SECONDS_KEY = "notifications.idle_seconds"
export const DEFAULT_IDLE_SECONDS = 15

const KIND_KEYS: Record<NotificationKind, string> = {
  ready: "notifications.on_ready",
  finished: "notifications.on_finished",
  error: "notifications.on_error",
  idle: "notifications.on_idle",
}

function isEnabled(kind: NotificationKind): boolean {
  if (settingsRepo.get(NOTIFY_ENABLED_KEY) === "0") return false
  return settingsRepo.get(KIND_KEYS[kind]) !== "0"
}

/** Seconds to wait for a ready URL before assuming the app is running. */
export function idleSeconds(): number {
  const raw = Number(settingsRepo.get(NOTIFY_IDLE_SECONDS_KEY))
  if (!Number.isFinite(raw) || raw < 1) return DEFAULT_IDLE_SECONDS
  return Math.min(Math.round(raw), 600)
}

export function notifyApp(
  kind: NotificationKind,
  title: string,
  body?: string
): void {
  if (!isEnabled(kind)) return
  void notify({ title, body })
}
