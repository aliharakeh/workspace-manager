import type { Genkit } from "genkit"
import { z } from "genkit"
import type { ConfigSetDetail, RunMode } from "@/lib/types"
import type { AppAIChatTurn, AppAIPatch, AppAIToolCall } from "@/lib/app-ai"
import { readProjectFile } from "./fs"
import { searchProjectFiles } from "./gitignore-glob"

export const APP_AI_AGENT_SYSTEM = `You are a configuration assistant for Workspace Manager.
You edit ONLY the currently selected config set of the current app.

Use tools to inspect and change this set:
- env vars: list_vars, get_var, update_var, delete_var
- templates: list_templates, get_template, update_template (Handlebars {{VAR_NAME}} for env placeholders)
- run config: get_run_config, update_run_config
- project files: search_files (glob; gitignored omitted), read_file (relative path)

You must NOT:
- create, rename, or delete config sets
- invent template file paths; only update templates list_templates returns
- reformat template files unless the user asked

When updating run commands, send the full command list.
Edits are staged for the user to review. Reply in short markdown (lists, inline code). No JSON. No headings.`

export function buildAppAIAgentPrompt(input: {
  appName: string
  projectPath: string
  configSetId: number
  configSetName: string
  history: AppAIChatTurn[]
  instruction: string
}): string {
  const historyBlock =
    input.history.length > 0
      ? input.history
          .map((t) => `${t.role === "user" ? "User" : "Assistant"}: ${t.text}`)
          .join("\n")
      : "(none)"

  return `App: ${input.appName}
Project path: ${input.projectPath}

Active config set (ONLY edit this one):
id: ${input.configSetId}
name: ${input.configSetName}

Use tools to read or edit env vars, templates, and run config. Use search_files and read_file when you need project files.

Prior conversation:
${historyBlock}

User instruction:
${input.instruction.trim()}`
}

type RunCmd = { label: string | null; command: string }

export class AppAIAgentState {
  private env = new Map<string, string>()
  private origEnv = new Map<string, string>()
  private deletedEnv = new Set<string>()
  private templates = new Map<string, string>()
  private origTemplates = new Map<string, string>()
  private runMode: RunMode
  private runCommands: RunCmd[]
  private runDirty = false
  readonly toolCalls: AppAIToolCall[] = []
  onTool?: (call: AppAIToolCall) => void
  projectPath: string

  constructor(detail: ConfigSetDetail, projectPath: string) {
    this.projectPath = projectPath
    for (const v of detail.env_vars) this.env.set(v.key, v.value)
    for (const t of detail.templates) this.templates.set(t.file_path, t.content)
    this.origEnv = new Map(this.env)
    this.origTemplates = new Map(this.templates)
    this.runMode = detail.run_config?.mode ?? "parallel"
    this.runCommands = (detail.run_config?.commands ?? []).map((c) => ({
      label: c.label,
      command: c.command,
    }))
  }

  patch(): AppAIPatch {
    const upsert: { key: string; value: string }[] = []
    for (const [key, value] of this.env) {
      if (this.origEnv.get(key) !== value) upsert.push({ key, value })
    }
    const del = [...this.deletedEnv]
    const templates: { file_path: string; content: string }[] = []
    for (const [file_path, content] of this.templates) {
      if (this.origTemplates.get(file_path) !== content) {
        templates.push({ file_path, content })
      }
    }
    return {
      message: "",
      env:
        upsert.length || del.length
          ? {
              upsert: upsert.length ? upsert : undefined,
              delete: del.length ? del : undefined,
            }
          : undefined,
      templates: templates.length ? templates : undefined,
      run: this.runDirty
        ? { mode: this.runMode, commands: this.runCommands }
        : undefined,
    }
  }

  listVars() {
    return [...this.env.entries()].map(([key, value]) => ({ key, value }))
  }

  getVar(key: string) {
    const k = key.trim()
    if (!k) return { error: "key is required" }
    if (!this.env.has(k)) return { error: `env var not found: ${k}` }
    return { key: k, value: this.env.get(k)! }
  }

  updateVar(key: string, value: string) {
    const k = key.trim()
    if (!k) return { error: "key is required" }
    this.deletedEnv.delete(k)
    this.env.set(k, value)
    return { key: k, value, ok: true }
  }

  deleteVar(key: string) {
    const k = key.trim()
    if (!k) return { error: "key is required" }
    if (!this.env.has(k) && !this.origEnv.has(k)) {
      return { error: `env var not found: ${k}` }
    }
    this.env.delete(k)
    if (this.origEnv.has(k)) this.deletedEnv.add(k)
    return { ok: true, key: k }
  }

  listTemplates() {
    return [...this.templates.keys()]
  }

  getTemplate(filePath: string) {
    const p = filePath.trim().replace(/\\/g, "/")
    if (!p) return { error: "file_path is required" }
    if (!this.templates.has(p)) return { error: `template not found: ${p}` }
    return { file_path: p, content: this.templates.get(p)! }
  }

  updateTemplate(filePath: string, content: string) {
    const p = filePath.trim().replace(/\\/g, "/")
    if (!p) return { error: "file_path is required" }
    if (!this.templates.has(p)) {
      return { error: `template not on this config set: ${p}` }
    }
    this.templates.set(p, content)
    return { ok: true, file_path: p }
  }

  getRun() {
    return { mode: this.runMode, commands: this.runCommands }
  }

  updateRun(input: { mode?: string; commands?: RunCmd[] }) {
    if (input.mode === "parallel" || input.mode === "sequential") {
      this.runMode = input.mode
      this.runDirty = true
    } else if (input.mode != null && input.mode !== "") {
      return { error: 'mode must be "parallel" or "sequential"' }
    }
    if (input.commands) {
      const commands: RunCmd[] = []
      for (const c of input.commands) {
        const command = c.command?.trim()
        if (!command) continue
        commands.push({
          label: c.label?.trim() || null,
          command,
        })
      }
      this.runCommands = commands
      this.runDirty = true
    }
    if (!this.runDirty) return { error: "provide mode and/or commands" }
    return { ok: true, ...this.getRun() }
  }

  searchFiles(pattern: string) {
    return searchProjectFiles(this.projectPath, pattern)
  }

  readFile(path: string) {
    const got = readProjectFile(this.projectPath, path)
    if (!got.ok) return { error: got.error }
    return { file_path: got.relative_path, content: got.content }
  }
}

function tap<T>(
  state: AppAIAgentState,
  name: string,
  input: unknown,
  output: T
): T {
  const call: AppAIToolCall = { name, input, output }
  state.toolCalls.push(call)
  state.onTool?.(call)
  return output
}

export function bindAppAITools(ai: Genkit, state: AppAIAgentState) {
  return [
    ai.dynamicTool(
      {
        name: "list_vars",
        description: "List env var keys and values on the active config set.",
        inputSchema: z.object({}),
      },
      async (input) => tap(state, "list_vars", input, { vars: state.listVars() })
    ),
    ai.dynamicTool(
      {
        name: "get_var",
        description: "Get one env var by key.",
        inputSchema: z.object({ key: z.string() }),
      },
      async (input) => tap(state, "get_var", input, state.getVar(input.key))
    ),
    ai.dynamicTool(
      {
        name: "update_var",
        description: "Create or update an env var on the active config set.",
        inputSchema: z.object({ key: z.string(), value: z.string() }),
      },
      async (input) =>
        tap(state, "update_var", input, state.updateVar(input.key, input.value))
    ),
    ai.dynamicTool(
      {
        name: "delete_var",
        description: "Delete an env var from the active config set.",
        inputSchema: z.object({ key: z.string() }),
      },
      async (input) => tap(state, "delete_var", input, state.deleteVar(input.key))
    ),
    ai.dynamicTool(
      {
        name: "list_templates",
        description:
          "List template file paths on the active config set (no content).",
        inputSchema: z.object({}),
      },
      async (input) =>
        tap(state, "list_templates", input, { file_paths: state.listTemplates() })
    ),
    ai.dynamicTool(
      {
        name: "get_template",
        description: "Get the full content of one template by file_path.",
        inputSchema: z.object({ file_path: z.string() }),
      },
      async (input) =>
        tap(state, "get_template", input, state.getTemplate(input.file_path))
    ),
    ai.dynamicTool(
      {
        name: "update_template",
        description:
          "Replace the content of an existing template. file_path must already be on this config set.",
        inputSchema: z.object({
          file_path: z.string(),
          content: z.string(),
        }),
      },
      async (input) =>
        tap(
          state,
          "update_template",
          input,
          state.updateTemplate(input.file_path, input.content)
        )
    ),
    ai.dynamicTool(
      {
        name: "get_run_config",
        description: "Get run mode and commands for the active config set.",
        inputSchema: z.object({}),
      },
      async (input) => tap(state, "get_run_config", input, state.getRun())
    ),
    ai.dynamicTool(
      {
        name: "update_run_config",
        description:
          "Update run mode and/or replace the full command list for this config set.",
        inputSchema: z.object({
          mode: z.enum(["parallel", "sequential"]).optional(),
          commands: z
            .array(
              z.object({
                label: z.string().nullable().optional(),
                command: z.string(),
              })
            )
            .optional(),
        }),
      },
      async (input) =>
        tap(
          state,
          "update_run_config",
          input,
          state.updateRun({
            mode: input.mode,
            commands: input.commands?.map((c) => ({
              label: c.label ?? null,
              command: c.command,
            })),
          })
        )
    ),
    ai.dynamicTool(
      {
        name: "search_files",
        description:
          "Glob search under the app project directory. Respects .gitignore. Returns matching relative paths.",
        inputSchema: z.object({
          pattern: z
            .string()
            .describe('Glob relative to the app directory, e.g. "**/*.env"'),
        }),
      },
      async (input) =>
        tap(state, "search_files", input, state.searchFiles(input.pattern))
    ),
    ai.dynamicTool(
      {
        name: "read_file",
        description:
          "Read a text file under the app project directory. Path must be relative and stay inside the project.",
        inputSchema: z.object({
          path: z.string().describe("Project-relative path, e.g. package.json"),
        }),
      },
      async (input) => tap(state, "read_file", input, state.readFile(input.path))
    ),
  ]
}
