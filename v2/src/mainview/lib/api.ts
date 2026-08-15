import { rpc } from "./electrobun"
import type { CopyParts, RunMode } from "./types"

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
    list: () => call(() => rpc.request.workspacesList({})),
    create: (body: { name: string; icon?: string | null }) =>
      call(() => rpc.request.workspacesCreate(body)),
    update: (id: number, body: { name?: string; icon?: string | null }) =>
      call(() => rpc.request.workspacesUpdate({ id, ...body })),
    delete: (id: number) => call(() => rpc.request.workspacesDelete({ id })),
  },

  apps: {
    list: (workspaceId: number) =>
      call(() => rpc.request.appsList({ workspaceId })),
    get: (id: number) => call(() => rpc.request.appsGet({ id })),
    create: (
      workspaceId: number,
      body: { name: string; project_path: string }
    ) =>
      call(() =>
        rpc.request.appsCreate({ workspaceId, ...body })
      ),
    update: (id: number, body: { name?: string; project_path?: string }) =>
      call(() => rpc.request.appsUpdate({ id, ...body })),
    delete: (id: number) => call(() => rpc.request.appsDelete({ id })),
    openInEditor: (id: number) =>
      call(() => rpc.request.appsOpenInEditor({ id })),
  },

  configSets: {
    list: (appId: number) =>
      call(() => rpc.request.configSetsList({ appId })),
    getDetail: (id: number) =>
      call(() => rpc.request.configSetsGetDetail({ id })),
    create: (
      appId: number,
      body: {
        name: string
        copy_from_id?: number
        activate?: boolean
        parts?: CopyParts
      }
    ) => call(() => rpc.request.configSetsCreate({ appId, ...body })),
    update: (id: number, body: { name: string }) =>
      call(() => rpc.request.configSetsUpdate({ id, name: body.name })),
    delete: (id: number) => call(() => rpc.request.configSetsDelete({ id })),
    activate: (id: number) =>
      call(() => rpc.request.configSetsActivate({ id })),
    copyFrom: (id: number, sourceId: number, parts?: CopyParts) =>
      call(() => rpc.request.configSetsCopyFrom({ id, sourceId, parts })),
  },

  envVars: {
    list: (appId: number) => call(() => rpc.request.envVarsList({ appId })),
    create: (appId: number, body: { key: string; value?: string }) =>
      call(() => rpc.request.envVarsCreate({ appId, ...body })),
    update: (id: number, body: { key?: string; value?: string }) =>
      call(() => rpc.request.envVarsUpdate({ id, ...body })),
    delete: (id: number) => call(() => rpc.request.envVarsDelete({ id })),
    importEnv: (appId: number) =>
      call(() => rpc.request.envVarsImport({ appId })),
  },

  templates: {
    list: (appId: number) =>
      call(() => rpc.request.templatesList({ appId })),
    create: (appId: number, body: { file_path: string; content?: string }) =>
      call(() => rpc.request.templatesCreate({ appId, ...body })),
    update: (id: number, body: { file_path?: string; content?: string }) =>
      call(() => rpc.request.templatesUpdate({ id, ...body })),
    delete: (id: number) => call(() => rpc.request.templatesDelete({ id })),
  },

  runConfig: {
    get: (appId: number) => call(() => rpc.request.runConfigGet({ appId })),
    save: (
      appId: number,
      body: {
        mode?: RunMode
        commands?: Array<{ label?: string | null; command: string }>
      }
    ) => call(() => rpc.request.runConfigSave({ appId, ...body })),
  },

  runner: {
    status: (appId: number) =>
      call(() => rpc.request.runnerStatus({ appId })),
    workspaceStatus: (workspaceId: number) =>
      call(() => rpc.request.runnerWorkspaceStatus({ workspaceId })),
    logs: (appId: number) => call(() => rpc.request.runnerLogs({ appId })),
    run: (appId: number) => call(() => rpc.request.runnerRun({ appId })),
    stop: (appId: number) => call(() => rpc.request.runnerStop({ appId })),
    reload: (appId: number) =>
      call(() => rpc.request.runnerReload({ appId })),
  },

  ports: {
    list: () => call(() => rpc.request.portsList({})),
    kill: (pid: number) => call(() => rpc.request.portsKill({ pid })),
  },

  readyUrlPatterns: {
    list: () => call(() => rpc.request.readyUrlPatternsList({})),
    create: (body: { label: string; pattern: string; flags?: string }) =>
      call(() => rpc.request.readyUrlPatternsCreate(body)),
    update: (
      id: number,
      body: { label?: string; pattern?: string; flags?: string }
    ) => call(() => rpc.request.readyUrlPatternsUpdate({ id, ...body })),
    delete: (id: number) =>
      call(() => rpc.request.readyUrlPatternsDelete({ id })),
  },

  settings: {
    get: () => call(() => rpc.request.settingsGet({})),
    set: (key: string, value: string) =>
      call(() => rpc.request.settingsSet({ key, value })),
  },

  fs: {
    validatePath: (path: string) =>
      call(() => rpc.request.fsValidatePath({ path })),
    pickFolder: (opts?: { startDir?: string }) =>
      call(() => rpc.request.fsPickFolder(opts ?? {})),
    pickFile: (opts?: { startDir?: string; appId?: number }) =>
      call(() => rpc.request.fsPickFile(opts ?? {})),
    pickAppFile: (appId: number) =>
      call(() => rpc.request.fsPickAppFile({ appId })),
    readAppFile: (appId: number, path: string) =>
      call(() => rpc.request.fsReadAppFile({ appId, path })),
  },

  openExternal: (url: string) =>
    call(() => rpc.request.openExternal({ url })),
}

export function handleReadyUrlClick(
  event: { preventDefault(): void; stopPropagation(): void },
  url: string
) {
  event.preventDefault()
  event.stopPropagation()
  void api.openExternal(url)
}

export { ApiError }
