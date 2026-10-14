import type { GitWorktreeAddInput, RunMode } from "./types"

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
  dev?: AppAICommandPatch
  prod?: AppAICommandPatch
  build?: AppAICommandPatch
  /** Git worktree changes; they run against the repo when the patch is applied. */
  worktrees?: {
    add?: GitWorktreeAddInput[]
    remove?: { path: string; force: boolean }[]
  }
}

/** A staged change to a run or build config. */
export type AppAICommandPatch = {
  mode?: RunMode
  commands?: { label?: string | null; command: string }[]
}

export function patchHasEdits(patch: AppAIPatch): boolean {
  return !!(
    patch.env?.upsert?.length ||
    patch.env?.delete?.length ||
    patch.templates?.length ||
    patch.dev ||
    patch.prod ||
    patch.build ||
    patch.worktrees?.add?.length ||
    patch.worktrees?.remove?.length
  )
}
