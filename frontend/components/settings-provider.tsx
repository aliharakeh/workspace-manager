/* eslint-disable react-refresh/only-export-components */
import * as React from "react"
import { toast } from "sonner"
import { api } from "@/lib/api"
import { DEFAULT_SHORTCUTS } from "@/lib/shortcuts"

type SettingsProviderState = {
  /** All settings merged over their defaults. */
  settings: Record<string, string>
  /** True once server settings have been loaded. */
  ready: boolean
  setSetting: (key: string, value: string) => Promise<void>
  /** Restores a setting to its default (removes it from the server). */
  resetSetting: (key: string) => Promise<void>
}

const SettingsContext = React.createContext<SettingsProviderState | undefined>(
  undefined
)

type SettingsProviderProps = {
  children: React.ReactNode
}

export function SettingsProvider({ children }: SettingsProviderProps) {
  const [settings, setSettings] = React.useState<Record<string, string>>(() => ({
    ...DEFAULT_SHORTCUTS,
  }))
  const [ready, setReady] = React.useState(false)

  React.useEffect(() => {
    let cancelled = false
    api.settings
      .get()
      .then((server) => {
        if (cancelled) return
        setSettings((prev) => ({ ...prev, ...server }))
      })
      .catch((err) => {
        if (cancelled) return
        toast.error(err instanceof Error ? err.message : "Failed to load settings")
      })
      .finally(() => {
        if (!cancelled) setReady(true)
      })
    return () => {
      cancelled = true
    }
  }, [])

  const setSetting = React.useCallback(
    async (key: string, value: string) => {
      const previous = settings
      setSettings((prev) => ({ ...prev, [key]: value }))
      try {
        await api.settings.set(key, value)
      } catch (err) {
        setSettings(previous)
        toast.error(err instanceof Error ? err.message : "Failed to save setting")
      }
    },
    [settings]
  )

  const resetSetting = React.useCallback(
    async (key: string) => {
      const fallback = DEFAULT_SHORTCUTS[key]
      if (fallback) await setSetting(key, fallback)
    },
    [setSetting]
  )

  const value = React.useMemo(
    () => ({ settings, ready, setSetting, resetSetting }),
    [settings, ready, setSetting, resetSetting]
  )

  return (
    <SettingsContext.Provider value={value}>{children}</SettingsContext.Provider>
  )
}

export const useSettings = () => {
  const context = React.useContext(SettingsContext)
  if (context === undefined) {
    throw new Error("useSettings must be used within a SettingsProvider")
  }
  return context
}
