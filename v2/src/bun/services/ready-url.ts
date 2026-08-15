/**
 * Ready-URL detection from process log lines.
 *
 * Patterns are stored in SQLite and managed via Settings → Log URL patterns.
 * Each pattern runs against a single ANSI-stripped log line (no trailing newline).
 *
 * Capture groups (use named groups):
 * - `url`  — full URL (preferred), e.g. http://localhost:5173/
 * - `port` — port only; becomes http://localhost:{port}
 *
 * If both are present, `url` wins.
 */

import { readyUrlPatternsRepo } from "../db/ready-url-patterns"

export type ReadyUrlPattern = {
  id: string
  label: string
  pattern: RegExp
}

export type ReadyUrlMatch = {
  url: string
  patternId: string
  label: string
}

let cached: ReadyUrlPattern[] | null = null

export function invalidateReadyUrlPatternsCache() {
  cached = null
}

export function validateReadyUrlPattern(
  source: string,
  flags = "i"
): { ok: true; pattern: RegExp } | { ok: false; error: string } {
  let re: RegExp
  try {
    re = new RegExp(source, flags)
  } catch (err) {
    return {
      ok: false,
      error: err instanceof Error ? err.message : "Invalid regular expression",
    }
  }

  if (!/\(\?<url>/.test(source) && !/\(\?<port>/.test(source)) {
    return {
      ok: false,
      error: "Pattern must include a named group `url` and/or `port`",
    }
  }

  return { ok: true, pattern: re }
}

function loadPatterns(): ReadyUrlPattern[] {
  readyUrlPatternsRepo.ensureSeeded()
  const rows = readyUrlPatternsRepo.list()
  const compiled: ReadyUrlPattern[] = []
  for (const row of rows) {
    try {
      compiled.push({
        id: String(row.id),
        label: row.label,
        pattern: new RegExp(row.pattern, row.flags),
      })
    } catch {
      // Skip broken rows so one bad pattern doesn't break matching.
    }
  }
  return compiled
}

function getPatterns(): ReadyUrlPattern[] {
  if (!cached) cached = loadPatterns()
  return cached
}

/** Trim trailing junk that often sticks to URLs in logs. */
function cleanUrl(raw: string): string {
  return raw.replace(/[)\].,;:'">]+$/g, "")
}

function fromPort(port: string): string | null {
  const n = Number(port)
  if (!Number.isInteger(n) || n < 1 || n > 65535) return null
  return `http://localhost:${n}`
}

/**
 * Try every configured pattern against one log line.
 * Returns the first match, or null.
 */
export function matchReadyUrl(line: string): ReadyUrlMatch | null {
  const text = line.trim()
  if (!text) return null

  for (const { id, label, pattern } of getPatterns()) {
    pattern.lastIndex = 0
    const m = pattern.exec(text)
    if (!m?.groups) continue

    const urlGroup = m.groups.url
    if (urlGroup) {
      const url = cleanUrl(urlGroup)
      if (/^https?:\/\//i.test(url)) {
        return { url, patternId: id, label }
      }
      continue
    }

    const portGroup = m.groups.port
    if (portGroup) {
      const url = fromPort(portGroup)
      if (url) return { url, patternId: id, label }
    }
  }

  return null
}
