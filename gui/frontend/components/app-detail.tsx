import { useMemo, useState } from "react"
import {
  ExternalLinkIcon,
  GitBranchIcon,
  PencilIcon,
  Trash2Icon,
} from "lucide-react"
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
import { GitPanel } from "@/components/git-panel"
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
  /** Bumped when another client writes this app's data (see api.DataChanged). */
  dataEpoch: number
  onTabChange: (tab: AppTab) => void
  onEdit: () => void
  onDelete: () => void
  onStatus: (status: StatusEvent) => void
  onAppChange: (app: App) => void
}

const TAB_TRIGGER =
  "h-8 flex-none rounded-full border-transparent px-4 text-sm font-medium text-muted-foreground hover:bg-background/60 hover:text-foreground data-active:border-border data-active:bg-background data-active:text-foreground data-active:shadow-sm dark:data-active:border-border dark:data-active:bg-background dark:data-active:text-foreground"

export function AppDetail({
  app,
  status,
  tab,
  dataEpoch,
  onTabChange,
  onEdit,
  onDelete,
  onStatus,
  onAppChange,
}: AppDetailProps) {
  const running = !!status?.running
  // An agent's run (prepared through the MCP server) cannot be driven from here.
  const byAgent = running && !!status?.owner
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

  const panelKey = `${app.active_config_set_id ?? "none"}-${panelEpoch}-${dataEpoch}`

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
              {byAgent ? " · by agent" : null}
            </Badge>
          </div>
          <p className="mt-1 truncate font-mono text-xs text-muted-foreground">
            {app.project_path}
          </p>
          {running && status?.worktree ? (
            <p className="mt-1 flex items-center gap-1 truncate text-xs text-muted-foreground">
              <GitBranchIcon className="size-3.5 shrink-0" />
              In worktree
              <span className="truncate font-mono">{status.worktree}</span>
            </p>
          ) : null}
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
        <div className="flex shrink-0 flex-wrap items-center gap-2">
          {byAgent ? (
            <span className="text-xs text-muted-foreground">
              Run by an agent
            </span>
          ) : (
            <AppRunControls
              appId={app.id}
              running={running}
              kind={status?.kind}
              onStatus={handleStatus}
              variant="icon"
              showSetup
            />
          )}
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
        <TabsList className="max-w-full shrink-0 flex-wrap justify-start gap-1 rounded-full border border-border/60 bg-muted/70 p-1 group-data-horizontal/tabs:h-auto">
          <TabsTrigger
            value="env"
            className={TAB_TRIGGER}
          >
            Env
          </TabsTrigger>
          <TabsTrigger
            value="templates"
            className={TAB_TRIGGER}
          >
            Templates
          </TabsTrigger>
          <TabsTrigger
            value="setup"
            className={TAB_TRIGGER}
          >
            Setup
          </TabsTrigger>
          <TabsTrigger
            value="dev"
            className={TAB_TRIGGER}
          >
            Dev
          </TabsTrigger>
          <TabsTrigger
            value="build"
            className={TAB_TRIGGER}
          >
            Build
          </TabsTrigger>
          <TabsTrigger value="prod" className={TAB_TRIGGER}>
            Prod
          </TabsTrigger>
          <TabsTrigger
            value="git"
            className={TAB_TRIGGER}
          >
            Git
          </TabsTrigger>
          <TabsTrigger
            value="ai"
            className={TAB_TRIGGER}
          >
            AI
          </TabsTrigger>
          <TabsTrigger
            value="logs"
            className={TAB_TRIGGER}
          >
            Logs
          </TabsTrigger>
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
          />
        </TabsContent>
        <TabsContent value="dev" className="mt-4 min-h-0 overflow-y-auto">
          <CommandConfigPanel key={panelKey} appId={app.id} kind="dev" />
        </TabsContent>
        <TabsContent value="prod" className="mt-4 min-h-0 overflow-y-auto">
          <CommandConfigPanel key={panelKey} appId={app.id} kind="prod" />
        </TabsContent>
        <TabsContent value="build" className="mt-4 min-h-0 overflow-y-auto">
          <CommandConfigPanel key={panelKey} appId={app.id} kind="build" />
        </TabsContent>
        <TabsContent value="git" className="mt-4 min-h-0 overflow-y-auto">
          <GitPanel
            appId={app.id}
            active={tab === "git"}
            dataEpoch={dataEpoch}
            activeConfigSetId={app.active_config_set_id ?? null}
            status={status}
            onStatus={handleStatus}
          />
        </TabsContent>
        <TabsContent value="ai"className="flex min-h-0 flex-1 flex-col overflow-hidden pt-4" keepMounted>
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
