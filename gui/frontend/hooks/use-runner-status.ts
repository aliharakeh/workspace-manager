import { useCallback, useEffect, useState } from "react"
import { api, onDataChanged, onRunnerEvent } from "@/lib/api"
import type { StatusEvent } from "@/lib/types"

type State = { appId: number | null; status: StatusEvent | null }

/** Live run status (processes, ready URLs) of one app. The terminal output
 * itself is not kept here: each terminal loads and streams its own. */
export function useRunnerStatus(appId: number | null) {
  const [state, setState] = useState<State>({ appId: null, status: null })

  useEffect(() => {
    if (appId == null) return
    let cancelled = false
    let gotEvent = false

    const unsubscribe = onRunnerEvent((eventAppId, event) => {
      if (cancelled || eventAppId !== appId || event.type !== "status") return
      gotEvent = true
      setState({ appId, status: event })
    }, appId)

    const load = (force: boolean) =>
      api.runner
        .status(appId)
        .then((status) => {
          if (!cancelled && (force || !gotEvent)) {
            setState({ appId, status: status as StatusEvent })
          }
        })
        .catch(() => {
          if (!cancelled) setState({ appId, status: null })
        })
    void load(false)

    // An agent that takes or releases the app's run session (through the MCP
    // server) does not produce runner events here, so read the status again.
    const unsubscribeChanges = onDataChanged((event) => {
      if (event.entity !== "session") return
      if (event.ids?.length && !event.ids.includes(appId)) return
      void load(true)
    })

    return () => {
      cancelled = true
      unsubscribe()
      unsubscribeChanges()
    }
  }, [appId])

  const setStatus = useCallback(
    (status: StatusEvent) => setState({ appId, status }),
    [appId]
  )

  const current = state.appId === appId ? state.status : null
  return { status: current, setStatus }
}
