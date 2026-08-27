import { useEffect, useMemo, useRef, useState, type ReactNode } from "react"
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  CloudDownloadIcon,
  ExternalLinkIcon,
  GitMergeIcon,
  Loader2Icon,
  RefreshCwIcon,
  SearchIcon,
  XIcon,
} from "lucide-react"
import { toast } from "sonner"
import { api, handleReadyUrlClick } from "@/lib/api"
import type {
  GitBranchInfo,
  GitRemoteInfo,
  GitRepoGraph,
} from "@/lib/types"
import { TimelineGraph, laneName, type GitInspect } from "@/components/timeline-graph"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group"
import { ScrollArea } from "@/components/ui/scroll-area"

const DEFAULT_BRANCHES = 10
const CHUNK_MONTHS = 5

function authorName(c: { author?: string }) {
  return c.author || "(unknown)"
}

function localDay(ts: string) {
  const d = new Date(ts)
  return `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`
}

function lanesByUpdated(list: GitBranchInfo[]) {
  const latest = new Map<string, number>()
  for (const b of list || []) {
    const name = laneName(b.name)
    const t = b.updated ? +new Date(b.updated) : 0
    if (!latest.has(name) || t > latest.get(name)!) latest.set(name, t)
  }
  return [...latest.keys()].sort((a, b) => latest.get(b)! - latest.get(a)!)
}

function addMonths(ms: number, n: number) {
  const d = new Date(ms)
  d.setMonth(d.getMonth() + n)
  return +d
}

function iso(ms: number) {
  return new Date(ms).toISOString()
}

function mergeGraphs(prev: GitRepoGraph | null, chunk: GitRepoGraph | null) {
  if (!prev) return chunk
  if (!chunk) return prev
  const seen = new Set((prev.commits || []).map((c) => c.hash))
  const commits = [...(prev.commits || [])]
  for (const c of chunk.commits || []) {
    if (!seen.has(c.hash)) commits.push(c)
  }
  const mseen = new Set((prev.merges || []).map((m) => `${m.hash}:${m.sourceHash}`))
  const merges = [...(prev.merges || [])]
  for (const m of chunk.merges || []) {
    const k = `${m.hash}:${m.sourceHash}`
    if (!mseen.has(k)) merges.push(m)
  }
  const bseen = new Set(prev.branches || [])
  const branches = [...(prev.branches || [])]
  for (const b of chunk.branches || []) {
    if (!bseen.has(b)) branches.push(b)
  }
  return { ...prev, branches, commits, merges }
}

function branchKey(names: string[]) {
  return [...names].sort().join("\n")
}

function chunkEmpty(chunk: GitRepoGraph | null) {
  return !(chunk?.commits?.length || chunk?.merges?.length)
}

function viewCovered(
  loaded: { from: number; to: number; pastDone: boolean; futureDone: boolean },
  from: number,
  to: number
) {
  if (!loaded.from && !loaded.to) return false
  const slack = 1000
  return (
    (from >= loaded.from - slack || loaded.pastDone) &&
    (to <= loaded.to + slack || loaded.futureDone)
  )
}

function monthQueue(
  loaded: { from: number; to: number; pastDone: boolean; futureDone: boolean },
  want: { from: number; to: number }
) {
  let n = 0
  if (!loaded.pastDone) {
    for (let t = loaded.from, i = 0; t > want.from && i < 200; i++) {
      t = addMonths(t, -CHUNK_MONTHS)
      n++
    }
  }
  const cap = Math.min(want.to, Date.now())
  if (!loaded.futureDone) {
    for (let t = loaded.to, i = 0; t < cap && i < 200; i++) {
      t = addMonths(t, CHUNK_MONTHS)
      n++
    }
  }
  return n
}

type GitGraphPanelProps = {
  appId: number
  projectPath: string
}

export function GitGraphPanel({ appId, projectPath }: GitGraphPanelProps) {
  const [graph, setGraph] = useState<GitRepoGraph | null>(null)
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)
  const [remote, setRemote] = useState<GitRemoteInfo | null>(null)
  const [fetching, setFetching] = useState(false)
  const [query, setQuery] = useState("")
  const [msgQuery, setMsgQuery] = useState("")
  const [hitIndex, setHitIndex] = useState(-1)
  const [jumpTo, setJumpTo] = useState<{ hash: string; n: number } | null>(null)
  const [authorQuery, setAuthorQuery] = useState("")
  const [focused, setFocused] = useState("")
  const [selected, setSelected] = useState<GitInspect | null>(null)
  const lastSelected = useRef<GitInspect | null>(null)
  if (selected) lastSelected.current = selected
  const inspect = selected || lastSelected.current
  const [catalog, setCatalog] = useState<GitBranchInfo[]>([])
  const [axisRange, setAxisRange] = useState<[number, number] | null>(null)
  const [visible, setVisible] = useState(() => new Set<string>())
  const [authors, setAuthors] = useState(() => new Set<string>())
  const [historyLeft, setHistoryLeft] = useState(0)
  const [viewGen, setViewGen] = useState(0)
  const visibleRef = useRef(visible)
  visibleRef.current = visible
  const loadSeq = useRef(0)
  const reloadTimer = useRef(0)
  const loadedRef = useRef({
    from: 0,
    to: 0,
    branches: "",
    pastDone: false,
    futureDone: false,
    emptyPast: 0,
    emptyFuture: 0,
  })
  const wantRef = useRef({ from: 0, to: 0 })
  const filling = useRef(false)

  async function load({ reset = true, branches }: { reset?: boolean; branches?: string[] } = {}) {
    const gen = ++loadSeq.current
    setLoading(true)
    setError("")
    if (reset) {
      setSelected(null)
      setFocused("")
    }
    try {
      let selectedBranches = branches
      if (reset) {
        const list = await api.apps.git.branches(appId)
        if (gen !== loadSeq.current) return
        setCatalog(list)
        const names = lanesByUpdated(list)
        selectedBranches = names.slice(0, DEFAULT_BRANCHES)
        setVisible(new Set(selectedBranches))
        setMsgQuery("")
        setAuthorQuery("")
      } else if (!selectedBranches) {
        selectedBranches = [...visibleRef.current]
      }
      const now = Date.now()
      const viewTo = now
      const viewFrom = addMonths(now, -CHUNK_MONTHS)
      const from = reset || !loadedRef.current.from ? addMonths(now, -CHUNK_MONTHS) : loadedRef.current.from
      const to = reset || !loadedRef.current.to ? now : loadedRef.current.to
      if (reset || !loadedRef.current.from) {
        setAxisRange([viewFrom, viewTo])
        wantRef.current = { from: viewFrom, to: viewTo }
      }
      const data = await api.apps.git.load(appId, {
        branches: selectedBranches,
        since: iso(from),
        until: iso(to),
      })
      if (gen !== loadSeq.current) return
      loadedRef.current = {
        from,
        to,
        branches: branchKey(selectedBranches),
        pastDone: false,
        futureDone: false,
        emptyPast: 0,
        emptyFuture: 0,
      }
      setGraph(data)
      try {
        setRemote(await api.apps.git.remote(appId))
      } catch {
        setRemote(null)
      }
      const names = (data.commits || []).map(authorName)
      if (reset) setAuthors(new Set(names))
      else {
        setAuthors((prev) => {
          const next = new Set(prev)
          for (const n of names) next.add(n)
          return next
        })
      }
    } catch (err) {
      if (gen !== loadSeq.current) return
      if (reset) {
        setGraph(null)
        setCatalog([])
        setRemote(null)
      }
      setError(err instanceof Error ? err.message : "Failed to load git graph")
    } finally {
      if (gen === loadSeq.current) setLoading(false)
    }
  }

  useEffect(() => {
    void load({ reset: true })
    return () => {
      loadSeq.current++
      window.clearTimeout(reloadTimer.current)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- reload when the app folder changes
  }, [appId, projectPath])

  function applyVisible(next: Set<string>, { debounce = false } = {}) {
    setVisible(next)
    window.clearTimeout(reloadTimer.current)
    const run = () => load({ reset: false, branches: [...next] })
    if (debounce) reloadTimer.current = window.setTimeout(run, 150)
    else void run()
  }

  function addAuthors(commits: { author?: string }[] | undefined) {
    setAuthors((prev) => {
      const next = new Set(prev)
      for (const c of commits || []) next.add(authorName(c))
      return next
    })
  }

  async function ensureRange(viewFrom: number, viewTo: number) {
    if (filling.current) return
    const selectedBranches = [...visibleRef.current]
    const key = branchKey(selectedBranches)
    const loaded = loadedRef.current
    if (loaded.branches && loaded.branches !== key) return
    wantRef.current = { from: viewFrom, to: viewTo }
    const left = monthQueue(loaded, wantRef.current)
    if (viewCovered(loaded, viewFrom, viewTo)) {
      if (!filling.current) setHistoryLeft(0)
      return
    }
    setHistoryLeft(left)
    filling.current = true
    const gen = loadSeq.current
    try {
      while (gen === loadSeq.current) {
        const cur = loadedRef.current
        const want = wantRef.current
        const queued = monthQueue(cur, want)
        setHistoryLeft(queued)
        if (cur.branches !== key || viewCovered(cur, want.from, want.to) || !queued) break
        if (!selectedBranches.length) break
        if (cur.from > want.from && !cur.pastDone) {
          const until = cur.from
          const since = addMonths(until, -CHUNK_MONTHS)
          const chunk = await api.apps.git.load(appId, {
            branches: selectedBranches,
            since: iso(since),
            until: iso(until),
          })
          if (gen !== loadSeq.current) return
          const empty = chunkEmpty(chunk)
          const emptyPast = empty ? cur.emptyPast + 1 : 0
          if (!empty) {
            setGraph((prev) => mergeGraphs(prev, chunk))
            addAuthors(chunk?.commits)
          }
          loadedRef.current = { ...loadedRef.current, from: since, emptyPast, pastDone: emptyPast >= 3 }
          continue
        }
        if (cur.to < want.to && !cur.futureDone) {
          const since = cur.to
          const until = Math.min(addMonths(since, CHUNK_MONTHS), Date.now())
          if (until <= since) {
            loadedRef.current = { ...loadedRef.current, futureDone: true }
            break
          }
          const chunk = await api.apps.git.load(appId, {
            branches: selectedBranches,
            since: iso(since),
            until: iso(until),
          })
          if (gen !== loadSeq.current) return
          const empty = chunkEmpty(chunk)
          const emptyFuture = empty ? cur.emptyFuture + 1 : 0
          if (!empty) {
            setGraph((prev) => mergeGraphs(prev, chunk))
            addAuthors(chunk?.commits)
          }
          loadedRef.current = { ...loadedRef.current, to: until, emptyFuture, futureDone: emptyFuture >= 3 }
          continue
        }
        break
      }
    } catch (err) {
      if (gen === loadSeq.current) {
        setError(err instanceof Error ? err.message : "Failed to load history")
      }
    } finally {
      filling.current = false
      setHistoryLeft(0)
    }
  }

  function refreshLogs() {
    filling.current = false
    loadedRef.current = {
      from: 0,
      to: 0,
      branches: "",
      pastDone: false,
      futureDone: false,
      emptyPast: 0,
      emptyFuture: 0,
    }
    setViewGen((n) => n + 1)
    void load({ reset: false })
  }

  async function fetchNow() {
    setFetching(true)
    try {
      await api.apps.git.fetch(appId)
      toast.success(`Fetched ${remote?.name || "origin"}`)
      await load({ reset: true })
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Fetch failed")
    } finally {
      setFetching(false)
    }
  }

  const rankedBranches = useMemo(() => lanesByUpdated(catalog), [catalog])

  const branches = useMemo(() => {
    const q = query.trim().toLowerCase()
    return q ? rankedBranches.filter((b) => b.toLowerCase().includes(q)) : rankedBranches
  }, [rankedBranches, query])

  const highlight = useMemo(() => {
    if (focused) return focused
    const q = query.trim().toLowerCase()
    if (!q) return ""
    const exact = rankedBranches.find((b) => b.toLowerCase() === q)
    if (exact) return exact
    return branches.length === 1 ? branches[0]! : ""
  }, [focused, query, rankedBranches, branches])

  const authorList = useMemo(() => {
    if (!graph) return []
    return [...new Set((graph.commits || []).map(authorName))].sort((a, b) =>
      a.localeCompare(b, undefined, { sensitivity: "base" })
    )
  }, [graph])

  const shownAuthors = useMemo(() => {
    const q = authorQuery.trim().toLowerCase()
    return q ? authorList.filter((a) => a.toLowerCase().includes(q)) : authorList
  }, [authorList, authorQuery])

  const selectedAuthors = useMemo(() => {
    if (!authorList.length || authors.size === authorList.length) return undefined
    return authors
  }, [authors, authorList])

  const visibleGraph = useMemo(() => {
    if (!graph) return null
    const names = rankedBranches.filter((b) => visible.has(b))
    const shown = new Set(names)
    const merges = (graph.merges || [])
      .map((m) => ({
        ...m,
        sourceBranch: laneName(m.sourceBranch),
        targetBranch: laneName(m.targetBranch),
      }))
      .filter((m) => {
        if (m.kind === "branch") return shown.has(m.sourceBranch) && shown.has(m.targetBranch)
        return shown.has(m.targetBranch) || shown.has(m.sourceBranch)
      })
    return {
      ...graph,
      branches: names,
      commits: (graph.commits || [])
        .map((c) => ({
          ...c,
          branch: laneName(c.branch),
          on: (c.on || [c.branch]).map(laneName),
        }))
        .filter((c) => shown.has(c.branch)),
      merges,
    }
  }, [graph, rankedBranches, visible])

  const searchHits = useMemo(() => {
    const msg = msgQuery.trim().toLowerCase()
    if (!msg || !visibleGraph) return []
    return visibleGraph.commits
      .filter(
        (c) =>
          String(c.subject || "").toLowerCase().includes(msg) ||
          (c.tags || []).some((t) => String(t).toLowerCase().includes(msg))
      )
      .sort((a, b) => +new Date(b.timestamp) - +new Date(a.timestamp))
  }, [visibleGraph, msgQuery])
  const matchHashes = useMemo(() => searchHits.map((c) => c.hash), [searchHits])
  const curHit = hitIndex >= 0 && hitIndex < searchHits.length ? hitIndex : -1

  function goHit(dir: number) {
    const n = searchHits.length
    if (!n || !visibleGraph) return
    const i = curHit < 0 ? (dir > 0 ? 0 : n - 1) : (curHit + dir + n) % n
    const c = searchHits[i]!
    const day = localDay(c.timestamp)
    const commits = visibleGraph.commits
      .filter((x) => x.branch === c.branch && localDay(x.timestamp) === day)
      .sort((a, b) => +new Date(a.timestamp) - +new Date(b.timestamp))
    setHitIndex(i)
    setSelected(
      commits.length > 1
        ? { kind: "cluster", ...c, count: commits.length, commits }
        : { kind: "commit", ...c }
    )
    setJumpTo({ hash: c.hash, n: (jumpTo?.n || 0) + 1 })
  }

  function showTop(n: number) {
    applyVisible(new Set(rankedBranches.slice(0, Math.min(Math.max(n, 0), rankedBranches.length))), {
      debounce: true,
    })
  }

  function toggleVisible(name: string) {
    const next = new Set(visible)
    if (next.has(name)) next.delete(name)
    else next.add(name)
    applyVisible(next)
  }

  function toggleAuthor(name: string) {
    setAuthors((prev) => {
      const next = new Set(prev)
      if (next.has(name)) next.delete(name)
      else next.add(name)
      return next
    })
  }

  return (
    <div className="relative flex min-h-0 flex-1 overflow-hidden">
      <aside className="flex w-72 shrink-0 flex-col overflow-hidden border-r pr-3">
        <ScrollArea className="min-h-0 flex-1">
          <div className="flex flex-col gap-4 pr-3 pb-4">
            {error ? <p className="text-sm text-destructive">{error}</p> : null}
            {remote ? (
              <FieldGroup>
                <Field>
                  <FieldLabel>Remote</FieldLabel>
                  <div className="flex min-w-0 items-center gap-2">
                    <Badge variant="outline">{remote.name}</Badge>
                    {remote.web ? (
                      <Button
                        variant="link"
                        size="sm"
                        className="h-auto min-w-0 flex-1 justify-start px-0"
                        nativeButton={false}
                        render={<a href={remote.web} />}
                        onClick={(event) => handleReadyUrlClick(event, remote.web!)}
                      >
                        <span className="truncate">{remote.web.replace(/^https:\/\//, "")}</span>
                        <ExternalLinkIcon data-icon="inline-end" />
                      </Button>
                    ) : (
                      <span className="truncate font-mono text-xs">{remote.url}</span>
                    )}
                  </div>
                  <Button
                    variant="outline"
                    onClick={() => void fetchNow()}
                    disabled={fetching || loading}
                  >
                    {fetching ? (
                      <Loader2Icon data-icon="inline-start" className="animate-spin" />
                    ) : (
                      <CloudDownloadIcon data-icon="inline-start" />
                    )}
                    {fetching ? "Fetching…" : `Fetch ${remote.name}`}
                  </Button>
                </Field>
              </FieldGroup>
            ) : null}

            {rankedBranches.length > 0 ? (
              <FieldGroup>
                <Field>
                  <FieldLabel>Branches</FieldLabel>
                <InputGroup>
                  <InputGroupAddon>
                    <SearchIcon />
                  </InputGroupAddon>
                  <InputGroupInput
                    value={query}
                    placeholder="Search branches…"
                    onChange={(e) => setQuery(e.target.value)}
                  />
                  {query ? (
                    <InputGroupAddon align="inline-end">
                      <InputGroupButton size="icon-xs" onClick={() => setQuery("")}>
                        <XIcon />
                      </InputGroupButton>
                    </InputGroupAddon>
                  ) : null}
                </InputGroup>
                <div className="flex items-center justify-between gap-2">
                  <span className="font-mono text-xs text-muted-foreground">
                    {visible.size} / {rankedBranches.length}
                  </span>
                  <div className="flex gap-1">
                    <Button variant="ghost" size="sm" onClick={() => showTop(rankedBranches.length)}>
                      All
                    </Button>
                    <Button variant="ghost" size="sm" onClick={() => showTop(0)}>
                      None
                    </Button>
                  </div>
                </div>
                <div className="thin-scrollbar flex max-h-52 flex-col gap-1 overflow-y-auto rounded-lg border p-1">
                  {branches.length === 0 ? (
                    <p className="px-2 py-3 text-xs text-muted-foreground">No branches loaded</p>
                  ) : (
                    branches.map((name) => (
                      <Field
                        key={name}
                        orientation="horizontal"
                        className={`min-w-0 rounded-md px-1 ${highlight === name ? "bg-primary/15" : ""} ${!visible.has(name) ? "opacity-40" : ""}`}
                      >
                        <Checkbox
                          checked={visible.has(name)}
                          onCheckedChange={() => toggleVisible(name)}
                        />
                        <FieldLabel
                          className="min-w-0 flex-1 cursor-pointer truncate font-normal"
                          onClick={() => setFocused(highlight === name ? "" : name)}
                        >
                          {name}
                        </FieldLabel>
                        {highlight === name ? <Badge>focus</Badge> : null}
                      </Field>
                    ))
                  )}
                </div>
                {highlight ? (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setFocused("")
                      setQuery("")
                    }}
                  >
                    Clear highlight
                  </Button>
                ) : null}
              </Field>
            </FieldGroup>
            ) : null}

            {graph ? (
              <FieldGroup>
                <Field>
                  <div className="flex items-center justify-between gap-2">
                    <FieldLabel>Authors</FieldLabel>
                    <span className="font-mono text-xs text-muted-foreground">
                      {authors.size} / {authorList.length}
                    </span>
                  </div>
                  <InputGroup>
                    <InputGroupAddon>
                      <SearchIcon />
                    </InputGroupAddon>
                    <InputGroupInput
                      value={authorQuery}
                      placeholder="Search authors…"
                      onChange={(e) => setAuthorQuery(e.target.value)}
                    />
                    {authorQuery ? (
                      <InputGroupAddon align="inline-end">
                        <InputGroupButton size="icon-xs" onClick={() => setAuthorQuery("")}>
                          <XIcon />
                        </InputGroupButton>
                      </InputGroupAddon>
                    ) : null}
                  </InputGroup>
                  <div className="flex gap-1">
                    <Button variant="ghost" size="sm" className="flex-1" onClick={() => setAuthors(new Set(authorList))}>
                      All
                    </Button>
                    <Button variant="ghost" size="sm" className="flex-1" onClick={() => setAuthors(new Set())}>
                      None
                    </Button>
                  </div>
                  <div className="thin-scrollbar flex max-h-40 flex-col gap-1 overflow-y-auto rounded-lg border p-1">
                    {shownAuthors.length === 0 ? (
                      <p className="px-2 py-3 text-xs text-muted-foreground">No authors</p>
                    ) : (
                      shownAuthors.map((name) => (
                        <Field
                          key={name}
                          orientation="horizontal"
                          className={`min-w-0 ${!authors.has(name) ? "opacity-40" : ""}`}
                        >
                          <Checkbox checked={authors.has(name)} onCheckedChange={() => toggleAuthor(name)} />
                          <FieldLabel className="min-w-0 truncate font-normal">{name}</FieldLabel>
                        </Field>
                      ))
                    )}
                  </div>
                </Field>
              </FieldGroup>
            ) : null}
          </div>
        </ScrollArea>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <div className="flex flex-wrap items-center gap-2 border-b pb-2">
          <div className="min-w-0">
            <div className="text-sm font-medium">Network timeline</div>
            <p className="text-xs text-muted-foreground">
              Scroll to zoom · drag to pan · double-click to reset
            </p>
          </div>
          {historyLeft > 0 ? (
            <Badge variant="secondary">
              <Loader2Icon data-icon="inline-start" className="animate-spin" />
              Loading history…
            </Badge>
          ) : null}
          {graph && visibleGraph ? (
            <div className="ml-auto flex flex-wrap items-center gap-2">
              <Button
                variant="outline"
                size="icon-sm"
                onClick={refreshLogs}
                disabled={loading}
                title="Refresh graph"
              >
                <RefreshCwIcon className={loading ? "animate-spin" : undefined} />
              </Button>
              <InputGroup className="w-52">
                <InputGroupAddon>
                  <SearchIcon />
                </InputGroupAddon>
                <InputGroupInput
                  value={msgQuery}
                  placeholder="Search commits…"
                  onChange={(e) => {
                    setMsgQuery(e.target.value)
                    setHitIndex(-1)
                  }}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      e.preventDefault()
                      goHit(e.shiftKey ? -1 : 1)
                    }
                  }}
                />
                {msgQuery ? (
                  <InputGroupAddon align="inline-end">
                    <InputGroupButton
                      size="icon-xs"
                      onClick={() => {
                        setMsgQuery("")
                        setHitIndex(-1)
                      }}
                    >
                      <XIcon />
                    </InputGroupButton>
                  </InputGroupAddon>
                ) : null}
              </InputGroup>
              {msgQuery.trim() ? (
                <>
                  <span className="min-w-10 text-center font-mono text-xs tabular-nums text-muted-foreground">
                    {searchHits.length ? (curHit < 0 ? searchHits.length : `${curHit + 1}/${searchHits.length}`) : "0"}
                  </span>
                  <Button variant="outline" size="icon-sm" disabled={!searchHits.length} onClick={() => goHit(-1)}>
                    <ChevronLeftIcon />
                  </Button>
                  <Button variant="outline" size="icon-sm" disabled={!searchHits.length} onClick={() => goHit(1)}>
                    <ChevronRightIcon />
                  </Button>
                </>
              ) : null}
              <Badge variant="outline">
                {visibleGraph.branches.length}/{rankedBranches.length} branches
              </Badge>
              <Badge variant="outline">{visibleGraph.merges.length} merges</Badge>
              <Badge variant="outline">{visibleGraph.commits.length} commits</Badge>
            </div>
          ) : null}
        </div>

        <div className="min-h-0 flex-1">
          {!graph ? (
            <Empty className="h-full border-0">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <GitMergeIcon />
                </EmptyMedia>
                <EmptyTitle>{loading ? "Loading repository…" : "No git graph"}</EmptyTitle>
                <EmptyDescription>
                  {loading
                    ? "Reading branches and commit history."
                    : "Open a git repository as this app’s project folder to plot merge flow."}
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            <TimelineGraph
              graph={visibleGraph!}
              focused={highlight}
              selectedHash={selected?.hash}
              matchHashes={matchHashes}
              selectedAuthors={selectedAuthors}
              jumpTo={msgQuery.trim() ? jumpTo : null}
              onSelect={setSelected}
              rangeStart={axisRange?.[0]}
              rangeEnd={axisRange?.[1]}
              onViewChange={(from, to) => {
                wantRef.current = { from, to }
                if (!filling.current) void ensureRange(from, to)
              }}
              fitKey={`${visibleGraph!.path}:${visibleGraph!.branches.join("\n")}:${viewGen}`}
              showTags
            />
          )}
        </div>
      </div>

      {selected && inspect ? (
        <Card className="absolute inset-y-0 right-0 z-10 w-80 overflow-y-auto rounded-none border-y-0 border-r-0 shadow-lg">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle>Inspector</CardTitle>
            <Button variant="ghost" size="icon-sm" onClick={() => setSelected(null)}>
              <XIcon />
              <span className="sr-only">Close</span>
            </Button>
          </CardHeader>
          <CardContent>
            <InspectBody inspect={inspect} commitUrl={graph?.commitUrl} />
          </CardContent>
        </Card>
      ) : null}
    </div>
  )
}

function InspectBody({
  inspect,
  commitUrl,
}: {
  inspect: GitInspect
  commitUrl?: string
}) {
  if (inspect.kind === "merge") {
    return (
      <dl className="flex flex-col gap-2 text-xs">
        <Row
          label="Merge commit"
          value={inspect.hash}
          mono
          action={<CommitLink prefix={commitUrl} hash={inspect.hash} />}
        />
        <Row label="Message" value={inspect.subject || "—"} />
        {inspect.tags?.length ? <Row label="Tags" value={inspect.tags.join(" · ")} /> : null}
        <Row label="Source branch" value={inspect.sourceBranch} />
        <Row label="Target branch" value={inspect.targetBranch} />
        <Row label="Timestamp" value={<TimeChip ts={inspect.timestamp} withDate />} />
        <Row label="Author" value={inspect.author} />
        <Row label="Commit count" value={String(inspect.commitCount)} />
      </dl>
    )
  }
  if (inspect.kind === "cluster") {
    return (
      <dl className="flex flex-col gap-2 text-xs">
        <Row label="Branch" value={inspect.branch} />
        <Row
          label="Date"
          value={new Date(inspect.timestamp).toLocaleDateString(undefined, {
            year: "numeric",
            month: "short",
            day: "numeric",
          })}
        />
        <Row label="Commits" value={String(inspect.count)} />
        <div className="flex flex-col gap-2">
          {(inspect.commits || []).map((c) => (
            <div
              key={c.hash}
              className={`flex flex-col gap-1 ${c.hash === inspect.hash ? "rounded-md bg-primary/15 p-1.5" : ""}`}
            >
              <div className="flex items-center gap-2">
                <TimeChip ts={c.timestamp} />
                <Badge variant="secondary" className="max-w-[8rem] truncate" title={authorName(c)}>
                  {authorName(c)}
                </Badge>
                <CommitLink prefix={commitUrl} hash={c.hash} />
              </div>
              <dd className="min-w-0 break-words font-medium">
                {c.subject || c.hash}
                {c.isMerge ? (
                  <Badge variant="outline" className="ml-1">
                    merge
                  </Badge>
                ) : null}
                {c.tags?.length ? (
                  <span className="ml-1 text-primary">{c.tags.join(" · ")}</span>
                ) : null}
              </dd>
            </div>
          ))}
        </div>
      </dl>
    )
  }
  return (
    <dl className="flex flex-col gap-2 text-xs">
      <Row
        label="Commit"
        value={inspect.hash}
        mono
        action={<CommitLink prefix={commitUrl} hash={inspect.hash} />}
      />
      <Row label="Message" value={inspect.subject || "—"} />
      <Row label="Branch" value={inspect.branch} />
      {inspect.tags?.length ? <Row label="Tags" value={inspect.tags.join(" · ")} /> : null}
      <Row label="Timestamp" value={<TimeChip ts={inspect.timestamp} withDate />} />
      <Row label="Author" value={inspect.author} />
    </dl>
  )
}

function Row({
  label,
  value,
  mono,
  action,
}: {
  label: string
  value: ReactNode
  mono?: boolean
  action?: ReactNode
}) {
  return (
    <div>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className={`flex items-start gap-1.5 ${mono ? "break-all font-mono text-[11px]" : "break-words font-medium"}`}>
        <span className="min-w-0">{value}</span>
        {action}
      </dd>
    </div>
  )
}

function CommitLink({ prefix, hash }: { prefix?: string; hash?: string }) {
  if (!prefix || !hash) return null
  const url = prefix + hash
  return (
    <Button
      variant="ghost"
      size="icon-xs"
      title="Open commit"
      nativeButton={false}
      render={<a href={url} />}
      onClick={(event) => handleReadyUrlClick(event, url)}
    >
      <ExternalLinkIcon />
    </Button>
  )
}

function TimeChip({ ts, withDate }: { ts: string; withDate?: boolean }) {
  const text = withDate
    ? new Date(ts).toLocaleString(undefined, {
        year: "numeric",
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      })
    : new Date(ts).toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" })
  return <Badge variant="secondary">{text}</Badge>
}
