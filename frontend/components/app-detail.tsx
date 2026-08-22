import { useMemo, useState } from "react"
import { toast } from "sonner"
import { CodeIcon, ExternalLinkIcon, PencilIcon, Trash2Icon } from "lucide-react"
import { api, handleReadyUrlClick } from "@/lib/api"
import type { App, StatusEvent } from "@/lib/types"
import type { LogLine } from "@/hooks/use-runner-logs"
import type { AppTab } from "@/lib/routes"
import { AppRunControls } from "@/components/app-run-controls"
import { ConfigSetSwitcher } from "@/components/config-set-switcher"
import { AppAIPanel } from "@/components/app-ai-panel"
import { EnvVarsPanel } from "@/components/env-vars-panel"
import { TemplatesPanel } from "@/components/templates-panel"
import { RunConfigPanel } from "@/components/run-config-panel"
import { LogsPanel } from "@/components/logs-panel"
import { Button, buttonVariants } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Separator } from "@/components/ui/separator"
import { cn } from "@/lib/utils"

type AppDetailProps = {
  app: App
  status: StatusEvent | null
  logs: LogLine[]
  connected: boolean
  tab: AppTab
  onTabChange: (tab: AppTab) => void
  onEdit: () => void
  onDelete: () => void
  onStatus: (status: StatusEvent) => void
  onAppChange: (app: App) => void
}

export function AppDetail({
  app,
  status,
  logs,
  connected,
  tab,
  onTabChange,
  onEdit,
  onDelete,
  onStatus,
  onAppChange,
}: AppDetailProps) {
  const running = !!status?.running
  const [panelEpoch, setPanelEpoch] = useState(0)
  const readyUrls = useMemo(() => {
    if (!running) return []
    return [
      ...new Set((status?.processes ?? []).flatMap((p) => p.urls ?? [])),
    ]
  }, [running, status?.processes])

  function handleStatus(next: StatusEvent) {
    onStatus(next)
    if (next.running) onTabChange("logs")
  }

  function handleAppChange(next: App) {
    onAppChange(next)
    setPanelEpoch((n) => n + 1)
  }

  const panelKey = `${app.active_config_set_id ?? "none"}-${panelEpoch}`

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4 p-4 md:p-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <h1 className="truncate text-xl font-medium tracking-tight">
              {app.name}
            </h1>
            <Badge variant={running ? "default" : "outline"}>
              {running ? "Running" : "Idle"}
            </Badge>
          </div>
          <p className="mt-1 truncate font-mono text-xs text-muted-foreground">
            {app.project_path}
          </p>
          <div className="mt-3">
            <ConfigSetSwitcher app={app} onAppChange={handleAppChange} />
          </div>
          {status?.error ? (
            <p className="mt-2 text-sm text-destructive">{status.error}</p>
          ) : null}
        </div>
        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              void api.apps
                .openInEditor(app.id)
                .then(() => toast.success("Opened in editor"))
                .catch((err) =>
                  toast.error(
                    err instanceof Error
                      ? err.message
                      : "Failed to open in editor"
                  )
                )
            }}
          >
            <CodeIcon data-icon="inline-start" />
            Editor
          </Button>
          <Button variant="outline" size="sm" onClick={onEdit}>
            <PencilIcon data-icon="inline-start" />
            Edit
          </Button>
          <Button variant="outline" size="sm" onClick={onDelete}>
            <Trash2Icon data-icon="inline-start" />
            Delete
          </Button>
          <Separator orientation="vertical" className="hidden h-8 sm:block" />
          <AppRunControls
            appId={app.id}
            running={running}
            onStatus={handleStatus}
          />
          {readyUrls.map((url) => (
            <a
              key={url}
              href={url}
              target="_blank"
              rel="noreferrer"
              onClick={(event) => handleReadyUrlClick(event, url)}
              className={cn(
                buttonVariants({ variant: "outline", size: "sm" }),
                "max-w-64"
              )}
              title={url}
            >
              <ExternalLinkIcon data-icon="inline-start" />
              <span className="truncate font-mono text-xs">{url}</span>
            </a>
          ))}
        </div>
      </div>

      <Tabs
        value={tab}
        onValueChange={(value) => {
          if (value) onTabChange(value as AppTab)
        }}
        className="min-h-0 flex-1"
      >
        <TabsList>
          <TabsTrigger value="env">Env</TabsTrigger>
          <TabsTrigger value="templates">Templates</TabsTrigger>
          <TabsTrigger value="run">Run</TabsTrigger>
          <TabsTrigger value="ai">AI</TabsTrigger>
          <TabsTrigger value="logs">Logs</TabsTrigger>
        </TabsList>
        <TabsContent value="env" className="mt-4">
          <EnvVarsPanel key={panelKey} appId={app.id} />
        </TabsContent>
        <TabsContent value="templates" className="mt-4 min-h-0">
          <TemplatesPanel key={panelKey} appId={app.id} />
        </TabsContent>
        <TabsContent value="run" className="mt-4">
          <RunConfigPanel key={panelKey} appId={app.id} />
        </TabsContent>
        <TabsContent value="ai" className="mt-4 flex min-h-0 flex-col" keepMounted>
          <AppAIPanel
            app={app}
            onApplied={() => setPanelEpoch((n) => n + 1)}
          />
        </TabsContent>
        <TabsContent value="logs" className="mt-4">
          <LogsPanel status={status} logs={logs} connected={connected} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
