import { useCallback, useEffect, useRef, useState } from "react"
import { api } from "@/lib/api"
import { onRunnerEvent } from "@/lib/electrobun"
import type { StatusEvent } from "@/lib/types"

export function useAppStatuses(workspaceIds: number[]) {
  const [statusByAppId, setStatusByAppId] = useState<
    Record<number, StatusEvent>
  >({})
  const idsKey = workspaceIds.slice().sort((a, b) => a - b).join(",")
  const idsRef = useRef(workspaceIds)
  idsRef.current = workspaceIds

  const refresh = useCallback(async () => {
    const ids = idsRef.current
    if (ids.length === 0) {
      setStatusByAppId({})
      return
    }
    try {
      const results = await Promise.all(
        ids.map((id) => api.runner.workspaceStatus(id))
      )
      setStatusByAppId(
        Object.fromEntries(
          results.flat().map((status) => [status.appId, status])
        )
      )
    } catch {
      // keep last known status on transient failures
    }
  }, [])

  const setAppStatus = useCallback((status: StatusEvent) => {
    setStatusByAppId((prev) => ({ ...prev, [status.appId]: status }))
  }, [])

  useEffect(() => {
    void refresh()
    const unsubscribe = onRunnerEvent((_appId, event) => {
      if (event.type !== "status") return
      setAppStatus(event)
    })
    return unsubscribe
  }, [idsKey, refresh, setAppStatus])

  return { statusByAppId, setAppStatus, refresh }
}
