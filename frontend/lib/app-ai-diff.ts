import type { AppAIPatch } from "./app-ai"
import type { ConfigSetDetail, RunMode } from "./types"

export type AppAIFileDiff = {
  path: string
  oldText: string
  newText: string
}

export type AppAIDiff = {
  files: AppAIFileDiff[]
  skipped: string[]
}

function dumpEnv(pairs: { key: string; value: string }[]): string {
  return pairs.map((p) => `${p.key}=${p.value}`).join("\n")
}

function dumpRun(
  mode: RunMode,
  commands: { label: string; command: string }[]
): string {
  const lines = [`mode: ${mode}`]
  if (commands.length === 0) {
    lines.push("(no commands)")
  } else {
    commands.forEach((c, i) => {
      lines.push(`${i + 1}. ${c.label ? `${c.label}: ` : ""}${c.command}`)
    })
  }
  return lines.join("\n")
}

function cmdKey(c: { label?: string | null; command: string }) {
  return { label: c.label ?? "", command: c.command }
}

export function normalizeNewlines(text: string): string {
  return text.replace(/\r\n/g, "\n").replace(/\r/g, "\n")
}

export function langForPath(path: string): string {
  const name = path.replace(/\\/g, "/").split("/").pop() ?? path
  if (name === "env" || name === ".env" || name.endsWith(".env")) return "ini"
  if (name === "run") return "txt"
  const ext = name.includes(".") ? name.slice(name.lastIndexOf(".") + 1) : "txt"
  const map: Record<string, string> = {
    ts: "typescript",
    tsx: "tsx",
    js: "javascript",
    jsx: "jsx",
    json: "json",
    yml: "yaml",
    yaml: "yaml",
    md: "markdown",
    html: "html",
    css: "css",
    toml: "ini",
    properties: "ini",
  }
  return map[ext] ?? ext
}

export function buildAppAIDiff(
  current: ConfigSetDetail,
  patch: AppAIPatch
): AppAIDiff {
  const files: AppAIFileDiff[] = []
  const skipped: string[] = []

  if (patch.env?.upsert?.length || patch.env?.delete?.length) {
    const oldPairs = current.env_vars.map((v) => ({
      key: v.key,
      value: v.value,
    }))
    const next = new Map(oldPairs.map((p) => [p.key, p.value]))
    for (const key of patch.env.delete ?? []) next.delete(key)
    for (const item of patch.env.upsert ?? []) next.set(item.key, item.value)
    const newPairs: { key: string; value: string }[] = []
    for (const p of oldPairs) {
      if (next.has(p.key)) {
        newPairs.push({ key: p.key, value: next.get(p.key)! })
        next.delete(p.key)
      }
    }
    for (const [key, value] of next) newPairs.push({ key, value })
    const oldText = dumpEnv(oldPairs)
    const newText = dumpEnv(newPairs)
    if (oldText !== newText) {
      files.push({ path: "env", oldText, newText })
    }
  }

  const byPath = new Map(current.templates.map((t) => [t.file_path, t]))
  for (const t of patch.templates ?? []) {
    const existing = byPath.get(t.file_path)
    if (!existing) {
      skipped.push(t.file_path)
      continue
    }
    if (existing.content === t.content) continue
    if (normalizeNewlines(existing.content) === normalizeNewlines(t.content)) {
      continue
    }
    files.push({
      path: t.file_path,
      oldText: existing.content,
      newText: t.content,
    })
  }

  if (patch.run) {
    const modeFrom = current.run_config?.mode ?? "parallel"
    const modeTo = patch.run.mode ?? modeFrom
    const from = (current.run_config?.commands ?? []).map(cmdKey)
    const to = (patch.run.commands ?? current.run_config?.commands ?? []).map(
      cmdKey
    )
    const oldText = dumpRun(modeFrom, from)
    const newText = dumpRun(modeTo, to)
    if (oldText !== newText) {
      files.push({ path: "run", oldText, newText })
    }
  }

  return { files, skipped }
}

export function appAIDiffCount(diff: AppAIDiff): number {
  return diff.files.length + diff.skipped.length
}
