import { useEffect, useMemo, useRef, useState } from "react"
import type { LogLine } from "@/hooks/use-runner-logs"
import type { ProcessState, StatusEvent } from "@/lib/types"
import { Badge } from "@/components/ui/badge"
import { ScrollArea } from "@/components/ui/scroll-area"
import { cn } from "@/lib/utils"

type LogsPanelProps = {
  status: StatusEvent | null
  logs: LogLine[]
  connected?: boolean
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

export function LogsPanel({ status, logs, connected }: LogsPanelProps) {
  const processes = status?.processes ?? []
  const [active, setActive] = useState<string>("all")
  const bottomRef = useRef<HTMLDivElement>(null)

  const activeId = useMemo(() => {
    if (active === "all") return "all"
    if (processes.some((p) => String(p.commandId) === active)) return active
    return "all"
  }, [processes, active])

  const visibleLogs = useMemo(() => {
    if (activeId === "all") return logs
    const commandId = Number(activeId)
    return logs.filter((l) => l.commandId === commandId)
  }, [logs, activeId])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" })
  }, [visibleLogs.length, activeId])

  if (processes.length === 0 && logs.length === 0) {
    return (
      <div className="rounded-lg border border-dashed p-6 text-sm text-muted-foreground">
        <div className="flex items-center gap-2">
          <span
            className={cn(
              "size-2 rounded-full",
              connected ? "bg-emerald-500" : "bg-muted-foreground/40"
            )}
          />
          {connected
            ? "SSE connected — run the app to stream process logs."
            : "Connecting to log stream…"}
        </div>
        {status?.error ? (
          <p className="mt-2 text-destructive">{status.error}</p>
        ) : null}
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-2 text-xs text-muted-foreground">
        <span
          className={cn(
            "size-2 rounded-full",
            connected ? "bg-emerald-500" : "bg-muted-foreground/40"
          )}
        />
        {connected ? "Streaming via SSE" : "Reconnecting…"}
        {status?.running ? <Badge variant="default">Live</Badge> : null}
      </div>

      <div className="flex flex-wrap gap-1">
        <button
          type="button"
          className={cn(
            "rounded-md px-2.5 py-1 text-sm",
            activeId === "all" ? "bg-muted font-medium" : "hover:bg-muted/60"
          )}
          onClick={() => setActive("all")}
        >
          All
        </button>
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

      <ScrollArea className="h-80 rounded-lg border bg-muted/30">
        <pre className="p-3 font-mono text-xs leading-relaxed whitespace-pre-wrap">
          {visibleLogs.length === 0
            ? "No output yet."
            : visibleLogs.map((line) => (
                <span
                  key={line.id}
                  className={cn(
                    "block",
                    line.stream === "stderr" && "text-destructive",
                    line.stream === "system" && "text-muted-foreground"
                  )}
                >
                  {line.text}
                </span>
              ))}
          <div ref={bottomRef} />
        </pre>
      </ScrollArea>
    </div>
  )
}
