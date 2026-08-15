import { useEffect, useRef, useState, type Dispatch, type SetStateAction } from "react"
import { api, onRunnerEvent } from "@/lib/api"
import type { LogEvent, RunnerEvent, StatusEvent } from "@/lib/types"

export type LogLine = LogEvent & { id: number }

function applyEvent(
  data: RunnerEvent,
  sessionIdRef: { current: string },
  lineIdRef: { current: number },
  setStatus: (status: StatusEvent) => void,
  setLogs: Dispatch<SetStateAction<LogLine[]>>
) {
  if (data.type === "status") {
    if (data.sessionId && data.sessionId !== sessionIdRef.current) {
      const hadSession = sessionIdRef.current !== ""
      sessionIdRef.current = data.sessionId
      if (hadSession) {
        setLogs([])
        lineIdRef.current = 0
      }
    }
    setStatus(data)
  } else if (data.type === "log") {
    const id = ++lineIdRef.current
    setLogs((prev) => [...prev, { ...data, id }])
  }
}

export function useRunnerLogs(appId: number | null) {
  const [status, setStatus] = useState<StatusEvent | null>(null)
  const [logs, setLogs] = useState<LogLine[]>([])
  const [connected, setConnected] = useState(false)
  const lineIdRef = useRef(0)
  const sessionIdRef = useRef("")

  useEffect(() => {
    if (appId == null) {
      setStatus(null)
      setLogs([])
      setConnected(false)
      sessionIdRef.current = ""
      return
    }

    setLogs([])
    lineIdRef.current = 0
    sessionIdRef.current = ""
    let cancelled = false

    void api.runner
      .logs(appId)
      .then((snapshot) => {
        if (cancelled) return
        if (snapshot.status.sessionId) {
          sessionIdRef.current = snapshot.status.sessionId
        }
        setStatus(snapshot.status)
        const next: LogLine[] = []
        for (const log of snapshot.logs) {
          next.push({ ...log, id: ++lineIdRef.current })
        }
        setLogs(next)
        setConnected(true)
      })
      .catch(() => {
        if (!cancelled) setConnected(false)
      })

    const unsubscribe = onRunnerEvent((eventAppId, event) => {
      if (cancelled || eventAppId !== appId) return
      applyEvent(event, sessionIdRef, lineIdRef, setStatus, setLogs)
    }, appId)

    return () => {
      cancelled = true
      setConnected(false)
      unsubscribe()
    }
  }, [appId])

  return { status, logs, setLogs, setStatus, connected }
}
