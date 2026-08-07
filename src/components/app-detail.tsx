import { useState } from "react"
import { PencilIcon, Trash2Icon } from "lucide-react"
import type { App, StatusEvent } from "@/lib/types"
import type { LogLine } from "@/hooks/use-runner-logs"
import { AppRunControls } from "@/components/app-run-controls"
import { EnvVarsPanel } from "@/components/env-vars-panel"
import { TemplatesPanel } from "@/components/templates-panel"
import { RunConfigPanel } from "@/components/run-config-panel"
import { LogsPanel } from "@/components/logs-panel"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Separator } from "@/components/ui/separator"

type AppDetailProps = {
  app: App
  status: StatusEvent | null
  logs: LogLine[]
  connected: boolean
  onEdit: () => void
  onDelete: () => void
  onStatus: (status: StatusEvent) => void
}

export function AppDetail({
  app,
  status,
  logs,
  connected,
  onEdit,
  onDelete,
  onStatus,
}: AppDetailProps) {
  const running = !!status?.running
  const [tab, setTab] = useState("env")

  function handleStatus(next: StatusEvent) {
    onStatus(next)
    if (next.running) setTab("logs")
  }

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
          {status?.error ? (
            <p className="mt-2 text-sm text-destructive">{status.error}</p>
          ) : null}
        </div>
        <div className="flex flex-wrap gap-2">
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
        </div>
      </div>

      <Tabs
        value={tab}
        onValueChange={(value) => {
          if (value) setTab(value)
        }}
        className="min-h-0 flex-1"
      >
        <TabsList>
          <TabsTrigger value="env">Env</TabsTrigger>
          <TabsTrigger value="templates">Templates</TabsTrigger>
          <TabsTrigger value="run">Run</TabsTrigger>
          <TabsTrigger value="logs">Logs</TabsTrigger>
        </TabsList>
        <TabsContent value="env" className="mt-4">
          <EnvVarsPanel appId={app.id} />
        </TabsContent>
        <TabsContent value="templates" className="mt-4 min-h-0">
          <TemplatesPanel appId={app.id} />
        </TabsContent>
        <TabsContent value="run" className="mt-4">
          <RunConfigPanel appId={app.id} />
        </TabsContent>
        <TabsContent value="logs" className="mt-4">
          <LogsPanel status={status} logs={logs} connected={connected} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
