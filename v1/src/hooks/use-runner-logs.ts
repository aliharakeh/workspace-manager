import { useEffect, useRef, useState } from "react"
import type { LogEvent, RunnerEvent, StatusEvent } from "@/lib/types"

export type LogLine = LogEvent & { id: number }

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
    let source: EventSource | null = null
    let retryTimer: ReturnType<typeof setTimeout> | undefined

    const connect = () => {
      if (cancelled) return
      // Same-origin via Vite SSE middleware (avoids CORS + http-proxy buffering)
      source = new EventSource(`/api/apps/${appId}/run/logs`)

      source.onopen = () => {
        if (!cancelled) setConnected(true)
      }

      source.onmessage = (event) => {
        if (cancelled) return
        try {
          const data = JSON.parse(event.data) as RunnerEvent
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
        } catch {
          // ignore malformed payloads
        }
      }

      source.onerror = () => {
        setConnected(false)
        source?.close()
        source = null
        if (cancelled) return
        retryTimer = setTimeout(connect, 1500)
      }
    }

    connect()

    return () => {
      cancelled = true
      setConnected(false)
      if (retryTimer) clearTimeout(retryTimer)
      source?.close()
    }
  }, [appId])

  return { status, logs, setLogs, setStatus, connected }
}
