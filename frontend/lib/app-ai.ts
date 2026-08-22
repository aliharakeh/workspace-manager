import type { ConfigSetDetail, RunMode } from "./types"

export const APP_AI_SYSTEM_PROMPT = `You are a configuration assistant for Workspace Manager.
You edit ONLY the currently selected config set of the current app.

You may:
- upsert or delete environment variables for this config set
- update content of existing templates (Handlebars {{VAR_NAME}} for env placeholders)
- change run mode and run commands for this config set

You must NOT:
- create, rename, delete, or edit any other config set
- invent template file paths that are not listed
- reformat template files unless the user asked

Reply with a single JSON object (no markdown fences, no extra text):
{
  "message": "short reply to the user",
  "env": {
    "upsert": [{ "key": "NAME", "value": "..." }],
    "delete": ["OLD_KEY"]
  },
  "templates": [{ "file_path": "exact/path", "content": "full new file contents" }],
  "run": {
    "mode": "parallel" | "sequential",
    "commands": [{ "label": "web", "command": "npm run dev" }]
  }
}

Omit env, templates, and/or run when unchanged. Always include message.
For questions with no edits, return only { "message": "..." }.
When updating a template, return its complete new content.
When updating run commands, return the full command list (not a partial diff).`

export type AppAIChatTurn = {
  role: "user" | "assistant"
  text: string
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
  history,
  instruction,
}: {
  appName: string
  projectPath: string
  configSet: ConfigSetDetail
  history: AppAIChatTurn[]
  instruction: string
}): string {
  const envLines =
    configSet.env_vars.length > 0
      ? configSet.env_vars.map((v) => `${v.key}=${v.value}`).join("\n")
      : "(none)"

  const templateBlocks =
    configSet.templates.length > 0
      ? configSet.templates
          .map(
            (t) =>
              `--- file_path: ${t.file_path}\n${t.content}`
          )
          .join("\n\n")
      : "(none)"

  const run = configSet.run_config
  const runBlock = run
    ? `mode: ${run.mode}\ncommands:\n${
        run.commands.length > 0
          ? run.commands
              .map(
                (c, i) =>
                  `  ${i + 1}. label=${c.label ?? ""} command=${c.command}`
              )
              .join("\n")
          : "  (none)"
      }`
    : "(none)"

  const historyBlock =
    history.length > 0
      ? history
          .map((t) => `${t.role === "user" ? "User" : "Assistant"}: ${t.text}`)
          .join("\n")
      : "(none)"

  return `App: ${appName}
Project path: ${projectPath}

Active config set (ONLY edit this one):
id: ${configSet.id}
name: ${configSet.name}

Environment variables:
${envLines}

Templates:
${templateBlocks}

Run config:
${runBlock}

Prior conversation:
${historyBlock}

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
