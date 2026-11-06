import { OpenEditorButton } from "@/components/open-editor-button"
import { OpenFolderButton } from "@/components/open-folder-button"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { appStateLabel } from "@/lib/app-state"
import type { CommandKind } from "@/lib/types"
import {
  EraserIcon,
  GitBranchIcon,
  HammerIcon,
  Link2Icon,
  PlayIcon,
  PlusIcon,
  RefreshCwIcon,
  RocketIcon,
  SquareIcon,
  Trash2Icon,
  WrenchIcon,
} from "lucide-react"
import { BulkBar, IconAction } from "./list-actions"
import type { GitPanelModel } from "./use-git-panel"
const START_KINDS: {
  kind: CommandKind
  label: string
  Icon: typeof PlayIcon
}[] = [
  { kind: "setup", label: "Setup", Icon: WrenchIcon },
  { kind: "dev", label: "Dev", Icon: PlayIcon },
  { kind: "build", label: "Build", Icon: HammerIcon },
  { kind: "prod", label: "Prod", Icon: RocketIcon },
]

type WorktreesSectionModel = Pick<
  GitPanelModel,
  | "appId"
  | "activeConfigSetId"
  | "status"
  | "info"
  | "sets"
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
  | "setFor"
  | "setName"
  | "chooseSet"
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
    sets,
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
    setFor,
    setName,
    chooseSet,
    handlePrune,
    handleStart,
    handleLink,
    handleStop,
    worktrees,
    runningIn,
    removableWorktrees,
    selectedWorktrees,
  } = model
  return (
    <section className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <p className="text-sm font-medium">Worktrees</p>
            <Badge variant="secondary">{worktrees.length}</Badge>
          </div>
          <p className="text-sm text-muted-foreground">
            Setup, Run and Build use this app's config, with the config set
            picked per worktree, inside that worktree. Setup first links the
            shared files.
          </p>
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
                      <Badge variant="outline">
                        {appStateLabel(true, status?.kind)}
                        {setName(status?.configSetId)
                          ? ` · ${setName(status?.configSetId)}`
                          : ""}
                      </Badge>
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
                    <Select
                      items={sets.map((s) => ({
                        value: String(s.id),
                        label: s.name,
                      }))}
                      value={setFor(w) ? String(setFor(w)) : null}
                      onValueChange={(value) => {
                        if (value != null) chooseSet(w, Number(value))
                      }}
                      disabled={runningIn(w) || sets.length === 0}
                    >
                      <SelectTrigger
                        size="sm"
                        className="w-40"
                        aria-label={`Config set for ${w.path}`}
                      >
                        <SelectValue placeholder="Config set" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectGroup>
                          {sets.map((s) => (
                            <SelectItem key={s.id} value={String(s.id)}>
                              {s.name}
                              {s.id === activeConfigSetId ? " (active)" : ""}
                            </SelectItem>
                          ))}
                        </SelectGroup>
                      </SelectContent>
                    </Select>
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
                        disabled={starting}
                        onClick={() => void handleStop()}
                      >
                        <SquareIcon />
                      </IconAction>
                    ) : (
                      START_KINDS.map(({ kind, label, Icon }) => (
                        <IconAction
                          key={kind}
                          label={`${label} in this worktree`}
                          disabled={starting || running || !setFor(w)}
                          onClick={() => void handleStart(w, kind, label)}
                        >
                          <Icon />
                        </IconAction>
                      ))
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
