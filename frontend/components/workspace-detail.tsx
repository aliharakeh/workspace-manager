import { useState } from "react"
import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core"
import {
  SortableContext,
  arrayMove,
  sortableKeyboardCoordinates,
  useSortable,
  rectSortingStrategy,
} from "@dnd-kit/sortable"
import { CSS } from "@dnd-kit/utilities"
import {
  AppWindowIcon,
  ExternalLinkIcon,
  GripVerticalIcon,
  LayoutTemplateIcon,
  PlusIcon,
} from "lucide-react"
import { toast } from "sonner"
import { api, handleReadyUrlClick } from "@/lib/api"
import type { App, StatusEvent, Workspace } from "@/lib/types"
import { AppRunControls } from "@/components/app-run-controls"
import { ConfigSetPicker } from "@/components/config-set-picker"
import { OpenEditorButton } from "@/components/open-editor-button"
import { OpenFolderButton } from "@/components/open-folder-button"
import { WorkspaceColorPicker } from "@/components/workspace-color-picker"
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
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { cn } from "@/lib/utils"

type WorkspaceDetailProps = {
  workspace: Workspace
  apps: App[]
  statusByAppId: Record<number, StatusEvent>
  onSelectApp: (appId: number) => void
  onCreateApp: () => void
  onCreateAppFromBlueprint: () => void
  onStatus: (status: StatusEvent) => void
  onAppChange: (app: App) => void
  /** Called with the saved workspace after its color changes. */
  onWorkspaceChange: (workspace: Workspace) => void
  /** Called with every app id in the new order after a drag. */
  onReorderApps: (orderedIds: number[]) => void
}

export function WorkspaceDetail({
  workspace,
  apps,
  statusByAppId,
  onSelectApp,
  onCreateApp,
  onCreateAppFromBlueprint,
  onStatus,
  onAppChange,
  onWorkspaceChange,
  onReorderApps,
}: WorkspaceDetailProps) {
  const [savingColor, setSavingColor] = useState(false)

  async function handleColorChange(color: string | null) {
    if (color === workspace.color) return
    setSavingColor(true)
    try {
      onWorkspaceChange(
        await api.workspaces.update(workspace.id, { color: color ?? "" }) // "" clears it
      )
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to save color")
    } finally {
      setSavingColor(false)
    }
  }

  const colorPicker = (compact: boolean) => (
    <WorkspaceColorPicker
      value={workspace.color}
      onChange={(color) => void handleColorChange(color)}
      disabled={savingColor}
      compact={compact}
    />
  )

  const sensors = useSensors(
    // A small drag distance keeps a plain click on the handle from starting a drag.
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  )

  function handleDragEnd({ active, over }: DragEndEvent) {
    if (!over || active.id === over.id) return
    const ids = apps.map((app) => app.id)
    const from = ids.indexOf(Number(active.id))
    const to = ids.indexOf(Number(over.id))
    if (from < 0 || to < 0) return
    onReorderApps(arrayMove(ids, from, to))
  }

  const buildingCount = apps.filter((app) => {
    const status = statusByAppId[app.id]
    return status?.running && status.kind === "build"
  }).length
  const runningCount =
    apps.filter((app) => statusByAppId[app.id]?.running).length - buildingCount

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
              This workspace has no apps yet. Add one to configure env,
              templates, and run commands.
            </EmptyDescription>
          </EmptyHeader>
          <EmptyContent>
            <Button onClick={onCreateApp}>
              <PlusIcon data-icon="inline-start" />
              Add app
            </Button>
            <Button variant="outline" onClick={onCreateAppFromBlueprint}>
              <LayoutTemplateIcon data-icon="inline-start" />
              From blueprint
            </Button>
          </EmptyContent>
          {colorPicker(false)}
        </Empty>
      </div>
    )
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4 p-4 md:p-6">
      <div className="flex items-start justify-between gap-4">
        <div className="flex min-w-0 flex-col gap-1">
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
            {buildingCount > 0 ? (
              <Badge variant="default">{buildingCount} building</Badge>
            ) : null}
          </div>
          <p className="text-sm text-muted-foreground">
            Overview of apps in this workspace. Open an app for env, templates,
            and logs. Drag an app by its handle to reorder it.
          </p>
        </div>
        <div className="shrink-0 pt-1.5">{colorPicker(true)}</div>
      </div>

      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragEnd={handleDragEnd}
      >
        <SortableContext
          items={apps.map((app) => app.id)}
          strategy={rectSortingStrategy}
        >
          <ul className="grid grid-cols-2 gap-4">
            {apps.map((app) => (
              <SortableAppCard
                key={app.id}
                app={app}
                status={statusByAppId[app.id]}
                onSelectApp={onSelectApp}
                onStatus={onStatus}
                onAppChange={onAppChange}
              />
            ))}
          </ul>
        </SortableContext>
      </DndContext>
    </div>
  )
}

type SortableAppCardProps = {
  app: App
  status: StatusEvent | undefined
  onSelectApp: (appId: number) => void
  onStatus: (status: StatusEvent) => void
  onAppChange: (app: App) => void
}

function SortableAppCard({
  app,
  status,
  onSelectApp,
  onStatus,
  onAppChange,
}: SortableAppCardProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    setActivatorNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: app.id })

  const running = !!status?.running
  const building = running && status?.kind === "build"
  const readyUrls = running
    ? [...new Set((status?.processes ?? []).flatMap((p) => p.urls ?? []))]
    : []

  return (
    <li
      ref={setNodeRef}
      style={{
        transform: CSS.Translate.toString(transform),
        transition,
      }}
      className={cn("relative min-w-0", isDragging && "z-10")}
    >
      <div
        className={cn(
          "flex h-full flex-col gap-3 rounded-xl bg-background p-4",
          !running
            ? "ring-1 ring-foreground/10"
            : building
              ? "ring-2 ring-amber-500"
              : "ring-2 ring-emerald-500",
          isDragging && "shadow-lg"
        )}
      >
        <div className="flex items-start gap-2">
          <button
            type="button"
            className="min-w-0 flex-1 text-left outline-none focus-visible:ring-2 focus-visible:ring-ring"
            onClick={() => onSelectApp(app.id)}
          >
            <span className="block truncate text-base font-semibold tracking-tight text-foreground">
              {app.name}
            </span>
            <p className="mt-1 truncate font-mono text-xs text-muted-foreground">
              {app.project_path}
            </p>
            {status?.error ? (
              <p className="mt-1 text-xs text-destructive">{status.error}</p>
            ) : null}
          </button>
          <div className="-mt-1 -mr-2 flex shrink-0 items-center gap-1">
            <OpenFolderButton
              appId={app.id}
              iconOnly
              className="border-yellow-200 bg-yellow-100 text-yellow-900 hover:bg-yellow-200 hover:text-yellow-900 dark:border-transparent dark:bg-yellow-500/20 dark:text-yellow-200 dark:hover:bg-yellow-500/30"
            />
            <OpenEditorButton
              appId={app.id}
              iconOnly
              className="border-blue-200 bg-blue-100 text-blue-900 hover:bg-blue-200 hover:text-blue-900 aria-expanded:bg-blue-200 aria-expanded:text-blue-900 dark:border-transparent dark:bg-blue-500/20 dark:text-blue-200 dark:hover:bg-blue-500/30 dark:aria-expanded:bg-blue-500/30"
            />
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    ref={setActivatorNodeRef}
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    className="shrink-0 cursor-grab touch-none text-muted-foreground active:cursor-grabbing"
                    aria-label={`Reorder ${app.name}`}
                    {...attributes}
                    {...listeners}
                  />
                }
              >
                <GripVerticalIcon />
              </TooltipTrigger>
              <TooltipContent>Drag to reorder</TooltipContent>
            </Tooltip>
          </div>
        </div>
        {readyUrls.length > 0 ? (
          <ul className="flex flex-wrap gap-1.5">
            {readyUrls.map((url) => (
              <li key={url} className="max-w-full min-w-0">
                <Badge
                  variant="secondary"
                  className="max-w-full font-mono"
                  title={url}
                  render={
                    <a
                      href={url}
                      target="_blank"
                      rel="noreferrer"
                      onClick={(event) => handleReadyUrlClick(event, url)}
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
        <div className="mt-auto flex flex-wrap items-center justify-between gap-2 border-t border-border/60 pt-3">
          <ConfigSetPicker
            app={app}
            onAppChange={onAppChange}
            stopPropagation
          />
          <AppRunControls
            appId={app.id}
            running={running}
            building={building}
            onStatus={onStatus}
            variant="icon"
          />
        </div>
      </div>
    </li>
  )
}
