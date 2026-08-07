export type Workspace = {
  id: number
  name: string
  icon: string | null
  created_at: string
  updated_at: string
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

export type EnvVar = {
  id: number
  config_set_id: number
  key: string
  value: string
  created_at: string
  updated_at: string
}

export type Template = {
  id: number
  config_set_id: number
  file_path: string
  content: string
  created_at: string
  updated_at: string
}

export type RunMode = "sequential" | "parallel"

export type RunCommand = {
  id: number
  run_config_id: number
  label: string | null
  command: string
  sort_order: number
  created_at: string
  updated_at: string
}

export type RunConfig = {
  id: number
  config_set_id: number
  mode: RunMode
  created_at: string
  updated_at: string
  commands: RunCommand[]
}

export type ProcessStatus =
  | "pending"
  | "running"
  | "exited"
  | "killed"
  | "error"

export type ProcessState = {
  commandId: number
  label: string
  command: string
  status: ProcessStatus
  exitCode: number | null
  pid: number | null
}

export type StatusEvent = {
  type?: "status"
  sessionId?: string
  appId: number
  running: boolean
  processes: ProcessState[]
  error?: string
  ts?: number
}

export type LogEvent = {
  type: "log"
  commandId: number
  stream: "stdout" | "stderr" | "system"
  text: string
  ts: number
}

/** OS process holding a listening TCP user port (1024–49151). */
export type ListeningProcess = {
  port: number
  pid: number
  name: string
}

export type RunnerEvent = LogEvent | (StatusEvent & { type: "status" })
