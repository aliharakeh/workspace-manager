import { useEffect, useState } from "react"
import { BellRingIcon } from "lucide-react"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { useSettings } from "@/components/settings-provider"
import {
  DEFAULT_NOTIFICATIONS,
  NOTIFY_ENABLED_KEY,
  NOTIFY_IDLE_SECONDS_KEY,
  NOTIFY_ON_ERROR_KEY,
  NOTIFY_ON_FINISHED_KEY,
  NOTIFY_ON_IDLE_KEY,
  NOTIFY_ON_READY_KEY,
  settingOn,
} from "@/lib/notifications"

const EVENT_ROWS: { key: string; label: string; description: string }[] = [
  {
    key: NOTIFY_ON_READY_KEY,
    label: "App is ready",
    description: "A ready-URL pattern matched the logs.",
  },
  {
    key: NOTIFY_ON_IDLE_KEY,
    label: "App is running (no URL detected)",
    description: "Output went quiet without a ready URL.",
  },
  {
    key: NOTIFY_ON_FINISHED_KEY,
    label: "App finished",
    description: "All run commands exited successfully.",
  },
  {
    key: NOTIFY_ON_ERROR_KEY,
    label: "App failed",
    description: "A command failed to start or exited non-zero.",
  },
]

export function NotificationsPanel() {
  const { settings, setSetting } = useSettings()
  const enabled = settingOn(settings[NOTIFY_ENABLED_KEY])

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4">
      <div className="flex flex-col gap-1">
        <p className="text-sm font-medium">System notifications</p>
        <p className="text-sm text-muted-foreground">
          Show an OS notification when an app starts, finishes, or fails.
        </p>
      </div>

      <ul className="divide-y overflow-hidden rounded-lg border">
        <li className="flex items-center gap-3 px-3 py-3">
          <Checkbox
            id={NOTIFY_ENABLED_KEY}
            checked={enabled}
            onCheckedChange={(checked) =>
              void setSetting(NOTIFY_ENABLED_KEY, checked === true ? "1" : "0")
            }
          />
          <label
            htmlFor={NOTIFY_ENABLED_KEY}
            className="flex min-w-0 flex-1 items-center gap-2 text-sm font-medium"
          >
            <BellRingIcon className="size-4 shrink-0 text-muted-foreground" />
            Enable notifications
          </label>
        </li>

        {EVENT_ROWS.map((row) => (
          <li key={row.key} className="flex items-center gap-3 px-3 py-3">
            <Checkbox
              id={row.key}
              checked={settingOn(settings[row.key])}
              disabled={!enabled}
              onCheckedChange={(checked) =>
                void setSetting(row.key, checked === true ? "1" : "0")
              }
            />
            <div className="min-w-0 flex-1">
              <label htmlFor={row.key} className="text-sm font-medium">
                {row.label}
              </label>
              <p className="mt-0.5 text-xs text-muted-foreground">
                {row.description}
              </p>
            </div>
          </li>
        ))}

        <li className="flex items-center gap-3 px-3 py-3">
          <div className="min-w-0 flex-1">
            <label
              htmlFor={NOTIFY_IDLE_SECONDS_KEY}
              className="text-sm font-medium"
            >
              Idle delay (seconds)
            </label>
            <p className="mt-0.5 text-xs text-muted-foreground">
              Assume the app is running after this long without new log output.
            </p>
          </div>
          <IdleSecondsInput disabled={!enabled} />
        </li>
      </ul>
    </div>
  )
}

function IdleSecondsInput({ disabled }: { disabled: boolean }) {
  const { settings, setSetting } = useSettings()
  const stored =
    settings[NOTIFY_IDLE_SECONDS_KEY] ??
    DEFAULT_NOTIFICATIONS[NOTIFY_IDLE_SECONDS_KEY]
  const [value, setValue] = useState(stored)

  useEffect(() => {
    setValue(stored)
  }, [stored])

  const commit = () => {
    const n = Math.round(Number(value))
    if (Number.isFinite(n) && n >= 1) {
      void setSetting(NOTIFY_IDLE_SECONDS_KEY, String(Math.min(n, 600)))
    } else {
      setValue(stored)
    }
  }

  return (
    <Input
      id={NOTIFY_IDLE_SECONDS_KEY}
      type="number"
      min={1}
      max={600}
      className="w-24"
      value={value}
      disabled={disabled}
      onChange={(event) => setValue(event.target.value)}
      onBlur={commit}
      aria-label="No-URL idle delay in seconds"
    />
  )
}
