import { useMemo, useState } from "react"
import { ExternalLinkIcon, PencilIcon, Trash2Icon } from "lucide-react"
import { handleReadyUrlClick } from "@/lib/api"
import type { App, StatusEvent } from "@/lib/types"
import type { AppTab } from "@/lib/routes"
import { AppRunControls } from "@/components/app-run-controls"
import { OpenEditorButton } from "@/components/open-editor-button"
import { OpenFolderButton } from "@/components/open-folder-button"
import { appStateLabel } from "@/lib/app-state"
import { ConfigSetSwitcher } from "@/components/config-set-switcher"
import { AppAIPanel } from "@/components/app-ai-panel"
import { EnvVarsPanel } from "@/components/env-vars-panel"
import { TemplatesPanel } from "@/components/templates-panel"
import { CommandConfigPanel } from "@/components/command-config-panel"
import { LogsPanel } from "@/components/logs-panel"
import { Button, buttonVariants } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Separator } from "@/components/ui/separator"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { cn } from "@/lib/utils"

type AppDetailProps = {
  app: App
  status: StatusEvent | null
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
    <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden p-4 md:p-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <h1 className="truncate text-xl font-medium tracking-tight">
              {app.name}
            </h1>
            <Badge variant={running ? "default" : "outline"}>
              {appStateLabel(running, status?.kind)}
            </Badge>
          </div>
          <p className="mt-1 truncate font-mono text-xs text-muted-foreground">
            {app.project_path}
          </p>
          <div className="mt-3">
            <ConfigSetSwitcher app={app} onAppChange={handleAppChange} />
          </div>
          {readyUrls.length > 0 ? (
            <div className="mt-3 flex flex-wrap gap-2">
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
          ) : null}
          {status?.error ? (
            <p className="mt-2 text-sm text-destructive">{status.error}</p>
          ) : null}
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <AppRunControls
            appId={app.id}
            running={running}
            kind={status?.kind}
            onStatus={handleStatus}
            variant="icon"
          />
          <Separator orientation="vertical" className="hidden h-8 sm:block" />
          <OpenFolderButton appId={app.id} iconOnly />
          <OpenEditorButton appId={app.id} iconOnly />
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant="outline"
                  size="icon-sm"
                  aria-label="Edit app"
                  onClick={onEdit}
                />
              }
            >
              <PencilIcon />
            </TooltipTrigger>
            <TooltipContent>Edit the app's name and project path</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant="outline"
                  size="icon-sm"
                  aria-label="Delete app"
                  onClick={onDelete}
                />
              }
            >
              <Trash2Icon />
            </TooltipTrigger>
            <TooltipContent>Delete this app</TooltipContent>
          </Tooltip>
        </div>
      </div>

      <Tabs
        value={tab}
        onValueChange={(value) => {
          if (value) onTabChange(value as AppTab)
        }}
        className="min-h-0 flex-1 overflow-hidden"
      >
        <TabsList>
          <TabsTrigger value="env">Env</TabsTrigger>
          <TabsTrigger value="templates">Templates</TabsTrigger>
          <TabsTrigger value="setup">Setup</TabsTrigger>
          <TabsTrigger value="run">Run</TabsTrigger>
          <TabsTrigger value="build">Build</TabsTrigger>
          <TabsTrigger value="ai">AI</TabsTrigger>
          <TabsTrigger value="logs">Logs</TabsTrigger>
        </TabsList>
        <TabsContent value="env" className="mt-4 min-h-0 overflow-y-auto">
          <EnvVarsPanel key={panelKey} appId={app.id} />
        </TabsContent>
        <TabsContent value="templates" className="mt-4 min-h-0 overflow-y-auto">
          <TemplatesPanel key={panelKey} appId={app.id} />
        </TabsContent>
        <TabsContent value="setup" className="mt-4 min-h-0 overflow-y-auto">
          <CommandConfigPanel
            key={panelKey}
            appId={app.id}
            kind="setup"
            status={status}
            onStatus={handleStatus}
          />
        </TabsContent>
        <TabsContent value="run" className="mt-4 min-h-0 overflow-y-auto">
          <CommandConfigPanel key={panelKey} appId={app.id} kind="run" />
        </TabsContent>
        <TabsContent value="build" className="mt-4 min-h-0 overflow-y-auto">
          <CommandConfigPanel key={panelKey} appId={app.id} kind="build" />
        </TabsContent>
        <TabsContent value="ai" className="flex min-h-0 flex-1 flex-col overflow-hidden pt-4" keepMounted>
          <AppAIPanel
            app={app}
            onApplied={() => setPanelEpoch((n) => n + 1)}
          />
        </TabsContent>
        <TabsContent value="logs" className="mt-4 min-h-0 overflow-y-auto">
          <LogsPanel appId={app.id} status={status} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
