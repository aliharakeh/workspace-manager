/**
 * Find free TCP ports by querying OS listening-port tools
 * (netstat / lsof / ss) — no temporary socket servers.
 */

import { run } from "./run"

/** Registered / user ports (excludes well-known and dynamic/ephemeral). */
export const USER_PORT_MIN = 1024
export const USER_PORT_MAX = 49151

export type ListeningProcess = {
  port: number
  pid: number
  name: string
}

/** Local port from `host:port` / `[ipv6]:port` / `*:port`. */
function parseLocalPort(address: string): number | null {
  const match = address.match(/:(\d+)$/)
  return match ? Number(match[1]) : null
}

function isUserPort(port: number): boolean {
  return port >= USER_PORT_MIN && port <= USER_PORT_MAX
}

/** Deduplicate by port+pid; sort by port then pid. */
function finalizeEntries(entries: ListeningProcess[]): ListeningProcess[] {
  const seen = new Set<string>()
  const out: ListeningProcess[] = []
  for (const entry of entries) {
    if (!isUserPort(entry.port) || entry.pid <= 0) continue
    const key = `${entry.port}:${entry.pid}`
    if (seen.has(key)) continue
    seen.add(key)
    out.push(entry)
  }
  out.sort((a, b) => a.port - b.port || a.pid - b.pid)
  return out
}

async function windowsProcessNames(
  pids: Iterable<number>
): Promise<Map<number, string>> {
  const wanted = new Set(pids)
  const names = new Map<number, string>()
  if (wanted.size === 0) return names

  const { stdout, code } = await run(["tasklist", "/FO", "CSV", "/NH"])
  if (code !== 0 && !stdout.trim()) return names

  for (const line of stdout.split(/\r?\n/)) {
    // "name.exe","1234","Session","1","12,345 K"
    const match = line.match(/^"([^"]+)","(\d+)"/)
    if (!match) continue
    const pid = Number(match[2])
    if (!wanted.has(pid)) continue
    names.set(pid, match[1] ?? "Unknown")
  }
  return names
}

async function listListeningProcessesWin(): Promise<ListeningProcess[]> {
  // `netstat -ano -p TCP` — local address col 2, state col 4, PID last.
  const { stdout } = await run(["netstat", "-ano", "-p", "TCP"])
  const raw: Array<{ port: number; pid: number }> = []

  for (const line of stdout.split(/\r?\n/)) {
    const parts = line.trim().split(/\s+/)
    if (parts.length < 5 || parts[0] !== "TCP") continue
    if (parts[3] !== "LISTENING") continue
    const port = parseLocalPort(parts[1] ?? "")
    const pid = Number(parts[4])
    if (port == null || !Number.isInteger(pid)) continue
    raw.push({ port, pid })
  }

  const names = await windowsProcessNames(raw.map((r) => r.pid))
  return finalizeEntries(
    raw.map(({ port, pid }) => ({
      port,
      pid,
      name: names.get(pid) ?? "Unknown",
    }))
  )
}

async function listListeningProcessesUnix(): Promise<ListeningProcess[]> {
  // Prefer lsof — includes COMMAND and PID.
  const lsof = await run(["lsof", "-nP", "-iTCP", "-sTCP:LISTEN"])
  if (lsof.code === 0 || lsof.stdout.trim()) {
    const entries: ListeningProcess[] = []
    for (const line of lsof.stdout.split(/\r?\n/).slice(1)) {
      const parts = line.trim().split(/\s+/)
      if (parts.length < 9) continue
      const name = parts[0] ?? "Unknown"
      const pid = Number(parts[1])
      const addr = parts.at(-1)?.replace(/\s*\(.*\)$/, "") ?? ""
      const port = parseLocalPort(addr)
      if (port == null || !Number.isInteger(pid)) continue
      entries.push({ port, pid, name })
    }
    if (entries.length > 0 || lsof.code === 0) {
      return finalizeEntries(entries)
    }
  }

  // ss -lntp: users:(("node",pid=123,fd=23))
  const ss = await run(["ss", "-lntp"])
  if (ss.code === 0 || ss.stdout.trim()) {
    const entries: ListeningProcess[] = []
    for (const line of ss.stdout.split(/\r?\n/)) {
      const parts = line.trim().split(/\s+/)
      if (!parts[0]?.startsWith("LISTEN") && parts[0] !== "tcp") continue
      const local =
        parts.find((p) => /:\d+$/.test(p)) ??
        parts[3] ??
        parts[4]
      const port = local ? parseLocalPort(local) : null
      const users = line.match(/users:\(\("([^"]*)",pid=(\d+)/)
      const pid = users ? Number(users[2]) : NaN
      const name = users?.[1] || "Unknown"
      if (port == null || !Number.isInteger(pid)) continue
      entries.push({ port, pid, name })
    }
    if (entries.length > 0 || ss.code === 0) {
      return finalizeEntries(entries)
    }
  }

  throw new Error(
    "Could not list listening processes (need netstat on Windows, or lsof/ss on Unix)"
  )
}

/**
 * Listening TCP sockets on user ports (1024–49151), with owning process name.
 */
export async function listListeningProcesses(): Promise<ListeningProcess[]> {
  if (process.platform === "win32") {
    return listListeningProcessesWin()
  }
  return listListeningProcessesUnix()
}

/** Listening TCP ports currently held on this machine. */
async function listListeningPorts(): Promise<Set<number>> {
  const ports = new Set<number>()

  if (process.platform === "win32") {
    // `netstat -ano -p TCP` — local address is column 2, state column 4.
    const { stdout } = await run(["netstat", "-ano", "-p", "TCP"])
    for (const line of stdout.split(/\r?\n/)) {
      const parts = line.trim().split(/\s+/)
      if (parts.length < 4 || parts[0] !== "TCP") continue
      if (parts[3] !== "LISTENING") continue
      const port = parseLocalPort(parts[1] ?? "")
      if (port != null) ports.add(port)
    }
    return ports
  }

  // macOS / Linux: prefer lsof, then ss, then netstat.
  const lsof = await run(["lsof", "-nP", "-iTCP", "-sTCP:LISTEN"])
  if (lsof.code === 0 || lsof.stdout.trim()) {
    for (const line of lsof.stdout.split(/\r?\n/).slice(1)) {
      const name = line.trim().split(/\s+/).at(-1)
      if (!name) continue
      const port = parseLocalPort(name.replace(/\s*\(.*\)$/, ""))
      if (port != null) ports.add(port)
    }
    if (ports.size > 0 || lsof.code === 0) return ports
  }

  const ss = await run(["ss", "-lntH"])
  if (ss.code === 0 || ss.stdout.trim()) {
    for (const line of ss.stdout.split(/\r?\n/)) {
      const parts = line.trim().split(/\s+/)
      // Local address is usually the 4th field: `LISTEN 0 511 0.0.0.0:3000 …`
      const local = parts[3] ?? parts.find((p) => /:\d+$/.test(p))
      const port = local ? parseLocalPort(local) : null
      if (port != null) ports.add(port)
    }
    if (ports.size > 0 || ss.code === 0) return ports
  }

  const netstat = await run(["netstat", "-lnt"])
  if (netstat.code === 0 || netstat.stdout.trim()) {
    for (const line of netstat.stdout.split(/\r?\n/)) {
      const parts = line.trim().split(/\s+/)
      if (!parts[0]?.startsWith("tcp")) continue
      const local = parts[3]
      const port = local ? parseLocalPort(local) : null
      if (port != null) ports.add(port)
    }
    return ports
  }

  throw new Error(
    "Could not list listening ports (need netstat on Windows, or lsof/ss/netstat on Unix)"
  )
}

/** Returns the first free TCP port at or after `preferred`. */
export async function findAvailablePort(
  preferred: number,
  maxAttempts = 40
): Promise<number> {
  const used = await listListeningPorts()

  for (let i = 0; i < maxAttempts; i++) {
    const port = preferred + i
    if (!used.has(port)) return port
  }

  throw new Error(
    `No free TCP port found between ${preferred} and ${preferred + maxAttempts - 1}`
  )
}
