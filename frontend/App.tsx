import { AppDetail } from '@/components/app-detail'
import { AppDialog, DeleteAppDialog } from '@/components/app-dialogs'
import { AppSidebar } from '@/components/app-sidebar'
import { CommandPalette, type PaletteItem } from '@/components/command-palette'
import { NewAppFromBlueprintDialog } from '@/components/new-app-from-blueprint-dialog'
import { SettingsDialog, type SettingsTab } from '@/components/settings-dialog'
import { SettingsProvider, useSettings } from '@/components/settings-provider'
import { isEditableTarget, useTheme } from '@/components/theme-provider'
import { Button } from '@/components/ui/button'
import {
    Empty,
    EmptyContent,
    EmptyDescription,
    EmptyHeader,
    EmptyMedia,
    EmptyTitle,
} from '@/components/ui/empty'
import { Kbd } from '@/components/ui/kbd'
import { Separator } from '@/components/ui/separator'
import { SidebarInset, SidebarProvider, SidebarTrigger } from '@/components/ui/sidebar'
import { Toaster } from '@/components/ui/sonner'
import { TooltipProvider } from '@/components/ui/tooltip'
import { WorkspaceDetail } from '@/components/workspace-detail'
import { DeleteWorkspaceDialog, WorkspaceDialog } from '@/components/workspace-dialogs'
import { useAppStatuses } from '@/hooks/use-app-statuses'
import { useRoute } from '@/hooks/use-route'
import { useRunnerStatus } from '@/hooks/use-runner-status'
import { api, onFileDrop } from '@/lib/api'
import { formatRoute, type AppTab } from '@/lib/routes'
import {
    DEFAULT_SHORTCUTS,
    SEARCH_SHORTCUT_KEY,
    THEME_SHORTCUT_KEY,
    formatShortcut,
    isShortcutRecorderActive,
    matchesShortcut,
    shortcutParts,
} from '@/lib/shortcuts'
import type { App as AppEntity, StatusEvent, Workspace } from '@/lib/types'
import { FolderIcon, LayoutTemplateIcon, PlusIcon, SearchIcon } from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { toast } from 'sonner'

// Last path segment, e.g. "C:\code\my-app\" -> "my-app". Falls back to the
// whole path for roots like "C:\".
function folderName(path: string) {
    const parts = path.split(/[\\/]/).filter(Boolean)
    return parts[parts.length - 1] ?? path
}

export function App() {
    return (
        <SettingsProvider>
            <AppContent />
        </SettingsProvider>
    )
}

function AppContent() {
    const { settings } = useSettings()
    const { toggleTheme } = useTheme()
    const searchShortcut = settings[SEARCH_SHORTCUT_KEY] ?? DEFAULT_SHORTCUTS[SEARCH_SHORTCUT_KEY]
    const themeShortcut = settings[THEME_SHORTCUT_KEY] ?? DEFAULT_SHORTCUTS[THEME_SHORTCUT_KEY]
    const [paletteOpen, setPaletteOpen] = useState(false)

    const [workspaces, setWorkspaces] = useState<Workspace[]>([])
    const [appsByWorkspace, setAppsByWorkspace] = useState<Record<number, AppEntity[]>>({})
    const { route, navigate } = useRoute()
    const selectedWorkspaceId = route.workspaceId
    const selectedAppId = route.appId
    const [loading, setLoading] = useState(true)

    const [workspaceDialogOpen, setWorkspaceDialogOpen] = useState(false)
    const [editingWorkspace, setEditingWorkspace] = useState<Workspace | null>(null)
    const [deletingWorkspace, setDeletingWorkspace] = useState<Workspace | null>(null)

    const [appDialogOpen, setAppDialogOpen] = useState(false)
    const [appDialogWorkspaceId, setAppDialogWorkspaceId] = useState<number | null>(null)
    const [editingApp, setEditingApp] = useState<AppEntity | null>(null)
    const [deletingApp, setDeletingApp] = useState<AppEntity | null>(null)

    const [blueprintWorkspaceId, setBlueprintWorkspaceId] = useState<number | null>(null)
    const [blueprintDialogOpen, setBlueprintDialogOpen] = useState(false)
    const [settingsOpen, setSettingsOpen] = useState(false)
    const [settingsTab, setSettingsTab] = useState<SettingsTab>('ports')
    // Set when Settings was opened from the blueprint dialog, to return there.
    const [returnToBlueprintDialog, setReturnToBlueprintDialog] = useState(false)

    const { status, setStatus } = useRunnerStatus(selectedAppId)
    const workspaceIds = useMemo(() => workspaces.map(workspace => workspace.id), [workspaces])
    const { statusByAppId, setAppStatus } = useAppStatuses(workspaceIds)

    const selectedWorkspace = useMemo(
        () => workspaces.find(w => w.id === selectedWorkspaceId) ?? null,
        [workspaces, selectedWorkspaceId],
    )

    const blueprintWorkspace = useMemo(
        () => workspaces.find(w => w.id === blueprintWorkspaceId) ?? null,
        [workspaces, blueprintWorkspaceId],
    )

    const selectedApps = useMemo(
        () => (selectedWorkspaceId != null ? (appsByWorkspace[selectedWorkspaceId] ?? []) : []),
        [appsByWorkspace, selectedWorkspaceId],
    )

    const selectedApp = useMemo(() => {
        if (selectedAppId == null) return null
        return selectedApps.find(a => a.id === selectedAppId) ?? null
    }, [selectedApps, selectedAppId])

    const handleStatus = useCallback(
        (next: StatusEvent) => {
            setAppStatus(next)
            if (selectedAppId === next.appId) setStatus(next)
        },
        [selectedAppId, setAppStatus, setStatus],
    )

    useEffect(() => {
        if (status) setAppStatus(status)
    }, [status, setAppStatus])

    const loadApps = useCallback(async (workspaceId: number) => {
        const apps = await api.apps.list(workspaceId)
        setAppsByWorkspace(prev => ({ ...prev, [workspaceId]: apps }))
        return apps
    }, [])

    const bootstrap = useCallback(async () => {
        setLoading(true)
        try {
            const list = await api.workspaces.list()
            setWorkspaces(list)
            if (list.length) {
                await Promise.all(list.map(w => loadApps(w.id)))
            }
        } catch (err) {
            toast.error(err instanceof Error ? err.message : 'Failed to load data')
        } finally {
            setLoading(false)
        }
    }, [loadApps])

    useEffect(() => {
        void bootstrap()
    }, [bootstrap])

    function handleAppSaved(app: AppEntity) {
        setAppsByWorkspace(prev => {
            const list = prev[app.workspace_id] ?? []
            const exists = list.some(a => a.id === app.id)
            const next = exists ? list.map(a => (a.id === app.id ? app : a)) : [...list, app]
            return {
                ...prev,
                [app.workspace_id]: next.sort((a, b) => a.name.localeCompare(b.name)),
            }
        })
        const workspace = workspaces.find(w => w.id === app.workspace_id)
        if (workspace) handleGoApp(workspace, app)
    }

    // Dropped folders become apps in the open workspace, or in a new "Default"
    // workspace when none is open.
    async function handleDropPaths(paths: string[]) {
        if (loading) return
        const folders: { name: string; path: string }[] = []
        for (const dropped of paths) {
            try {
                const result = await api.fs.validatePath(dropped)
                if (result.ok && result.path) {
                    folders.push({ name: folderName(result.path), path: result.path })
                } else {
                    toast.error(result.error ?? `Cannot add ${dropped}`)
                }
            } catch (err) {
                toast.error(err instanceof Error ? err.message : 'Failed to read dropped item')
            }
        }
        if (!folders.length) return

        let target: Workspace
        if (selectedWorkspace) {
            target = selectedWorkspace
        } else {
            try {
                const created = (await api.workspaces.create({ name: 'Default' })) as Workspace
                target = created
                setWorkspaces(prev =>
                    [...prev, created].sort((a, b) => a.name.localeCompare(b.name)),
                )
                setAppsByWorkspace(prev => ({ ...prev, [created.id]: [] }))
            } catch (err) {
                toast.error(err instanceof Error ? err.message : 'Failed to create workspace')
                return
            }
        }

        let last: AppEntity | null = null
        for (const folder of folders) {
            try {
                const app = (await api.apps.create(target.id, {
                    name: folder.name,
                    project_path: folder.path,
                })) as AppEntity
                last = app
                setAppsByWorkspace(prev => ({
                    ...prev,
                    [target.id]: [...(prev[target.id] ?? []), app].sort((a, b) =>
                        a.name.localeCompare(b.name),
                    ),
                }))
                toast.success(`Added ${app.name} to ${target.name}`)
            } catch (err) {
                toast.error(err instanceof Error ? err.message : `Failed to add ${folder.name}`)
            }
        }
        if (last) handleGoApp(target, last)
        else handleGoWorkspace(target)
    }

    const handleDropPathsRef = useRef(handleDropPaths)
    useEffect(() => {
        handleDropPathsRef.current = handleDropPaths
    })
    useEffect(() => onFileDrop(paths => void handleDropPathsRef.current(paths)), [])

    function handleSelectWorkspace(id: number) {
        const workspace = workspaces.find(w => w.id === id)
        if (!workspace) return
        if (!appsByWorkspace[id]) void loadApps(id)
        navigate(
            formatRoute(
                { workspaceId: id, appId: null, tab: null, configSetId: null },
                workspace.name,
            ),
        )
    }

    function handleSelectApp(workspaceId: number, appId: number) {
        const workspace = workspaces.find(w => w.id === workspaceId)
        const app = (appsByWorkspace[workspaceId] ?? []).find(a => a.id === appId)
        if (!workspace || !app) return
        navigate(
            formatRoute(
                { workspaceId, appId, tab: 'env', configSetId: app.active_config_set_id },
                workspace.name,
                app.name,
            ),
        )
    }

    function handleTabChange(tab: AppTab) {
        if (!selectedWorkspace || !selectedApp) return
        navigate(
            formatRoute(
                {
                    workspaceId: selectedWorkspace.id,
                    appId: selectedApp.id,
                    tab,
                    configSetId: selectedApp.active_config_set_id,
                },
                selectedWorkspace.name,
                selectedApp.name,
            ),
        )
    }

    function handleGoWorkspace(workspace: Workspace) {
        navigate(
            formatRoute(
                { workspaceId: workspace.id, appId: null, tab: null, configSetId: null },
                workspace.name,
            ),
        )
    }

    function handleGoApp(workspace: Workspace, app: AppEntity, tab: AppTab = 'env') {
        navigate(
            formatRoute(
                {
                    workspaceId: workspace.id,
                    appId: app.id,
                    tab,
                    configSetId: app.active_config_set_id,
                },
                workspace.name,
                app.name,
            ),
        )
    }

    function handleGoHome() {
        navigate('/')
    }

    // Open the search palette from the configured keyboard shortcut.
    useEffect(() => {
        const onKeyDown = (event: KeyboardEvent) => {
            if (isShortcutRecorderActive()) return
            if (matchesShortcut(searchShortcut, event)) {
                event.preventDefault()
                setPaletteOpen(true)
            }
        }
        window.addEventListener('keydown', onKeyDown)
        return () => window.removeEventListener('keydown', onKeyDown)
    }, [searchShortcut])

    // Toggle the theme from the configured keyboard shortcut.
    useEffect(() => {
        const onKeyDown = (event: KeyboardEvent) => {
            if (isShortcutRecorderActive()) return
            if (isEditableTarget(event.target)) return
            if (matchesShortcut(themeShortcut, event)) {
                event.preventDefault()
                toggleTheme()
            }
        }
        window.addEventListener('keydown', onKeyDown)
        return () => window.removeEventListener('keydown', onKeyDown)
    }, [themeShortcut, toggleTheme])

    const paletteItems = useMemo<PaletteItem[]>(
        () => [
            ...workspaces.map<PaletteItem>(w => ({
                type: 'workspace' as const,
                id: w.id,
                workspaceId: null,
                title: w.name,
                subtitle: 'Workspace',
            })),
            ...workspaces.flatMap<PaletteItem>(w =>
                (appsByWorkspace[w.id] ?? []).map(a => ({
                    type: 'app' as const,
                    id: a.id,
                    workspaceId: w.id,
                    title: a.name,
                    subtitle: w.name,
                })),
            ),
        ],
        [workspaces, appsByWorkspace],
    )

    function handlePaletteSelect(item: PaletteItem) {
        setPaletteOpen(false)
        if (item.type === 'workspace') {
            const workspace = workspaces.find(w => w.id === item.id)
            if (workspace) handleGoWorkspace(workspace)
            return
        }
        const workspace = workspaces.find(w => w.id === item.workspaceId)
        const app = (appsByWorkspace[item.workspaceId ?? -1] ?? []).find(a => a.id === item.id)
        if (workspace && app) handleGoApp(workspace, app)
    }

    // Update an app's stored data and, if it's the selected app, re-navigate so
    // the URL reflects the (possibly changed) active config set.
    function handleAppChange(app: AppEntity) {
        setAppsByWorkspace(prev => {
            const list = prev[app.workspace_id] ?? []
            return {
                ...prev,
                [app.workspace_id]: list.map(a => (a.id === app.id ? app : a)),
            }
        })
        if (selectedAppId === app.id && selectedWorkspace) {
            handleGoApp(selectedWorkspace, app, route.tab ?? 'env')
        }
    }

    // Normalize the URL whenever data loads or entities disappear so the URL
    // never points at a missing workspace/app (deep links, deletions, renames).
    useEffect(() => {
        if (loading) return
        if (selectedWorkspaceId == null) {
            const first = workspaces[0]
            if (first) {
                navigate(
                    formatRoute(
                        { workspaceId: first.id, appId: null, tab: null, configSetId: null },
                        first.name,
                    ),
                )
            }
            return
        }
        const workspace = workspaces.find(w => w.id === selectedWorkspaceId)
        if (!workspace) {
            navigate('/')
            return
        }
        if (selectedAppId != null) {
            const app = (appsByWorkspace[selectedWorkspaceId] ?? []).find(
                a => a.id === selectedAppId,
            )
            if (!app) {
                navigate(
                    formatRoute(
                        { workspaceId: workspace.id, appId: null, tab: null, configSetId: null },
                        workspace.name,
                    ),
                )
            } else if (route.configSetId !== app.active_config_set_id) {
                navigate(
                    formatRoute(
                        {
                            workspaceId: workspace.id,
                            appId: app.id,
                            tab: route.tab ?? 'env',
                            configSetId: app.active_config_set_id,
                        },
                        workspace.name,
                        app.name,
                    ),
                )
            }
        }
    }, [
        loading,
        selectedWorkspaceId,
        selectedAppId,
        route.configSetId,
        route.tab,
        workspaces,
        appsByWorkspace,
        navigate,
    ])

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
                    onEditWorkspace={workspace => {
                        setEditingWorkspace(workspace)
                        setWorkspaceDialogOpen(true)
                    }}
                    onDeleteWorkspace={setDeletingWorkspace}
                    onCreateApp={workspaceId => {
                        setAppDialogWorkspaceId(workspaceId)
                        setEditingApp(null)
                        setAppDialogOpen(true)
                    }}
                    onCreateAppFromBlueprint={workspaceId => {
                        setBlueprintWorkspaceId(workspaceId)
                        setBlueprintDialogOpen(true)
                    }}
                    onOpenSettings={() => setSettingsOpen(true)}
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
                                    {selectedApp ? ` / ${selectedApp.name}` : ''}
                                </span>
                            ) : (
                                <span className="text-muted-foreground">Workspace Manager</span>
                            )}
                        </div>
                        <Button
                            variant="outline"
                            size="sm"
                            className="ml-auto gap-1.5 text-muted-foreground"
                            onClick={() => setPaletteOpen(true)}
                            title={`Search (${formatShortcut(searchShortcut)})`}
                        >
                            <SearchIcon />
                            <span className="hidden lg:inline">Search</span>
                            <span className="hidden items-center gap-0.5 md:flex">
                                {shortcutParts(searchShortcut).map(part => (
                                    <Kbd key={part}>{part}</Kbd>
                                ))}
                            </span>
                            <span className="sr-only">Search</span>
                        </Button>
                        {selectedWorkspace && !selectedApp ? (
                            <Button
                                variant="outline"
                                size="sm"
                                onClick={() => {
                                    setBlueprintWorkspaceId(selectedWorkspace.id)
                                    setBlueprintDialogOpen(true)
                                }}
                            >
                                <LayoutTemplateIcon data-icon="inline-start" />
                                From blueprint
                            </Button>
                        ) : null}
                        {selectedWorkspace && !selectedApp ? (
                            <Button
                                size="sm"
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
                            key={selectedApp.id}
                            app={selectedApp}
                            status={status ?? statusByAppId[selectedApp.id] ?? null}
                            tab={route.tab ?? 'env'}
                            onTabChange={handleTabChange}
                            onEdit={() => {
                                setAppDialogWorkspaceId(selectedApp.workspace_id)
                                setEditingApp(selectedApp)
                                setAppDialogOpen(true)
                            }}
                            onDelete={() => setDeletingApp(selectedApp)}
                            onStatus={handleStatus}
                            onAppChange={handleAppChange}
                        />
                    ) : selectedWorkspace ? (
                        <WorkspaceDetail
                            workspace={selectedWorkspace}
                            apps={selectedApps}
                            statusByAppId={statusByAppId}
                            onSelectApp={appId => handleSelectApp(selectedWorkspace.id, appId)}
                            onCreateApp={() => {
                                setAppDialogWorkspaceId(selectedWorkspace.id)
                                setEditingApp(null)
                                setAppDialogOpen(true)
                            }}
                            onCreateAppFromBlueprint={() => {
                                setBlueprintWorkspaceId(selectedWorkspace.id)
                                setBlueprintDialogOpen(true)
                            }}
                            onStatus={handleStatus}
                            onAppChange={handleAppChange}
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
                    onSaved={workspace => {
                        setWorkspaces(prev => {
                            const exists = prev.some(w => w.id === workspace.id)
                            const next = exists
                                ? prev.map(w => (w.id === workspace.id ? workspace : w))
                                : [...prev, workspace]
                            return next.sort((a, b) => a.name.localeCompare(b.name))
                        })
                        handleGoWorkspace(workspace)
                        if (!appsByWorkspace[workspace.id]) {
                            setAppsByWorkspace(prev => ({ ...prev, [workspace.id]: [] }))
                        }
                    }}
                />

                <DeleteWorkspaceDialog
                    workspace={deletingWorkspace}
                    open={!!deletingWorkspace}
                    onOpenChange={open => {
                        if (!open) setDeletingWorkspace(null)
                    }}
                    onDeleted={id => {
                        setWorkspaces(prev => prev.filter(w => w.id !== id))
                        setAppsByWorkspace(prev => {
                            const next = { ...prev }
                            delete next[id]
                            return next
                        })
                        if (selectedWorkspaceId === id) {
                            handleGoHome()
                        }
                    }}
                />

                {appDialogWorkspaceId != null ? (
                    <AppDialog
                        open={appDialogOpen}
                        onOpenChange={setAppDialogOpen}
                        workspaceId={appDialogWorkspaceId}
                        app={editingApp}
                        onSaved={handleAppSaved}
                    />
                ) : null}

                <NewAppFromBlueprintDialog
                    open={blueprintDialogOpen}
                    onOpenChange={setBlueprintDialogOpen}
                    workspace={blueprintWorkspace}
                    onCreated={handleAppSaved}
                    onManage={() => {
                        setBlueprintDialogOpen(false)
                        setReturnToBlueprintDialog(true)
                        setSettingsTab('blueprints')
                        setSettingsOpen(true)
                    }}
                />

                <SettingsDialog
                    open={settingsOpen}
                    onOpenChange={open => {
                        setSettingsOpen(open)
                        if (!open && returnToBlueprintDialog) {
                            setReturnToBlueprintDialog(false)
                            setBlueprintDialogOpen(true)
                        }
                    }}
                    tab={settingsTab}
                    onTabChange={setSettingsTab}
                />

                <DeleteAppDialog
                    app={deletingApp}
                    open={!!deletingApp}
                    onOpenChange={open => {
                        if (!open) setDeletingApp(null)
                    }}
                    onDeleted={id => {
                        if (deletingApp) {
                            setAppsByWorkspace(prev => ({
                                ...prev,
                                [deletingApp.workspace_id]: (
                                    prev[deletingApp.workspace_id] ?? []
                                ).filter(a => a.id !== id),
                            }))
                        }
                    }}
                />

                <CommandPalette
                    open={paletteOpen}
                    onOpenChange={setPaletteOpen}
                    items={paletteItems}
                    shortcut={searchShortcut}
                    onSelect={handlePaletteSelect}
                />

                <Toaster />
            </SidebarProvider>
        </TooltipProvider>
    )
}

export default App
