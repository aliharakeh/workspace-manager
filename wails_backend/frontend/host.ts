import type { AIProviderConfig, CopyParts, RunMode, RunnerEvent } from "@/lib/types"
import type { AppAIChatResult, AppAIStreamEvent } from "@/lib/app-ai"
import { EventsOn } from "./wailsjs/runtime/runtime"
import * as Go from "./wailsjs/go/main/App"

type RunnerEventHandler = (appId: number, event: RunnerEvent) => void

const listeners = new Set<RunnerEventHandler>()

EventsOn("runnerEvent", (payload: { appId: number; event: RunnerEvent }) => {
  if (!payload) return
  for (const handler of listeners) handler(payload.appId, payload.event)
})

export function onRunnerEvent(handler: RunnerEventHandler, appId?: number) {
  void appId
  listeners.add(handler)
  return () => {
    listeners.delete(handler)
  }
}

class ApiError extends Error {
  status: number
  constructor(message: string, status = 500) {
    super(message)
    this.status = status
  }
}

async function call<T>(fn: () => Promise<T>): Promise<T> {
  try {
    return await fn()
  } catch (err) {
    throw new ApiError(err instanceof Error ? err.message : String(err))
  }
}

export const api = {
  workspaces: {
    list: () => call(() => Go.WorkspacesList()),
    create: (body: { name: string; icon?: string | null }) =>
      call(() => Go.WorkspacesCreate(body)),
    update: (id: number, body: { name?: string; icon?: string | null }) =>
      call(() => Go.WorkspacesUpdate(id, body)),
    delete: (id: number) => call(() => Go.WorkspacesDelete(id)),
  },

  apps: {
    list: (workspaceId: number) => call(() => Go.AppsList(workspaceId)),
    get: (id: number) => call(() => Go.AppsGet(id)),
    create: (
      workspaceId: number,
      body: { name: string; project_path: string }
    ) => call(() => Go.AppsCreate(workspaceId, body)),
    update: (id: number, body: { name?: string; project_path?: string }) =>
      call(() => Go.AppsUpdate(id, body)),
    delete: (id: number) => call(() => Go.AppsDelete(id)),
    openInEditor: (id: number) => call(() => Go.AppsOpenInEditor(id)),
    git: {
      branches: (id: number) => call(() => Go.GitGraphBranches(id)),
      remote: (id: number) => call(() => Go.GitGraphRemote(id)),
      fetch: (id: number) => call(() => Go.GitGraphFetch(id)),
      load: (
        id: number,
        body: { branches: string[]; since?: string; until?: string }
      ) =>
        call(() =>
          Go.GitGraphLoad(id, {
            branches: body.branches,
            since: body.since ?? "",
            until: body.until ?? "",
          })
        ),
    },
  },

  configSets: {
    list: (appId: number) => call(() => Go.ConfigSetsList(appId)),
    getDetail: (id: number) => call(() => Go.ConfigSetsGetDetail(id)),
    create: (
      appId: number,
      body: {
        name: string
        copy_from_id?: number
        activate?: boolean
        parts?: CopyParts
      }
    ) => call(() => Go.ConfigSetsCreate(appId, body)),
    update: (id: number, body: { name: string }) =>
      call(() => Go.ConfigSetsUpdate(id, body)),
    delete: (id: number) => call(() => Go.ConfigSetsDelete(id)),
    activate: (id: number) => call(() => Go.ConfigSetsActivate(id)),
    copyFrom: (id: number, sourceId: number, parts?: CopyParts) =>
      call(() => Go.ConfigSetsCopyFrom(id, sourceId, parts ?? {})),
  },

  envVars: {
    list: (appId: number) => call(() => Go.EnvVarsList(appId)),
    create: (appId: number, body: { key: string; value?: string }) =>
      call(() => Go.EnvVarsCreate(appId, body)),
    update: (id: number, body: { key?: string; value?: string }) =>
      call(() => Go.EnvVarsUpdate(id, body)),
    delete: (id: number) => call(() => Go.EnvVarsDelete(id)),
    importEnv: (appId: number) => call(() => Go.EnvVarsImport(appId)),
  },

  templates: {
    list: (appId: number) => call(() => Go.TemplatesList(appId)),
    create: (appId: number, body: { file_path: string; content?: string }) =>
      call(() => Go.TemplatesCreate(appId, body)),
    update: (id: number, body: { file_path?: string; content?: string }) =>
      call(() => Go.TemplatesUpdate(id, body)),
    delete: (id: number) => call(() => Go.TemplatesDelete(id)),
  },

  runConfig: {
    get: (appId: number) => call(() => Go.RunConfigGet(appId)),
    save: (
      appId: number,
      body: {
        mode?: RunMode
        commands?: Array<{ label?: string | null; command: string }>
      }
    ) => call(() => Go.RunConfigSave(appId, body)),
  },

  runner: {
    status: (appId: number) => call(() => Go.RunnerStatus(appId)),
    workspaceStatus: (workspaceId: number) =>
      call(() => Go.RunnerWorkspaceStatus(workspaceId)),
    logs: (appId: number) => call(() => Go.RunnerLogs(appId)),
    run: (appId: number) => call(() => Go.RunnerRun(appId)),
    stop: (appId: number) => call(() => Go.RunnerStop(appId)),
    reload: (appId: number) => call(() => Go.RunnerReload(appId)),
  },

  ports: {
    list: () => call(() => Go.PortsList()),
    kill: (pid: number) => call(() => Go.PortsKill(pid)),
  },

  readyUrlPatterns: {
    list: () => call(() => Go.ReadyUrlPatternsList()),
    create: (body: { label: string; pattern: string; flags?: string }) =>
      call(() => Go.ReadyUrlPatternsCreate(body)),
    update: (
      id: number,
      body: { label?: string; pattern?: string; flags?: string }
    ) => call(() => Go.ReadyUrlPatternsUpdate(id, body)),
    delete: (id: number) => call(() => Go.ReadyUrlPatternsDelete(id)),
  },

  settings: {
    get: () => call(() => Go.SettingsGet()),
    set: (key: string, value: string) => call(() => Go.SettingsSet(key, value)),
  },

  ai: {
    getConfig: () => call(() => Go.AIConfigGet()),
    saveConfig: (body: AIProviderConfig) => call(() => Go.AIConfigSave(body)),
    deleteConfig: (name: string) => call(() => Go.AIConfigDelete(name)),
    activate: (name: string) => call(() => Go.AIConfigActivate({ name })),
    chat: (body: { system?: string; prompt: string }) =>
      call(() => Go.AIChat(body)),
    appChat: async (
      body: {
        appId: number
        configSetId: number
        history?: { role: "user" | "assistant"; text: string }[]
        instruction: string
      },
      onEvent?: (ev: AppAIStreamEvent) => void
    ): Promise<AppAIChatResult> => {
      const off = onEvent
        ? EventsOn("appAIEvent", (ev: AppAIStreamEvent) => onEvent(ev))
        : undefined
      try {
        return (await call(() => Go.AIAppChat(body))) as AppAIChatResult
      } finally {
        off?.()
      }
    },
    test: (body: AIProviderConfig) => call(() => Go.AITest(body)),
  },

  fs: {
    validatePath: (path: string) => call(() => Go.FsValidatePath(path)),
    pickFolder: (opts?: { startDir?: string }) =>
      call(() => Go.FsPickFolder(opts ?? {})),
    pickFile: (opts?: { startDir?: string; appId?: number }) =>
      call(() => Go.FsPickFile(opts ?? {})),
    pickAppFile: (appId: number) => call(() => Go.FsPickAppFile(appId)),
    readAppFile: (appId: number, path: string) =>
      call(() => Go.FsReadAppFile(appId, path)),
  },

  openExternal: (url: string) => call(() => Go.OpenExternal(url)),
}

export { ApiError }
