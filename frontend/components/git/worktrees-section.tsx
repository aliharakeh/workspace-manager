import { OpenEditorButton } from "@/components/open-editor-button"
import { OpenFolderButton } from "@/components/open-folder-button"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Skeleton } from "@/components/ui/skeleton"
import { AppStateBadge } from "@/components/app-run-controls"
import { useRunConfigs } from "@/lib/use-run-configs"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  EraserIcon,
  GitBranchIcon,
  Link2Icon,
  PlayIcon,
  PlusIcon,
  RefreshCwIcon,
  SquareIcon,
  Trash2Icon,
} from "lucide-react"
import { BulkBar, IconAction } from "./list-actions"
import type { GitPanelModel } from "./use-git-panel"
type WorktreesSectionModel = Pick<
  GitPanelModel,
  | "appId"
  | "activeConfigSetId"
  | "status"
  | "info"
  | "links"
  | "setLinksOpen"
  | "linking"
  | "loading"
  | "pruning"
  | "setAddOpen"
  | "setPendingRemove"
  | "setForce"
  | "starting"
  | "selWorktrees"
  | "setSelWorktrees"
  | "setBulk"
  | "setBulkForce"
  | "setBulkWithBranches"
  | "running"
  | "load"
  | "setName"
  | "handlePrune"
  | "handleStart"
  | "handleLink"
  | "handleStop"
  | "worktrees"
  | "runningIn"
  | "removableWorktrees"
  | "selectedWorktrees"
>

export function WorktreesSection({ model }: { model: WorktreesSectionModel }) {
  const {
    appId,
    activeConfigSetId,
    status,
    info,
    links,
    setLinksOpen,
    linking,
    loading,
    pruning,
    setAddOpen,
    setPendingRemove,
    setForce,
    starting,
    selWorktrees,
    setSelWorktrees,
    setBulk,
    setBulkForce,
    setBulkWithBranches,
    running,
    load,
    setName,
    handlePrune,
    handleStart,
    handleLink,
    handleStop,
    worktrees,
    runningIn,
    removableWorktrees,
    selectedWorktrees,
  } = model
  const { configs } = useRunConfigs(appId, activeConfigSetId ?? 0)
  return (
    <section className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <p className="text-sm font-medium">Worktrees</p>
            <Badge variant="secondary">{worktrees.length}</Badge>
          </div>
        </div>
        <div className="ml-auto flex flex-wrap justify-end gap-2">
          <Button
            variant="outline"
            disabled={loading}
            onClick={() => void load()}
          >
            <RefreshCwIcon data-icon="inline-start" />
            Refresh
          </Button>
          <Button
            variant="outline"
            title="Files and folders worktrees link to from this app's folder, e.g. node_modules"
            onClick={() => setLinksOpen(true)}
          >
            <Link2Icon data-icon="inline-start" />
            Shared files
            {links.length ? (
              <Badge variant="secondary">{links.length}</Badge>
            ) : null}
          </Button>
          <Button
            variant="outline"
            disabled={pruning || !info}
            title="git worktree prune: forget worktrees whose folder was deleted"
            onClick={() => void handlePrune()}
          >
            <EraserIcon data-icon="inline-start" />
            {pruning ? "Pruning…" : "Prune"}
          </Button>
          <Button
            variant="outline"
            disabled={!info}
            onClick={() => setAddOpen(true)}
          >
            <PlusIcon data-icon="inline-start" />
            Add worktree
          </Button>
        </div>
      </div>

      {!info ? (
        <div className="flex flex-col gap-2">
          <Skeleton className="h-14 w-full" />
          <Skeleton className="h-14 w-full" />
        </div>
      ) : (
        <>
          {removableWorktrees.length > 0 ? (
            <BulkBar
              label="worktrees"
              total={removableWorktrees.length}
              selected={selectedWorktrees.length}
              onToggleAll={(all) =>
                setSelWorktrees(
                  all ? removableWorktrees.map((w) => w.path) : []
                )
              }
              onDelete={() => {
                setBulkForce(false)
                setBulkWithBranches(false)
                setBulk("worktrees")
              }}
            />
          ) : null}
          <ul className="divide-y rounded-lg border">
            {worktrees.map((w) => (
              <li
                key={w.path}
                className="flex flex-wrap items-center gap-3 px-3 py-2"
              >
                {removableWorktrees.length > 0 ? (
                  <Checkbox
                    aria-label={`Select worktree ${w.branch || w.path}`}
                    disabled={!removableWorktrees.includes(w)}
                    checked={selWorktrees.includes(w.path)}
                    onCheckedChange={(checked) =>
                      setSelWorktrees((cur) =>
                        checked
                          ? [...cur, w.path]
                          : cur.filter((p) => p !== w.path)
                      )
                    }
                  />
                ) : null}
                <div className="min-w-48 flex-1">
                  <div className="flex flex-wrap items-center gap-1.5">
                    <GitBranchIcon className="size-3.5 text-muted-foreground" />
                    <span className="text-sm font-medium">
                      {w.branch ||
                        (w.bare ? "bare" : `detached @ ${w.head.slice(0, 7)}`)}
                    </span>
                    {w.main ? <Badge variant="secondary">main</Badge> : null}
                    {w.current ? <Badge>this app</Badge> : null}
                    {w.locked ? <Badge variant="outline">locked</Badge> : null}
                    {w.shared_links ? (
                      <Badge
                        variant="outline"
                        title={`Shared files for this worktree only: ${w.shared_links.join(", ")}`}
                      >
                        own shared files
                      </Badge>
                    ) : null}
                    {w.prunable ? (
                      <Badge variant="destructive">missing</Badge>
                    ) : null}
                    {runningIn(w) ? (
                      <AppStateBadge
                        running
                        kind={status?.kind}
                        configName={
                          [status?.configName, setName(status?.configSetId)]
                            .filter(Boolean)
                            .join(" · ") || undefined
                        }
                      />
                    ) : null}
                  </div>
                  <p
                    className="truncate font-mono text-xs text-muted-foreground"
                    title={w.path}
                  >
                    {w.path}
                  </p>
                </div>
                {!w.current && !w.bare && !w.prunable ? (
                  <div className="flex flex-wrap items-center gap-2">
                    {(w.shared_links ?? links).length ? (
                      <IconAction
                        label="Link shared files"
                        disabled={linking || runningIn(w)}
                        onClick={() => void handleLink(w)}
                      >
                        <Link2Icon />
                      </IconAction>
                    ) : null}
                    {runningIn(w) ? (
                      <IconAction
                        label="Stop"
                        variant="stop"
                        disabled={starting}
                        onClick={() => void handleStop()}
                      >
                        <SquareIcon className="fill-current" />
                      </IconAction>
                    ) : (
                      <DropdownMenu>
                        <DropdownMenuTrigger
                          render={
                            <Button
                              size="sm"
                              variant="run"
                              disabled={starting || running || !configs.length}
                            />
                          }
                        >
                          <PlayIcon data-icon="inline-start" className="fill-current" />
                          Run
                        </DropdownMenuTrigger>
                        <DropdownMenuContent>
                          <DropdownMenuGroup>
                            <DropdownMenuLabel>
                              Run in this worktree
                            </DropdownMenuLabel>
                            {configs.map((c) => (
                              <DropdownMenuItem
                                key={c.kind}
                                disabled={!c.commands.length}
                                onClick={() =>
                                  void handleStart(w, c.kind, c.name)
                                }
                              >
                                {c.name}
                              </DropdownMenuItem>
                            ))}
                          </DropdownMenuGroup>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    )}
                  </div>
                ) : null}
                {!w.bare && !w.prunable ? (
                  <div className="flex items-center gap-2">
                    <OpenFolderButton
                      appId={appId}
                      worktreePath={w.current ? undefined : w.path}
                      iconOnly
                    />
                    <OpenEditorButton
                      appId={appId}
                      worktreePath={w.current ? undefined : w.path}
                      iconOnly
                    />
                  </div>
                ) : null}
                {!w.main && !w.current ? (
                  <IconAction
                    label="Remove worktree"
                    variant="ghost"
                    disabled={runningIn(w)}
                    onClick={() => {
                      setForce(false)
                      setPendingRemove(w)
                    }}
                  >
                    <Trash2Icon />
                  </IconAction>
                ) : null}
              </li>
            ))}
          </ul>
        </>
      )}
    </section>
  )
}
