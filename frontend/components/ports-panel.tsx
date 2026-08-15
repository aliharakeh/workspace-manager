import { useDeferredValue, useEffect, useMemo, useState } from "react"
import {
  RefreshCwIcon,
  SearchIcon,
  SkullIcon,
  XIcon,
} from "lucide-react"
import { toast } from "sonner"
import { api } from "@/lib/api"
import type { ListeningProcess } from "@/lib/types"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Skeleton } from "@/components/ui/skeleton"

type PortsPanelProps = {
  active: boolean
}

function matchesQuery(entry: ListeningProcess, query: string): boolean {
  if (!query) return true
  const q = query.toLowerCase()
  return (
    entry.name.toLowerCase().includes(q) ||
    String(entry.port).includes(q) ||
    String(entry.pid).includes(q)
  )
}

export function PortsPanel({ active }: PortsPanelProps) {
  const [processes, setProcesses] = useState<ListeningProcess[]>([])
  const [range, setRange] = useState({ min: 1024, max: 49151 })
  const [query, setQuery] = useState("")
  const deferredQuery = useDeferredValue(query)
  const [loading, setLoading] = useState(false)
  const [pendingKill, setPendingKill] = useState<ListeningProcess | null>(null)
  const [killing, setKilling] = useState(false)

  async function load() {
    setLoading(true)
    try {
      const data = await api.ports.list()
      setProcesses(data.processes)
      setRange({ min: data.min, max: data.max })
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to list ports")
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (active) void load()
    else setQuery("")
  }, [active])

  const filtered = useMemo(
    () =>
      processes.filter((entry) => matchesQuery(entry, deferredQuery.trim())),
    [processes, deferredQuery]
  )

  async function handleConfirmKill() {
    if (!pendingKill) return
    setKilling(true)
    try {
      await api.ports.kill(pendingKill.pid)
      toast.success(`Killed ${pendingKill.name} (pid ${pendingKill.pid})`)
      setPendingKill(null)
      await load()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to kill process")
    } finally {
      setKilling(false)
    }
  }

  return (
    <>
      <div className="flex min-h-0 flex-1 flex-col gap-4">
        <div className="flex flex-col gap-1">
          <div className="flex items-center justify-between gap-2">
            <p className="text-sm font-medium">Listening ports</p>
            <Badge variant="secondary">{processes.length}</Badge>
          </div>
          <p className="text-sm text-muted-foreground">
            Processes on ports {range.min}–{range.max}. Kill frees a port for
            other apps.
          </p>
        </div>

        <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
          <InputGroup className="flex-1">
            <InputGroupAddon align="inline-start">
              <SearchIcon />
            </InputGroupAddon>
            <InputGroupInput
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search name, port, or PID…"
              aria-label="Search processes"
            />
            {query ? (
              <InputGroupAddon align="inline-end">
                <InputGroupButton
                  size="icon-xs"
                  aria-label="Clear search"
                  onClick={() => setQuery("")}
                >
                  <XIcon />
                </InputGroupButton>
              </InputGroupAddon>
            ) : null}
          </InputGroup>
          <Button
            variant="outline"
            disabled={loading}
            onClick={() => void load()}
          >
            <RefreshCwIcon data-icon="inline-start" />
            Refresh
          </Button>
        </div>

        <ScrollArea className="min-h-0 flex-1 rounded-lg border">
          {loading && processes.length === 0 ? (
            <div className="flex flex-col gap-2 p-3">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : processes.length === 0 ? (
            <Empty className="min-h-48 border-0">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <SkullIcon />
                </EmptyMedia>
                <EmptyTitle>No listening processes</EmptyTitle>
                <EmptyDescription>
                  Nothing is listening in this port range right now.
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : filtered.length === 0 ? (
            <Empty className="min-h-48 border-0">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <SearchIcon />
                </EmptyMedia>
                <EmptyTitle>No matches</EmptyTitle>
                <EmptyDescription>
                  Nothing matches “{deferredQuery.trim()}”.
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            <ul className="divide-y">
              {filtered.map((entry) => (
                <li
                  key={`${entry.port}-${entry.pid}`}
                  className="flex items-center gap-3 px-3 py-2.5"
                >
                  <div className="min-w-0 flex-1">
                    <div className="truncate font-medium">{entry.name}</div>
                    <div className="flex flex-wrap items-center gap-x-2 gap-y-0.5 text-xs text-muted-foreground">
                      <span className="font-mono">:{entry.port}</span>
                      <span aria-hidden>·</span>
                      <span className="font-mono">pid {entry.pid}</span>
                    </div>
                  </div>
                  <Button
                    variant="destructive"
                    size="xs"
                    disabled={killing || loading}
                    onClick={() => setPendingKill(entry)}
                    title={`Kill ${entry.name}`}
                  >
                    <SkullIcon data-icon="inline-start" />
                    Kill
                  </Button>
                </li>
              ))}
            </ul>
          )}
        </ScrollArea>

        {processes.length > 0 && deferredQuery.trim() ? (
          <p className="text-xs text-muted-foreground">
            Showing {filtered.length} of {processes.length}
          </p>
        ) : null}
      </div>

      <AlertDialog
        open={!!pendingKill}
        onOpenChange={(next) => {
          if (!next && !killing) setPendingKill(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Kill process?</AlertDialogTitle>
            <AlertDialogDescription>
              This force-kills the process and frees its listening ports. This
              cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          {pendingKill ? (
            <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1.5 rounded-lg border bg-muted/40 px-3 py-2.5 text-sm">
              <dt className="text-muted-foreground">Process</dt>
              <dd className="truncate font-medium">{pendingKill.name}</dd>
              <dt className="text-muted-foreground">Port</dt>
              <dd className="font-mono">{pendingKill.port}</dd>
              <dt className="text-muted-foreground">PID</dt>
              <dd className="font-mono">{pendingKill.pid}</dd>
            </dl>
          ) : null}
          <AlertDialogFooter>
            <AlertDialogCancel disabled={killing}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={killing}
              onClick={(e) => {
                e.preventDefault()
                void handleConfirmKill()
              }}
            >
              {killing ? "Killing…" : "Kill process"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
