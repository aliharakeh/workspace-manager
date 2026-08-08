import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type RefObject,
} from "react"
import { ExternalLinkIcon } from "lucide-react"
import type { LogLine } from "@/hooks/use-runner-logs"
import type { ProcessState, StatusEvent } from "@/lib/types"
import { stripAnsi } from "@/lib/ansi"
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

function LogStreamView({
  label,
  empty,
  lines,
  tone,
  bottomRef,
}: {
  label: string
  empty: string
  lines: LogLine[]
  tone?: "destructive"
  bottomRef?: RefObject<HTMLDivElement | null>
}) {
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-1.5">
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs font-medium text-muted-foreground">{label}</span>
        <span className="text-xs text-muted-foreground/70">{lines.length}</span>
      </div>
      <ScrollArea className="h-56 rounded-lg border bg-muted/30">
        <pre
          className={cn(
            "p-3 font-mono text-xs leading-relaxed whitespace-pre-wrap",
            tone === "destructive" && "text-destructive"
          )}
        >
          {lines.length === 0
            ? empty
            : lines.map((line) => (
                <span
                  key={line.id}
                  className={cn(
                    "block",
                    line.stream === "system" && "text-muted-foreground"
                  )}
                >
                  {stripAnsi(line.text)}
                </span>
              ))}
          {bottomRef ? <div ref={bottomRef} /> : null}
        </pre>
      </ScrollArea>
    </div>
  )
}

export function LogsPanel({ status, logs, connected }: LogsPanelProps) {
  const processes = status?.processes ?? []
  const readyUrls = useMemo(() => {
    if (!status?.running) return []
    return [...new Set(processes.flatMap((p) => p.urls ?? []))]
  }, [status?.running, processes])
  const [active, setActive] = useState<string | null>(null)
  const stdoutBottomRef = useRef<HTMLDivElement>(null)
  const stderrBottomRef = useRef<HTMLDivElement>(null)

  const activeId = useMemo(() => {
    if (active && processes.some((p) => String(p.commandId) === active)) {
      return active
    }
    return processes[0] ? String(processes[0].commandId) : null
  }, [processes, active])

  const visibleLogs = useMemo(() => {
    if (!activeId) return []
    const commandId = Number(activeId)
    return logs.filter((l) => l.commandId === commandId)
  }, [logs, activeId])

  const stdoutLogs = useMemo(
    () => visibleLogs.filter((l) => l.stream !== "stderr"),
    [visibleLogs]
  )
  const stderrLogs = useMemo(
    () => visibleLogs.filter((l) => l.stream === "stderr"),
    [visibleLogs]
  )

  useEffect(() => {
    stdoutBottomRef.current?.scrollIntoView({ behavior: "smooth" })
    stderrBottomRef.current?.scrollIntoView({ behavior: "smooth" })
  }, [stdoutLogs.length, stderrLogs.length, activeId])

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

      {processes.length > 0 ? (
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
      ) : null}

      {readyUrls.length > 0 ? (
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-xs text-muted-foreground">Open</span>
          {readyUrls.map((url) => (
            <a
              key={url}
              href={url}
              target="_blank"
              rel="noreferrer"
              className="inline-flex max-w-full items-center gap-1 truncate rounded-md border px-2 py-1 font-mono text-xs hover:bg-muted"
            >
              <ExternalLinkIcon className="size-3 shrink-0" />
              <span className="truncate">{url}</span>
            </a>
          ))}
        </div>
      ) : null}

      <div className="flex flex-col gap-3 lg:flex-row">
        <LogStreamView
          label="stdout"
          empty="No stdout yet."
          lines={stdoutLogs}
          bottomRef={stdoutBottomRef}
        />
        <LogStreamView
          label="stderr"
          empty="No stderr yet."
          lines={stderrLogs}
          tone="destructive"
          bottomRef={stderrBottomRef}
        />
      </div>
    </div>
  )
}
