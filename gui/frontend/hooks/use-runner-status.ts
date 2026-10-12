import { useCallback, useEffect, useState } from "react"
import { api, onRunnerEvent } from "@/lib/api"
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

    void api.runner
      .status(appId)
      .then((status) => {
        if (!cancelled && !gotEvent) {
          setState({ appId, status: status as StatusEvent })
        }
      })
      .catch(() => {
        if (!cancelled) setState({ appId, status: null })
      })

    return () => {
      cancelled = true
      unsubscribe()
    }
  }, [appId])

  const setStatus = useCallback(
    (status: StatusEvent) => setState({ appId, status }),
    [appId]
  )

  const current = state.appId === appId ? state.status : null
  return { status: current, setStatus }
}
