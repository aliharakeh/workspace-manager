import { useCallback, useEffect, useState } from "react"
import { api, onDataChanged } from "@/lib/api"
import type { RunConfig } from "@/lib/types"

const selectionEvent = "runConfigurationSelected"

export function useRunConfigs(appId: number, configSetId = 0) {
  const storageKey = `run-config:${appId}`
  const context = `${appId}:${configSetId}`
  const [outcome, setOutcome] = useState<{
    context: string
    configs: RunConfig[]
    error: string | null
  } | null>(null)
  const [selectedKey, setSelectedKey] = useState("")

  useEffect(() => {
    let cancelled = false
    let request = 0
    const load = async () => {
      const current = ++request
      try {
        const next = await api.runConfig.list(appId, configSetId)
        if (!cancelled && current === request) {
          setOutcome({ context, configs: next, error: null })
          setSelectedKey(localStorage.getItem(storageKey) ?? "")
        }
      } catch (e) {
        if (!cancelled && current === request)
          setOutcome({
            context,
            configs: [],
            error:
              e instanceof Error ? e.message : "Failed to load configurations",
          })
      }
    }
    void load()
    const off = onDataChanged((event) => {
      if (
        ["runConfig", "configSet", "app"].includes(event.entity) &&
        (event.ids ?? []).includes(appId)
      )
        void load()
    })
    const sync = () => setSelectedKey(localStorage.getItem(storageKey) ?? "")
    window.addEventListener(selectionEvent, sync)
    return () => {
      cancelled = true
      off()
      window.removeEventListener(selectionEvent, sync)
    }
  }, [appId, configSetId, storageKey, context])

  const select = useCallback(
    (key: string) => {
      localStorage.setItem(storageKey, key)
      setSelectedKey(key)
      window.dispatchEvent(new Event(selectionEvent))
    },
    [storageKey]
  )

  const fresh = outcome?.context === context ? outcome : null
  const configs = fresh?.configs ?? []
  const selected =
    configs.find((c) => c.kind === selectedKey) ??
    configs.find((c) => c.kind === "dev") ??
    configs[0]
  return {
    configs,
    selected,
    select,
    loading: fresh === null,
    error: fresh?.error ?? null,
  }
}
