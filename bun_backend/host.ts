import type {
  AIConfigInfo,
  AIProviderConfig,
  App,
  ConfigSet,
  ConfigSetDetail,
  CopyParts,
  EnvVar,
  GitBranchInfo,
  GitRemoteInfo,
  GitRepoGraph,
  ListeningProcess,
  ReadyUrlPattern,
  RunConfig,
  RunMode,
  RunnerEvent,
  RunnerLogsSnapshot,
  StatusEvent,
  Template,
  Workspace,
} from "@/lib/types"
import type { AppAIChatResult, AppAIStreamEvent } from "@/lib/app-ai"

class ApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.status = status
  }
}

type RunnerEventHandler = (appId: number, event: RunnerEvent) => void

const listeners = new Set<RunnerEventHandler>()

type AppWatch = {
  refs: number
  source: EventSource | null
  retry?: ReturnType<typeof setTimeout>
}

const watches = new Map<number, AppWatch>()

function dispatch(appId: number, event: RunnerEvent) {
  for (const handler of listeners) handler(appId, event)
}

function connect(appId: number, watch: AppWatch) {
  if (watch.refs <= 0) return
  const source = new EventSource(`/api/apps/${appId}/run/logs`)
  watch.source = source
  source.onmessage = (event) => {
    try {
      dispatch(appId, JSON.parse(event.data) as RunnerEvent)
    } catch {
      // ignore malformed payloads
    }
  }
  source.onerror = () => {
    source.close()
    if (watch.source === source) watch.source = null
    if (watch.refs <= 0) return
    watch.retry = setTimeout(() => connect(appId, watch), 1500)
  }
}

function watchApp(appId: number) {
  let watch = watches.get(appId)
  if (!watch) {
    watch = { refs: 0, source: null }
    watches.set(appId, watch)
  }
  watch.refs++
  if (watch.refs === 1) connect(appId, watch)
  return () => {
    watch.refs--
    if (watch.refs > 0) return
    if (watch.retry) clearTimeout(watch.retry)
    watch.source?.close()
    watches.delete(appId)
  }
}

export function onRunnerEvent(handler: RunnerEventHandler, appId?: number) {
  listeners.add(handler)
  const release = appId != null ? watchApp(appId) : undefined
  return () => {
    listeners.delete(handler)
    release?.()
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...init?.headers,
    },
  })

  if (res.status === 204) return undefined as T

  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new ApiError(
      typeof data.error === "string" ? data.error : res.statusText,
      res.status
    )
  }
  return data as T
}

async function readAppChatStream(
  res: Response,
  onEvent?: (ev: AppAIStreamEvent) => void
): Promise<AppAIChatResult> {
  if (!res.body) throw new ApiError("empty stream", 502)
  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buf = ""
  let result: AppAIChatResult | undefined
  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buf += decoder.decode(value, { stream: true })
    const parts = buf.split("\n\n")
    buf = parts.pop() ?? ""
    for (const part of parts) {
      for (const line of part.split("\n")) {
        if (!line.startsWith("data: ")) continue
        const ev = JSON.parse(line.slice(6)) as AppAIStreamEvent
        if (ev.type === "error") throw new ApiError(ev.error, 500)
        if (ev.type === "done") {
          result = {
            text: ev.text,
            patch: ev.patch,
            toolCalls: ev.toolCalls,
          }
          continue
        }
        onEvent?.(ev)
      }
    }
  }
  if (!result) throw new ApiError("AI stream ended early", 502)
  return result
}

export const api = {
  workspaces: {
    list: () => request<Workspace[]>("/api/workspaces"),
    create: (body: { name: string; icon?: string | null }) =>
      request<Workspace>("/api/workspaces", {
        method: "POST",
        body: JSON.stringify(body),
      }),
    update: (id: number, body: { name?: string; icon?: string | null }) =>
      request<Workspace>(`/api/workspaces/${id}`, {
        method: "PATCH",
        body: JSON.stringify(body),
      }),
    delete: (id: number) =>
      request<void>(`/api/workspaces/${id}`, { method: "DELETE" }),
  },

  apps: {
    list: (workspaceId: number) =>
      request<App[]>(`/api/workspaces/${workspaceId}/apps`),
    get: (id: number) => request<App>(`/api/apps/${id}`),
    create: (
      workspaceId: number,
      body: { name: string; project_path: string }
    ) =>
      request<App>(`/api/workspaces/${workspaceId}/apps`, {
        method: "POST",
        body: JSON.stringify(body),
      }),
    update: (id: number, body: { name?: string; project_path?: string }) =>
      request<App>(`/api/apps/${id}`, {
        method: "PATCH",
        body: JSON.stringify(body),
      }),
    delete: (id: number) =>
      request<void>(`/api/apps/${id}`, { method: "DELETE" }),
    openInEditor: (id: number) =>
      request<{ ok: true }>(`/api/apps/${id}/open-in-editor`, {
        method: "POST",
      }),
    git: {
      branches: (id: number) =>
        request<GitBranchInfo[]>(`/api/apps/${id}/git/branches`),
      remote: (id: number) =>
        request<GitRemoteInfo>(`/api/apps/${id}/git/remote`),
      fetch: (id: number) =>
        request<{ ok: true }>(`/api/apps/${id}/git/fetch`, { method: "POST" }),
      load: (
        id: number,
        body: { branches: string[]; since?: string; until?: string }
      ) =>
        request<GitRepoGraph>(`/api/apps/${id}/git/graph`, {
          method: "POST",
          body: JSON.stringify(body),
        }),
    },
  },

  configSets: {
    list: (appId: number) =>
      request<ConfigSet[]>(`/api/apps/${appId}/config-sets`),
    getDetail: (id: number) =>
      request<ConfigSetDetail>(`/api/config-sets/${id}`),
    create: (
      appId: number,
      body: {
        name: string
        copy_from_id?: number
        activate?: boolean
        parts?: CopyParts
      }
    ) =>
      request<ConfigSet>(`/api/apps/${appId}/config-sets`, {
        method: "POST",
        body: JSON.stringify(body),
      }),
    update: (id: number, body: { name: string }) =>
      request<ConfigSet>(`/api/config-sets/${id}`, {
        method: "PATCH",
        body: JSON.stringify(body),
      }),
    delete: (id: number) =>
      request<void>(`/api/config-sets/${id}`, { method: "DELETE" }),
    activate: (id: number) =>
      request<{ id: number; app_id: number; name: string; app: App }>(
        `/api/config-sets/${id}/activate`,
        { method: "POST" }
      ),
    copyFrom: (id: number, sourceId: number, parts?: CopyParts) =>
      request<ConfigSet>(`/api/config-sets/${id}/copy-from`, {
        method: "POST",
        body: JSON.stringify({ source_id: sourceId, parts }),
      }),
  },

  envVars: {
    list: (appId: number) => request<EnvVar[]>(`/api/apps/${appId}/env-vars`),
    create: (
      appId: number,
      body: { key: string; value?: string; include_in_ai?: boolean }
    ) =>
      request<EnvVar>(`/api/apps/${appId}/env-vars`, {
        method: "POST",
        body: JSON.stringify(body),
      }),
    update: (
      id: number,
      body: { key?: string; value?: string; include_in_ai?: boolean }
    ) =>
      request<EnvVar>(`/api/env-vars/${id}`, {
        method: "PATCH",
        body: JSON.stringify(body),
      }),
    delete: (id: number) =>
      request<void>(`/api/env-vars/${id}`, { method: "DELETE" }),
    importEnv: (appId: number) =>
      request<{
        cancelled: boolean
        path?: string
        format?: string
        imported?: number
        vars?: EnvVar[]
        template?: {
          id: number
          file_path: string
          created: boolean
        } | null
      }>(`/api/apps/${appId}/env-vars/import`, { method: "POST" }),
  },

  templates: {
    list: (appId: number) =>
      request<Template[]>(`/api/apps/${appId}/templates`),
    create: (appId: number, body: { file_path: string; content?: string }) =>
      request<Template>(`/api/apps/${appId}/templates`, {
        method: "POST",
        body: JSON.stringify(body),
      }),
    update: (id: number, body: { file_path?: string; content?: string }) =>
      request<Template>(`/api/templates/${id}`, {
        method: "PATCH",
        body: JSON.stringify(body),
      }),
    delete: (id: number) =>
      request<void>(`/api/templates/${id}`, { method: "DELETE" }),
  },

  runConfig: {
    get: (appId: number) => request<RunConfig>(`/api/apps/${appId}/run-config`),
    save: (
      appId: number,
      body: {
        mode?: RunMode
        commands?: Array<{ label?: string | null; command: string }>
      }
    ) =>
      request<RunConfig>(`/api/apps/${appId}/run-config`, {
        method: "PUT",
        body: JSON.stringify(body),
      }),
  },

  runner: {
    status: (appId: number) =>
      request<StatusEvent>(`/api/apps/${appId}/run/status`),
    workspaceStatus: (workspaceId: number) =>
      request<StatusEvent[]>(`/api/workspaces/${workspaceId}/run-status`),
    logs: (appId: number) =>
      request<RunnerLogsSnapshot>(`/api/apps/${appId}/run/snapshot`),
    run: (appId: number) =>
      request<StatusEvent>(`/api/apps/${appId}/run`, { method: "POST" }),
    stop: (appId: number) =>
      request<StatusEvent>(`/api/apps/${appId}/stop`, { method: "POST" }),
    reload: (appId: number) =>
      request<StatusEvent>(`/api/apps/${appId}/reload`, { method: "POST" }),
  },

  ports: {
    list: () =>
      request<{
        min: number
        max: number
        processes: ListeningProcess[]
      }>("/api/ports"),
    kill: (pid: number) =>
      request<{ ok: true; pid: number }>(`/api/ports/${pid}/kill`, {
        method: "POST",
      }),
  },

  readyUrlPatterns: {
    list: () => request<ReadyUrlPattern[]>("/api/ready-url-patterns"),
    create: (body: { label: string; pattern: string; flags?: string }) =>
      request<ReadyUrlPattern>("/api/ready-url-patterns", {
        method: "POST",
        body: JSON.stringify(body),
      }),
    update: (
      id: number,
      body: { label?: string; pattern?: string; flags?: string }
    ) =>
      request<ReadyUrlPattern>(`/api/ready-url-patterns/${id}`, {
        method: "PATCH",
        body: JSON.stringify(body),
      }),
    delete: (id: number) =>
      request<void>(`/api/ready-url-patterns/${id}`, { method: "DELETE" }),
  },

  settings: {
    get: () => request<Record<string, string>>("/api/settings"),
    set: (key: string, value: string) =>
      request<Record<string, string>>("/api/settings", {
        method: "PUT",
        body: JSON.stringify({ key, value }),
      }),
  },

  ai: {
    getConfig: () => request<AIConfigInfo>("/api/ai/config"),
    saveConfig: (body: AIProviderConfig) =>
      request<AIConfigInfo>("/api/ai/config", {
        method: "PUT",
        body: JSON.stringify(body),
      }),
    deleteConfig: (name: string) =>
      request<AIConfigInfo>(`/api/ai/config/${encodeURIComponent(name)}`, {
        method: "DELETE",
      }),
    activate: (name: string) =>
      request<AIConfigInfo>("/api/ai/config/activate", {
        method: "POST",
        body: JSON.stringify({ name }),
      }),
    chat: (body: { system?: string; prompt: string }) =>
      request<{ text: string }>("/api/ai/chat", {
        method: "POST",
        body: JSON.stringify(body),
      }),
    appChat: async (
      body: {
        appId: number
        configSetId: number
        history?: { role: "user" | "assistant"; text: string }[]
        instruction: string
      },
      onEvent?: (ev: AppAIStreamEvent) => void
    ) => {
      const res = await fetch("/api/ai/app-chat", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Accept: "text/event-stream",
        },
        body: JSON.stringify(body),
      })
      if (!res.ok) {
        const data = await res.json().catch(() => ({}))
        throw new ApiError(
          typeof data.error === "string" ? data.error : res.statusText,
          res.status
        )
      }
      return readAppChatStream(res, onEvent)
    },
    test: (body: AIProviderConfig) =>
      request<{ ok: true; text: string }>("/api/ai/test", {
        method: "POST",
        body: JSON.stringify(body),
      }),
  },

  fs: {
    validatePath: (path: string) =>
      request<{ ok: boolean; path?: string; error?: string }>(
        "/api/fs/validate-path",
        {
          method: "POST",
          body: JSON.stringify({ path }),
        }
      ),
    pickFolder: (opts?: { startDir?: string }) =>
      request<{ cancelled: boolean; path?: string }>("/api/fs/pick-folder", {
        method: "POST",
        body: JSON.stringify(opts ?? {}),
      }),
    pickFile: (opts?: { startDir?: string; appId?: number }) =>
      request<{
        cancelled: boolean
        path?: string
        relative_path?: string
        content?: string
      }>("/api/fs/pick-file", {
        method: "POST",
        body: JSON.stringify(opts ?? {}),
      }),
    pickAppFile: (appId: number) =>
      request<{
        cancelled: boolean
        path?: string
        relative_path?: string
        content?: string
      }>(`/api/apps/${appId}/pick-file`, { method: "POST" }),
    readAppFile: (appId: number, path: string) =>
      request<{ ok: true; content: string; relative_path: string }>(
        `/api/apps/${appId}/read-file`,
        {
          method: "POST",
          body: JSON.stringify({ path }),
        }
      ),
  },

  openExternal: (url: string) => {
    window.open(url, "_blank", "noopener,noreferrer")
    return Promise.resolve({ ok: true as const })
  },
}

export { ApiError }
