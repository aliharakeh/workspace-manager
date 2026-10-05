import { useEffect, useState, type CSSProperties } from "react"
import {
  AppWindowIcon,
  ChevronRightIcon,
  FolderIcon,
  MoonIcon,
  MoreHorizontalIcon,
  PlusIcon,
  SettingsIcon,
  SunIcon,
} from "lucide-react"
import type { App, StatusEvent, Workspace } from "@/lib/types"
import { AppRunControls, AppStatusDot } from "@/components/app-run-controls"
import { useTheme } from "@/components/theme-provider"
import { cn } from "@/lib/utils"
import { workspaceTint } from "@/lib/workspace-colors"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupAction,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuAction,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarRail,
} from "@/components/ui/sidebar"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

type AppSidebarProps = {
  workspaces: Workspace[]
  appsByWorkspace: Record<number, App[]>
  statusByAppId: Record<number, StatusEvent>
  selectedWorkspaceId: number | null
  selectedAppId: number | null
  onSelectWorkspace: (id: number) => void
  onSelectApp: (workspaceId: number, appId: number) => void
  onCreateWorkspace: () => void
  onEditWorkspace: (workspace: Workspace) => void
  onDeleteWorkspace: (workspace: Workspace) => void
  onCreateApp: (workspaceId: number) => void
  onCreateAppFromBlueprint: (workspaceId: number) => void
  onOpenSettings: () => void
  onStatus: (status: StatusEvent) => void
}

const COLLAPSED_STORAGE_KEY = "workspace-manager.sidebar.collapsed"

// Workspaces the user has collapsed; every other workspace shows its apps.
function loadCollapsed(): Set<number> {
  try {
    const raw = localStorage.getItem(COLLAPSED_STORAGE_KEY)
    const ids: unknown = raw ? JSON.parse(raw) : []
    return new Set(Array.isArray(ids) ? ids.filter(Number.isInteger) : [])
  } catch {
    return new Set()
  }
}

export function AppSidebar({
  workspaces,
  appsByWorkspace,
  statusByAppId,
  selectedWorkspaceId,
  selectedAppId,
  onSelectWorkspace,
  onSelectApp,
  onCreateWorkspace,
  onEditWorkspace,
  onDeleteWorkspace,
  onCreateApp,
  onCreateAppFromBlueprint,
  onOpenSettings,
  onStatus,
}: AppSidebarProps) {
  const { resolvedTheme, toggleTheme } = useTheme()
  const dark = resolvedTheme === "dark"

  const [collapsed, setCollapsed] = useState(loadCollapsed)
  const [seenWorkspaceId, setSeenWorkspaceId] = useState(selectedWorkspaceId)

  // Opening a workspace (from the palette, a link or a click) expands it.
  if (seenWorkspaceId !== selectedWorkspaceId) {
    setSeenWorkspaceId(selectedWorkspaceId)
    if (selectedWorkspaceId != null && collapsed.has(selectedWorkspaceId)) {
      const next = new Set(collapsed)
      next.delete(selectedWorkspaceId)
      setCollapsed(next)
    }
  }

  useEffect(() => {
    try {
      localStorage.setItem(
        COLLAPSED_STORAGE_KEY,
        JSON.stringify([...collapsed])
      )
    } catch {
      // Storage can be unavailable; the sidebar still works without it.
    }
  }, [collapsed])

  function toggleWorkspace(id: number) {
    setCollapsed((prev) => {
      const next = new Set(prev)
      if (!next.delete(id)) next.add(id)
      return next
    })
  }

  return (
    <Sidebar collapsible="icon">
      <SidebarHeader className="border-b">
        <div className="flex items-center gap-2 px-2 py-1.5">
          <div className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <AppWindowIcon className="size-4" />
          </div>
          <div className="flex min-w-0 flex-1 flex-col group-data-[collapsible=icon]:hidden">
            <span className="truncate text-sm font-medium">
              Workspace Manager
            </span>
            <span className="truncate text-xs text-muted-foreground">
              Local workspaces
            </span>
          </div>
        </div>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Workspaces</SidebarGroupLabel>
          <SidebarGroupAction title="New workspace" onClick={onCreateWorkspace}>
            <PlusIcon />
            <span className="sr-only">New workspace</span>
          </SidebarGroupAction>
          <SidebarGroupContent>
            <SidebarMenu>
              {workspaces.length === 0 ? (
                <SidebarMenuItem>
                  <SidebarMenuButton disabled>
                    <FolderIcon />
                    <span>No workspaces yet</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ) : (
                workspaces.map((workspace) => {
                  const apps = appsByWorkspace[workspace.id] ?? []
                  const isActive = selectedWorkspaceId === workspace.id
                  const expanded = !collapsed.has(workspace.id)
                  const colorStyle = workspace.color
                    ? { color: workspace.color }
                    : undefined
                  const lineProps = workspace.color
                    ? {
                        className: "border-l-2",
                        style: { borderColor: workspace.color },
                      }
                    : {}
                  const runningInWorkspace = apps.filter(
                    (app) => statusByAppId[app.id]?.running
                  ).length
                  return (
                    <SidebarMenuItem
                      key={workspace.id}
                      className={workspace.color ? "rounded-md" : undefined}
                      style={
                        workspace.color
                          ? ({
                              backgroundColor: workspaceTint(workspace.color, 5),
                              "--ws-row": workspaceTint(workspace.color, 11),
                              "--ws-row-hover": workspaceTint(workspace.color, 16),
                              "--ws-row-active": workspaceTint(workspace.color, 22),
                            } as CSSProperties)
                          : undefined
                      }
                    >
                      <SidebarMenuButton
                        isActive={isActive && !selectedAppId}
                        tooltip={workspace.name}
                        className={cn(
                          "group-has-data-[sidebar=menu-action]/menu-item:pr-14",
                          // The row's own fill, also for hover and the selected state, which
                          // would otherwise paint the neutral sidebar-accent grey.
                          workspace.color &&
                            "bg-(--ws-row) hover:bg-(--ws-row-hover) active:bg-(--ws-row-active) data-active:bg-(--ws-row-active) data-open:hover:bg-(--ws-row-hover)"
                        )}
                        onClick={() => onSelectWorkspace(workspace.id)}
                      >
                        <FolderIcon style={colorStyle} />
                        <span>{workspace.name}</span>
                        {runningInWorkspace > 0 ? (
                          <AppStatusDot
                            running
                            className="group-data-[collapsible=icon]:hidden"
                          />
                        ) : null}
                      </SidebarMenuButton>
                      <SidebarMenuAction
                        className="right-7"
                        style={colorStyle}
                        title={
                          expanded
                            ? `Collapse ${workspace.name}`
                            : `Expand ${workspace.name}`
                        }
                        aria-expanded={expanded}
                        onClick={() => toggleWorkspace(workspace.id)}
                      >
                        <ChevronRightIcon
                          className={cn(
                            "transition-transform",
                            expanded && "rotate-90"
                          )}
                        />
                        <span className="sr-only">
                          {expanded ? "Collapse" : "Expand"} {workspace.name}
                        </span>
                      </SidebarMenuAction>
                      <DropdownMenu>
                        <DropdownMenuTrigger
                          render={<SidebarMenuAction showOnHover style={colorStyle} />}
                        >
                          <MoreHorizontalIcon />
                          <span className="sr-only">Workspace menu</span>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent side="right" align="start">
                          <DropdownMenuItem
                            onClick={() => onCreateApp(workspace.id)}
                          >
                            Add app
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            onClick={() =>
                              onCreateAppFromBlueprint(workspace.id)
                            }
                          >
                            New app from blueprint
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            onClick={() => onEditWorkspace(workspace)}
                          >
                            Edit
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            variant="destructive"
                            onClick={() => onDeleteWorkspace(workspace)}
                          >
                            Delete
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                      {expanded && apps.length > 0 ? (
                        <SidebarMenuSub {...lineProps}>
                          {apps.map((app) => {
                            const running = !!statusByAppId[app.id]?.running
                            const building =
                              running && statusByAppId[app.id]?.kind === "build"
                            return (
                              <SidebarMenuSubItem key={app.id}>
                                <div className="flex w-full min-w-0 items-center justify-between gap-1">
                                  <SidebarMenuSubButton
                                    isActive={selectedAppId === app.id}
                                    className="min-w-0 flex-1"
                                    onClick={() =>
                                      onSelectApp(workspace.id, app.id)
                                    }
                                  >
                                    <span className="truncate">{app.name}</span>
                                  </SidebarMenuSubButton>
                                  <AppRunControls
                                    className="shrink-0"
                                    appId={app.id}
                                    running={running}
                                    building={building}
                                    onStatus={onStatus}
                                    variant="compact"
                                  />
                                </div>
                              </SidebarMenuSubItem>
                            )
                          })}
                        </SidebarMenuSub>
                      ) : null}
                      {expanded && apps.length === 0 ? (
                        <SidebarMenuSub {...lineProps}>
                          <SidebarMenuSubItem>
                            <SidebarMenuSubButton
                              onClick={() => onCreateApp(workspace.id)}
                            >
                              <PlusIcon />
                              <span>Add app</span>
                            </SidebarMenuSubButton>
                          </SidebarMenuSubItem>
                        </SidebarMenuSub>
                      ) : null}
                    </SidebarMenuItem>
                  )
                })
              )}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter className="border-t">
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              tooltip={dark ? "Switch to light theme" : "Switch to dark theme"}
              onClick={toggleTheme}
            >
              {dark ? <SunIcon /> : <MoonIcon />}
              <span>Theme</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
          <SidebarMenuItem>
            <SidebarMenuButton tooltip="Settings" onClick={onOpenSettings}>
              <SettingsIcon />
              <span>Settings</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  )
}
