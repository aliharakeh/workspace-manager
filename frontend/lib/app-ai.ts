import type { ConfigSetDetail, RunMode } from "./types"

export const APP_AI_SYSTEM_PROMPT = `You are a configuration assistant for Workspace Manager.
You edit ONLY the currently selected config set of the current app.

Use tools to inspect and change this set. Do not expect env vars, templates, or run config in the user message.

You must NOT:
- create, rename, delete, or edit any other config set
- invent template file paths
- reformat template files unless the user asked

Reply in short markdown (lists, inline code). No JSON. No headings. Edits are applied through tools.`

export type AppAIChatTurn = {
  role: "user" | "assistant"
  text: string
  /** Prior tool calls for this assistant turn (replayed as Genkit tool messages). */
  tools?: AppAIToolCall[]
}

export type AppAIToolCall = {
  name: string
  input: unknown
  output: unknown
}

export type AppAIStreamEvent =
  | { type: "tool"; call: AppAIToolCall }
  | { type: "text"; text: string }
  | {
      type: "done"
      text: string
      patch: AppAIPatch
      toolCalls?: AppAIToolCall[]
    }
  | { type: "error"; error: string }

export type AppAIChatResult = {
  text: string
  patch: AppAIPatch
  toolCalls?: AppAIToolCall[]
}

export function summarizeAIToolInput(input: unknown, max = 72): string {
  let s: string
  try {
    s = JSON.stringify(input) ?? ""
  } catch {
    s = String(input)
  }
  if (s === "{}" || s === "null" || s === "") return ""
  return s.length <= max ? s : `${s.slice(0, max - 1)}…`
}

export type AppAIEnvUpsert = { key: string; value: string }

export type AppAIPatch = {
  message: string
  env?: {
    upsert?: AppAIEnvUpsert[]
    delete?: string[]
  }
  templates?: { file_path: string; content: string }[]
  run?: {
    mode?: RunMode
    commands?: { label?: string | null; command: string }[]
  }
}

export function buildAppAIPrompt({
  appName,
  projectPath,
  configSet,
  instruction,
}: {
  appName: string
  projectPath: string
  configSet: ConfigSetDetail
  /** @deprecated prior turns are Genkit messages, not prompt text */
  history?: AppAIChatTurn[]
  instruction: string
  /** @deprecated templates are fetched via tools */
  templatePaths?: string[]
}): string {
  return `App: ${appName}
Project path: ${projectPath}

Active config set (ONLY edit this one):
id: ${configSet.id}
name: ${configSet.name}

Use tools to read or edit env vars, templates, and run config. Use search_files and read_file when you need project files.

User instruction:
${instruction.trim()}`
}

export function stripAIFences(raw: string): string {
  let text = raw.trim()
  const codeBlockMatch = text.match(
    /^```(?:[a-zA-Z0-9_.-]+)?\r?\n([\s\S]*?)\r?\n```$/
  )
  if (codeBlockMatch && codeBlockMatch[1] !== undefined) {
    return codeBlockMatch[1]
  }
  if (text.startsWith("```") && text.endsWith("```")) {
    text = text.slice(3, -3).trim()
    const firstNewline = text.indexOf("\n")
    if (firstNewline !== -1) {
      const firstLine = text.slice(0, firstNewline).trim()
      if (/^[a-zA-Z0-9_.-]+$/.test(firstLine)) {
        text = text.slice(firstNewline + 1)
      }
    }
    return text
  }
  return text
}

function asString(v: unknown): string | undefined {
  return typeof v === "string" ? v : undefined
}

function parseEnv(raw: unknown): AppAIPatch["env"] | undefined {
  if (!raw || typeof raw !== "object") return undefined
  const obj = raw as Record<string, unknown>
  const upsert: AppAIEnvUpsert[] = []
  if (Array.isArray(obj.upsert)) {
    for (const item of obj.upsert) {
      if (!item || typeof item !== "object") continue
      const rec = item as Record<string, unknown>
      const key = asString(rec.key)?.trim()
      if (!key) continue
      upsert.push({ key, value: asString(rec.value) ?? "" })
    }
  }
  const del: string[] = []
  if (Array.isArray(obj.delete)) {
    for (const item of obj.delete) {
      if (typeof item === "string" && item.trim()) del.push(item.trim())
    }
  }
  if (upsert.length === 0 && del.length === 0) return undefined
  return {
    upsert: upsert.length ? upsert : undefined,
    delete: del.length ? del : undefined,
  }
}

function parseTemplates(raw: unknown): AppAIPatch["templates"] | undefined {
  if (!Array.isArray(raw)) return undefined
  const out: { file_path: string; content: string }[] = []
  for (const item of raw) {
    if (!item || typeof item !== "object") continue
    const rec = item as Record<string, unknown>
    const file_path = asString(rec.file_path)?.trim()
    const content = asString(rec.content)
    if (!file_path || content == null) continue
    out.push({ file_path, content })
  }
  return out.length ? out : undefined
}

function parseRun(raw: unknown): AppAIPatch["run"] | undefined {
  if (!raw || typeof raw !== "object") return undefined
  const obj = raw as Record<string, unknown>
  const modeRaw = asString(obj.mode)
  const mode: RunMode | undefined =
    modeRaw === "parallel" || modeRaw === "sequential" ? modeRaw : undefined
  const commands: { label?: string | null; command: string }[] = []
  if (Array.isArray(obj.commands)) {
    for (const item of obj.commands) {
      if (!item || typeof item !== "object") continue
      const rec = item as Record<string, unknown>
      const command = asString(rec.command)?.trim()
      if (!command) continue
      const label = asString(rec.label)?.trim() || null
      commands.push({ label, command })
    }
  }
  if (!mode && commands.length === 0) return undefined
  return {
    mode,
    commands: commands.length ? commands : undefined,
  }
}

export function parseAppAIResponse(raw: string): AppAIPatch {
  const text = stripAIFences(raw)
  const start = text.indexOf("{")
  const end = text.lastIndexOf("}")
  if (start === -1 || end <= start) {
    return { message: text || "Done." }
  }
  try {
    const obj = JSON.parse(text.slice(start, end + 1)) as Record<string, unknown>
    const message = asString(obj.message)?.trim() || "Done."
    return {
      message,
      env: parseEnv(obj.env),
      templates: parseTemplates(obj.templates),
      run: parseRun(obj.run),
    }
  } catch {
    return { message: text || "Done." }
  }
}

export function patchHasEdits(patch: AppAIPatch): boolean {
  return !!(
    patch.env?.upsert?.length ||
    patch.env?.delete?.length ||
    patch.templates?.length ||
    patch.run
  )
}
