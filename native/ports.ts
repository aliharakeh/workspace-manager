/**
 * Find free TCP ports by querying OS listening-port tools
 * (netstat / lsof / ss) — no temporary socket servers.
 */

import { run } from "./run"

/** Local port from `host:port` / `[ipv6]:port` / `*:port`. */
function parseLocalPort(address: string): number | null {
  const match = address.match(/:(\d+)$/)
  return match ? Number(match[1]) : null
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
