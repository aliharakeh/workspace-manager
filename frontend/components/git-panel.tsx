import { useEffect, useState } from "react"
import {
  DownloadCloudIcon,
  EraserIcon,
  FolderGit2Icon,
  GitBranchIcon,
  HammerIcon,
  Link2Icon,
  PlayIcon,
  PlusIcon,
  RefreshCwIcon,
  SquareIcon,
  Trash2Icon,
  WrenchIcon,
} from "lucide-react"
import { toast } from "sonner"
import { api } from "@/lib/api"
import { appStateLabel } from "@/lib/app-state"
import type {
  CommandKind,
  ConfigSet,
  GitInfo,
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
  { kind: "run", label: "Run", Icon: PlayIcon },
  { kind: "build", label: "Build", Icon: HammerIcon },
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
  }, [active, appId])

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
      toast.error(err instanceof Error ? err.message : `Failed to start ${kind}`)
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
  // The worktree the app's session is in: the app's own one when status names none.
  const runningIn = (w: GitWorktree) =>
    running && (status?.worktree ? status.worktree === w.path : w.current)

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
                picked per worktree, inside that worktree. Setup first links
                the shared files.
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
            <ul className="divide-y rounded-lg border">
              {worktrees.map((w) => (
                <li
                  key={w.path}
                  className="flex flex-wrap items-center gap-3 px-3 py-2"
                >
                  <div className="min-w-48 flex-1">
                    <div className="flex flex-wrap items-center gap-1.5">
                      <GitBranchIcon className="size-3.5 text-muted-foreground" />
                      <span className="text-sm font-medium">
                        {w.branch ||
                          (w.bare
                            ? "bare"
                            : `detached @ ${w.head.slice(0, 7)}`)}
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
              branch{pendingRemove?.branch ? ` ${pendingRemove.branch}` : ""}{" "}
              is kept.
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

/** Common candidates, offered as one-click additions. */
const LINK_SUGGESTIONS = ["node_modules", ".env", ".env.local", ".venv", "vendor"]

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
            added (unless turned off there), before Setup runs in a worktree,
            or with the link button on its row.
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
              <FieldLabel htmlFor="git-link-shared">Link shared files</FieldLabel>
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
                    This worktree keeps its own list for later setups; the
                    app's Shared files stay unchanged.{" "}
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
