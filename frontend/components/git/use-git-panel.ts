import { api } from "@/lib/api"
import type {
  CommandKind,
  ConfigSet,
  GitInfo,
  GitLocalBranch,
  GitPullResult,
  GitWorktree,
  StatusEvent,
} from "@/lib/types"
import { useEffect, useState } from "react"
import { toast } from "sonner"
import { linkSharedPaths } from "./worktree-links"

export type GitPanelProps = {
  appId: number
  active: boolean
  /** Bumped when another client writes this app's git state, to reload it. */
  dataEpoch: number
  /** The app's active config set: used by every GUI launch. */
  activeConfigSetId: number | null
  status: StatusEvent | null
  onStatus: (status: StatusEvent) => void
}

function fetchGitState(appId: number) {
  return Promise.all([
    api.git.info(appId),
    api.configSets.list(appId) as Promise<ConfigSet[]>,
    api.git.linksGet(appId),
  ])
}

export function useGitPanel({
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

  function setName(id: number | undefined) {
    return sets.find((s) => s.id === id)?.name
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
      onStatus(await api.git.worktreeStart(appId, w.path, kind, 0))
      toast.success(
        `${label} started in ${w.branch || w.path} with ${setName(activeConfigSetId ?? undefined) ?? "the active set"}`
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
  return {
    appId,
    activeConfigSetId,
    status,
    info,
    sets,
    links,
    setLinks,
    linksOpen,
    setLinksOpen,
    linking,
    loading,
    fetching,
    pruning,
    output,
    setOutput,
    addOpen,
    setAddOpen,
    pendingRemove,
    setPendingRemove,
    force,
    setForce,
    removing,
    starting,
    branchCreateOpen,
    setBranchCreateOpen,
    renaming,
    setRenaming,
    pendingDelete,
    setPendingDelete,
    forceDelete,
    setForceDelete,
    deleting,
    checkingOut,
    selWorktrees,
    setSelWorktrees,
    selBranches,
    setSelBranches,
    bulk,
    setBulk,
    bulkForce,
    setBulkForce,
    bulkWithBranches,
    setBulkWithBranches,
    bulkBusy,
    dateFrom,
    setDateFrom,
    dateTo,
    setDateTo,
    pulling,
    conflicts,
    setConflicts,
    pushing,
    rejected,
    setRejected,
    running,
    load,
    setName,
    handleFetch,
    handlePrune,
    handleStart,
    handleLink,
    handleStop,
    handleConfirmRemove,
    handleCheckout,
    handlePush,
    handlePull,
    handleConfirmDeleteBranch,
    handleBulkConfirm,
    worktrees,
    allBranches,
    filtering,
    localBranches,
    currentBranch,
    runningIn,
    removableWorktrees,
    deletableBranches,
    selectedWorktrees,
    selectedBranches,
  }
}

export type GitPanelModel = ReturnType<typeof useGitPanel>
