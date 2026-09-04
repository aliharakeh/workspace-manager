import { useState } from "react"
import { toast } from "sonner"
import { PlayIcon, RefreshCwIcon, SquareIcon } from "lucide-react"
import { api } from "@/lib/api"
import type { StatusEvent } from "@/lib/types"
import { Button } from "@/components/ui/button"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { cn } from "@/lib/utils"

type AppRunControlsProps = {
  appId: number
  running: boolean
  onStatus: (status: StatusEvent) => void
  variant?: "default" | "compact"
  className?: string
}

export function AppRunControls({
  appId,
  running,
  onStatus,
  variant = "default",
  className,
}: AppRunControlsProps) {
  const [busy, setBusy] = useState(false)
  const compact = variant === "compact"

  async function runAction(
    action: "run" | "stop" | "reload",
    success: string,
    failure: string
  ) {
    setBusy(true)
    try {
      const next = await api.runner[action](appId)
      onStatus(next)
      toast.success(success)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : failure)
    } finally {
      setBusy(false)
    }
  }

  if (compact) {
    const label = running ? "Stop" : "Start"
    return (
      <div
        className={cn("flex items-center", className)}
        onClick={(event) => event.stopPropagation()}
        onKeyDown={(event) => event.stopPropagation()}
      >
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                type="button"
                variant="ghost"
                size="icon-xs"
                disabled={busy}
                onClick={() =>
                  running
                    ? void runAction("stop", "Stopped", "Failed to stop")
                    : void runAction("run", "Started", "Failed to run")
                }
              />
            }
          >
            {running ? <SquareIcon /> : <PlayIcon />}
            <span className="sr-only">{label}</span>
          </TooltipTrigger>
          <TooltipContent>{label}</TooltipContent>
        </Tooltip>
      </div>
    )
  }

  return (
    <div className={cn("flex flex-wrap gap-2", className)}>
      <Button
        size="sm"
        disabled={busy || running}
        onClick={() => void runAction("run", "Started", "Failed to run")}
      >
        <PlayIcon data-icon="inline-start" />
        Run
      </Button>
      <Button
        size="sm"
        variant="secondary"
        disabled={busy || !running}
        onClick={() => void runAction("stop", "Stopped", "Failed to stop")}
      >
        <SquareIcon data-icon="inline-start" />
        Stop
      </Button>
      <Button
        size="sm"
        variant="outline"
        disabled={busy}
        onClick={() => void runAction("reload", "Reloaded", "Failed to reload")}
      >
        <RefreshCwIcon data-icon="inline-start" />
        Reload
      </Button>
    </div>
  )
}

export function AppStatusDot({
  running,
  className,
}: {
  running: boolean
  className?: string
}) {
  return (
    <span
      className={cn(
        "size-2 shrink-0 rounded-full",
        running ? "bg-emerald-500" : "bg-muted-foreground/35",
        className
      )}
      title={running ? "Running" : "Idle"}
      aria-label={running ? "Running" : "Idle"}
    />
  )
}
