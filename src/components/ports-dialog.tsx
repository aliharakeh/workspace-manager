import { useEffect, useState } from "react"
import { RefreshCwIcon, SkullIcon } from "lucide-react"
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
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { ScrollArea } from "@/components/ui/scroll-area"

type PortsDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function PortsDialog({ open, onOpenChange }: PortsDialogProps) {
  const [processes, setProcesses] = useState<ListeningProcess[]>([])
  const [range, setRange] = useState({ min: 1024, max: 49151 })
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
    if (open) void load()
  }, [open])

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
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Listening ports</DialogTitle>
            <DialogDescription>
              Processes using registered ports {range.min}–{range.max}. Kill
              frees the port for other apps.
            </DialogDescription>
          </DialogHeader>

          <ScrollArea className="h-72 rounded-lg border">
            {loading && processes.length === 0 ? (
              <p className="p-4 text-sm text-muted-foreground">Loading…</p>
            ) : processes.length === 0 ? (
              <p className="p-4 text-sm text-muted-foreground">
                No listening processes in this range.
              </p>
            ) : (
              <ul className="divide-y">
                {processes.map((entry) => (
                  <li
                    key={`${entry.port}-${entry.pid}`}
                    className="flex items-center gap-3 px-3 py-2"
                  >
                    <div className="min-w-0 flex-1">
                      <div className="truncate font-medium">{entry.name}</div>
                      <div className="truncate text-xs text-muted-foreground">
                        Port {entry.port} · PID {entry.pid}
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

          <DialogFooter>
            <Button
              variant="outline"
              disabled={loading}
              onClick={() => void load()}
            >
              <RefreshCwIcon data-icon="inline-start" />
              Refresh
            </Button>
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

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
