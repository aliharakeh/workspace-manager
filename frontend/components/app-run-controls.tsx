import { useState } from "react"
import { toast } from "sonner"
import {
  ChevronDownIcon,
  PlayIcon,
  RefreshCwIcon,
  SquareIcon,
} from "lucide-react"
import { api } from "@/lib/api"
import {
  appStateLabel,
  appVisualState,
  type AppVisualState,
} from "@/lib/app-state"
import type { StatusEvent } from "@/lib/types"
import { useRunConfigs } from "@/lib/use-run-configs"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { cn } from "@/lib/utils"

type AppRunControlsProps = {
  appId: number
  running: boolean
  kind?: string
  onStatus: (status: StatusEvent) => void
  variant?: "default" | "compact" | "icon"
  showSetup?: boolean
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
  const { configs, selected, select, loading, error } = useRunConfigs(appId)
  const compact = variant === "compact"
  async function action(action: "run" | "stop" | "reload") {
    setBusy(true)
    try {
      const next =
        action === "run" && selected
          ? await api.runner.start(appId, selected.kind)
          : action === "stop"
            ? await api.runner.stop(appId)
            : await api.runner.reload(appId)
      onStatus(next)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Runner action failed")
    } finally {
      setBusy(false)
    }
  }
  const label = running
    ? "Stop"
    : selected
      ? `Run ${selected.name}`
      : "Add a run configuration first"
  // Stop, and Run with no configuration to run, are icon-only; only a
  // runnable configuration shows its name next to the play icon.
  const iconOnly = compact || running || !selected
  return (
    <div
      className={cn("flex items-center gap-1", className)}
      onClick={(e) => e.stopPropagation()}
      onKeyDown={(e) => e.stopPropagation()}
    >
      <Button
        type="button"
        variant={running ? "stop" : compact ? "ghost" : "run"}
        size={compact ? "icon-xs" : iconOnly ? "icon-sm" : "sm"}
        className={cn(
          // In the sidebar an idle app keeps a quiet button with a green icon,
          // so a list of apps does not turn into a wall of green.
          compact && !running && "text-emerald-600 dark:text-emerald-500"
        )}
        title={error ?? label}
        aria-label={label}
        disabled={
          busy ||
          (!running &&
            (loading || !selected || !!error || selected.commands.length === 0))
        }
        onClick={() => void action(running ? "stop" : "run")}
      >
        {running ? (
          <SquareIcon data-icon="inline-start" className="fill-current" />
        ) : (
          <PlayIcon data-icon="inline-start" className="fill-current" />
        )}
        {iconOnly ? null : selected?.name}
      </Button>
      {!running && configs.length > 0 ? (
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                type="button"
                variant="ghost"
                size={compact ? "icon-xs" : "icon-sm"}
                aria-label="Choose run configuration"
                disabled={busy || loading}
              />
            }
          >
            <ChevronDownIcon />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuGroup>
              <DropdownMenuLabel>Run configuration</DropdownMenuLabel>
              {configs.map((c) => (
                <DropdownMenuItem key={c.kind} onClick={() => select(c.kind)}>
                  {c.name}
                  {selected?.kind === c.kind ? " ✓" : ""}
                </DropdownMenuItem>
              ))}
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      ) : null}
      {running && !compact ? (
        <Button
          type="button"
          size="icon-sm"
          variant="outline"
          title="Restart this configuration"
          aria-label="Restart this configuration"
          disabled={busy}
          onClick={() => void action("reload")}
        >
          <RefreshCwIcon />
        </Button>
      ) : null}
    </div>
  )
}

const DOT_COLOR: Record<AppVisualState, string> = {
  running: "bg-emerald-500",
  task: "bg-amber-500",
  failed: "bg-red-500",
  idle: "bg-muted-foreground/35",
}

/** A small dot for an app's state; it pulses while something is going. */
export function AppStatusDot({
  running,
  kind,
  error,
  className,
}: {
  running: boolean
  kind?: string
  error?: string
  className?: string
}) {
  const state = appVisualState(running, kind, error)
  const label = state === "failed" ? "Failed" : appStateLabel(running, kind)
  return (
    <span
      className={cn("relative inline-flex size-2 shrink-0", className)}
      title={state === "failed" ? error : label}
      aria-label={label}
      role="img"
    >
      {running ? (
        <span
          className={cn(
            "absolute inset-0 animate-ping rounded-full opacity-60",
            DOT_COLOR[state]
          )}
        />
      ) : null}
      <span
        className={cn("relative size-2 rounded-full", DOT_COLOR[state])}
      />
    </span>
  )
}

const BADGE_STYLE: Record<AppVisualState, string> = {
  running:
    "border-emerald-500/40 bg-emerald-500/15 text-emerald-700 dark:text-emerald-400",
  task: "border-amber-500/40 bg-amber-500/15 text-amber-700 dark:text-amber-400",
  failed: "border-red-500/40 bg-red-500/15 text-red-700 dark:text-red-400",
  idle: "border-border bg-muted text-muted-foreground",
}

/** A colored pill with a status dot: the running configuration's name, Building, Failed or Idle. */
export function AppStateBadge({
  running,
  kind,
  configName,
  error,
  className,
}: {
  running: boolean
  kind?: string
  configName?: string
  error?: string
  className?: string
}) {
  const state = appVisualState(running, kind, error)
  const label =
    state === "failed"
      ? "Failed"
      : state === "running" && configName
        ? configName
        : appStateLabel(running, kind)
  const title =
    state === "failed"
      ? error
      : state === "running" && configName
        ? `Running ${configName}`
        : label
  return (
    <span
      role="status"
      title={title}
      className={cn(
        "inline-flex h-5 w-fit shrink-0 items-center gap-1.5 rounded-4xl border px-2 text-xs font-medium whitespace-nowrap",
        BADGE_STYLE[state],
        className
      )}
    >
      <AppStatusDot running={running} kind={kind} error={error} />
      {label}
    </span>
  )
}
