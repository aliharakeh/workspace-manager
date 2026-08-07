import { existsSync, readFileSync, statSync } from "node:fs"
import { isAbsolute, normalize, relative, resolve, sep } from "node:path"

export function validateProjectPath(projectPath: string): {
  ok: boolean
  error?: string
  resolved?: string
} {
  const trimmed = projectPath.trim()
  if (!trimmed) {
    return { ok: false, error: "project_path is required" }
  }

  const resolvedPath = resolve(trimmed)
  if (!existsSync(resolvedPath)) {
    return { ok: false, error: `Path does not exist: ${resolvedPath}` }
  }

  try {
    if (!statSync(resolvedPath).isDirectory()) {
      return { ok: false, error: `Path is not a directory: ${resolvedPath}` }
    }
  } catch {
    return { ok: false, error: `Cannot access path: ${resolvedPath}` }
  }

  return { ok: true, resolved: resolvedPath }
}

/** Resolve a project-relative path safely (no escape). */
export function resolveProjectFile(
  projectPath: string,
  relativePath: string
): { ok: true; absolute: string } | { ok: false; error: string } {
  const cleaned = relativePath.trim().replace(/\\/g, "/").replace(/^\/+/, "")
  if (!cleaned) return { ok: false, error: "file path is required" }

  const root = normalize(resolve(projectPath))
  const full = normalize(resolve(root, cleaned))
  if (full !== root && !full.startsWith(root + sep)) {
    return { ok: false, error: "Path escapes project directory" }
  }
  if (!existsSync(full)) {
    return { ok: false, error: `File not found: ${cleaned}` }
  }
  try {
    if (!statSync(full).isFile()) {
      return { ok: false, error: `Not a file: ${cleaned}` }
    }
  } catch {
    return { ok: false, error: `Cannot access file: ${cleaned}` }
  }
  return { ok: true, absolute: full }
}

export function readProjectFile(
  projectPath: string,
  relativePath: string
):
  | { ok: true; content: string; relative_path: string }
  | { ok: false; error: string } {
  const resolved = resolveProjectFile(projectPath, relativePath)
  if (!resolved.ok) return resolved
  try {
    const content = readFileSync(resolved.absolute, "utf8")
    const rel = relative(resolve(projectPath), resolved.absolute).replace(
      /\\/g,
      "/"
    )
    if (!rel || isAbsolute(rel) || rel.startsWith("..")) {
      return { ok: false, error: "Path escapes project directory" }
    }
    return { ok: true, content, relative_path: rel }
  } catch (err) {
    return {
      ok: false,
      error: err instanceof Error ? err.message : "Failed to read file",
    }
  }
}
