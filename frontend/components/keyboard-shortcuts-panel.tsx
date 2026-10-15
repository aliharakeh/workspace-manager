import { useCallback, useEffect, useState } from "react"
import { KeyboardIcon, RotateCcwIcon } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Kbd } from "@/components/ui/kbd"
import {
  DEFAULT_SHORTCUTS,
  SEARCH_SHORTCUT_KEY,
  THEME_SHORTCUT_KEY,
  eventToShortcut,
  formatShortcut,
  isValidShortcut,
  setShortcutRecorderActive,
  shortcutParts,
} from "@/lib/shortcuts"
import { useSettings } from "@/components/settings-provider"

const SHORTCUT_DEFINITIONS: {
  key: string
  label: string
  description: string
}[] = [
  {
    key: SEARCH_SHORTCUT_KEY,
    label: "Search workspaces and apps",
    description: "Open the global search palette.",
  },
  {
    key: THEME_SHORTCUT_KEY,
    label: "Toggle light/dark theme",
    description: "Switch between the light and dark theme.",
  },
]

export function KeyboardShortcutsPanel() {
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4">
      <div className="flex flex-col gap-1">
        <p className="text-sm font-medium">Keyboard shortcuts</p>
        <p className="text-sm text-muted-foreground">
          Click a shortcut to record a new key combination. Shortcuts must
          include at least one modifier key (Ctrl, Alt, Shift, or ⌘).
        </p>
      </div>

      <ul className="divide-y overflow-hidden rounded-lg border">
        {SHORTCUT_DEFINITIONS.map((definition) => (
          <ShortcutRow key={definition.key} definition={definition} />
        ))}
      </ul>
    </div>
  )
}

function ShortcutRow({
  definition,
}: {
  definition: { key: string; label: string; description: string }
}) {
  const { settings, setSetting, resetSetting } = useSettings()
  const value = settings[definition.key]
  const isDefault = value === DEFAULT_SHORTCUTS[definition.key]

  const handleRecord = useCallback(
    (combo: string) => {
      void setSetting(definition.key, combo)
    },
    [definition.key, setSetting]
  )

  return (
    <li className="flex flex-wrap items-center gap-x-4 gap-y-2 px-3 py-3 sm:flex-nowrap">
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2 text-sm font-medium">
          <KeyboardIcon className="size-4 shrink-0 text-muted-foreground" />
          <span>{definition.label}</span>
        </div>
        <p className="mt-0.5 text-xs text-muted-foreground">
          {definition.description}
        </p>
      </div>
      <div className="flex items-center gap-1.5">
        {!isDefault ? (
          <Button
            variant="ghost"
            size="icon-xs"
            title="Reset to default"
            onClick={() => void resetSetting(definition.key)}
          >
            <RotateCcwIcon />
            <span className="sr-only">Reset to default</span>
          </Button>
        ) : null}
        <ShortcutRecorder
          value={value}
          defaultLabel={formatShortcut(DEFAULT_SHORTCUTS[definition.key])}
          onRecord={handleRecord}
        />
      </div>
    </li>
  )
}

function ShortcutRecorder({
  value,
  defaultLabel,
  onRecord,
}: {
  value: string | undefined
  defaultLabel: string
  onRecord: (combo: string) => void
}) {
  const [recording, setRecording] = useState(false)

  useEffect(() => {
    if (!recording) return
    setShortcutRecorderActive(true)
    const onKeyDown = (event: KeyboardEvent) => {
      event.preventDefault()
      event.stopPropagation()
      if (event.key === "Escape") {
        setRecording(false)
        return
      }
      const combo = eventToShortcut(event)
      if (combo && isValidShortcut(combo)) {
        onRecord(combo)
        setRecording(false)
      }
    }
    window.addEventListener("keydown", onKeyDown, { capture: true })
    return () => {
      window.removeEventListener("keydown", onKeyDown, { capture: true })
      setShortcutRecorderActive(false)
    }
  }, [recording, onRecord])

  if (recording) {
    return (
      <Button
        variant="secondary"
        size="sm"
        className="animate-pulse font-normal"
        onClick={() => setRecording(false)}
        title="Click to cancel, or press Esc"
      >
        Press keys…
      </Button>
    )
  }

  return (
    <Button
      variant="outline"
      size="sm"
      className="gap-1 font-normal"
      onClick={() => setRecording(true)}
      title={`Click to rebind (current: ${formatShortcut(value)})`}
    >
      {value ? (
        shortcutParts(value).map((part) => <Kbd key={part}>{part}</Kbd>)
      ) : (
        <span className="text-muted-foreground">{defaultLabel}</span>
      )}
    </Button>
  )
}
