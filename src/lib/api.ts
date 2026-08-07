import type {
  App,
  ConfigSet,
  EnvVar,
  RunConfig,
  RunMode,
  StatusEvent,
  Template,
  Workspace,
} from "./types"

class ApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.status = status
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
  },

  configSets: {
    list: (appId: number) =>
      request<ConfigSet[]>(`/api/apps/${appId}/config-sets`),
    create: (
      appId: number,
      body: {
        name: string
        copy_from_id?: number
        activate?: boolean
        parts?: { env?: boolean; templates?: boolean; run?: boolean }
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
    copyFrom: (
      id: number,
      sourceId: number,
      parts?: { env?: boolean; templates?: boolean; run?: boolean }
    ) =>
      request<ConfigSet>(`/api/config-sets/${id}/copy-from`, {
        method: "POST",
        body: JSON.stringify({ source_id: sourceId, parts }),
      }),
  },

  envVars: {
    list: (appId: number) => request<EnvVar[]>(`/api/apps/${appId}/env-vars`),
    create: (appId: number, body: { key: string; value?: string }) =>
      request<EnvVar>(`/api/apps/${appId}/env-vars`, {
        method: "POST",
        body: JSON.stringify(body),
      }),
    update: (id: number, body: { key?: string; value?: string }) =>
      request<EnvVar>(`/api/env-vars/${id}`, {
        method: "PATCH",
        body: JSON.stringify(body),
      }),
    delete: (id: number) =>
      request<void>(`/api/env-vars/${id}`, { method: "DELETE" }),
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
    run: (appId: number) =>
      request<StatusEvent>(`/api/apps/${appId}/run`, { method: "POST" }),
    stop: (appId: number) =>
      request<StatusEvent>(`/api/apps/${appId}/stop`, { method: "POST" }),
    reload: (appId: number) =>
      request<StatusEvent>(`/api/apps/${appId}/reload`, { method: "POST" }),
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
    /** Opens the OS native folder dialog. */
    pickFolder: (opts?: { startDir?: string }) =>
      request<{ cancelled: boolean; path?: string }>("/api/fs/pick-folder", {
        method: "POST",
        body: JSON.stringify(opts ?? {}),
      }),
    /** Opens the OS native file dialog. Paths are relative when scoped to an app. */
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
}

export { ApiError }
