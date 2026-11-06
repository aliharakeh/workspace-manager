import type {
    ConfigSetDetail,
    RunConfig,
    App,
    BlueprintInput,
    BlueprintLogEvent,
    BlueprintRunInput,
    CommandKind,
    CopyParts,
    DataChangedEvent,
    SharedEnvVar,
    SharedLinkInput,
    SharedSet,
    SharedSetDetail,
    GitInfo,
    GitPullResult,
    GitWorktreeAddInput,
    GitWorktreeAddResult,
    RunMode,
    RunnerEvent,
    StatusEvent,
    WorktreeLinkState,
    WorkspaceBundle,
    WorkspaceBundleFileResult,
    WorkspaceExportFileResult,
    WorkspaceExportOptions,
    WorkspaceImportOptions,
    WorkspaceImportResult,
} from '@/lib/types'
import * as Go from './bindings/workspace-manager/gui/app'
import * as types from './bindings/workspace-manager/shared/api/types/models'
import { Events } from '@wailsio/runtime'

type RunnerEventHandler = (appId: number, event: RunnerEvent) => void

const listeners = new Set<RunnerEventHandler>()

Events.On('runnerEvent', ({ data: payload }: { data: { appId: number; event: RunnerEvent } }) => {
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
    return Events.On('blueprintEvent', ({ data: event }: { data: BlueprintLogEvent }) => {
        if (event) handler(event)
    })
}

/** Calls `handler` for every write the server reports, whoever made it. */
export function onDataChanged(handler: (event: DataChangedEvent) => void) {
    return Events.On('dataChanged', ({ data: event }: { data: DataChangedEvent }) => {
        if (event) handler(event)
    })
}

/** Calls `handler` with the absolute paths of files/folders dropped on the window. */
export function onFileDrop(handler: (paths: string[]) => void) {
    return Events.On('filesDropped', ({ data: paths }: { data: string[] }) => {
        if (paths?.length) handler(paths)
    })
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
    envProvider: {
        run: (shared: boolean, id: number) => Go.EnvProviderRun(shared, id),
        get: (shared: boolean, id: number) => Go.EnvProviderGet(shared, id),
        save: (shared: boolean, id: number, body: { command: string; directory: string; timeout: number }) => Go.EnvProviderSave(shared, id, new types.EnvProvider(body)),
    },
    workspaces: {
        list: () => call(() => Go.WorkspacesList()),
        create: (body: { name: string; icon?: string | null; color?: string | null }) =>
            call(() => Go.WorkspacesCreate(types.WorkspaceCreateInput.createFrom(body))),
        update: (
            id: number,
            body: { name?: string; icon?: string | null; color?: string | null },
        ) => call(() => Go.WorkspacesUpdate(id, types.WorkspaceUpdateInput.createFrom(body))),
        delete: (id: number) => call(() => Go.WorkspacesDelete(id)),
        /** Shows a save dialog and writes the workspace bundle to the chosen file. */
        exportToFile: (id: number, options: WorkspaceExportOptions) =>
            call(() => Go.WorkspaceExportToFile(id, options) as Promise<WorkspaceExportFileResult>),
        /** Shows an open dialog and reads a bundle from the chosen file. */
        openBundleFile: () =>
            call(() => Go.WorkspaceImportOpenFile() as Promise<WorkspaceBundleFileResult>),
        /** Creates a workspace from a bundle; with `dry_run` only previews it. */
        import: (bundle: WorkspaceBundle, options: WorkspaceImportOptions) =>
            call(
                () =>
                    Go.WorkspacesImport(
                        bundle as unknown as types.WorkspaceBundle,
                        options as unknown as types.WorkspaceImportOptions,
                    ) as Promise<WorkspaceImportResult>,
            ),
        reorder: (ids: number[]) => call(() => Go.WorkspacesReorder(ids)),
        setPinned: (id: number, pinned: boolean) => call(() => Go.WorkspacesSetPinned(id, pinned)),
    },

    apps: {
        list: (workspaceId: number) => call(() => Go.AppsList(workspaceId)),
        get: (id: number) => call(() => Go.AppsGet(id)),
        create: (workspaceId: number, body: { name: string; project_path: string }) =>
            call(() => Go.AppsCreate(workspaceId, body)),
        update: (
            id: number,
            body: {
                name?: string
                project_path?: string
                exclude_from_start_all?: boolean
            },
        ) =>
            call(
                () =>
                    Go.AppsUpdate(
                        id,
                        types.AppUpdateInput.createFrom(body),
                    ) as unknown as Promise<App>,
            ),
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
        getDetail: (id: number) =>
            call(() => Go.ConfigSetsGetDetail(id) as Promise<ConfigSetDetail>),
        create: (
            appId: number,
            body: {
                name: string
                copy_from_id?: number
                activate?: boolean
                parts?: CopyParts
            },
        ) => call(() => Go.ConfigSetsCreate(appId, types.ConfigSetCreateInput.createFrom(body))),
        update: (id: number, body: { name: string }) => call(() => Go.ConfigSetsUpdate(id, body)),
        delete: (id: number) => call(() => Go.ConfigSetsDelete(id)),
        activate: (id: number) => call(() => Go.ConfigSetsActivate(id)),
        compare: (leftId: number, rightId: number) =>
            call(() => Go.ConfigSetsCompare(leftId, rightId)),
        copyFrom: (id: number, sourceId: number, parts?: CopyParts) =>
            call(() => Go.ConfigSetsCopyFrom(id, sourceId, parts ?? {})),
    },

    /** Workspace shared config: sets a workspace owns that app config sets link to. */
    sharedSets: {
        list: (workspaceId: number) =>
            call(() => Go.SharedSetsList(workspaceId) as unknown as Promise<SharedSet[]>),
        get: (id: number) =>
            call(() => Go.SharedSetsGet(id) as unknown as Promise<SharedSetDetail>),
        create: (workspaceId: number, name: string) =>
            call(() => Go.SharedSetsCreate(workspaceId, name) as unknown as Promise<SharedSet>),
        rename: (id: number, name: string) =>
            call(() => Go.SharedSetsRename(id, name) as unknown as Promise<SharedSet>),
        delete: (id: number) => call(() => Go.SharedSetsDelete(id)),
        createVar: (
            setId: number,
            body: { key: string; value?: string; private?: boolean; options?: types.EnvVarOption[]; selected?: string },
        ) =>
            call(
                () =>
                    Go.SharedVarsCreate(
                        setId,
                        types.EnvVarCreateInput.createFrom(body),
                    ) as unknown as Promise<SharedEnvVar>,
            ),
        updateVar: (
            id: number,
            body: { key?: string; value?: string; private?: boolean; options?: types.EnvVarOption[]; selected?: string },
        ) =>
            call(
                () =>
                    Go.SharedVarsUpdate(
                        id,
                        types.EnvVarUpdateInput.createFrom(body),
                    ) as unknown as Promise<SharedEnvVar>,
            ),
        deleteVar: (id: number) => call(() => Go.SharedVarsDelete(id)),
        /** Links an app config set to a shared set, taking all of its variables. */
        link: (configSetId: number, body: SharedLinkInput) =>
            call(() => Go.ConfigSetsLinkShared(configSetId, types.SharedLinkInput.createFrom(body))),
        /** Adds shared variables that appeared since linking to every linked config set (runs no script). */
        refresh: () => call(() => Go.SharedLinksRefresh()),
        /** Stops sharing; with `keep` the shared variables stay as the app's own. */
        unlink: (configSetId: number, keep: boolean) =>
            call(() => Go.ConfigSetsUnlinkShared(configSetId, keep)),
    },

    envVars: {
        list: (appId: number) => call(() => Go.EnvVarsList(appId)),
        create: (appId: number, body: { key: string; value?: string; private?: boolean; options?: types.EnvVarOption[]; selected?: string }) =>
            call(() => Go.EnvVarsCreate(appId, types.EnvVarCreateInput.createFrom(body))),
        update: (id: number, body: { key?: string; value?: string; private?: boolean; options?: types.EnvVarOption[]; selected?: string }) =>
            call(() => Go.EnvVarsUpdate(id, types.EnvVarUpdateInput.createFrom(body))),
        delete: (id: number) => call(() => Go.EnvVarsDelete(id)),
        importEnv: (appId: number) => call(() => Go.EnvVarsImport(appId)),
    },

    templates: {
        list: (appId: number) => call(() => Go.TemplatesList(appId)),
        create: (
            appId: number,
            body: { file_path: string; content?: string; private?: boolean },
        ) => call(() => Go.TemplatesCreate(appId, types.TemplateCreateInput.createFrom(body))),
        update: (
            id: number,
            body: { file_path?: string; content?: string; private?: boolean },
        ) => call(() => Go.TemplatesUpdate(id, types.TemplateUpdateInput.createFrom(body))),
        delete: (id: number) => call(() => Go.TemplatesDelete(id)),
    },

    packageScripts: {
        list: (appId: number) => call(() => Go.PackageScriptsList(appId)),
    },

    /** The active config set's setup, dev, build or prod commands, by `kind`. */
    runConfig: {
        get: (appId: number, kind: CommandKind) =>
            call(() => Go.RunConfigGet(appId, kind) as Promise<RunConfig>),
        save: (
            appId: number,
            kind: CommandKind,
            body: {
                mode?: RunMode
                commands?: Array<{ label?: string | null; command: string }>
            },
        ) =>
            call(
                () =>
                    Go.RunConfigSave(
                        appId,
                        kind,
                        types.RunConfigSaveInput.createFrom(body),
                    ) as Promise<RunConfig>,
            ),
    },

    runner: {
        status: (appId: number) => call(() => Go.RunnerStatus(appId) as Promise<StatusEvent>),
        workspaceStatus: (workspaceId: number) =>
            call(() => Go.RunnerWorkspaceStatus(workspaceId) as Promise<StatusEvent[]>),
        output: (appId: number, commandId: number) => call(() => Go.RunnerOutput(appId, commandId)),
        resize: (appId: number, commandId: number, cols: number, rows: number) =>
            call(() => Go.RunnerResize(appId, commandId, cols, rows)),
        run: (appId: number, configSetId = 0) => call(() => Go.RunnerRun(appId, configSetId) as Promise<StatusEvent>),
        dev: (appId: number, configSetId = 0) => call(() => Go.RunnerDev(appId, configSetId) as Promise<StatusEvent>),
        prod: (appId: number, configSetId = 0) => call(() => Go.RunnerProd(appId, configSetId) as Promise<StatusEvent>),
        build: (appId: number, configSetId = 0) => call(() => Go.RunnerBuild(appId, configSetId) as Promise<StatusEvent>),
        setup: (appId: number, configSetId = 0) => call(() => Go.RunnerSetup(appId, configSetId) as Promise<StatusEvent>),
        stop: (appId: number) => call(() => Go.RunnerStop(appId) as Promise<StatusEvent>),
        reload: (appId: number) => call(() => Go.RunnerReload(appId) as Promise<StatusEvent>),
        startWorkspace: (workspaceId: number, excludeAppIds: number[], sequential: boolean) =>
            call(() => Go.RunnerStartWorkspace(workspaceId, excludeAppIds, sequential)),
        stopWorkspace: (workspaceId: number, excludeAppIds: number[]) =>
            call(() => Go.RunnerStopWorkspace(workspaceId, excludeAppIds)),
    },

    git: {
        info: (appId: number) => call(() => Go.GitInfo(appId)) as Promise<GitInfo>,
        fetchAll: (appId: number) => call(() => Go.GitFetchAll(appId)),
        /** Creates a local branch at `base` (empty: HEAD) without checking it out. */
        branchCreate: (appId: number, name: string, base: string) =>
            call(() => Go.GitBranchCreate(appId, name, base)),
        /** Switches the app's own folder to the local branch; refused while the app runs. */
        branchCheckout: (appId: number, name: string) =>
            call(() => Go.GitBranchCheckout(appId, name)),
        /** Pulls the branch's upstream in the worktree that has it checked out; conflicts come back in the result. */
        branchPull: (appId: number, name: string) =>
            call(() => Go.GitBranchPull(appId, name)) as Promise<GitPullResult>,
        /** Pushes the branch to its upstream, or publishes it to a remote when it has none; `force` uses --force-with-lease. */
        branchPush: (appId: number, name: string, force: boolean) =>
            call(() => Go.GitBranchPush(appId, name, force)),
        /** Opens a file inside one of the repo's worktrees (e.g. a conflicted one) in an editor. */
        fileOpenInEditor: (appId: number, file: string, editor: string) =>
            call(() => Go.GitFileOpenInEditor(appId, file, editor)),
        branchRename: (appId: number, oldName: string, newName: string) =>
            call(() => Go.GitBranchRename(appId, oldName, newName)),
        /** With `force`, also deletes a branch that is not fully merged. */
        branchDelete: (appId: number, name: string, force: boolean) =>
            call(() => Go.GitBranchDelete(appId, name, force)),
        /** With `link_shared`, also links the app's shared paths into the new worktree. */
        worktreeAdd: (appId: number, body: GitWorktreeAddInput) =>
            call(() => Go.GitWorktreeAdd(appId, body)) as Promise<GitWorktreeAddResult>,
        worktreeRemove: (appId: number, path: string, force: boolean) =>
            call(() => Go.GitWorktreeRemove(appId, path, force)),
        worktreePrune: (appId: number) => call(() => Go.GitWorktreePrune(appId)),
        /** Open the app's folder inside the worktree at `path`. */
        worktreeOpenFolder: (appId: number, path: string) =>
            call(() => Go.GitWorktreeOpenFolder(appId, path)),
        worktreeOpenInEditor: (appId: number, path: string, editor: string) =>
            call(() => Go.GitWorktreeOpenInEditor(appId, path, editor)),
        /** Runs the app's commands inside the worktree at `path`, with config set `configSetId` (0: the active one). */
        worktreeStart: (appId: number, path: string, kind: CommandKind, configSetId: number) =>
            call(() => Go.GitWorktreeStart(appId, path, kind, configSetId)) as Promise<StatusEvent>,
        /** The app's shared paths (relative to its folder) that worktrees link to. */
        linksGet: (appId: number) => call(() => Go.WorktreeLinksGet(appId)) as Promise<string[]>,
        linksSave: (appId: number, paths: string[]) =>
            call(() => Go.WorktreeLinksSave(appId, paths)) as Promise<string[]>,
        /** State of each shared path in the worktree at `path`; with `create`, links the missing ones first. */
        worktreeLinks: (appId: number, path: string, create: boolean) =>
            call(() => Go.GitWorktreeLinks(appId, path, create)) as Promise<WorktreeLinkState[]>,
    },

    ports: {
        list: () => call(() => Go.PortsList()),
        kill: (pid: number) => call(() => Go.PortsKill(pid)),
    },

    readyUrlPatterns: {
        list: () => call(() => Go.ReadyUrlPatternsList()),
        create: (body: { label: string; pattern: string; flags?: string }) =>
            call(() =>
                Go.ReadyUrlPatternsCreate(types.ReadyUrlPatternCreateInput.createFrom(body)),
            ),
        update: (id: number, body: { label?: string; pattern?: string; flags?: string }) =>
            call(() =>
                Go.ReadyUrlPatternsUpdate(id, types.ReadyUrlPatternUpdateInput.createFrom(body)),
            ),
        delete: (id: number) => call(() => Go.ReadyUrlPatternsDelete(id)),
    },

    settings: {
        get: () => call(() => Go.SettingsGet() as Promise<Record<string, string>>),
        set: (key: string, value: string) => call(() => Go.SettingsSet(key, value)),
    },

    fs: {
        validatePath: (path: string) => call(() => Go.FsValidatePath(path)),
        pickFolder: (opts?: { startDir?: string }) =>
            call(() => Go.FsPickFolder(types.FsPickFolderInput.createFrom(opts ?? {}))),
        pickFile: (opts?: { startDir?: string; appId?: number }) =>
            call(() => Go.FsPickFile(types.FsPickFileInput.createFrom(opts ?? {}))),
        pickAppFile: (appId: number) => call(() => Go.FsPickAppFile(appId)),
        readAppFile: (appId: number, path: string) => call(() => Go.FsReadAppFile(appId, path)),
    },

    openExternal: (url: string) => call(() => Go.OpenExternal(url)),
}

export { ApiError }
