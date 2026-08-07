import { useCallback, useEffect, useMemo, useState } from "react"
import { toast } from "sonner"
import { PlusIcon, FolderIcon } from "lucide-react"
import { api } from "@/lib/api"
import type { App as AppEntity, StatusEvent, Workspace } from "@/lib/types"
import { useRunnerLogs } from "@/hooks/use-runner-logs"
import { useAppStatuses } from "@/hooks/use-app-statuses"
import { AppSidebar } from "@/components/app-sidebar"
import { AppDetail } from "@/components/app-detail"
import { WorkspaceDetail } from "@/components/workspace-detail"
import { AppDialog, DeleteAppDialog } from "@/components/app-dialogs"
import {
  WorkspaceDialog,
  DeleteWorkspaceDialog,
} from "@/components/workspace-dialogs"
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
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar"
import { Separator } from "@/components/ui/separator"
import { Toaster } from "@/components/ui/sonner"
import { TooltipProvider } from "@/components/ui/tooltip"

export function App() {
  const [workspaces, setWorkspaces] = useState<Workspace[]>([])
  const [appsByWorkspace, setAppsByWorkspace] = useState<
    Record<number, AppEntity[]>
  >({})
  const [selectedWorkspaceId, setSelectedWorkspaceId] = useState<number | null>(
    null
  )
  const [selectedAppId, setSelectedAppId] = useState<number | null>(null)
  const [loading, setLoading] = useState(true)

  const [workspaceDialogOpen, setWorkspaceDialogOpen] = useState(false)
  const [editingWorkspace, setEditingWorkspace] = useState<Workspace | null>(
    null
  )
  const [deletingWorkspace, setDeletingWorkspace] = useState<Workspace | null>(
    null
  )

  const [appDialogOpen, setAppDialogOpen] = useState(false)
  const [appDialogWorkspaceId, setAppDialogWorkspaceId] = useState<
    number | null
  >(null)
  const [editingApp, setEditingApp] = useState<AppEntity | null>(null)
  const [deletingApp, setDeletingApp] = useState<AppEntity | null>(null)

  const { status, logs, setStatus, connected } = useRunnerLogs(selectedAppId)
  const workspaceIds = useMemo(
    () => workspaces.map((workspace) => workspace.id),
    [workspaces]
  )
  const { statusByAppId, setAppStatus } = useAppStatuses(workspaceIds)

  const selectedWorkspace = useMemo(
    () => workspaces.find((w) => w.id === selectedWorkspaceId) ?? null,
    [workspaces, selectedWorkspaceId]
  )

  const selectedApps = useMemo(
    () =>
      selectedWorkspaceId != null
        ? (appsByWorkspace[selectedWorkspaceId] ?? [])
        : [],
    [appsByWorkspace, selectedWorkspaceId]
  )

  const selectedApp = useMemo(() => {
    if (selectedAppId == null) return null
    return selectedApps.find((a) => a.id === selectedAppId) ?? null
  }, [selectedApps, selectedAppId])

  const handleStatus = useCallback(
    (next: StatusEvent) => {
      setAppStatus(next)
      if (selectedAppId === next.appId) setStatus(next)
    },
    [selectedAppId, setAppStatus, setStatus]
  )

  useEffect(() => {
    if (status) setAppStatus(status)
  }, [status, setAppStatus])

  const loadApps = useCallback(async (workspaceId: number) => {
    const apps = await api.apps.list(workspaceId)
    setAppsByWorkspace((prev) => ({ ...prev, [workspaceId]: apps }))
    return apps
  }, [])

  const bootstrap = useCallback(async () => {
    setLoading(true)
    try {
      const list = await api.workspaces.list()
      setWorkspaces(list)
      if (list.length) {
        const first = list[0]!
        setSelectedWorkspaceId((prev) => prev ?? first.id)
        await Promise.all(list.map((w) => loadApps(w.id)))
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to load data")
    } finally {
      setLoading(false)
    }
  }, [loadApps])

  useEffect(() => {
    void bootstrap()
  }, [bootstrap])

  function handleSelectWorkspace(id: number) {
    setSelectedWorkspaceId(id)
    setSelectedAppId(null)
    if (!appsByWorkspace[id]) void loadApps(id)
  }

  function handleSelectApp(workspaceId: number, appId: number) {
    setSelectedWorkspaceId(workspaceId)
    setSelectedAppId(appId)
  }

  return (
    <TooltipProvider>
      <SidebarProvider>
        <AppSidebar
          workspaces={workspaces}
          appsByWorkspace={appsByWorkspace}
          statusByAppId={statusByAppId}
          selectedWorkspaceId={selectedWorkspaceId}
          selectedAppId={selectedAppId}
          onSelectWorkspace={handleSelectWorkspace}
          onSelectApp={handleSelectApp}
          onCreateWorkspace={() => {
            setEditingWorkspace(null)
            setWorkspaceDialogOpen(true)
          }}
          onEditWorkspace={(workspace) => {
            setEditingWorkspace(workspace)
            setWorkspaceDialogOpen(true)
          }}
          onDeleteWorkspace={setDeletingWorkspace}
          onCreateApp={(workspaceId) => {
            setAppDialogWorkspaceId(workspaceId)
            setEditingApp(null)
            setAppDialogOpen(true)
          }}
          onStatus={handleStatus}
        />
        <SidebarInset>
          <header className="flex h-12 items-center gap-2 border-b px-3">
            <SidebarTrigger />
            <Separator orientation="vertical" className="h-4" />
            <div className="min-w-0 text-sm">
              {selectedWorkspace ? (
                <span className="truncate">
                  {selectedWorkspace.name}
                  {selectedApp ? ` / ${selectedApp.name}` : ""}
                </span>
              ) : (
                <span className="text-muted-foreground">App Runner</span>
              )}
            </div>
            {selectedWorkspace && !selectedApp ? (
              <Button
                size="sm"
                className="ml-auto"
                onClick={() => {
                  setAppDialogWorkspaceId(selectedWorkspace.id)
                  setEditingApp(null)
                  setAppDialogOpen(true)
                }}
              >
                <PlusIcon data-icon="inline-start" />
                Add app
              </Button>
            ) : null}
          </header>

          {loading ? (
            <div className="flex flex-1 items-center justify-center text-sm text-muted-foreground">
              Loading…
            </div>
          ) : selectedApp ? (
            <AppDetail
              app={selectedApp}
              status={status ?? statusByAppId[selectedApp.id] ?? null}
              logs={logs}
              connected={connected}
              onEdit={() => {
                setAppDialogWorkspaceId(selectedApp.workspace_id)
                setEditingApp(selectedApp)
                setAppDialogOpen(true)
              }}
              onDelete={() => setDeletingApp(selectedApp)}
              onStatus={handleStatus}
            />
          ) : selectedWorkspace ? (
            <WorkspaceDetail
              workspace={selectedWorkspace}
              apps={selectedApps}
              statusByAppId={statusByAppId}
              onSelectApp={(appId) =>
                handleSelectApp(selectedWorkspace.id, appId)
              }
              onCreateApp={() => {
                setAppDialogWorkspaceId(selectedWorkspace.id)
                setEditingApp(null)
                setAppDialogOpen(true)
              }}
              onStatus={handleStatus}
            />
          ) : (
            <div className="flex flex-1 items-center justify-center p-6">
              <Empty className="max-w-md border">
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <FolderIcon />
                  </EmptyMedia>
                  <EmptyTitle>Create a workspace</EmptyTitle>
                  <EmptyDescription>
                    Workspaces hold the apps you configure and run locally.
                  </EmptyDescription>
                </EmptyHeader>
                <EmptyContent>
                  <Button
                    onClick={() => {
                      setEditingWorkspace(null)
                      setWorkspaceDialogOpen(true)
                    }}
                  >
                    <PlusIcon data-icon="inline-start" />
                    New workspace
                  </Button>
                </EmptyContent>
              </Empty>
            </div>
          )}
        </SidebarInset>

        <WorkspaceDialog
          open={workspaceDialogOpen}
          onOpenChange={setWorkspaceDialogOpen}
          workspace={editingWorkspace}
          onSaved={(workspace) => {
            setWorkspaces((prev) => {
              const exists = prev.some((w) => w.id === workspace.id)
              const next = exists
                ? prev.map((w) => (w.id === workspace.id ? workspace : w))
                : [...prev, workspace]
              return next.sort((a, b) => a.name.localeCompare(b.name))
            })
            setSelectedWorkspaceId(workspace.id)
            if (!appsByWorkspace[workspace.id]) {
              setAppsByWorkspace((prev) => ({ ...prev, [workspace.id]: [] }))
            }
          }}
        />

        <DeleteWorkspaceDialog
          workspace={deletingWorkspace}
          open={!!deletingWorkspace}
          onOpenChange={(open) => {
            if (!open) setDeletingWorkspace(null)
          }}
          onDeleted={(id) => {
            setWorkspaces((prev) => prev.filter((w) => w.id !== id))
            setAppsByWorkspace((prev) => {
              const next = { ...prev }
              delete next[id]
              return next
            })
            if (selectedWorkspaceId === id) {
              setSelectedWorkspaceId(null)
              setSelectedAppId(null)
            }
          }}
        />

        {appDialogWorkspaceId != null ? (
          <AppDialog
            open={appDialogOpen}
            onOpenChange={setAppDialogOpen}
            workspaceId={appDialogWorkspaceId}
            app={editingApp}
            onSaved={(app) => {
              setAppsByWorkspace((prev) => {
                const list = prev[app.workspace_id] ?? []
                const exists = list.some((a) => a.id === app.id)
                const next = exists
                  ? list.map((a) => (a.id === app.id ? app : a))
                  : [...list, app]
                return {
                  ...prev,
                  [app.workspace_id]: next.sort((a, b) =>
                    a.name.localeCompare(b.name)
                  ),
                }
              })
              setSelectedWorkspaceId(app.workspace_id)
              setSelectedAppId(app.id)
            }}
          />
        ) : null}

        <DeleteAppDialog
          app={deletingApp}
          open={!!deletingApp}
          onOpenChange={(open) => {
            if (!open) setDeletingApp(null)
          }}
          onDeleted={(id) => {
            if (deletingApp) {
              setAppsByWorkspace((prev) => ({
                ...prev,
                [deletingApp.workspace_id]: (
                  prev[deletingApp.workspace_id] ?? []
                ).filter((a) => a.id !== id),
              }))
            }
            if (selectedAppId === id) setSelectedAppId(null)
          }}
        />

        <Toaster />
      </SidebarProvider>
    </TooltipProvider>
  )
}

export default App
