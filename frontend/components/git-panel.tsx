import { useEffect, useState } from "react"
import {
  ArrowDownToLineIcon,
  ArrowUpFromLineIcon,
  CalendarIcon,
  CheckIcon,
  DownloadCloudIcon,
  EraserIcon,
  FolderGit2Icon,
  GitBranchIcon,
  HammerIcon,
  Link2Icon,
  PencilIcon,
  PlayIcon,
  PlusIcon,
  RefreshCwIcon,
  RocketIcon,
  SquareIcon,
  Trash2Icon,
  WrenchIcon,
} from "lucide-react"
import { toast } from "sonner"
import { api } from "@/lib/api"
import { appStateLabel } from "@/lib/app-state"
import { cn } from "@/lib/utils"
import type {
  CommandKind,
  ConfigSet,
  GitInfo,
  GitLocalBranch,
  GitPullResult,
  GitWorktree,
  StatusEvent,
  WorktreeLinkState,
} from "@/lib/types"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Calendar } from "@/components/ui/calendar"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { OpenEditorButton } from "@/components/open-editor-button"
import { OpenFolderButton } from "@/components/open-folder-button"

type GitPanelProps = {
  appId: number
  active: boolean
  /** Bumped when another client writes this app's git state, to reload it. */
  dataEpoch: number
  /** The app's active config set: the default for a worktree with none chosen. */
  activeConfigSetId: number | null
  status: StatusEvent | null
  onStatus: (status: StatusEvent) => void
}

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

/** An icon-only button; the label is its tooltip and accessible name. */
function IconAction({
  label,
  variant = "outline",
  disabled,
  onClick,
  children,
}: {
  label: string
  variant?: React.ComponentProps<typeof Button>["variant"]
  disabled?: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant={variant}
            size="icon-sm"
            aria-label={label}
            disabled={disabled}
            onClick={onClick}
          />
        }
      >
        {children}
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  )
}

/** Config set chosen per worktree path, remembered per app on this machine. */
type ChosenSets = Record<string, number>

function chosenSetsKey(appId: number) {
  return `git-worktree-sets:${appId}`
}

function fetchGitState(appId: number) {
  return Promise.all([
    api.git.info(appId),
    api.configSets.list(appId) as Promise<ConfigSet[]>,
    api.git.linksGet(appId),
  ])
}

const LINK_STATE_LABELS: Record<WorktreeLinkState["state"], string> = {
  linked: "linked",
  missing: "not linked",
  exists: "skipped, real copy present",
  other_link: "skipped, links elsewhere",
  no_source: "skipped, not in the app folder",
  error: "failed",
}

/** One line per shared path, for the "Last git output" box. */
function formatLinkStates(states: WorktreeLinkState[]): string {
  return states
    .map(
      (s) =>
        `${s.path}: ${LINK_STATE_LABELS[s.state]}${s.message ? ` (${s.message})` : ""}`
    )
    .join("\n")
}

/** Links the shared paths into the worktree at path and reports the outcome. */
async function linkSharedPaths(
  appId: number,
  path: string,
  setOutput: (output: string) => void
) {
  reportLinkStates(await api.git.worktreeLinks(appId, path, true), setOutput)
}

/** Shows the outcome of linking shared paths in the output box and a toast. */
function reportLinkStates(
  states: WorktreeLinkState[],
  setOutput: (output: string) => void
) {
  setOutput(formatLinkStates(states))
  const linked = states.filter((s) => s.state === "linked").length
  const failed = states.filter((s) => s.state === "error").length
  const message = `${linked} of ${states.length} shared paths linked`
  if (failed) toast.error(`${message}, ${failed} failed`)
  else if (linked < states.length) toast.warning(`${message}, see the output`)
  else toast.success(message)
}

function readChosenSets(appId: number): ChosenSets {
  try {
    const raw = localStorage.getItem(chosenSetsKey(appId))
    const parsed: unknown = raw ? JSON.parse(raw) : null
    return parsed && typeof parsed === "object" ? (parsed as ChosenSets) : {}
  } catch {
    return {}
  }
}

/** Mirrors DefaultWorktreePath in services/git.go: sibling "<repo>-<branch>". */
function defaultWorktreePath(mainPath: string, branch: string): string {
  const slug = branch.replace(/[^A-Za-z0-9._-]+/g, "-").replace(/^-+|-+$/g, "")
  return `${mainPath}-${slug}`
}

export function GitPanel({
  appId,
  active,
  dataEpoch,
  activeConfigSetId,
  status,
  onStatus,
}: GitPanelProps) {
  const [info, setInfo] = useState<GitInfo | null>(null)
  const [sets, setSets] = useState<ConfigSet[]>([])
  const [links, setLinks] = useState<string[]>([])
  const [linksOpen, setLinksOpen] = useState(false)
  const [linking, setLinking] = useState(false)
  const [chosen, setChosen] = useState(() => readChosenSets(appId))
  // The panel is reused across apps: reload the remembered choices on a switch.
  const [chosenFor, setChosenFor] = useState(appId)
  if (chosenFor !== appId) {
    setChosenFor(appId)
    setChosen(readChosenSets(appId))
  }
  const [loading, setLoading] = useState(false)
  const [fetching, setFetching] = useState(false)
  const [pruning, setPruning] = useState(false)
  const [output, setOutput] = useState<string | null>(null)
  const [addOpen, setAddOpen] = useState(false)
  const [pendingRemove, setPendingRemove] = useState<GitWorktree | null>(null)
  const [force, setForce] = useState(false)
  const [removing, setRemoving] = useState(false)
  const [starting, setStarting] = useState(false)
  const [branchCreateOpen, setBranchCreateOpen] = useState(false)
  const [renaming, setRenaming] = useState<GitLocalBranch | null>(null)
  const [pendingDelete, setPendingDelete] = useState<GitLocalBranch | null>(
    null
  )
  const [forceDelete, setForceDelete] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [checkingOut, setCheckingOut] = useState(false)
  // Bulk selection (worktree paths, branch names) and the bulk delete dialog.
  const [selWorktrees, setSelWorktrees] = useState<string[]>([])
  const [selBranches, setSelBranches] = useState<string[]>([])
  const [bulk, setBulk] = useState<"worktrees" | "branches" | null>(null)
  const [bulkForce, setBulkForce] = useState(false)
  const [bulkWithBranches, setBulkWithBranches] = useState(false)
  const [bulkBusy, setBulkBusy] = useState(false)
  // The branch being pulled, and the result of a pull that hit conflicts.
  const [dateFrom, setDateFrom] = useState("")
  const [dateTo, setDateTo] = useState("")
  const [pulling, setPulling] = useState<string | null>(null)
  const [conflicts, setConflicts] = useState<GitPullResult | null>(null)
  // The branch being pushed, and one whose push was rejected (offered a force).
  const [pushing, setPushing] = useState<string | null>(null)
  const [rejected, setRejected] = useState<GitLocalBranch | null>(null)

  const running = !!status?.running

  async function load() {
    setLoading(true)
    try {
      const [nextInfo, nextSets, nextLinks] = await fetchGitState(appId)
      setInfo(nextInfo)
      setSets(nextSets)
      setLinks(nextLinks)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to read git")
    } finally {
      setLoading(false)
    }
  }

  // The config set a worktree runs with: its remembered choice while that set
  // still exists, else the app's active set.
  function setFor(w: GitWorktree): number {
    const id = chosen[w.path]
    if (id && sets.some((s) => s.id === id)) return id
    return activeConfigSetId ?? sets[0]?.id ?? 0
  }

  function setName(id: number | undefined) {
    return sets.find((s) => s.id === id)?.name
  }

  function chooseSet(w: GitWorktree, id: number) {
    const next = { ...chosen, [w.path]: id }
    setChosen(next)
    try {
      localStorage.setItem(chosenSetsKey(appId), JSON.stringify(next))
    } catch {
      // Not remembered; the choice still applies until the panel reloads.
    }
  }

  // Load when the tab opens first, on an app switch, and after another client
  // wrote this app's git state (dataEpoch).
  useEffect(() => {
    if (!active) return
    let cancelled = false
    fetchGitState(appId)
      .then(([nextInfo, nextSets, nextLinks]) => {
        if (cancelled) return
        setInfo(nextInfo)
        setSets(nextSets)
        setLinks(nextLinks)
      })
      .catch((err) => {
        if (!cancelled) {
          toast.error(err instanceof Error ? err.message : "Failed to read git")
        }
      })
    return () => {
      cancelled = true
    }
  }, [active, appId, dataEpoch])

  async function handleFetch() {
    setFetching(true)
    try {
      const res = await api.git.fetchAll(appId)
      setOutput(res.output)
      toast.success("Fetched all remotes")
      await load()
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Fetch failed"
      setOutput(msg)
      toast.error("Fetch failed")
    } finally {
      setFetching(false)
    }
  }

  async function handlePrune() {
    setPruning(true)
    try {
      const res = await api.git.worktreePrune(appId)
      setOutput(res.output)
      toast.success("Pruned worktrees")
      await load()
    } catch (err) {
      setOutput(err instanceof Error ? err.message : "Prune failed")
      toast.error("Prune failed")
    } finally {
      setPruning(false)
    }
  }

  async function handleStart(w: GitWorktree, kind: CommandKind, label: string) {
    setStarting(true)
    try {
      onStatus(await api.git.worktreeStart(appId, w.path, kind, setFor(w)))
      toast.success(
        `${label} started in ${w.branch || w.path} with ${setName(setFor(w)) ?? "the active set"}`
      )
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : `Failed to start ${kind}`
      )
    } finally {
      setStarting(false)
    }
  }

  async function handleLink(w: GitWorktree) {
    setLinking(true)
    try {
      await linkSharedPaths(appId, w.path, setOutput)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to link")
    } finally {
      setLinking(false)
    }
  }

  async function handleStop() {
    setStarting(true)
    try {
      onStatus((await api.runner.stop(appId)) as StatusEvent)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to stop")
    } finally {
      setStarting(false)
    }
  }

  async function handleConfirmRemove() {
    if (!pendingRemove) return
    setRemoving(true)
    try {
      await api.git.worktreeRemove(appId, pendingRemove.path, force)
      toast.success("Worktree removed")
      setPendingRemove(null)
      await load()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to remove")
    } finally {
      setRemoving(false)
    }
  }

  async function handleCheckout(b: GitLocalBranch) {
    setCheckingOut(true)
    try {
      await api.git.branchCheckout(appId, b.name)
      toast.success(`Switched to ${b.name}`)
      await load()
    } catch (err) {
      setOutput(err instanceof Error ? err.message : "Checkout failed")
      toast.error("Checkout failed, see the output")
    } finally {
      setCheckingOut(false)
    }
  }

  async function handlePush(b: GitLocalBranch, force = false) {
    setPushing(b.name)
    try {
      const res = await api.git.branchPush(appId, b.name, force)
      setOutput(res.output)
      toast.success(
        b.upstream && !b.gone ? `Pushed ${b.name}` : `Published ${b.name}`
      )
      setRejected(null)
      await load()
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Push failed"
      setOutput(msg)
      // A non-fast-forward push is the one failure a force can fix.
      if (!force && /rejected|non-fast-forward|fetch first/i.test(msg)) {
        setRejected(b)
        toast.error(`Push of ${b.name} was rejected`)
      } else {
        toast.error("Push failed, see the output")
      }
    } finally {
      setPushing(null)
    }
  }

  async function handlePull(b: GitLocalBranch) {
    setPulling(b.name)
    try {
      const res = await api.git.branchPull(appId, b.name)
      setOutput(res.output)
      if (res.conflicts.length) {
        setConflicts(res)
        toast.warning(
          `Pull of ${b.name} stopped on ${res.conflicts.length} conflicted file${res.conflicts.length === 1 ? "" : "s"}`
        )
      } else {
        toast.success(`Pulled ${b.upstream} into ${b.name}`)
      }
      await load()
    } catch (err) {
      setOutput(err instanceof Error ? err.message : "Pull failed")
      toast.error("Pull failed, see the output")
    } finally {
      setPulling(null)
    }
  }

  async function handleConfirmDeleteBranch() {
    if (!pendingDelete) return
    setDeleting(true)
    try {
      await api.git.branchDelete(appId, pendingDelete.name, forceDelete)
      toast.success(`Branch ${pendingDelete.name} deleted`)
      setPendingDelete(null)
      await load()
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed to delete branch"
      setOutput(msg)
      toast.error("Failed to delete branch, see the output")
    } finally {
      setDeleting(false)
    }
  }

  async function handleBulkConfirm() {
    if (!bulk) return
    setBulkBusy(true)
    const failed: string[] = []
    let ok = 0
    const msg = (err: unknown) =>
      err instanceof Error ? err.message : "failed"
    if (bulk === "worktrees") {
      for (const w of selectedWorktrees) {
        try {
          await api.git.worktreeRemove(appId, w.path, bulkForce)
          ok++
        } catch (err) {
          failed.push(`${w.branch || w.path}: ${msg(err)}`)
          continue
        }
        if (bulkWithBranches && w.branch) {
          try {
            await api.git.branchDelete(appId, w.branch, bulkForce)
          } catch (err) {
            failed.push(`branch ${w.branch}: ${msg(err)}`)
          }
        }
      }
    } else {
      for (const b of selectedBranches) {
        try {
          await api.git.branchDelete(appId, b.name, bulkForce)
          ok++
        } catch (err) {
          failed.push(`${b.name}: ${msg(err)}`)
        }
      }
    }
    const noun = bulk === "worktrees" ? "worktree" : "branch"
    if (ok > 0) toast.success(`Deleted ${ok} ${noun}${ok === 1 ? "" : "s"}`)
    if (failed.length > 0) {
      setOutput(failed.join("\n"))
      toast.error(`${failed.length} failed, see the output`)
    }
    setSelWorktrees([])
    setSelBranches([])
    setBulk(null)
    setBulkBusy(false)
    await load()
  }

  if (info && !info.is_repo) {
    return (
      <Empty className="border">
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <FolderGit2Icon />
          </EmptyMedia>
          <EmptyTitle>Not a git repository</EmptyTitle>
          <EmptyDescription>
            The project path is not inside a git repository.
          </EmptyDescription>
        </EmptyHeader>
        <Button variant="outline" onClick={() => void load()}>
          <RefreshCwIcon data-icon="inline-start" />
          Check again
        </Button>
      </Empty>
    )
  }

  const worktrees = info?.worktrees ?? []
  const allBranches = info?.local_branches ?? []
  // Branches whose tip commit falls inside the chosen date-time range.
  const fromMs = dateFrom ? new Date(dateFrom).getTime() : NaN
  const toMs = dateTo ? new Date(dateTo).getTime() : NaN
  const filtering = !Number.isNaN(fromMs) || !Number.isNaN(toMs)
  const localBranches = filtering
    ? allBranches.filter((b) => {
        const t = b.date * 1000
        return (
          (Number.isNaN(fromMs) || t >= fromMs) &&
          (Number.isNaN(toMs) || t <= toMs)
        )
      })
    : allBranches
  // The branch the app's own folder is on (empty when detached).
  const currentBranch = worktrees.find((w) => w.current)?.branch ?? ""
  // The worktree the app's session is in: the app's own one when status names none.
  const runningIn = (w: GitWorktree) =>
    running && (status?.worktree ? status.worktree === w.path : w.current)
  // What a bulk delete may touch: not the main or app worktree, not one running,
  // and no branch that is checked out.
  const removableWorktrees = worktrees.filter(
    (w) => !w.main && !w.current && !runningIn(w)
  )
  const deletableBranches = localBranches.filter((b) => !b.worktree)
  const selectedWorktrees = removableWorktrees.filter((w) =>
    selWorktrees.includes(w.path)
  )
  const selectedBranches = deletableBranches.filter((b) =>
    selBranches.includes(b.name)
  )

  return (
    <>
      <div className="flex flex-col gap-6">
        <section className="flex flex-col gap-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div>
              <p className="text-sm font-medium">Remotes</p>
              <p className="text-sm text-muted-foreground">
                Fetch every remote and prune branches deleted upstream.
              </p>
            </div>
            <Button disabled={fetching} onClick={() => void handleFetch()}>
              <DownloadCloudIcon data-icon="inline-start" />
              {fetching ? "Fetching…" : "Fetch all & prune"}
            </Button>
          </div>
        </section>

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
                            (w.bare
                              ? "bare"
                              : `detached @ ${w.head.slice(0, 7)}`)}
                        </span>
                        {w.main ? (
                          <Badge variant="secondary">main</Badge>
                        ) : null}
                        {w.current ? <Badge>this app</Badge> : null}
                        {w.locked ? (
                          <Badge variant="outline">locked</Badge>
                        ) : null}
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
                                  {s.id === activeConfigSetId
                                    ? " (active)"
                                    : ""}
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

        <section className="flex flex-col gap-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <p className="text-sm font-medium">Local branches</p>
                <Badge variant="secondary">
                  {filtering
                    ? `${localBranches.length} of ${allBranches.length}`
                    : localBranches.length}
                </Badge>
              </div>
              <p className="text-sm text-muted-foreground">
                Create, rename and delete local branches. A branch checked out
                in a worktree cannot be deleted.
              </p>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <DateRangeFilter
                disabled={!info || allBranches.length === 0}
                from={dateFrom}
                to={dateTo}
                onChange={(f, t) => {
                  setDateFrom(f)
                  setDateTo(t)
                }}
              />
              <Button
                variant="outline"
                disabled={!info}
                onClick={() => setBranchCreateOpen(true)}
              >
                <PlusIcon data-icon="inline-start" />
                New branch
              </Button>
            </div>
          </div>

          {!info ? (
            <Skeleton className="h-14 w-full" />
          ) : localBranches.length === 0 ? (
            <p className="rounded-lg border px-3 py-2 text-sm text-muted-foreground">
              {filtering
                ? "No branches have a last commit in this range."
                : "No local branches yet."}
            </p>
          ) : (
            <>
              {deletableBranches.length > 0 ? (
                <BulkBar
                  label="branches"
                  total={deletableBranches.length}
                  selected={selectedBranches.length}
                  onToggleAll={(all) =>
                    setSelBranches(
                      all ? deletableBranches.map((b) => b.name) : []
                    )
                  }
                  onDelete={() => {
                    setBulkForce(false)
                    setBulk("branches")
                  }}
                />
              ) : null}
              <ul className="divide-y rounded-lg border">
                {localBranches.map((b) => (
                  <li
                    key={b.name}
                    className="flex flex-wrap items-center gap-3 px-3 py-2"
                  >
                    {deletableBranches.length > 0 ? (
                      <Checkbox
                        aria-label={`Select branch ${b.name}`}
                        disabled={!!b.worktree}
                        checked={selBranches.includes(b.name)}
                        onCheckedChange={(checked) =>
                          setSelBranches((cur) =>
                            checked
                              ? [...cur, b.name]
                              : cur.filter((n) => n !== b.name)
                          )
                        }
                      />
                    ) : null}
                    <div className="min-w-48 flex-1">
                      <div className="flex flex-wrap items-center gap-1.5">
                        <GitBranchIcon className="size-3.5 text-muted-foreground" />
                        <span className="text-sm font-medium">{b.name}</span>
                        {b.name === currentBranch ? (
                          <Badge title={b.worktree}>this app</Badge>
                        ) : b.worktree ? (
                          <Badge variant="secondary" title={b.worktree}>
                            checked out
                          </Badge>
                        ) : null}
                        {b.upstream ? (
                          <Badge
                            variant="outline"
                            title={b.gone ? "Upstream branch was deleted" : ""}
                          >
                            {b.upstream}
                            {b.gone ? " · gone" : ""}
                            {b.ahead ? ` · ↑${b.ahead}` : ""}
                            {b.behind ? ` · ↓${b.behind}` : ""}
                          </Badge>
                        ) : null}
                      </div>
                      <p className="truncate text-xs text-muted-foreground">
                        {b.date ? new Date(b.date * 1000).toLocaleString() : ""}
                      </p>
                    </div>
                    <div className="flex items-center gap-2">
                      {!b.upstream || b.gone || b.ahead > 0 ? (
                        <IconAction
                          label={
                            b.upstream && !b.gone
                              ? `Push ${b.ahead} commit${b.ahead === 1 ? "" : "s"} to ${b.upstream}`
                              : "Publish branch to a remote"
                          }
                          disabled={pushing !== null}
                          onClick={() => void handlePush(b)}
                        >
                          <ArrowUpFromLineIcon />
                        </IconAction>
                      ) : null}
                      {b.worktree && b.upstream && !b.gone ? (
                        <IconAction
                          label={
                            running
                              ? "Stop the app to pull"
                              : `Pull ${b.upstream} into ${b.name}`
                          }
                          disabled={pulling !== null || running}
                          onClick={() => void handlePull(b)}
                        >
                          <ArrowDownToLineIcon />
                        </IconAction>
                      ) : null}
                      {!b.worktree ? (
                        <IconAction
                          label={
                            running
                              ? "Stop the app to switch branches"
                              : "Check out in this app's folder"
                          }
                          disabled={checkingOut || running}
                          onClick={() => void handleCheckout(b)}
                        >
                          <CheckIcon />
                        </IconAction>
                      ) : null}
                      <IconAction
                        label="Rename branch"
                        onClick={() => setRenaming(b)}
                      >
                        <PencilIcon />
                      </IconAction>
                      <IconAction
                        label={
                          b.worktree
                            ? "Checked out in a worktree: cannot delete"
                            : "Delete branch"
                        }
                        variant="ghost"
                        disabled={!!b.worktree}
                        onClick={() => {
                          setForceDelete(false)
                          setPendingDelete(b)
                        }}
                      >
                        <Trash2Icon />
                      </IconAction>
                    </div>
                  </li>
                ))}
              </ul>
            </>
          )}
        </section>

        {output != null ? (
          <section className="flex flex-col gap-2">
            <p className="text-sm font-medium">Last git output</p>
            <pre className="max-h-40 overflow-auto rounded-md bg-muted p-2 font-mono text-xs whitespace-pre-wrap">
              {output}
            </pre>
          </section>
        ) : null}
      </div>

      {info ? (
        <AddWorktreeDialog
          appId={appId}
          info={info}
          links={links}
          open={addOpen}
          onOpenChange={setAddOpen}
          onAdded={() => void load()}
          onOutput={setOutput}
        />
      ) : null}

      <SharedLinksDialog
        appId={appId}
        links={links}
        open={linksOpen}
        onOpenChange={setLinksOpen}
        onSaved={setLinks}
      />

      {info ? (
        <BranchNameDialog
          appId={appId}
          branches={info.branches}
          renaming={renaming}
          open={branchCreateOpen || !!renaming}
          onOpenChange={(open) => {
            if (!open) {
              setBranchCreateOpen(false)
              setRenaming(null)
            }
          }}
          onDone={() => void load()}
          onOutput={setOutput}
        />
      ) : null}

      <Dialog
        open={!!conflicts}
        onOpenChange={(open) => {
          if (!open) setConflicts(null)
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Merge conflicts</DialogTitle>
            <DialogDescription>
              The pull stopped because these files conflict. Open them in an
              editor, resolve the conflict markers, then commit in{" "}
              <span className="font-mono">{conflicts?.path}</span> to finish the
              merge.
            </DialogDescription>
          </DialogHeader>
          <ul className="max-h-64 divide-y overflow-auto rounded-lg border">
            {conflicts?.conflicts.map((file) => (
              <li key={file} className="flex items-center gap-2 px-3 py-2">
                <span
                  className="min-w-0 flex-1 truncate font-mono text-xs"
                  title={file}
                >
                  {conflicts.path && file.startsWith(conflicts.path)
                    ? file.slice(conflicts.path.length).replace(/^[\\/]/, "")
                    : file}
                </span>
                <OpenEditorButton appId={appId} filePath={file} iconOnly />
              </li>
            ))}
          </ul>
          <DialogFooter>
            <Button onClick={() => setConflicts(null)}>Close</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={!!rejected}
        onOpenChange={(open) => {
          if (!open) setRejected(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Push rejected</AlertDialogTitle>
            <AlertDialogDescription>
              The remote has commits{" "}
              <span className="font-mono">{rejected?.name}</span> does not. Pull
              first to merge them, or force push with lease: it overwrites the
              remote branch, but only if nobody pushed to it since your last
              fetch.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={pushing !== null}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={pushing !== null}
              onClick={(event) => {
                // Keep the dialog open until the push finishes.
                event.preventDefault()
                if (rejected) void handlePush(rejected, true)
              }}
            >
              {pushing !== null ? "Pushing…" : "Force push"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={!!pendingDelete}
        onOpenChange={(open) => {
          if (!open) setPendingDelete(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete branch?</AlertDialogTitle>
            <AlertDialogDescription>
              Deletes the local branch{" "}
              <span className="font-mono">{pendingDelete?.name}</span>. A branch
              that is not fully merged is refused unless you force it. Its
              remote branch is not touched.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <Field orientation="horizontal" className="gap-2">
            <Checkbox
              id="git-delete-branch-force"
              checked={forceDelete}
              onCheckedChange={(checked) => setForceDelete(checked === true)}
            />
            <FieldLabel
              htmlFor="git-delete-branch-force"
              className="text-xs font-normal text-muted-foreground"
            >
              Force: delete even if it has unmerged commits
            </FieldLabel>
          </Field>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleting}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={deleting}
              onClick={() => void handleConfirmDeleteBranch()}
            >
              {deleting ? "Deleting…" : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={bulk !== null}
        onOpenChange={(open) => {
          if (!open && !bulkBusy) setBulk(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              Delete{" "}
              {bulk === "worktrees"
                ? selectedWorktrees.length
                : selectedBranches.length}{" "}
              {bulk === "worktrees" ? "worktrees" : "branches"}?
            </AlertDialogTitle>
            <AlertDialogDescription>
              {bulk === "worktrees"
                ? "Deletes these worktree folders:"
                : "Deletes these local branches (their remote branches are not touched):"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <ul className="max-h-40 overflow-y-auto rounded-md border p-2 font-mono text-xs">
            {bulk === "worktrees"
              ? selectedWorktrees.map((w) => (
                  <li key={w.path} className="truncate" title={w.path}>
                    {w.branch || w.path}
                  </li>
                ))
              : selectedBranches.map((b) => (
                  <li key={b.name} className="truncate">
                    {b.name}
                  </li>
                ))}
          </ul>
          {bulk === "worktrees" ? (
            <Field orientation="horizontal" className="gap-2">
              <Checkbox
                id="git-bulk-with-branches"
                checked={bulkWithBranches}
                onCheckedChange={(checked) =>
                  setBulkWithBranches(checked === true)
                }
              />
              <FieldLabel
                htmlFor="git-bulk-with-branches"
                className="text-xs font-normal text-muted-foreground"
              >
                Also delete their local branches
              </FieldLabel>
            </Field>
          ) : null}
          <Field orientation="horizontal" className="gap-2">
            <Checkbox
              id="git-bulk-force"
              checked={bulkForce}
              onCheckedChange={(checked) => setBulkForce(checked === true)}
            />
            <FieldLabel
              htmlFor="git-bulk-force"
              className="text-xs font-normal text-muted-foreground"
            >
              {bulk === "worktrees"
                ? "Force: also discard uncommitted changes and delete unmerged branches"
                : "Force: delete even if they have unmerged commits"}
            </FieldLabel>
          </Field>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={bulkBusy}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={bulkBusy}
              onClick={(event) => {
                event.preventDefault()
                void handleBulkConfirm()
              }}
            >
              {bulkBusy ? "Deleting…" : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={!!pendingRemove}
        onOpenChange={(open) => {
          if (!open) setPendingRemove(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove worktree?</AlertDialogTitle>
            <AlertDialogDescription>
              Deletes the folder{" "}
              <span className="font-mono">{pendingRemove?.path}</span>. The
              branch{pendingRemove?.branch ? ` ${pendingRemove.branch}` : ""} is
              kept.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <Field orientation="horizontal" className="gap-2">
            <Checkbox
              id="git-remove-force"
              checked={force}
              onCheckedChange={(checked) => setForce(checked === true)}
            />
            <FieldLabel
              htmlFor="git-remove-force"
              className="text-xs font-normal text-muted-foreground"
            >
              Force: also discard uncommitted changes
            </FieldLabel>
          </Field>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={removing}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={removing}
              onClick={() => void handleConfirmRemove()}
            >
              {removing ? "Removing…" : "Remove"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}

/** Creates a local branch, or with `renaming` set renames that one. */
/** Select-all checkbox with a "Delete selected" button above a list. */
// "YYYY-MM-DDTHH:mm" in local time, the value format of the range filter.
function splitStamp(v: string): { date?: Date; time: string } {
  if (!v) return { time: "" }
  const [d, t = ""] = v.split("T")
  const [y, m, day] = d.split("-").map(Number)
  return { date: new Date(y, m - 1, day), time: t }
}

function joinStamp(date: Date | undefined, time: string, fallback: string) {
  if (!date) return ""
  const p = (n: number) => String(n).padStart(2, "0")
  return `${date.getFullYear()}-${p(date.getMonth() + 1)}-${p(date.getDate())}T${time || fallback}`
}

function DateRangeFilter({
  from,
  to,
  disabled,
  onChange,
}: {
  from: string
  to: string
  disabled?: boolean
  onChange: (from: string, to: string) => void
}) {
  const f = splitStamp(from)
  const t = splitStamp(to)
  const fmt = (v: string) =>
    new Date(v).toLocaleString([], {
      dateStyle: "medium",
      timeStyle: "short",
    })
  const label =
    from || to
      ? `${from ? fmt(from) : "…"} – ${to ? fmt(to) : "…"}`
      : "Filter by date"
  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button variant="outline" disabled={disabled}>
            <CalendarIcon data-icon="inline-start" />
            {label}
          </Button>
        }
      />
      <PopoverContent align="end" className="w-auto gap-3">
        <Calendar
          mode="range"
          numberOfMonths={1}
          selected={{ from: f.date, to: t.date }}
          onSelect={(r) =>
            onChange(
              joinStamp(r?.from, f.time, "00:00"),
              joinStamp(r?.to, t.time, "23:59")
            )
          }
        />
        <div className="flex items-center gap-2">
          <Input
            type="time"
            aria-label="From time"
            disabled={!f.date}
            value={f.time}
            onChange={(e) =>
              onChange(joinStamp(f.date, e.target.value, "00:00"), to)
            }
          />
          <span className="text-muted-foreground">–</span>
          <Input
            type="time"
            aria-label="To time"
            disabled={!t.date}
            value={t.time}
            onChange={(e) =>
              onChange(from, joinStamp(t.date, e.target.value, "23:59"))
            }
          />
        </div>
        <Button
          variant="ghost"
          size="sm"
          disabled={!from && !to}
          onClick={() => onChange("", "")}
        >
          Clear
        </Button>
      </PopoverContent>
    </Popover>
  )
}

function BulkBar({
  label,
  total,
  selected,
  onToggleAll,
  onDelete,
}: {
  label: string
  total: number
  selected: number
  onToggleAll: (all: boolean) => void
  onDelete: () => void
}) {
  return (
    <div className="flex items-center gap-2 px-1 text-sm">
      <Checkbox
        aria-label={`Select all ${label}`}
        checked={selected === total}
        onCheckedChange={(checked) => onToggleAll(checked === true)}
      />
      <span className="text-xs text-muted-foreground">
        {selected > 0 ? `${selected} of ${total} selected` : `Select ${label}`}
      </span>
      {/* Always laid out, so showing it does not change the bar's height. */}
      <Button
        variant="destructive"
        size="sm"
        className={cn("ml-auto", selected === 0 && "invisible")}
        tabIndex={selected === 0 ? -1 : undefined}
        aria-hidden={selected === 0}
        onClick={onDelete}
      >
        <Trash2Icon data-icon="inline-start" />
        Delete selected
      </Button>
    </div>
  )
}

function BranchNameDialog({
  appId,
  branches,
  renaming,
  open,
  onOpenChange,
  onDone,
  onOutput,
}: {
  appId: number
  branches: string[]
  renaming: GitLocalBranch | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onDone: () => void
  onOutput: (output: string) => void
}) {
  const [name, setName] = useState("")
  const [base, setBase] = useState("")
  const [saving, setSaving] = useState(false)

  // The dialog stays mounted, so the form is reset each time it opens.
  const [wasOpen, setWasOpen] = useState(open)
  if (open !== wasOpen) {
    setWasOpen(open)
    if (open) {
      setName(renaming?.name ?? "")
      setBase("")
    }
  }

  const isRename = !!renaming

  async function handleSave() {
    const next = name.trim()
    if (!next) {
      toast.error("Branch name is required")
      return
    }
    setSaving(true)
    try {
      if (renaming) {
        await api.git.branchRename(appId, renaming.name, next)
        toast.success(`Renamed ${renaming.name} to ${next}`)
      } else {
        await api.git.branchCreate(appId, next, base.trim())
        toast.success(`Branch ${next} created`)
      }
      onOpenChange(false)
      onDone()
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed"
      onOutput(msg)
      toast.error(`${renaming ? "Rename" : "Create"} failed: ${msg}`)
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isRename ? "Rename branch" : "New branch"}</DialogTitle>
          <DialogDescription>
            {isRename ? (
              <>
                Renames <span className="font-mono">{renaming?.name}</span>. A
                worktree that has it checked out follows the new name; its
                upstream and remote branch are not renamed.
              </>
            ) : (
              "Creates a local branch without checking it out."
            )}
          </DialogDescription>
        </DialogHeader>
        <datalist id="git-base-branches">
          {branches.map((b) => (
            <option key={b} value={b} />
          ))}
        </datalist>
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="git-branch-name">
              {isRename ? "New name" : "Branch name"}
            </FieldLabel>
            <Input
              id="git-branch-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && !saving) void handleSave()
              }}
              placeholder="feature/my-change"
              autoFocus
            />
          </Field>
          {!isRename ? (
            <Field>
              <FieldLabel htmlFor="git-branch-base">Start from</FieldLabel>
              <Input
                id="git-branch-base"
                value={base}
                onChange={(e) => setBase(e.target.value)}
                placeholder="HEAD"
                list="git-base-branches"
              />
              <FieldDescription>
                A branch, tag or commit. Empty: the current HEAD of the app's
                folder.
              </FieldDescription>
            </Field>
          ) : null}
        </FieldGroup>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button disabled={saving} onClick={() => void handleSave()}>
            {saving
              ? isRename
                ? "Renaming…"
                : "Creating…"
              : isRename
                ? "Rename"
                : "Create"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

/** Common candidates, offered as one-click additions. */
const LINK_SUGGESTIONS = [
  "node_modules",
  ".env",
  ".env.local",
  ".venv",
  "vendor",
]

function SharedLinksDialog({
  appId,
  links,
  open,
  onOpenChange,
  onSaved,
}: {
  appId: number
  links: string[]
  open: boolean
  onOpenChange: (open: boolean) => void
  onSaved: (links: string[]) => void
}) {
  const [text, setText] = useState("")
  const [saving, setSaving] = useState(false)

  // The dialog stays mounted, so the draft is reset each time it opens.
  const [wasOpen, setWasOpen] = useState(open)
  if (open !== wasOpen) {
    setWasOpen(open)
    if (open) setText(links.join("\n"))
  }

  const draft = text
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean)

  function addSuggestion(path: string) {
    setText([...draft, path].join("\n"))
  }

  async function handleSave() {
    setSaving(true)
    try {
      onSaved(await api.git.linksSave(appId, draft))
      toast.success("Shared files saved")
      onOpenChange(false)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to save")
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Shared files</DialogTitle>
          <DialogDescription>
            Files and folders each worktree links to in this app's folder
            instead of having its own copy. They are linked when a worktree is
            added (unless turned off there), before Setup runs in a worktree, or
            with the link button on its row.
          </DialogDescription>
        </DialogHeader>
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="git-shared-links">
              Paths, one per line
            </FieldLabel>
            <Textarea
              id="git-shared-links"
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder={"node_modules\n.env"}
              rows={6}
              className="font-mono text-xs"
            />
            <FieldDescription>
              Relative to the app folder. A worktree that already has a real
              file or folder at a path keeps it: delete it there to share the
              app's one. Changes made through a link change the app's own copy.
            </FieldDescription>
          </Field>
          <div className="flex flex-wrap gap-1.5">
            {LINK_SUGGESTIONS.filter((s) => !draft.includes(s)).map((s) => (
              <Button
                key={s}
                variant="outline"
                size="sm"
                className="font-mono text-xs"
                onClick={() => addSuggestion(s)}
              >
                <PlusIcon data-icon="inline-start" />
                {s}
              </Button>
            ))}
          </div>
        </FieldGroup>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button disabled={saving} onClick={() => void handleSave()}>
            {saving ? "Saving…" : "Save"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function AddWorktreeDialog({
  appId,
  info,
  links,
  open,
  onOpenChange,
  onAdded,
  onOutput,
}: {
  appId: number
  info: GitInfo
  links: string[]
  open: boolean
  onOpenChange: (open: boolean) => void
  onAdded: () => void
  onOutput: (output: string) => void
}) {
  const [branch, setBranch] = useState("")
  const [newBranch, setNewBranch] = useState(true)
  const [base, setBase] = useState("")
  const [path, setPath] = useState("")
  const [link, setLink] = useState(true)
  const [linkText, setLinkText] = useState("")
  const [saving, setSaving] = useState(false)

  // The dialog stays mounted, so the form is reset each time it opens.
  const [wasOpen, setWasOpen] = useState(open)
  if (open !== wasOpen) {
    setWasOpen(open)
    if (open) {
      setBranch("")
      setNewBranch(true)
      setBase("")
      setPath("")
      setLink(links.length > 0)
      setLinkText(links.join("\n"))
    }
  }

  const linkPaths = linkText
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean)
  // Paths that differ from the app's become this worktree's own list.
  const ownLinks = linkPaths.join("\n") !== links.join("\n")
  const mainPath = info.worktrees[0]?.path ?? ""
  const defaultPath =
    branch.trim() && mainPath
      ? defaultWorktreePath(mainPath, branch.trim())
      : "Sibling folder <repo>-<branch>"

  async function handleAdd() {
    if (!branch.trim()) {
      toast.error("Branch is required")
      return
    }
    setSaving(true)
    try {
      const res = await api.git.worktreeAdd(appId, {
        branch: branch.trim(),
        new_branch: newBranch,
        base: newBranch ? base.trim() : "",
        path: path.trim(),
        link_shared: link && linkPaths.length > 0,
        links: link && ownLinks && linkPaths.length ? linkPaths : undefined,
      })
      toast.success(`Worktree created at ${res.path}`)
      onOpenChange(false)
      onAdded()
      if (res.links.length) reportLinkStates(res.links, onOutput)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to add worktree")
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add worktree</DialogTitle>
          <DialogDescription>
            Check out a branch into its own folder.
          </DialogDescription>
        </DialogHeader>
        <datalist id="git-branches">
          {info.branches.map((b) => (
            <option key={b} value={b} />
          ))}
        </datalist>
        <FieldGroup>
          <Field orientation="horizontal" className="gap-2">
            <Checkbox
              id="git-new-branch"
              checked={newBranch}
              onCheckedChange={(checked) => setNewBranch(checked === true)}
            />
            <FieldLabel htmlFor="git-new-branch" className="font-normal">
              Create a new branch
            </FieldLabel>
          </Field>
          <Field>
            <FieldLabel htmlFor="git-branch">
              {newBranch ? "New branch name" : "Branch"}
            </FieldLabel>
            <Input
              id="git-branch"
              value={branch}
              onChange={(e) => setBranch(e.target.value)}
              placeholder={newBranch ? "feature/my-change" : "main"}
              list={newBranch ? undefined : "git-branches"}
              autoFocus
            />
          </Field>
          {newBranch ? (
            <Field>
              <FieldLabel htmlFor="git-base">Start from</FieldLabel>
              <Input
                id="git-base"
                value={base}
                onChange={(e) => setBase(e.target.value)}
                placeholder="HEAD"
                list="git-branches"
              />
            </Field>
          ) : null}
          <Field>
            <FieldLabel htmlFor="git-path">Folder</FieldLabel>
            <Input
              id="git-path"
              value={path}
              onChange={(e) => setPath(e.target.value)}
              placeholder={defaultPath}
              className="font-mono text-xs"
            />
          </Field>
          <Field orientation="horizontal">
            <FieldContent>
              <FieldLabel htmlFor="git-link-shared">
                Link shared files
              </FieldLabel>
              <FieldDescription>
                Link these files and folders to the app's folder as soon as the
                worktree is created.
              </FieldDescription>
            </FieldContent>
            <Switch
              id="git-link-shared"
              checked={link}
              onCheckedChange={setLink}
            />
          </Field>
          {link ? (
            <Field>
              <Textarea
                id="git-link-paths"
                aria-label="Shared files for this worktree"
                value={linkText}
                onChange={(e) => setLinkText(e.target.value)}
                placeholder={"node_modules\n.env"}
                rows={4}
                className="font-mono text-xs"
              />
              <FieldDescription>
                {ownLinks ? (
                  <>
                    This worktree keeps its own list for later setups; the app's
                    Shared files stay unchanged.{" "}
                    <button
                      type="button"
                      className="underline underline-offset-4"
                      onClick={() => setLinkText(links.join("\n"))}
                    >
                      Use the app's list
                    </button>
                  </>
                ) : (
                  "One path per line, relative to the app folder. Starts as the app's Shared files; edit it to change this worktree only."
                )}
              </FieldDescription>
            </Field>
          ) : null}
        </FieldGroup>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button disabled={saving} onClick={() => void handleAdd()}>
            {saving ? "Adding…" : "Add"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
