import { useState } from "react"
import { toast } from "sonner"
import { ChevronDownIcon, PlayIcon, SquareIcon } from "lucide-react"
import { api } from "@/lib/api"
import type { App, StatusEvent, WorkspaceRunResult } from "@/lib/types"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

type WorkspaceRunControlsProps = {
  workspaceId: number
  apps: App[]
  statusByAppId: Record<number, StatusEvent>
  /** Called with the saved app after its "exclude from Start all" flag changes. */
  onAppChange: (app: App) => void
}

/** Summarizes a workspace-wide start or stop for a toast. */
function summarize(result: WorkspaceRunResult, done: "started" | "stopped") {
  const count = (status: string) =>
    result.results.filter((r) => r.status === status).length
  const errors = result.results.filter((r) => r.status === "error")
  const parts = [`${count(done)} ${done}`]
  if (count("skipped") > 0) parts.push(`${count("skipped")} skipped`)
  if (count("excluded") > 0) parts.push(`${count("excluded")} excluded`)
  if (errors.length > 0) parts.push(`${errors.length} failed`)
  return {
    title: parts.join(", "),
    errors: errors.map((r) => `${r.name}: ${r.message ?? "failed"}`).join("\n"),
    failed: errors.length > 0,
  }
}

export function WorkspaceRunControls({
  workspaceId,
  apps,
  statusByAppId,
  onAppChange,
}: WorkspaceRunControlsProps) {
  const [busy, setBusy] = useState<"start" | "stop" | null>(null)
  const [sequential, setSequential] = useState(false)

  const anyRunning = apps.some((app) => statusByAppId[app.id]?.running)
  const includedCount = apps.filter((app) => !app.exclude_from_start_all).length

  async function run(action: "start" | "stop") {
    setBusy(action)
    try {
      const result =
        action === "start"
          ? await api.runner.startWorkspace(workspaceId, [], sequential)
          : await api.runner.stopWorkspace(workspaceId, [])
      const { title, errors, failed } = summarize(
        result,
        action === "start" ? "started" : "stopped"
      )
      if (failed) toast.error(title, { description: errors })
      else toast.success(title)
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : `Failed to ${action} the workspace's apps`
      )
    } finally {
      setBusy(null)
    }
  }

  async function setExcluded(app: App, excluded: boolean) {
    try {
      onAppChange(
        await api.apps.update(app.id, { exclude_from_start_all: excluded })
      )
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to save the preference"
      )
    }
  }

  return (
    <div className="flex items-center gap-2">
      <Button
        variant="outline"
        size="sm"
        disabled={busy !== null || includedCount === 0}
        onClick={() => void run("start")}
      >
        <PlayIcon data-icon="inline-start" />
        Start all
      </Button>
      <Button
        variant="outline"
        size="sm"
        disabled={busy !== null || !anyRunning}
        onClick={() => void run("stop")}
      >
        <SquareIcon data-icon="inline-start" />
        Stop all
      </Button>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button
              variant="ghost"
              size="sm"
              aria-label="Start all options"
            />
          }
        >
          Options
          <ChevronDownIcon data-icon="inline-end" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="min-w-64">
          <DropdownMenuGroup>
            <DropdownMenuLabel>Include in Start all</DropdownMenuLabel>
            {apps.map((app) => (
              <DropdownMenuCheckboxItem
                key={app.id}
                checked={!app.exclude_from_start_all}
                onCheckedChange={(checked) => void setExcluded(app, !checked)}
              >
                <span className="truncate">{app.name}</span>
              </DropdownMenuCheckboxItem>
            ))}
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          <DropdownMenuCheckboxItem
            checked={sequential}
            onCheckedChange={setSequential}
          >
            Start one after another
          </DropdownMenuCheckboxItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}
