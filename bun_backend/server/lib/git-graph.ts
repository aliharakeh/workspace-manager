import { existsSync, statSync } from "node:fs"
import { resolve } from "node:path"
import { spawnSync } from "node:child_process"

export type GitCommitNode = {
  hash: string
  branch: string
  on?: string[]
  parents?: string[]
  timestamp: string
  author: string
  subject: string
  isMerge: boolean
  tags?: string[]
  lanes?: string[]
}

export type GitMergeEvent = {
  hash: string
  kind?: string
  sourceBranch: string
  targetBranch: string
  sourceHash: string
  timestamp: string
  author: string
  subject: string
  commitCount: number
}

export type GitRepoGraph = {
  path: string
  commitUrl?: string
  branches: string[]
  commits: GitCommitNode[]
  merges: GitMergeEvent[]
}

export type GitBranchInfo = {
  name: string
  updated?: string
}

export type GitRemoteInfo = {
  name: string
  url: string
  web?: string
  host?: string
  ssh: boolean
}

type BranchMeta = { name: string; hash: string; at: number }

type RawCommit = {
  hash: string
  parents: string[]
  author: string
  at: number
  subject: string
  branch: string
  assigned: boolean
  on: string[]
  fp: string[]
}

const mergeBranch =
  /^(?:Merge(?: remote-tracking)? branch '([^']+)'(?: of \S+)?(?: into '?([^'\s]+)'?)?)$/i
const mergeTag = /^(?:Merge tag '([^']+)'(?: into '?([^'\s]+)'?)?)$/i
const mergePR = /^(?:Merge pull request #\d+ from [^/\s]+\/(\S+?)(?: into \S+)?)$/i
const mergeBB = /^(?:Merged in (\S+) \(pull request #\d+\))/i
const nameRevJunk = /([~^][\d]+)+$/

function gitOutput(dir: string, args: string[], stdin = "", timeoutMs = 0): string {
  const r = spawnSync("git", ["-C", dir, ...args], {
    encoding: "utf8",
    windowsHide: true,
    maxBuffer: 50 * 1024 * 1024,
    input: stdin || undefined,
    timeout: timeoutMs || undefined,
  })
  if (r.error) {
    if ((r.error as NodeJS.ErrnoException).code === "ETIMEDOUT") {
      throw new Error("fetch timed out")
    }
    throw r.error
  }
  if (r.status !== 0) {
    const msg = (r.stderr || r.stdout || r.error?.message || "").trim()
    throw new Error(msg || "git failed")
  }
  return (r.stdout || "").trim()
}

export function gitRoot(path: string): string {
  const abs = resolve(path)
  if (!existsSync(abs) || !statSync(abs).isDirectory()) {
    throw new Error(existsSync(abs) ? `not a directory: ${abs}` : `path not found: ${abs}`)
  }
  try {
    return resolve(gitOutput(abs, ["rev-parse", "--show-toplevel"]))
  } catch {
    throw new Error(`not a git repository: ${abs}`)
  }
}

export async function listBranches(path: string): Promise<GitBranchInfo[]> {
  const root = gitRoot(path)
  const metas = listBranchMeta(root)
  metas.sort((a, b) => (a.at !== b.at ? b.at - a.at : a.name.localeCompare(b.name)))
  return metas.map((m) => {
    const info: GitBranchInfo = { name: m.name }
    if (m.at) info.updated = new Date(m.at).toISOString()
    return info
  })
}

export async function listRemote(path: string): Promise<GitRemoteInfo> {
  const root = gitRoot(path)
  let name = "origin"
  let u = ""
  try {
    u = gitOutput(root, ["remote", "get-url", "origin"])
  } catch {
    u = ""
  }
  if (!u) {
    const names = gitOutput(root, ["remote"])
    if (!names) throw new Error("no git remotes")
    name = names.split(/\s+/)[0]!
    u = gitOutput(root, ["remote", "get-url", name])
    if (!u) throw new Error("no git remotes")
  }
  const web = toWebBase(u)
  return { name, url: u, web: web || undefined, host: remoteHost(web) || undefined, ssh: isSSHURL(u) }
}

export async function fetchRemote(path: string): Promise<void> {
  const root = gitRoot(path)
  const info = await listRemote(root)
  gitOutput(root, ["fetch", "--prune", info.name], "", 60_000)
}

export async function loadGraphAt(
  path: string,
  only: string[] | null,
  since: Date | null,
  until: Date | null
): Promise<GitRepoGraph> {
  const root = gitRoot(path)
  const allTips = listBranchTips(root)
  let branchTips = allTips
  if (only != null) {
    branchTips = filterTips(allTips, only)
    if (Object.keys(branchTips).length === 0) {
      return {
        path: root,
        commitUrl: commitURLPrefix(toWebBase(remoteURL(root))) || undefined,
        branches: [],
        commits: [],
        merges: [],
      }
    }
  }
  const windowed = !!(since || until)
  const commits = windowed ? listCommitsRange(root, since, until) : listCommits(root)
  const tagByHash = listTags(root)
  const order = sortBranchNames(Object.keys(branchTips))
  const ranked = claimOrder(order)
  assignLanes(root, ranked, branchTips, commits)
  assignReachable(root, ranked, branchTips, commits, since, until)
  assignOffSpineMerges(commits, order)
  absorbDeletedMergeSources(root, branchTips, commits, since, until)

  const nodes: GitCommitNode[] = []
  const merges: GitMergeEvent[] = []
  const used: Record<string, boolean> = {}
  const known: Record<string, boolean> = {}
  for (const name of order) known[laneName(name)] = true
  preferParentLanes(commits, known)

  for (const c of Object.values(commits)) {
    if (!c.assigned || !c.branch) continue
    const iso = new Date(c.at).toISOString()
    const [msgSrc, msgDst] = parseMergeSubject(c.subject)
    let target = c.branch
    const d = knownLane(msgDst, known)
    if (d) target = d
    else if (incomingMerge(c.subject)) target = incomingDest(c, msgSrc, known)
    else if (!laneName(msgDst)) {
      const s = knownLane(msgSrc, known)
      if (s && s === laneName(c.branch)) target = s
    }
    let branch = c.branch
    if (c.parents.length > 1) branch = target
    used[branch] = true
    const node: GitCommitNode = {
      hash: c.hash,
      branch,
      timestamp: iso,
      author: c.author,
      subject: c.subject,
      isMerge: c.parents.length > 1,
    }
    if (c.on.length) node.on = c.on
    if (c.parents.length) node.parents = c.parents
    if (tagByHash[c.hash]?.length) node.tags = tagByHash[c.hash]
    const lanes = laneList(c)
    if (lanes.length) node.lanes = lanes
    nodes.push(node)
    if (c.parents.length < 2) continue
    for (const srcHash of c.parents.slice(1)) {
      const src = commits[srcHash]
      let srcBranch = laneName(msgSrc)
      if (!srcBranch && src) srcBranch = src.branch
      if (!srcBranch) srcBranch = shortHash(srcHash)
      srcBranch = laneName(srcBranch)
      if (srcBranch === laneName(target)) srcBranch = target
      else if (srcBranch && known[srcBranch]) used[srcBranch] = true
      merges.push({
        hash: c.hash,
        sourceBranch: srcBranch,
        targetBranch: target,
        sourceHash: srcHash,
        timestamp: iso,
        author: c.author,
        subject: c.subject,
        commitCount: countExclusive(srcHash, c.parents[0]!, commits),
      })
    }
  }

  merges.push(...branchStarts(commits, known))

  const branches: string[] = []
  const seenBr: Record<string, boolean> = {}
  for (const name of order) {
    const lane = laneName(name)
    if (used[lane] && !seenBr[lane]) {
      seenBr[lane] = true
      branches.push(lane)
    }
  }
  const rest = Object.keys(used).filter((lane) => !seenBr[lane]).sort()
  branches.push(...rest)
  nodes.sort((a, b) =>
    a.timestamp === b.timestamp ? a.hash.localeCompare(b.hash) : a.timestamp.localeCompare(b.timestamp)
  )
  merges.sort((a, b) => a.timestamp.localeCompare(b.timestamp))
  return {
    path: root,
    commitUrl: commitURLPrefix(toWebBase(remoteURL(root))) || undefined,
    branches,
    commits: nodes,
    merges,
  }
}

function listBranchTips(root: string): Record<string, string> {
  const tips: Record<string, string> = {}
  for (const m of listBranchMeta(root)) tips[m.name] = m.hash
  return tips
}

function listBranchMeta(root: string): BranchMeta[] {
  const format = "--format=%(refname:short)%00%(objectname)%00%(authordate:unix)%00%(committerdate:unix)"
  const tips = parseRefMeta(gitOutput(root, ["for-each-ref", format, "refs/heads"]))
  for (const [name, meta] of Object.entries(parseRefMeta(gitOutput(root, ["for-each-ref", format, "refs/remotes"])))) {
    if (name.endsWith("/HEAD")) continue
    const short = stripRemotePrefix(name)
    const existing = tips[short]
    if (!existing) {
      meta.name = short
      tips[short] = meta
    } else {
      if (meta.at > existing.at) existing.at = meta.at
      if (existing.hash !== meta.hash) tips[name] = meta
    }
  }
  return Object.values(tips)
}

function listTags(root: string): Record<string, string[]> {
  const out = gitOutput(root, ["for-each-ref", "--format=%(refname:short)%00%(*objectname)%00%(objectname)", "refs/tags"])
  const byHash: Record<string, string[]> = {}
  if (!out) return byHash
  for (const line of out.split("\n")) {
    const parts = line.split("\0")
    if (parts.length < 3 || !parts[0]) continue
    const hash = parts[1] || parts[2]
    if (!hash) continue
    ;(byHash[hash] ??= []).push(parts[0])
  }
  for (const tags of Object.values(byHash)) tags.sort()
  return byHash
}

function parseRefMeta(out: string): Record<string, BranchMeta> {
  const tips: Record<string, BranchMeta> = {}
  if (!out) return tips
  for (const line of out.split("\n")) {
    const parts = line.split("\0")
    if (parts.length < 2 || !parts[0] || !parts[1]) continue
    const meta: BranchMeta = { name: parts[0], hash: parts[1], at: 0 }
    let unix = 0
    for (const p of parts.slice(2)) {
      const n = Number.parseInt(p, 10)
      if (Number.isFinite(n) && n > unix) unix = n
    }
    if (unix > 0) meta.at = unix * 1000
    tips[parts[0]] = meta
  }
  return tips
}

function filterTips(tips: Record<string, string>, want: string[]): Record<string, string> {
  const allow = new Set<string>()
  for (const w of want) {
    if (!w) continue
    allow.add(w)
    allow.add(laneName(w))
  }
  const out: Record<string, string> = {}
  for (const [name, hash] of Object.entries(tips)) {
    if (allow.has(name) || allow.has(laneName(name))) out[name] = hash
  }
  return out
}

function listCommits(root: string): Record<string, RawCommit> {
  return parseLog(root, ["log", "--pretty=format:%H%x1f%P%x1f%an%x1f%aI%x1f%s", "--all"])
}

function listCommitsRange(root: string, since: Date | null, until: Date | null): Record<string, RawCommit> {
  return parseLog(root, rangeLogArgs("--all", since, until))
}

function rangeLogArgs(tip: string, since: Date | null, until: Date | null, ...extra: string[]): string[] {
  const args = ["log", "--pretty=format:%H%x1f%P%x1f%an%x1f%aI%x1f%s", ...extra, tip]
  if (since) args.push(`--since=${since.toISOString()}`)
  if (until) args.push(`--until=${until.toISOString()}`)
  return args
}

function parseLog(root: string, args: string[]): Record<string, RawCommit> {
  let out = ""
  try {
    out = gitOutput(root, args)
  } catch (err) {
    if (String(err).toLowerCase().includes("does not have any commits")) return {}
    throw err
  }
  const commits: Record<string, RawCommit> = {}
  if (!out) return commits
  for (const line of out.split("\n")) {
    const parts = line.split("\x1f")
    if (parts.length < 4) continue
    const subject = parts[4] ?? ""
    const at = Date.parse(parts[3]!)
    if (!Number.isFinite(at)) continue
    commits[parts[0]!] = {
      hash: parts[0]!,
      parents: parts[1] ? parts[1].split(/\s+/) : [],
      author: parts[2]!,
      at,
      subject,
      branch: "",
      assigned: false,
      on: [],
      fp: [],
    }
  }
  return commits
}

function assignLanes(
  root: string,
  order: string[],
  tips: Record<string, string>,
  commits: Record<string, RawCommit>
) {
  for (const name of order) {
    const lane = laneName(name)
    const tip = tips[name]
    if (!tip) continue
    let out = ""
    try {
      out = gitOutput(root, ["rev-list", "--first-parent", tip])
    } catch {
      continue
    }
    for (const line of out.split("\n")) {
      const c = commits[line.trim()]
      if (!c) continue
      markFP(c, lane)
      markOn(c, lane)
      if (!c.assigned) {
        c.assigned = true
        c.branch = lane
      }
    }
  }
}

function assignReachable(
  root: string,
  order: string[],
  tips: Record<string, string>,
  commits: Record<string, RawCommit>,
  since: Date | null,
  until: Date | null
) {
  for (const name of order) {
    const lane = laneName(name)
    const tip = tips[name]
    if (!tip) continue
    let chunk: Record<string, RawCommit>
    try {
      chunk = parseLog(root, rangeLogArgs(tip, since, until))
    } catch {
      continue
    }
    for (const hash of Object.keys(chunk)) {
      const c = commits[hash]
      if (!c) continue
      if (!c.assigned) {
        c.assigned = true
        c.branch = lane
      }
      markOn(c, lane)
    }
  }
}

function assignOffSpineMerges(commits: Record<string, RawCommit>, lanes: string[]) {
  const known: Record<string, boolean> = {}
  for (const name of lanes) known[laneName(name)] = true
  for (const c of Object.values(commits)) {
    if (c.assigned || c.parents.length < 2) continue
    const d = destLane(c.subject)
    if (known[d]) {
      c.assigned = true
      c.branch = d
      markOn(c, d)
      continue
    }
    if (d) continue
    const p = c.parents[0] ? commits[c.parents[0]] : undefined
    if (p?.assigned && known[p.branch]) {
      c.assigned = true
      c.branch = p.branch
      markOn(c, p.branch)
      continue
    }
    const knownKeys = Object.keys(known)
    if (knownKeys.length !== 1 || !(mergePR.test(c.subject) || mergeBB.test(c.subject))) continue
    const lane = knownKeys[0]!
    c.assigned = true
    c.branch = lane
    markOn(c, lane)
  }
}

function destLane(subject: string): string {
  return parseMergeSubject(subject)[1]
}

function branchStarts(commits: Record<string, RawCommit>, known: Record<string, boolean>): GitMergeEvent[] {
  const out: GitMergeEvent[] = []
  for (const c of Object.values(commits)) {
    if (!c.assigned || !c.branch || c.parents.length !== 1) continue
    const dst = laneName(c.branch)
    if (!known[dst]) continue
    const p = commits[c.parents[0]!]
    if (!p?.assigned || !p.branch) continue
    const src = laneName(p.branch)
    if (src === dst || !known[src]) continue
    out.push({
      hash: c.hash,
      kind: "branch",
      sourceBranch: src,
      targetBranch: dst,
      sourceHash: p.hash,
      timestamp: new Date(c.at).toISOString(),
      author: c.author,
      subject: "Branch from " + src,
      commitCount: 1,
    })
  }
  return out
}

function firstParentSet(root: string, tips: Record<string, string>): Record<string, boolean> {
  const hashes = Object.values(tips)
  if (!hashes.length) return {}
  let out = ""
  try {
    out = gitOutput(root, ["rev-list", "--first-parent", ...hashes])
  } catch {
    return {}
  }
  if (!out) return {}
  const live: Record<string, boolean> = {}
  for (const line of out.split("\n")) {
    const h = line.trim()
    if (h) live[h] = true
  }
  return live
}

function absorbDeletedMergeSources(
  root: string,
  tips: Record<string, string>,
  commits: Record<string, RawCommit>,
  since: Date | null,
  until: Date | null
) {
  const live = firstParentSet(root, tips)
  for (const c of Object.values(commits)) {
    if (c.parents.length < 2 || !c.assigned || !c.branch) continue
    let lane = c.branch
    const d = destLane(c.subject)
    if (d) {
      for (const name of Object.keys(tips)) {
        if (laneName(name) === d) {
          lane = d
          break
        }
      }
    }
    for (const srcHash of c.parents.slice(1)) {
      if (live[srcHash]) continue
      if (!commits[srcHash]) {
        try {
          const chunk = parseLog(root, rangeLogArgs(srcHash, since, until, "--first-parent"))
          for (const [h, nc] of Object.entries(chunk)) {
            if (!commits[h]) commits[h] = nc
          }
        } catch {
          continue
        }
      }
      if (!commits[srcHash]) continue
      assignLanes(root, [lane], { [lane]: srcHash }, commits)
    }
  }
}

function countExclusive(from: string, exclude: string, commits: Record<string, RawCommit>): number {
  const blocked: Record<string, boolean> = {}
  const stack = [exclude]
  while (stack.length) {
    const h = stack.pop()!
    if (!h || blocked[h]) continue
    blocked[h] = true
    const c = commits[h]
    if (c) stack.push(...c.parents)
  }
  let n = 0
  const seen: Record<string, boolean> = {}
  stack.push(from)
  while (stack.length) {
    const h = stack.pop()!
    if (!h || blocked[h] || seen[h]) continue
    seen[h] = true
    n++
    const c = commits[h]
    if (c) stack.push(...c.parents)
  }
  return n
}

export function parseMergeSubject(subject: string): [string, string] {
  const s = subject.trim()
  let m = mergeBranch.exec(s)
  if (m) return [cleanLaneName(m[1]!), cleanLaneName((m[2] ?? "").replaceAll("'", ""))]
  m = mergeTag.exec(s)
  if (m) return [cleanLaneName(m[1]!), cleanLaneName((m[2] ?? "").replaceAll("'", ""))]
  m = mergePR.exec(s)
  if (m) return [cleanLaneName(m[1]!), ""]
  m = mergeBB.exec(s)
  if (m) return [cleanLaneName(m[1]!), ""]
  return ["", ""]
}

function cleanLaneName(name: string): string {
  name = stripRemotePrefix(stripNameRev(name))
  if (!name || name === "undefined") return ""
  return name
}

function stripNameRev(name: string): string {
  name = name.replace(/^refs\/heads\//, "").replace(/^refs\/remotes\//, "").replace(/^refs\/tags\//, "")
  name = name.replace(/^remotes\//, "").replace(/^tags\//, "")
  return name.replace(nameRevJunk, "")
}

function stripRemotePrefix(name: string): string {
  name = name.replace(/^remotes\//, "")
  const i = name.indexOf("/")
  if (i > 0) {
    const origin = name.slice(0, i)
    if (origin === "origin" || origin === "upstream") return name.slice(i + 1)
  }
  return name
}

function laneName(name: string): string {
  return cleanLaneName(name) || name
}

function sortBranchNames(names: string[]): string[] {
  const rank: Record<string, number> = { main: 0, master: 1, trunk: 2, develop: 3, dev: 4 }
  return [...names].sort((a, b) => {
    const okI = a in rank
    const okJ = b in rank
    if (okI && okJ) return rank[a]! - rank[b]!
    if (okI !== okJ) return okI ? -1 : 1
    return a < b ? -1 : 1
  })
}

function claimOrder(names: string[]): string[] {
  const rank: Record<string, number> = { main: 0, master: 1, trunk: 2, develop: 3, dev: 4 }
  return [...names].sort((a, b) => {
    const ri = rank[laneName(a)] ?? 100
    const rj = rank[laneName(b)] ?? 100
    if (ri !== rj) return ri - rj
    return a < b ? -1 : 1
  })
}

function knownLane(name: string, known: Record<string, boolean>): string {
  name = laneName(name)
  return name && known[name] ? name : ""
}

function incomingMerge(subject: string): boolean {
  return mergeBB.test(subject) || mergePR.test(subject)
}

function incomingDest(c: RawCommit, msgSrc: string, known: Record<string, boolean>): string {
  const src = laneName(msgSrc)
  const other = c.on.filter((lane) => known[lane] && lane !== src)
  const t = pickTrunk(other)
  if (t) return t
  if (other.length) {
    other.sort()
    return other[0]!
  }
  return c.branch
}

function pickTrunk(lanes: string[]): string {
  for (const t of ["dev", "develop", "main", "master", "trunk"]) {
    if (lanes.includes(t)) return t
  }
  return ""
}

function preferParentLanes(commits: Record<string, RawCommit>, known: Record<string, boolean>) {
  const list = Object.values(commits).filter((c) => c.assigned)
  list.sort((a, b) => (a.at !== b.at ? a.at - b.at : a.hash.localeCompare(b.hash)))
  for (const c of list) {
    const b = pickShownLane(c, known, commits)
    if (b) c.branch = b
  }
}

function pickShownLane(
  c: RawCommit,
  known: Record<string, boolean>,
  commits: Record<string, RawCommit>
): string {
  return pickShownFrom(c, known, commits, c.fp) || pickShownFrom(c, known, commits, c.on) || (known[laneName(c.branch)] ? c.branch : "")
}

function pickShownFrom(
  c: RawCommit,
  known: Record<string, boolean>,
  commits: Record<string, RawCommit>,
  lanes: string[]
): string {
  const allow: Record<string, boolean> = {}
  for (const l of lanes) {
    if (known[l]) allow[l] = true
  }
  if (!Object.keys(allow).length) return ""
  for (const h of c.parents) {
    const p = commits[h]
    if (p && allow[laneName(p.branch)]) return p.branch
  }
  for (const l of lanes) {
    if (allow[l]) return l
  }
  return ""
}

function laneList(c: RawCommit): string[] {
  const seen: Record<string, boolean> = {}
  const out: string[] = []
  for (const l of [...c.fp, ...c.on]) {
    if (!l || seen[l]) continue
    seen[l] = true
    out.push(l)
  }
  return out
}

function markFP(c: RawCommit, lane: string) {
  if (!c.fp.includes(lane)) c.fp.push(lane)
}

function markOn(c: RawCommit, lane: string) {
  if (!c.on.includes(lane)) c.on.push(lane)
}

function remoteURL(root: string): string {
  try {
    const u = gitOutput(root, ["remote", "get-url", "origin"])
    if (u) return u
  } catch {
    /* fall through */
  }
  try {
    const names = gitOutput(root, ["remote"])
    if (!names) return ""
    return gitOutput(root, ["remote", "get-url", names.split(/\s+/)[0]!])
  } catch {
    return ""
  }
}

function toWebBase(remote: string): string {
  let u = remote.trim().replace(/\/+$/, "").replace(/\.git$/, "")
  if (!u) return ""
  if (u.startsWith("git@")) {
    const rest = u.slice(4)
    const i = rest.indexOf(":")
    if (i < 0) return ""
    const host = rest.slice(0, i)
    const path = rest.slice(i + 1)
    if (!host || !path) return ""
    return "https://" + host + "/" + path.replace(/^\//, "")
  }
  if (u.startsWith("ssh://")) {
    return "https://" + u.slice(6).replace(/^git@/, "")
  }
  const i = u.indexOf("://")
  if (i < 0) return ""
  const scheme = u.slice(0, i)
  let rest = u.slice(i + 3)
  if (scheme !== "http" && scheme !== "https") return ""
  const at = rest.lastIndexOf("@")
  if (at >= 0) rest = rest.slice(at + 1)
  return scheme + "://" + rest
}

function commitURLPrefix(base: string): string {
  if (!base) return ""
  let host = base
  const i = base.indexOf("://")
  if (i >= 0) host = base.slice(i + 3).split("/")[0]!
  if (host === "bitbucket.org" || host.endsWith(".bitbucket.org")) return base + "/commits/"
  if (host === "gitlab.com" || host.includes("gitlab")) return base + "/-/commit/"
  return base + "/commit/"
}

function shortHash(h: string): string {
  return h.length > 7 ? h.slice(0, 7) : h
}

function remoteHost(web: string): string {
  const i = web.indexOf("://")
  if (i < 0) return ""
  return web.slice(i + 3).split("/")[0] ?? ""
}

function isSSHURL(u: string): boolean {
  u = u.trim()
  return u.startsWith("git@") || u.startsWith("ssh://")
}

export function parseISO(s: string): Date | null {
  const t = s.trim()
  if (!t) return null
  const d = new Date(t)
  if (Number.isNaN(+d)) throw new Error("invalid date")
  return d
}
