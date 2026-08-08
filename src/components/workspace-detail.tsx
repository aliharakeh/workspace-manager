import { AppWindowIcon, ExternalLinkIcon, PlusIcon } from "lucide-react"
import type { App, StatusEvent, Workspace } from "@/lib/types"
import { AppRunControls, AppStatusDot } from "@/components/app-run-controls"
import { ConfigSetPicker } from "@/components/config-set-picker"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"

type WorkspaceDetailProps = {
  workspace: Workspace
  apps: App[]
  statusByAppId: Record<number, StatusEvent>
  onSelectApp: (appId: number) => void
  onCreateApp: () => void
  onStatus: (status: StatusEvent) => void
  onAppChange: (app: App) => void
}

export function WorkspaceDetail({
  workspace,
  apps,
  statusByAppId,
  onSelectApp,
  onCreateApp,
  onStatus,
  onAppChange,
}: WorkspaceDetailProps) {
  const runningCount = apps.filter(
    (app) => statusByAppId[app.id]?.running
  ).length

  if (apps.length === 0) {
    return (
      <div className="flex flex-1 items-center justify-center p-6">
        <Empty className="max-w-md border">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <AppWindowIcon />
            </EmptyMedia>
            <EmptyTitle>{workspace.name}</EmptyTitle>
            <EmptyDescription>
              This workspace has no apps yet. Add one to configure env, templates,
              and run commands.
            </EmptyDescription>
          </EmptyHeader>
          <EmptyContent>
            <Button onClick={onCreateApp}>
              <PlusIcon data-icon="inline-start" />
              Add app
            </Button>
          </EmptyContent>
        </Empty>
      </div>
    )
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4 p-4 md:p-6">
      <div className="flex flex-col gap-1">
        <div className="flex flex-wrap items-center gap-2">
          <h1 className="truncate text-xl font-medium tracking-tight">
            {workspace.name}
          </h1>
          <Badge variant="outline">
            {apps.length} app{apps.length === 1 ? "" : "s"}
          </Badge>
          <Badge variant={runningCount > 0 ? "default" : "outline"}>
            {runningCount} running
          </Badge>
        </div>
        <p className="text-sm text-muted-foreground">
          Overview of apps in this workspace. Open an app for env, templates, and
          logs.
        </p>
      </div>

      <ul className="flex flex-col gap-2">
        {apps.map((app) => {
          const status = statusByAppId[app.id]
          const running = !!status?.running
          const processCount = status?.processes.length ?? 0
          const activeProcesses = status?.processes.filter(
            (process) => process.status === "running"
          ).length
          const readyUrls = running
            ? [
                ...new Set(
                  (status?.processes ?? []).flatMap((p) => p.urls ?? [])
                ),
              ]
            : []

          return (
            <li key={app.id}>
              <div className="flex flex-col gap-3 rounded-xl px-4 py-3 ring-1 ring-foreground/10">
                <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <button
                    type="button"
                    className="min-w-0 flex-1 text-left outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    onClick={() => onSelectApp(app.id)}
                  >
                    <div className="flex items-center gap-2">
                      <AppStatusDot running={running} />
                      <span className="truncate font-medium">{app.name}</span>
                      <Badge variant={running ? "default" : "outline"}>
                        {running ? "Running" : "Idle"}
                      </Badge>
                    </div>
                    <p className="mt-1 truncate font-mono text-xs text-muted-foreground">
                      {app.project_path}
                    </p>
                    {running && processCount > 0 ? (
                      <p className="mt-1 text-xs text-muted-foreground">
                        {activeProcesses}/{processCount} process
                        {processCount === 1 ? "" : "es"} active
                      </p>
                    ) : null}
                    {status?.error ? (
                      <p className="mt-1 text-xs text-destructive">
                        {status.error}
                      </p>
                    ) : null}
                  </button>
                  <div className="flex flex-wrap items-center gap-2">
                    <ConfigSetPicker
                      app={app}
                      onAppChange={onAppChange}
                      stopPropagation
                    />
                    <AppRunControls
                      appId={app.id}
                      running={running}
                      onStatus={onStatus}
                    />
                  </div>
                </div>
                {readyUrls.length > 0 ? (
                  <ul className="flex flex-wrap gap-1.5 border-t border-border/60 pt-3">
                    {readyUrls.map((url) => (
                      <li key={url} className="min-w-0 max-w-full">
                        <Badge
                          variant="secondary"
                          className="max-w-full font-mono"
                          title={url}
                          render={
                            <a
                              href={url}
                              target="_blank"
                              rel="noreferrer"
                              onClick={(event) => event.stopPropagation()}
                            />
                          }
                        >
                          <ExternalLinkIcon data-icon="inline-start" />
                          <span className="truncate">{url}</span>
                        </Badge>
                      </li>
                    ))}
                  </ul>
                ) : null}
              </div>
            </li>
          )
        })}
      </ul>
    </div>
  )
}
