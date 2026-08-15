import { useState } from "react"
import {
  AppWindowIcon,
  FolderIcon,
  MoonIcon,
  MoreHorizontalIcon,
  PlusIcon,
  SettingsIcon,
  SunIcon,
} from "lucide-react"
import type { App, StatusEvent, Workspace } from "@/lib/types"
import { AppRunControls, AppStatusDot } from "@/components/app-run-controls"
import { SettingsDialog } from "@/components/settings-dialog"
import { useTheme } from "@/components/theme-provider"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import {
  Sidebar,
  SidebarContent,
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
  onStatus: (status: StatusEvent) => void
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
  onStatus,
}: AppSidebarProps) {
  const [settingsOpen, setSettingsOpen] = useState(false)
  const { resolvedTheme, toggleTheme } = useTheme()
  const dark = resolvedTheme === "dark"

  return (
    <Sidebar collapsible="icon">
      <SidebarHeader className="border-b">
        <div className="flex items-center gap-2 px-2 py-1.5">
          <div className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <AppWindowIcon className="size-4" />
          </div>
          <div className="flex min-w-0 flex-1 flex-col group-data-[collapsible=icon]:hidden">
            <span className="truncate text-sm font-medium">Workspace Manager</span>
            <span className="truncate text-xs text-muted-foreground">
              Local workspaces
            </span>
          </div>
          <Button
            variant="ghost"
            size="icon-sm"
            className="shrink-0"
            title={dark ? "Switch to light theme" : "Switch to dark theme"}
            onClick={toggleTheme}
          >
            {dark ? <SunIcon /> : <MoonIcon />}
            <span className="sr-only">Toggle theme</span>
          </Button>
          <Button
            variant="ghost"
            size="icon-sm"
            className="shrink-0"
            title="Settings"
            onClick={() => setSettingsOpen(true)}
          >
            <SettingsIcon />
            <span className="sr-only">Settings</span>
          </Button>
        </div>
      </SidebarHeader>
      <SettingsDialog open={settingsOpen} onOpenChange={setSettingsOpen} />
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
                  const runningInWorkspace = apps.filter(
                    (app) => statusByAppId[app.id]?.running
                  ).length
                  return (
                    <SidebarMenuItem key={workspace.id}>
                      <SidebarMenuButton
                        isActive={isActive && !selectedAppId}
                        tooltip={workspace.name}
                        onClick={() => onSelectWorkspace(workspace.id)}
                      >
                        <FolderIcon />
                        <span>{workspace.name}</span>
                        {runningInWorkspace > 0 ? (
                          <AppStatusDot
                            running
                            className="ml-auto group-data-[collapsible=icon]:hidden"
                          />
                        ) : null}
                      </SidebarMenuButton>
                      <DropdownMenu>
                        <DropdownMenuTrigger
                          render={<SidebarMenuAction showOnHover />}
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
                      {isActive && apps.length > 0 ? (
                        <SidebarMenuSub>
                          {apps.map((app) => {
                            const running = !!statusByAppId[app.id]?.running
                            return (
                              <SidebarMenuSubItem key={app.id}>
                                <SidebarMenuSubButton
                                  isActive={selectedAppId === app.id}
                                  className="pr-16"
                                  onClick={() =>
                                    onSelectApp(workspace.id, app.id)
                                  }
                                >
                                  <AppStatusDot running={running} />
                                  <span>{app.name}</span>
                                </SidebarMenuSubButton>
                                <div
                                  className={cn(
                                    "absolute top-0.5 right-0 transition-opacity",
                                    running || selectedAppId === app.id
                                      ? "opacity-100"
                                      : "opacity-0 group-hover/menu-sub-item:opacity-100 group-focus-within/menu-sub-item:opacity-100"
                                  )}
                                >
                                  <AppRunControls
                                    appId={app.id}
                                    running={running}
                                    onStatus={onStatus}
                                    variant="compact"
                                  />
                                </div>
                              </SidebarMenuSubItem>
                            )
                          })}
                        </SidebarMenuSub>
                      ) : null}
                      {isActive && apps.length === 0 ? (
                        <SidebarMenuSub>
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
      <SidebarRail />
    </Sidebar>
  )
}
