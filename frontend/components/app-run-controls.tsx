import { useState } from "react"
import { toast } from "sonner"
import {
  ChevronDownIcon,
  PlayIcon,
  RefreshCwIcon,
  SquareIcon,
} from "lucide-react"
import { api } from "@/lib/api"
import { appStateLabel, isTaskKind } from "@/lib/app-state"
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
  return (
    <div
      className={cn("flex items-center gap-1", className)}
      onClick={(e) => e.stopPropagation()}
      onKeyDown={(e) => e.stopPropagation()}
    >
      <Button
        type="button"
        variant={compact ? "ghost" : running ? "secondary" : "default"}
        size={compact ? "icon-xs" : "sm"}
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
          <SquareIcon data-icon="inline-start" />
        ) : (
          <PlayIcon data-icon="inline-start" />
        )}
        {compact ? null : running ? "Stop" : (selected?.name ?? "Run")}
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

export function AppStatusDot({
  running,
  kind,
  className,
}: {
  running: boolean
  kind?: string
  className?: string
}) {
  const label = appStateLabel(running, kind)
  return (
    <span
      className={cn(
        "size-2 shrink-0 rounded-full",
        !running
          ? "bg-muted-foreground/35"
          : isTaskKind(kind)
            ? "bg-amber-500"
            : "bg-emerald-500",
        className
      )}
      title={label}
      aria-label={label}
    />
  )
}
