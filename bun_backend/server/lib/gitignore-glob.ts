import { readdirSync, statSync } from "node:fs"
import { join, relative } from "node:path"
import { spawnSync } from "node:child_process"

const MAX_WALK = 10_000
const MAX_RESULTS = 200

export function globToRegExp(pattern: string): RegExp {
  let i = 0
  let out = "^"
  while (i < pattern.length) {
    if (pattern.startsWith("**/", i)) {
      out += "(?:.*/)?"
      i += 3
      continue
    }
    if (pattern.startsWith("**", i)) {
      out += ".*"
      i += 2
      continue
    }
    const ch = pattern[i]!
    if (ch === "*") out += "[^/]*"
    else if (ch === "?") out += "[^/]"
    else out += ch.replace(/[|\\{}()[\]^$+.]/g, "\\$&")
    i++
  }
  return new RegExp(out + "$")
}

export function sanitizeGlobPattern(pattern: string): string | { error: string } {
  const cleaned = pattern.trim().replace(/\\/g, "/")
  if (!cleaned) return { error: "pattern is required" }
  if (cleaned.startsWith("/") || /^[a-zA-Z]:/.test(cleaned)) {
    return { error: "pattern must be relative to the app directory" }
  }
  if (cleaned.split("/").includes("..")) {
    return { error: "pattern must not contain .." }
  }
  return cleaned
}

function gitVisibleFiles(root: string): string[] | null {
  const r = spawnSync(
    "git",
    ["-C", root, "ls-files", "-co", "--exclude-standard"],
    { encoding: "utf8", windowsHide: true }
  )
  if (r.status !== 0) return null
  return r.stdout
    .split(/\r?\n/)
    .map((s) => s.replace(/\\/g, "/"))
    .filter(Boolean)
}

function walkFiles(root: string): string[] {
  const out: string[] = []
  const stack = [root]
  while (stack.length && out.length < MAX_WALK) {
    const dir = stack.pop()!
    let entries
    try {
      entries = readdirSync(dir, { withFileTypes: true })
    } catch {
      continue
    }
    for (const ent of entries) {
      if (ent.name === ".git" || ent.name === "node_modules") continue
      const full = join(dir, ent.name)
      if (ent.isDirectory()) {
        stack.push(full)
        continue
      }
      if (!ent.isFile() && !ent.isSymbolicLink()) continue
      try {
        if (!statSync(full).isFile()) continue
      } catch {
        continue
      }
      const rel = relative(root, full).replace(/\\/g, "/")
      if (rel && !rel.startsWith("..")) out.push(rel)
    }
  }
  return out
}

export function searchProjectFiles(
  root: string,
  pattern: string
): { files: string[]; truncated: boolean } | { error: string } {
  const pat = sanitizeGlobPattern(pattern)
  if (typeof pat !== "string") return pat
  const re = globToRegExp(pat)
  const candidates = gitVisibleFiles(root) ?? walkFiles(root)
  const files: string[] = []
  for (const f of candidates) {
    if (!re.test(f)) continue
    files.push(f)
    if (files.length > MAX_RESULTS) {
      files.pop()
      return { files, truncated: true }
    }
  }
  return { files, truncated: false }
}
