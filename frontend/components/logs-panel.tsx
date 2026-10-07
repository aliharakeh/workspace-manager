import { useEffect, useMemo, useState } from "react"
import { ExternalLinkIcon } from "lucide-react"
import { api, handleReadyUrlClick, onRunnerEvent } from "@/lib/api"
import { decodeTerminalData } from "@/lib/terminal"
import type { LogEvent, ProcessState, StatusEvent } from "@/lib/types"
import { Badge } from "@/components/ui/badge"
import { Terminal, type TerminalHandle } from "@/components/terminal"
import { cn } from "@/lib/utils"

type LogsPanelProps = {
  appId: number
  status: StatusEvent | null
}

function statusVariant(
  status: ProcessState["status"]
): "default" | "secondary" | "destructive" | "outline" {
  switch (status) {
    case "running":
      return "default"
    case "error":
    case "killed":
      return "destructive"
    case "exited":
      return "secondary"
    default:
      return "outline"
  }
}

/**
 * The output of one run command in a read-only terminal. It loads what the
 * command printed so far, then follows the live chunks; offsets keep the two
 * from overlapping or leaving a gap.
 */
function RunnerTerminal({
  appId,
  commandId,
}: {
  appId: number
  commandId: number
}) {
  const [handle, setHandle] = useState<TerminalHandle | null>(null)

  useEffect(() => {
    if (!handle) return
    let cancelled = false
    // Bytes of the command shown so far; -1 until the snapshot has arrived.
    let end = -1
    const queued: LogEvent[] = []

    const apply = (event: LogEvent) => {
      const bytes = decodeTerminalData(event.data)
      const eventEnd = event.offset + bytes.length
      if (eventEnd <= end) return
      handle.write(
        event.offset < end ? bytes.subarray(end - event.offset) : bytes
      )
      end = eventEnd
    }
    const flush = () => queued.splice(0).forEach(apply)

    const unsubscribe = onRunnerEvent((eventAppId, event) => {
      if (
        cancelled ||
        eventAppId !== appId ||
        event.type !== "log" ||
        event.commandId !== commandId
      ) {
        return
      }
      if (end < 0) queued.push(event)
      else apply(event)
    }, appId)

    api.runner
      .output(appId, commandId)
      .then((snapshot) => {
        if (cancelled) return
        const bytes = decodeTerminalData(snapshot.data)
        if (bytes.length > 0) handle.write(bytes)
        end = snapshot.end
        flush()
      })
      .catch(() => {
        if (cancelled) return
        end = 0
        flush()
      })

    return () => {
      cancelled = true
      unsubscribe()
    }
  }, [handle, appId, commandId])

  return (
    <Terminal
      readOnly
      className="h-80"
      onHandle={setHandle}
      onResize={(cols, rows) => {
        void api.runner.resize(appId, commandId, cols, rows)
      }}
    />
  )
}

export function LogsPanel({ appId, status }: LogsPanelProps) {
  const processes = useMemo(() => status?.processes ?? [], [status?.processes])
  const readyUrls = useMemo(() => {
    if (!status?.running) return []
    return [...new Set(processes.flatMap((p) => p.urls ?? []))]
  }, [status?.running, processes])
  const [active, setActive] = useState<string | null>(null)

  const activeId = useMemo(() => {
    if (active && processes.some((p) => String(p.commandId) === active)) {
      return active
    }
    return processes[0] ? String(processes[0].commandId) : null
  }, [processes, active])

  if (processes.length === 0 || activeId === null) {
    return (
      <div className="rounded-lg border border-dashed p-6 text-sm text-muted-foreground">
        Run, build or set up the app to see its output here.
        {status?.error ? (
          <p className="mt-2 text-destructive">{status.error}</p>
        ) : null}
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
        <span>Read-only terminal output</span>
        {status?.running ? <Badge variant="default">Live</Badge> : null}
      </div>

      <div className="flex flex-wrap gap-1">
        {processes.map((p) => (
          <button
            key={p.commandId}
            type="button"
            className={cn(
              "inline-flex items-center gap-2 rounded-md px-2.5 py-1 text-sm",
              activeId === String(p.commandId)
                ? "bg-muted font-medium"
                : "hover:bg-muted/60"
            )}
            onClick={() => setActive(String(p.commandId))}
          >
            <span className="max-w-40 truncate">{p.label}</span>
            <Badge variant={statusVariant(p.status)}>{p.status}</Badge>
          </button>
        ))}
      </div>

      {readyUrls.length > 0 ? (
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-xs text-muted-foreground">Open</span>
          {readyUrls.map((url) => (
            <a
              key={url}
              href={url}
              target="_blank"
              rel="noreferrer"
              onClick={(event) => handleReadyUrlClick(event, url)}
              className="inline-flex max-w-full items-center gap-1 truncate rounded-md border px-2 py-1 font-mono text-xs hover:bg-muted"
            >
              <ExternalLinkIcon className="size-3 shrink-0" />
              <span className="truncate">{url}</span>
            </a>
          ))}
        </div>
      ) : null}

      {/* A new run (session) or another command starts a fresh terminal. */}
      <RunnerTerminal
        key={`${status?.sessionId ?? ""}:${activeId}`}
        appId={appId}
        commandId={Number(activeId)}
      />
    </div>
  )
}
