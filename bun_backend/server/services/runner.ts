import {
  killProcess,
  mergeSpawnEnv,
  spawnShell,
  type SpawnedProcess,
} from "../../native/process"
import { appsRepo } from "@db/apps"
import { configSetsRepo } from "@db/config-sets"
import { envVarsRepo } from "@db/env-vars"
import { runConfigsRepo } from "@db/run-configs"
import type { RunCommand, RunMode } from "@db/types"
import { matchReadyUrl } from "./ready-url"
import { applyTemplates, restoreTemplates } from "./templates"

export type ProcessStatus = "pending" | "running" | "exited" | "killed" | "error"

export type ProcessState = {
  commandId: number
  label: string
  command: string
  status: ProcessStatus
  exitCode: number | null
  pid: number | null
  /** URLs detected from process logs (Vite, Spring Boot, etc.) */
  urls: string[]
}

export type LogEvent = {
  type: "log"
  commandId: number
  stream: "stdout" | "stderr" | "system"
  text: string
  ts: number
}

export type StatusEvent = {
  type: "status"
  sessionId: string
  appId: number
  running: boolean
  processes: ProcessState[]
  error?: string
  ts: number
}

export type RunnerEvent = LogEvent | StatusEvent

type Session = {
  id: string
  appId: number
  mode: RunMode
  processes: ProcessState[]
  children: Map<number, SpawnedProcess>
  running: boolean
  abortSequential: boolean
  restored: boolean
  logBuffer: LogEvent[]
}

type RunnerStore = {
  sessions: Map<number, Session>
  listeners: Map<number, Set<(event: RunnerEvent) => void>>
}

const globalKey = "__workspaceManagerStore" as const

function getStore(): RunnerStore {
  const g = globalThis as typeof globalThis & {
    [globalKey]?: RunnerStore
  }
  if (!g[globalKey]) {
    g[globalKey] = {
      sessions: new Map(),
      listeners: new Map(),
    }
  }
  return g[globalKey]
}

const store = getStore()
const sessions = store.sessions
const globalListeners = store.listeners

const MAX_LOG_BUFFER = 5000

function emit(session: Session, event: RunnerEvent) {
  if (event.type === "log") {
    session.logBuffer.push(event)
    if (session.logBuffer.length > MAX_LOG_BUFFER) {
      session.logBuffer.splice(0, session.logBuffer.length - MAX_LOG_BUFFER)
    }
  }

  const globals = globalListeners.get(session.appId)
  if (!globals) return
  for (const listener of globals) {
    try {
      listener(event)
    } catch {
      // ignore broken subscribers
    }
  }
}

function statusEvent(session: Session, error?: string): StatusEvent {
  return {
    type: "status",
    sessionId: session.id,
    appId: session.appId,
    running: session.running,
    processes: session.processes.map((p) => ({ ...p })),
    error,
    ts: Date.now(),
  }
}

function systemLog(session: Session, commandId: number, text: string) {
  emit(session, {
    type: "log",
    commandId,
    stream: "system",
    text: text.endsWith("\n") ? text : `${text}\n`,
    ts: Date.now(),
  })
}

function noteReadyUrl(session: Session, commandId: number, line: string) {
  if (!session.running) return
  const match = matchReadyUrl(line)
  if (!match) return
  const processState = session.processes.find((p) => p.commandId === commandId)
  if (!processState) return
  if (processState.urls.includes(match.url)) return
  processState.urls.push(match.url)
  systemLog(
    session,
    commandId,
    `Detected URL (${match.label}): ${match.url}`
  )
  emit(session, statusEvent(session))
}

function clearReadyUrls(session: Session) {
  for (const processState of session.processes) {
    processState.urls = []
  }
}

function emitLogLine(
  session: Session,
  commandId: number,
  kind: "stdout" | "stderr",
  text: string
) {
  emit(session, {
    type: "log",
    commandId,
    stream: kind,
    text: text.endsWith("\n") ? text : `${text}\n`,
    ts: Date.now(),
  })
  noteReadyUrl(session, commandId, text)
}

async function pipeStream(
  session: Session,
  commandId: number,
  stream: ReadableStream<Uint8Array> | null,
  kind: "stdout" | "stderr"
) {
  if (!stream) return
  const reader = stream.getReader()
  const decoder = new TextDecoder()
  // Keep the raw buffer so partial ESC sequences are never stripped mid-chunk.
  let pending = ""
  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      const chunk = decoder.decode(value, { stream: true })
      if (!chunk) continue
      pending += chunk
      const parts = pending.split(/\r?\n/)
      pending = parts.pop() ?? ""
      for (const line of parts) {
        emitLogLine(session, commandId, kind, Bun.stripANSI(line))
      }
    }
    pending += decoder.decode()
    const rest = Bun.stripANSI(pending)
    if (rest) {
      emitLogLine(session, commandId, kind, rest)
    }
  } catch {
    // process killed / stream closed
  }
}

async function spawnCommand(
  session: Session,
  cmd: RunCommand,
  cwd: string,
  env: Record<string, string>
): Promise<number> {
  const processState = session.processes.find((p) => p.commandId === cmd.id)!
  processState.status = "running"
  emit(session, statusEvent(session))
  systemLog(session, cmd.id, `$ ${cmd.command}`)

  let child: SpawnedProcess
  try {
    child = spawnShell({ command: cmd.command, cwd, env })
  } catch (err) {
    processState.status = "error"
    processState.exitCode = 1
    systemLog(
      session,
      cmd.id,
      `Failed to start: ${err instanceof Error ? err.message : String(err)}`
    )
    emit(session, statusEvent(session))
    return 1
  }

  processState.pid = child.pid
  session.children.set(cmd.id, child)
  emit(session, statusEvent(session))

  const pipes = Promise.all([
    pipeStream(session, cmd.id, child.stdout, "stdout"),
    pipeStream(session, cmd.id, child.stderr, "stderr"),
  ])

  const exitCode = await child.exited
  await pipes
  session.children.delete(cmd.id)

  if (processState.status === "killed") {
    emit(session, statusEvent(session))
    return exitCode
  }

  processState.status = exitCode === 0 ? "exited" : "error"
  processState.exitCode = exitCode
  systemLog(
    session,
    cmd.id,
    exitCode === 0
      ? `Process exited with code ${exitCode}`
      : `Process failed with code ${exitCode}`
  )
  emit(session, statusEvent(session))
  return exitCode
}

async function runSession(session: Session) {
  const app = appsRepo.get(session.appId)
  if (!app) {
    session.running = false
    emit(session, statusEvent(session, "App not found"))
    return
  }

  try {
    applyTemplates(session.appId, session.id)
    systemLog(
      session,
      session.processes[0]?.commandId ?? 0,
      "Templates applied"
    )
  } catch (err) {
    session.running = false
    const message = err instanceof Error ? err.message : String(err)
    systemLog(
      session,
      session.processes[0]?.commandId ?? 0,
      `Template apply failed: ${message}`
    )
    emit(session, statusEvent(session, `Template apply failed: ${message}`))
    return
  }

  const set = configSetsRepo.resolveActive(session.appId)
  const env = mergeSpawnEnv(envVarsRepo.toRecord(set.id))
  const config = runConfigsRepo.getByConfigSet(set.id)
  const commands = config?.commands ?? []

  try {
    if (session.mode === "parallel") {
      await Promise.all(
        commands.map((cmd) => spawnCommand(session, cmd, app.project_path, env))
      )
    } else {
      for (const cmd of commands) {
        if (session.abortSequential) break
        const code = await spawnCommand(session, cmd, app.project_path, env)
        if (code !== 0) {
          systemLog(
            session,
            cmd.id,
            "Sequential run stopped due to non-zero exit"
          )
          break
        }
      }
    }
  } finally {
    session.running = false
    clearReadyUrls(session)
    if (!session.restored) {
      restoreTemplates(session.appId, session.id)
      session.restored = true
      systemLog(
        session,
        session.processes[0]?.commandId ?? 0,
        "Original files restored"
      )
    }
    emit(session, statusEvent(session))
  }
}

function createSession(appId: number): Session {
  const set = configSetsRepo.resolveActive(appId)
  const config = runConfigsRepo.getOrCreate(set.id)
  if (config.commands.length === 0) {
    throw new Error("No run commands configured")
  }

  return {
    id: `${appId}-${Date.now()}`,
    appId,
    mode: config.mode,
    processes: config.commands.map((cmd) => ({
      commandId: cmd.id,
      label: cmd.label || cmd.command,
      command: cmd.command,
      status: "pending",
      exitCode: null,
      pid: null,
      urls: [],
    })),
    children: new Map(),
    running: true,
    abortSequential: false,
    restored: false,
    logBuffer: [],
  }
}

export const runner = {
  getStatus(appId: number): StatusEvent | null {
    const session = sessions.get(appId)
    if (!session) return null
    return statusEvent(session)
  },

  getSnapshot(appId: number): {
    status: StatusEvent
    logs: LogEvent[]
  } {
    const session = sessions.get(appId)
    if (!session) {
      return {
        status: {
          type: "status",
          sessionId: "",
          appId,
          running: false,
          processes: [],
          ts: Date.now(),
        },
        logs: [],
      }
    }
    return {
      status: statusEvent(session),
      logs: [...session.logBuffer],
    }
  },

  subscribe(
    appId: number,
    listener: (event: RunnerEvent) => void,
    opts?: { replay?: boolean }
  ): () => void {
    let set = globalListeners.get(appId)
    if (!set) {
      set = new Set()
      globalListeners.set(appId, set)
    }
    set.add(listener)

    if (opts?.replay !== false) {
      const session = sessions.get(appId)
      if (session) {
        listener(statusEvent(session))
        for (const log of session.logBuffer) {
          listener(log)
        }
      } else {
        listener({
          type: "status",
          sessionId: "",
          appId,
          running: false,
          processes: [],
          ts: Date.now(),
        })
      }
    }

    return () => {
      set?.delete(listener)
      if (set && set.size === 0) globalListeners.delete(appId)
    }
  },

  async start(appId: number): Promise<StatusEvent> {
    if (!appsRepo.get(appId)) throw new Error("App not found")

    const existing = sessions.get(appId)
    if (existing?.running) {
      throw new Error("App is already running")
    }

    const session = createSession(appId)
    sessions.set(appId, session)
    emit(session, statusEvent(session))
    void runSession(session)
    return statusEvent(session)
  },

  async stop(appId: number): Promise<StatusEvent> {
    const session = sessions.get(appId)
    if (!session) throw new Error("No active session")

    session.abortSequential = true
    session.running = false
    clearReadyUrls(session)
    for (const processState of session.processes) {
      if (processState.status === "running") {
        processState.status = "killed"
      }
    }
    const children = [...session.children.values()]
    session.children.clear()
    // Emit cleared URLs immediately so UI updates before kill finishes
    emit(session, statusEvent(session))
    await Promise.all(children.map((child) => killProcess(child)))

    if (!session.restored) {
      restoreTemplates(appId, session.id)
      session.restored = true
      systemLog(
        session,
        session.processes[0]?.commandId ?? 0,
        "Stopped — original files restored"
      )
    }
    emit(session, statusEvent(session))
    return statusEvent(session)
  },

  async reload(appId: number): Promise<StatusEvent> {
    const existing = sessions.get(appId)
    if (existing?.running) {
      await this.stop(appId)
    }
    return this.start(appId)
  },
}
