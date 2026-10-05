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
  /** "icon" is the full set of actions as icon buttons with tooltips. */
  variant?: "default" | "compact" | "icon"
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
            {running ? (
              <SquareIcon className="text-red-600" />
            ) : (
              <PlayIcon className="text-emerald-600" />
            )}
            <span className="sr-only">{label}</span>
          </TooltipTrigger>
          <TooltipContent>{label}</TooltipContent>
        </Tooltip>
      </div>
    )
  }

  if (variant === "icon") {
    const actions = [
      {
        label: "Run",
        hint: "Run the app",
        icon: <PlayIcon />,
        variant: "default",
        className: running
          ? ""
          : "bg-emerald-600 text-white hover:bg-emerald-700",
        disabled: busy || running,
        onClick: () => runAction("run", "Started", "Failed to run"),
      },
      {
        label: "Build",
        hint: "Build the app",
        icon: <HammerIcon />,
        variant: "outline",
        disabled: busy || running,
        onClick: () => runAction("build", "Build started", "Failed to build"),
      },
      {
        label: "Stop",
        hint: building ? "Stop the build" : "Stop the app",
        icon: <SquareIcon />,
        variant: "secondary",
        className: running ? "bg-red-600 text-white hover:bg-red-700" : "",
        disabled: busy || !running,
        onClick: () => runAction("stop", "Stopped", "Failed to stop"),
      },
      {
        label: "Reload",
        hint: "Restart the app (stop, then run again)",
        icon: <RefreshCwIcon />,
        variant: "outline",
        className: running
          ? "border-transparent bg-yellow-500 text-black hover:bg-yellow-600 hover:text-black dark:border-transparent dark:bg-yellow-500 dark:hover:bg-yellow-600"
          : "",
        disabled: busy || building,
        onClick: () => runAction("reload", "Reloaded", "Failed to reload"),
      },
    ] as const
    return (
      <div className={cn("flex items-center gap-2", className)}>
        {actions.map((action) => (
          <Tooltip key={action.label}>
            <TooltipTrigger
              render={
                <Button
                  type="button"
                  size="icon-sm"
                  variant={action.variant}
                  className={"className" in action ? action.className : undefined}
                  aria-label={action.label}
                  disabled={action.disabled}
                  onClick={(event) => {
                    event.stopPropagation()
                    void action.onClick()
                  }}
                />
              }
            >
              {action.icon}
            </TooltipTrigger>
            <TooltipContent>{action.hint}</TooltipContent>
          </Tooltip>
        ))}
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
