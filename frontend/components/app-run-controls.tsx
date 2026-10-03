import { useState } from "react"
import { toast } from "sonner"
import { HammerIcon, PlayIcon, RefreshCwIcon, SquareIcon } from "lucide-react"
import { api } from "@/lib/api"
import { appStateLabel } from "@/lib/app-state"
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
  /** True while the app is running or building: both hold the app's one session. */
  running: boolean
  /** True while that session is a build; Stop ends it, Reload is unavailable. */
  building?: boolean
  onStatus: (status: StatusEvent) => void
  variant?: "default" | "compact"
  className?: string
}

export function AppRunControls({
  appId,
  running,
  building = false,
  onStatus,
  variant = "default",
  className,
}: AppRunControlsProps) {
  const [busy, setBusy] = useState(false)
  const compact = variant === "compact"

  async function runAction(
    action: "run" | "build" | "stop" | "reload",
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
                disabled={busy || running}
                onClick={() =>
                  void runAction("build", "Build started", "Failed to build")
                }
              />
            }
          >
            <HammerIcon />
            <span className="sr-only">Build</span>
          </TooltipTrigger>
          <TooltipContent>Build</TooltipContent>
        </Tooltip>
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
        variant="outline"
        disabled={busy || running}
        onClick={() => void runAction("build", "Build started", "Failed to build")}
      >
        <HammerIcon data-icon="inline-start" />
        Build
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
        disabled={busy || building}
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
  building = false,
  className,
}: {
  running: boolean
  building?: boolean
  className?: string
}) {
  const label = appStateLabel(running, building)
  return (
    <span
      className={cn(
        "size-2 shrink-0 rounded-full",
        !running
          ? "bg-muted-foreground/35"
          : building
            ? "bg-amber-500"
            : "bg-emerald-500",
        className
      )}
      title={label}
      aria-label={label}
    />
  )
}
