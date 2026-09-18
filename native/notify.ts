/**
 * Best-effort OS notifications. No dependencies — shells out per platform:
 * a PowerShell toast on Windows, `osascript` on macOS, `notify-send` on Linux.
 */

import { run } from "./run"

export type NotifyOptions = {
  title: string
  body?: string
}

export function notificationsSupported(): boolean {
  return (
    process.platform === "win32" ||
    process.platform === "darwin" ||
    process.platform === "linux"
  )
}

/** Escape a value for a PowerShell single-quoted literal. */
function psQuote(value: string): string {
  return `'${value.replace(/'/g, "''")}'`
}

/** Escape a value for an AppleScript double-quoted literal. */
function appleQuote(value: string): string {
  return `"${value.replace(/\\/g, "\\\\").replace(/"/g, '\\"')}"`
}

/**
 * The stock Windows PowerShell AUMID. It exists on every Windows install, so
 * toasts work without registering a Start Menu shortcut first.
 */
const WINDOWS_APP_ID =
  "{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\\WindowsPowerShell\\v1.0\\powershell.exe"

function windowsToastScript(title: string, body: string): string {
  return [
    "[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null",
    "$xml = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)",
    "$texts = $xml.GetElementsByTagName('text')",
    `$texts.Item(0).AppendChild($xml.CreateTextNode(${psQuote(title)})) > $null`,
    `$texts.Item(1).AppendChild($xml.CreateTextNode(${psQuote(body)})) > $null`,
    "$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)",
    `[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier(${psQuote(WINDOWS_APP_ID)}).Show($toast)`,
  ].join("; ")
}

/** Fire-and-forget OS notification. Never throws. */
export async function notify({ title, body = "" }: NotifyOptions): Promise<void> {
  if (!title || !notificationsSupported()) return
  try {
    if (process.platform === "win32") {
      await run([
        "powershell",
        "-NoProfile",
        "-NonInteractive",
        "-Command",
        windowsToastScript(title, body),
      ])
    } else if (process.platform === "darwin") {
      await run([
        "osascript",
        "-e",
        `display notification ${appleQuote(body)} with title ${appleQuote(title)}`,
      ])
    } else {
      await run(["notify-send", title, body])
    }
  } catch {
    // Notifications are best-effort; never let them break the runner.
  }
}
