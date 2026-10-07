export type Workspace = {
    id: number
    name: string
    icon: string | null
    color: string | null
    created_at: string
    updated_at: string
}

export type Editor = {
    id: string
    name: string
}

export type App = {
    id: number
    workspace_id: number
    name: string
    project_path: string
    active_config_set_id: number | null
    active_config_set_name: string | null
    created_at: string
    updated_at: string
}

export type ConfigSet = {
    id: number
    app_id: number
    name: string
    created_at: string
    updated_at: string
}

export type ConfigSetDetail = ConfigSet & {
    env_vars: EnvVar[]
    templates: Template[]
    run_config: RunConfig | null
    build_config: RunConfig | null
    setup_config: RunConfig | null
}

/**
 * What to copy from a source config set.
 * `true` copies everything, `false` skips, an array copies only the listed
 * items: env var keys, template file paths, or source run/build/setup command ids.
 */
export type CopyParts = {
    env?: boolean | string[]
    templates?: boolean | string[]
    run?: boolean | number[]
    build?: boolean | number[]
    setup?: boolean | number[]
}

export type EnvVar = {
    id: number
    config_set_id: number
    key: string
    value: string
    include_in_ai: boolean
    created_at: string
    updated_at: string
}

export type Template = {
    id: number
    config_set_id: number
    file_path: string
    content: string
    include_in_ai: boolean
    created_at: string
    updated_at: string
}

export type RunMode = 'sequential' | 'parallel'

/**
 * The command list a config set holds one of: `run` starts the app, `build`
 * builds it, `setup` prepares it (install dependencies and the like).
 */
export type CommandKind = 'run' | 'build' | 'setup'

export type RunCommand = {
    id: number
    run_config_id: number
    label: string | null
    command: string
    sort_order: number
    created_at: string
    updated_at: string
}

/** The command list of one kind of a config set; commands carry `run_config_id` for both. */
export type RunConfig = {
    id: number
    config_set_id: number
    kind: CommandKind
    mode: RunMode
    created_at: string
    updated_at: string
    commands: RunCommand[]
}

export type PackageScript = {
    name: string
    /** Raw script value from package.json. */
    script: string
    /** Full command line for the detected package manager, e.g. "bun run dev". */
    command: string
}

export type PackageScripts = {
    has_package_json: boolean
    package_manager: string
    scripts: PackageScript[]
}

export type ProcessStatus = 'pending' | 'running' | 'exited' | 'killed' | 'error'

export type ProcessState = {
    commandId: number
    label: string
    command: string
    status: ProcessStatus
    exitCode: number | null
    pid: number | null
    /** URLs detected from process logs (Vite, Spring Boot, etc.) */
    urls?: string[]
}

export type StatusEvent = {
    type?: 'status'
    sessionId?: string
    appId: number
    /** What the session executes; empty before the app was ever started. */
    kind?: CommandKind | ''
    /** Root of the git worktree the session runs in; absent for the app's own folder. */
    worktree?: string
    /** Config set the session runs with. */
    configSetId?: number
    running: boolean
    processes: ProcessState[]
    error?: string
    ts?: number
}

/** A chunk of terminal output of one run command. `data` is base64 of the raw
 * bytes (escape sequences included); `offset` is how many bytes the command had
 * produced before this chunk. */
export type LogEvent = {
    type: 'log'
    appId?: number
    commandId: number
    offset: number
    data: string
    ts: number
}

/** Recent terminal output of one run command (base64), up to `end` bytes. */
export type RunnerOutput = {
    sessionId: string
    data: string
    end: number
}

/** OS process holding a listening TCP user port (1024–49151). */
export type ListeningProcess = {
    port: number
    pid: number
    name: string
}

/** One entry of `git worktree list`. */
export type GitWorktree = {
    path: string
    head: string
    branch: string
    detached: boolean
    bare: boolean
    locked: boolean
    prunable: boolean
    /** The repository's main worktree (cannot be removed). */
    main: boolean
    /** The worktree the app's project path is in (cannot be removed). */
    current: boolean
}

export type GitInfo = {
    is_repo: boolean
    worktrees: GitWorktree[]
    /** Local branches, then remote-tracking ones (e.g. "origin/main"). */
    branches: string[]
}

/** Path is optional (default: sibling "<repo>-<branch>"); relative paths are from the project path. */
export type GitWorktreeAddInput = {
    path: string
    branch: string
    new_branch: boolean
    base: string
}

/** Regex used to detect ready URLs from process logs. */
export type ReadyUrlPattern = {
    id: number
    /** Stable id for built-in defaults; null for user-created patterns. */
    key: string | null
    label: string
    pattern: string
    flags: string
    sort_order: number
    created_at: string
    updated_at: string
}

export type RunnerEvent = LogEvent | (StatusEvent & { type: 'status' })

/** One saved AI connection (save payload). Each connection has a user-chosen
 * name (its key) plus the provider it talks to, so several connections can
 * share one provider. The API key is write-only: the UI never reads it back.
 * Empty apiKey keeps the stored key; clearApiKey removes it. */
export type AIProviderConfig = {
    name?: string
    provider: string
    baseURL?: string
    apiKey?: string
    model?: string
    temperature?: number
    opencodeSession?: boolean
    clearApiKey?: boolean
}

/** A saved connection as the UI sees it: no secret, plus hasApiKey. */
export type AIConnectionInfo = {
    name: string
    provider: string
    baseURL?: string
    model?: string
    hasApiKey: boolean
    temperature?: number
    opencodeSession?: boolean
}

/** All saved connections plus which name is the active default for chat. */
export type AIConfigInfo = {
    providers: AIConnectionInfo[]
    active: string
}

export type BlueprintCommand = {
    label?: string | null
    command: string
}

export type Blueprint = {
    id: number
    name: string
    description: string
    /** Example app name the new-app dialog starts with. */
    sample_name: string
    create_folder: boolean
    commands: BlueprintCommand[]
    created_at: string
    updated_at: string
}

export type BlueprintInput = {
    name: string
    description: string
    sample_name: string
    create_folder: boolean
    commands: BlueprintCommand[]
}

export type BlueprintAIInput = {
    instruction: string
    /** What the editor holds now, so the AI changes it instead of starting over. */
    draft: BlueprintInput | null
}

export type BlueprintAIResult = {
    blueprint: BlueprintInput
    /** The AI's short notes: assumptions and tools that must be installed. */
    message: string
}

export type BlueprintRunInput = {
    run_id: string
    blueprint_id: number
    /** Workspace the new app is added to. */
    workspace_id: number
    name: string
    parent_path: string
    folder_name: string
    create_folder: boolean
    /** Pause on a failed command and wait for `blueprints.resolve`. */
    ask_on_error: boolean
}

export type BlueprintRunResult = {
    app: App
    warning?: string
}

export type BlueprintLogEvent = {
    runId: string
    /** `data` is terminal output (base64 bytes in `data`); `failed` is a command
     * failure (message in `text`) waiting for a skip/abort answer. */
    stream: 'data' | 'failed'
    text: string
    data: string
    ts: number
}
