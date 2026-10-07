import type { AppAIChatResult, AppAIStreamEvent } from '@/lib/app-ai'
import type {
    AIProviderConfig,
    BlueprintAIInput,
    BlueprintInput,
    BlueprintLogEvent,
    BlueprintRunInput,
    CommandKind,
    CopyParts,
    GitInfo,
    GitWorktreeAddInput,
    RunMode,
    RunnerEvent,
    StatusEvent,
} from '@/lib/types'
import * as Go from './wailsjs/go/main/App'
import { types } from './wailsjs/go/models'
import { EventsOn, OnFileDrop, OnFileDropOff } from './wailsjs/runtime/runtime'

type RunnerEventHandler = (appId: number, event: RunnerEvent) => void

const listeners = new Set<RunnerEventHandler>()

EventsOn('runnerEvent', (payload: { appId: number; event: RunnerEvent }) => {
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

export function onBlueprintLog(handler: (event: BlueprintLogEvent) => void) {
    return EventsOn('blueprintEvent', (event: BlueprintLogEvent) => {
        if (event) handler(event)
    })
}

/** Calls `handler` with the absolute paths of files/folders dropped on the window. */
export function onFileDrop(handler: (paths: string[]) => void) {
    OnFileDrop((_x, _y, paths) => {
        if (paths?.length) handler(paths)
    }, false)
    return () => OnFileDropOff()
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
        create: (body: { name: string; icon?: string | null; color?: string | null }) =>
            call(() => Go.WorkspacesCreate(body)),
        update: (id: number, body: { name?: string; icon?: string | null; color?: string | null }) =>
            call(() => Go.WorkspacesUpdate(id, body)),
        delete: (id: number) => call(() => Go.WorkspacesDelete(id)),
    },

    apps: {
        list: (workspaceId: number) => call(() => Go.AppsList(workspaceId)),
        get: (id: number) => call(() => Go.AppsGet(id)),
        create: (workspaceId: number, body: { name: string; project_path: string }) =>
            call(() => Go.AppsCreate(workspaceId, body)),
        update: (id: number, body: { name?: string; project_path?: string }) =>
            call(() => Go.AppsUpdate(id, body)),
        reorder: (workspaceId: number, ids: number[]) =>
            call(() => Go.AppsReorder(workspaceId, ids)),
        delete: (id: number) => call(() => Go.AppsDelete(id)),
        openFolder: (id: number) => call(() => Go.AppsOpenFolder(id)),
        editors: () => call(() => Go.AppsEditors()),
        openInEditor: (id: number, editor: string) => call(() => Go.AppsOpenInEditor(id, editor)),
    },

    blueprints: {
        list: () => call(() => Go.BlueprintsList()),
        create: (body: BlueprintInput) =>
            call(() => Go.BlueprintsCreate(types.BlueprintInput.createFrom(body))),
        update: (id: number, body: BlueprintInput) =>
            call(() => Go.BlueprintsUpdate(id, types.BlueprintInput.createFrom(body))),
        delete: (id: number) => call(() => Go.BlueprintsDelete(id)),
        aiPropose: (body: BlueprintAIInput) =>
            call(() => Go.BlueprintsAIPropose(types.BlueprintAIInput.createFrom(body))),
        createApp: (body: BlueprintRunInput) => call(() => Go.BlueprintsCreateApp(body)),
        resolve: (runId: string, action: 'skip' | 'abort') =>
            call(() => Go.BlueprintsResolve(runId, action)),
        sendInput: (runId: string, data: string) => call(() => Go.BlueprintsSendInput(runId, data)),
        resize: (runId: string, cols: number, rows: number) =>
            call(() => Go.BlueprintsResize(runId, cols, rows)),
        cancel: (runId: string) => call(() => Go.BlueprintsCancel(runId)),
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
            },
        ) => call(() => Go.ConfigSetsCreate(appId, body)),
        update: (id: number, body: { name: string }) => call(() => Go.ConfigSetsUpdate(id, body)),
        delete: (id: number) => call(() => Go.ConfigSetsDelete(id)),
        activate: (id: number) => call(() => Go.ConfigSetsActivate(id)),
        copyFrom: (id: number, sourceId: number, parts?: CopyParts) =>
            call(() => Go.ConfigSetsCopyFrom(id, sourceId, parts ?? {})),
    },

    envVars: {
        list: (appId: number) => call(() => Go.EnvVarsList(appId)),
        create: (appId: number, body: { key: string; value?: string; include_in_ai?: boolean }) =>
            call(() => Go.EnvVarsCreate(appId, body)),
        update: (id: number, body: { key?: string; value?: string; include_in_ai?: boolean }) =>
            call(() => Go.EnvVarsUpdate(id, body)),
        delete: (id: number) => call(() => Go.EnvVarsDelete(id)),
        importEnv: (appId: number) => call(() => Go.EnvVarsImport(appId)),
    },

    templates: {
        list: (appId: number) => call(() => Go.TemplatesList(appId)),
        create: (appId: number, body: { file_path: string; content?: string; include_in_ai?: boolean }) =>
            call(() => Go.TemplatesCreate(appId, body)),
        update: (id: number, body: { file_path?: string; content?: string; include_in_ai?: boolean }) =>
            call(() => Go.TemplatesUpdate(id, body)),
        delete: (id: number) => call(() => Go.TemplatesDelete(id)),
    },

    packageScripts: {
        list: (appId: number) => call(() => Go.PackageScriptsList(appId)),
    },

    /** The active config set's run, build or setup command list, by `kind`. */
    runConfig: {
        get: (appId: number, kind: CommandKind) => call(() => Go.RunConfigGet(appId, kind)),
        save: (
            appId: number,
            kind: CommandKind,
            body: {
                mode?: RunMode
                commands?: Array<{ label?: string | null; command: string }>
            },
        ) => call(() => Go.RunConfigSave(appId, kind, body)),
    },

    runner: {
        status: (appId: number) => call(() => Go.RunnerStatus(appId)),
        workspaceStatus: (workspaceId: number) => call(() => Go.RunnerWorkspaceStatus(workspaceId)),
        output: (appId: number, commandId: number) => call(() => Go.RunnerOutput(appId, commandId)),
        resize: (appId: number, commandId: number, cols: number, rows: number) =>
            call(() => Go.RunnerResize(appId, commandId, cols, rows)),
        run: (appId: number) => call(() => Go.RunnerRun(appId)),
        build: (appId: number) => call(() => Go.RunnerBuild(appId)),
        setup: (appId: number) => call(() => Go.RunnerSetup(appId)),
        stop: (appId: number) => call(() => Go.RunnerStop(appId)),
        reload: (appId: number) => call(() => Go.RunnerReload(appId)),
    },

    git: {
        info: (appId: number) => call(() => Go.GitInfo(appId)) as Promise<GitInfo>,
        fetchAll: (appId: number) => call(() => Go.GitFetchAll(appId)),
        worktreeAdd: (appId: number, body: GitWorktreeAddInput) =>
            call(() => Go.GitWorktreeAdd(appId, body)),
        worktreeRemove: (appId: number, path: string, force: boolean) =>
            call(() => Go.GitWorktreeRemove(appId, path, force)),
        worktreePrune: (appId: number) => call(() => Go.GitWorktreePrune(appId)),
        /** Open the app's folder inside the worktree at `path`. */
        worktreeOpenFolder: (appId: number, path: string) =>
            call(() => Go.GitWorktreeOpenFolder(appId, path)),
        worktreeOpenInEditor: (appId: number, path: string, editor: string) =>
            call(() => Go.GitWorktreeOpenInEditor(appId, path, editor)),
        /** Runs the app's run/build/setup config inside the worktree at `path`, with config set `configSetId` (0: the active one). */
        worktreeStart: (appId: number, path: string, kind: CommandKind, configSetId: number) =>
            call(() => Go.GitWorktreeStart(appId, path, kind, configSetId)) as Promise<StatusEvent>,
    },

    ports: {
        list: () => call(() => Go.PortsList()),
        kill: (pid: number) => call(() => Go.PortsKill(pid)),
    },

    readyUrlPatterns: {
        list: () => call(() => Go.ReadyUrlPatternsList()),
        create: (body: { label: string; pattern: string; flags?: string }) =>
            call(() => Go.ReadyUrlPatternsCreate(body)),
        update: (id: number, body: { label?: string; pattern?: string; flags?: string }) =>
            call(() => Go.ReadyUrlPatternsUpdate(id, body)),
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
        chat: (body: { system?: string; prompt: string }) => call(() => Go.AIChat(body)),
        appChat: async (
            body: {
                appId: number
                configSetId: number
                history?: {
                    role: 'user' | 'assistant'
                    text: string
                    tools?: { name: string; input: unknown; output: unknown }[]
                }[]
                instruction: string
            },
            onEvent?: (ev: AppAIStreamEvent) => void,
        ): Promise<AppAIChatResult> => {
            const off = onEvent
                ? EventsOn('appAIEvent', (ev: AppAIStreamEvent) => onEvent(ev))
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
        pickFolder: (opts?: { startDir?: string }) => call(() => Go.FsPickFolder(opts ?? {})),
        pickFile: (opts?: { startDir?: string; appId?: number }) =>
            call(() => Go.FsPickFile(opts ?? {})),
        pickAppFile: (appId: number) => call(() => Go.FsPickAppFile(appId)),
        readAppFile: (appId: number, path: string) => call(() => Go.FsReadAppFile(appId, path)),
    },

    openExternal: (url: string) => call(() => Go.OpenExternal(url)),
}

export { ApiError }
