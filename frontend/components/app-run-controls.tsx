import { useId, useState, type ReactNode } from "react"
import { toast } from "sonner"
import {
  HammerIcon,
  PlayIcon,
  RefreshCwIcon,
  RocketIcon,
  SquareIcon,
  WrenchIcon,
} from "lucide-react"
import { api } from "@/lib/api"
import { appStateLabel, isTaskKind } from "@/lib/app-state"
import type { CommandKind, ConfigSet, StatusEvent } from "@/lib/types"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { cn } from "@/lib/utils"

type AppRunControlsProps = {
  appId: number
  activeConfigSetId: number | null
  /** True while the app is running, building or being set up: all hold the app's one session. */
  running: boolean
  /** Kind of the current session. For a build or setup, Stop ends it and Reload is unavailable. */
  kind?: string
  onStatus: (status: StatusEvent) => void
  /** "icon" is the full set of actions as icon buttons with tooltips. */
  variant?: "default" | "compact" | "icon"
  showSetup?: boolean
  className?: string
}

export function AppRunControls({
  appId,
  activeConfigSetId,
  running,
  kind,
  onStatus,
  variant = "default",
  showSetup = false,
  className,
}: AppRunControlsProps) {
  const [busy, setBusy] = useState(false)
  const [pending, setPending] = useState<{
    action: CommandKind
    success: string
    failure: string
    sets: ConfigSet[]
    configSetId: number
  } | null>(null)
  const selectId = useId()
  const task = running && isTaskKind(kind)
  const taskName = kind === "setup" ? "setup" : "build"
  const compact = variant === "compact"

  async function runAction(
    action: "setup" | "dev" | "prod" | "build" | "stop" | "reload",
    success: string,
    failure: string
  ) {
    if (action === "stop" || action === "reload") {
      return execute(action, success, failure)
    }
    setBusy(true)
    try {
      const sets = await api.configSets.list(appId)
      if (!sets.length) throw new Error("No config sets available")
      if (sets.length === 1) {
        await execute(action, success, failure, sets[0].id)
      } else {
        setPending({
          action,
          success,
          failure,
          sets,
          configSetId:
            sets.find((set) => set.id === activeConfigSetId)?.id ?? sets[0].id,
        })
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : failure)
    } finally {
      setBusy(false)
    }
  }

  async function execute(
    action: CommandKind | "stop" | "reload",
    success: string,
    failure: string,
    configSetId = 0
  ) {
    setBusy(true)
    try {
      const next =
        action === "stop" || action === "reload"
          ? await api.runner[action](appId)
          : await api.runner[action](appId, configSetId)
      setPending(null)
      onStatus(next)
      toast.success(success)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : failure)
    } finally {
      setBusy(false)
    }
  }

  function withDialog(controls: ReactNode) {
    return (
      <>
        {controls}
        <Dialog
          open={pending !== null}
          onOpenChange={(open) => {
            if (!open && !busy) setPending(null)
          }}
        >
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Start {pending?.action}</DialogTitle>
              <DialogDescription>
                Choose the config set for this run. Your app's active config set
                stays unchanged.
              </DialogDescription>
            </DialogHeader>
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor={selectId}>Config set</FieldLabel>
                <Select
                  items={
                    pending?.sets.map((set) => ({
                      value: String(set.id),
                      label: set.name,
                    })) ?? []
                  }
                  value={pending ? String(pending.configSetId) : null}
                  onValueChange={(value) => {
                    if (value !== null)
                      setPending((current) =>
                        current
                          ? { ...current, configSetId: Number(value) }
                          : null
                      )
                  }}
                  disabled={busy}
                >
                  <SelectTrigger id={selectId} className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {pending?.sets.map((set) => (
                        <SelectItem key={set.id} value={String(set.id)}>
                          {set.name}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </Field>
            </FieldGroup>
            <DialogFooter>
              <Button
                variant="outline"
                disabled={busy}
                onClick={() => setPending(null)}
              >
                Cancel
              </Button>
              <Button
                disabled={busy || running || !pending}
                onClick={() => {
                  if (pending)
                    void execute(
                      pending.action,
                      pending.success,
                      pending.failure,
                      pending.configSetId
                    )
                }}
              >
                Start {pending?.action}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </>
    )
  }

  if (compact) {
    const actions = [
      { action: "setup", label: "Setup", Icon: WrenchIcon },
      { action: "dev", label: "Dev", Icon: PlayIcon },
      { action: "build", label: "Build", Icon: HammerIcon },
      { action: "prod", label: "Prod", Icon: RocketIcon },
    ] as const
    return withDialog(
      <div
        className={cn("flex items-center", className)}
        onClick={(event) => event.stopPropagation()}
        onKeyDown={(event) => event.stopPropagation()}
      >
        {actions.filter(({ action }) => showSetup || action !== "setup").map(({ action, label, Icon }) => {
          const stop = running && action === "dev"
          const title = stop ? "Stop" : `Start ${label.toLowerCase()}`
          return (
            <Tooltip key={action}>
              <TooltipTrigger
                render={
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-xs"
                    aria-label={title}
                    disabled={busy || (running && !stop)}
                    onClick={() => void (stop
                      ? runAction("stop", "Stopped", "Failed to stop")
                      : runAction(action, `${label} started`, `Failed to start ${label.toLowerCase()}`))}
                  />
                }
              >
                {stop ? <SquareIcon /> : <Icon />}
              </TooltipTrigger>
              <TooltipContent>{title}</TooltipContent>
            </Tooltip>
          )
        })}
      </div>
    )
  }

  if (variant === "icon") {
    const actions = [
      {
        label: "Setup",
        hint: "Run setup commands",
        icon: <WrenchIcon />,
        variant: "outline",
        disabled: busy || running,
        onClick: () => runAction("setup", "Setup started", "Failed to run setup"),
      },
      {
        label: "Dev",
        hint: "Start development commands",
        icon: <PlayIcon />,
        variant: "default",
        className: running
          ? ""
          : "bg-emerald-600 text-white hover:bg-emerald-700",
        disabled: busy || running,
        onClick: () => runAction("dev", "Dev started", "Failed to start dev"),
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
        label: "Prod",
        hint: "Start production commands",
        icon: <RocketIcon />,
        variant: "outline",
        disabled: busy || running,
        onClick: () => runAction("prod", "Prod started", "Failed to start prod"),
      },
      {
        label: "Stop",
        hint: task ? `Stop the ${taskName}` : "Stop the app",
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
        disabled: busy || task,
        onClick: () => runAction("reload", "Reloaded", "Failed to reload"),
      },
    ] as const
    return withDialog(
      <div className={cn("flex flex-wrap items-center gap-2", className)}>
        {actions.filter(({ label }) => showSetup || label !== "Setup").map((action) => (
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

  return withDialog(
    <div className={cn("flex flex-wrap gap-2", className)}>
      {showSetup ? <Button
        size="sm"
        variant="outline"
        disabled={busy || running}
        onClick={() => void runAction("setup", "Setup started", "Failed to run setup")}
      >
        <WrenchIcon data-icon="inline-start" />
        Setup
      </Button> : null}
      <Button
        size="sm"
        disabled={busy || running}
        onClick={() => void runAction("dev", "Dev started", "Failed to start dev")}
      >
        <PlayIcon data-icon="inline-start" />
        Dev
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
        variant="outline"
        disabled={busy || running}
        onClick={() => void runAction("prod", "Prod started", "Failed to start prod")}
      >
        <RocketIcon data-icon="inline-start" />
        Prod
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
        disabled={busy || task}
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
