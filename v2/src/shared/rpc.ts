import type {
  App,
  ConfigSet,
  ConfigSetDetail,
  CopyParts,
  EnvVar,
  ListeningProcess,
  ReadyUrlPattern,
  RunConfig,
  RunMode,
  RunnerEvent,
  StatusEvent,
  Template,
  Workspace,
} from "./types"

type Ok = { ok: true }

export type PickFolderResult =
  | { cancelled: true }
  | { cancelled: false; path: string }

export type PickFileResult =
  | { cancelled: true }
  | {
      cancelled: false
      path: string
      relative_path?: string
      content: string
    }

export type ImportEnvResult =
  | { cancelled: true }
  | {
      cancelled: false
      path: string
      format: string
      imported: number
      vars: EnvVar[]
      template: { id: number; file_path: string; created: boolean } | null
    }

export type ValidatePathResult =
  | { ok: true; path: string }
  | { ok: false; error: string }

export type ReadAppFileResult = {
  ok: true
  content: string
  relative_path: string
}

export type RunnerLogsSnapshot = {
  status: StatusEvent
  logs: Extract<RunnerEvent, { type: "log" }>[]
}

export type AppRPC = {
  bun: {
    requests: {
      workspacesList: { params: Record<string, never>; response: Workspace[] }
      workspacesCreate: {
        params: { name: string; icon?: string | null }
        response: Workspace
      }
      workspacesUpdate: {
        params: { id: number; name?: string; icon?: string | null }
        response: Workspace
      }
      workspacesDelete: { params: { id: number }; response: Ok }

      appsList: { params: { workspaceId: number }; response: App[] }
      appsGet: { params: { id: number }; response: App }
      appsCreate: {
        params: { workspaceId: number; name: string; project_path: string }
        response: App
      }
      appsUpdate: {
        params: { id: number; name?: string; project_path?: string }
        response: App
      }
      appsDelete: { params: { id: number }; response: Ok }
      appsOpenInEditor: { params: { id: number }; response: Ok }

      configSetsList: { params: { appId: number }; response: ConfigSet[] }
      configSetsGetDetail: {
        params: { id: number }
        response: ConfigSetDetail
      }
      configSetsCreate: {
        params: {
          appId: number
          name: string
          copy_from_id?: number
          activate?: boolean
          parts?: CopyParts
        }
        response: ConfigSet
      }
      configSetsUpdate: {
        params: { id: number; name: string }
        response: ConfigSet
      }
      configSetsDelete: { params: { id: number }; response: Ok }
      configSetsActivate: {
        params: { id: number }
        response: { id: number; app_id: number; name: string; app: App }
      }
      configSetsCopyFrom: {
        params: { id: number; sourceId: number; parts?: CopyParts }
        response: ConfigSet
      }

      envVarsList: { params: { appId: number }; response: EnvVar[] }
      envVarsCreate: {
        params: { appId: number; key: string; value?: string }
        response: EnvVar
      }
      envVarsUpdate: {
        params: { id: number; key?: string; value?: string }
        response: EnvVar
      }
      envVarsDelete: { params: { id: number }; response: Ok }
      envVarsImport: { params: { appId: number }; response: ImportEnvResult }

      templatesList: { params: { appId: number }; response: Template[] }
      templatesCreate: {
        params: { appId: number; file_path: string; content?: string }
        response: Template
      }
      templatesUpdate: {
        params: { id: number; file_path?: string; content?: string }
        response: Template
      }
      templatesDelete: { params: { id: number }; response: Ok }

      runConfigGet: { params: { appId: number }; response: RunConfig }
      runConfigSave: {
        params: {
          appId: number
          mode?: RunMode
          commands?: Array<{ label?: string | null; command: string }>
        }
        response: RunConfig
      }

      runnerStatus: { params: { appId: number }; response: StatusEvent }
      runnerWorkspaceStatus: {
        params: { workspaceId: number }
        response: StatusEvent[]
      }
      runnerLogs: { params: { appId: number }; response: RunnerLogsSnapshot }
      runnerRun: { params: { appId: number }; response: StatusEvent }
      runnerStop: { params: { appId: number }; response: StatusEvent }
      runnerReload: { params: { appId: number }; response: StatusEvent }

      portsList: {
        params: Record<string, never>
        response: {
          min: number
          max: number
          processes: ListeningProcess[]
        }
      }
      portsKill: { params: { pid: number }; response: { ok: true; pid: number } }

      readyUrlPatternsList: {
        params: Record<string, never>
        response: ReadyUrlPattern[]
      }
      readyUrlPatternsCreate: {
        params: { label: string; pattern: string; flags?: string }
        response: ReadyUrlPattern
      }
      readyUrlPatternsUpdate: {
        params: {
          id: number
          label?: string
          pattern?: string
          flags?: string
        }
        response: ReadyUrlPattern
      }
      readyUrlPatternsDelete: { params: { id: number }; response: Ok }

      settingsGet: {
        params: Record<string, never>
        response: Record<string, string>
      }
      settingsSet: {
        params: { key: string; value: string }
        response: Record<string, string>
      }

      fsValidatePath: {
        params: { path: string }
        response: ValidatePathResult
      }
      fsPickFolder: {
        params: { startDir?: string }
        response: PickFolderResult
      }
      fsPickFile: {
        params: { startDir?: string; appId?: number }
        response: PickFileResult
      }
      fsPickAppFile: { params: { appId: number }; response: PickFileResult }
      fsReadAppFile: {
        params: { appId: number; path: string }
        response: ReadAppFileResult
      }

      openExternal: { params: { url: string }; response: Ok }
    }
    messages: Record<string, never>
  }
  webview: {
    requests: Record<string, never>
    messages: {
      runnerEvent: { appId: number; event: RunnerEvent }
    }
  }
}
