import { appsRepo } from "../db/apps"
import { envVarsRepo } from "../db/env-vars"
import { runConfigsRepo } from "../db/run-configs"
import type { RunCommand, RunMode } from "../db/types"
import { applyTemplates, restoreTemplates } from "./templates"

export type ProcessStatus = "pending" | "running" | "exited" | "killed" | "error"

export type ProcessState = {
  commandId: number
  label: string
  command: string
  status: ProcessStatus
  exitCode: number | null
  pid: number | null
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
  children: Map<number, ReturnType<typeof Bun.spawn>>
  running: boolean
  abortSequential: boolean
  restored: boolean
  logBuffer: LogEvent[]
}

type RunnerStore = {
  sessions: Map<number, Session>
  listeners: Map<number, Set<(event: RunnerEvent) => void>>
}

const globalKey = "__appRunnerStore" as const

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

async function pipeStream(
  session: Session,
  commandId: number,
  stream: ReadableStream<Uint8Array> | null,
  kind: "stdout" | "stderr"
) {
  if (!stream) return
  const reader = stream.getReader()
  const decoder = new TextDecoder()
  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      const text = decoder.decode(value, { stream: true })
      if (!text) continue
      emit(session, {
        type: "log",
        commandId,
        stream: kind,
        text,
        ts: Date.now(),
      })
    }
    const rest = decoder.decode()
    if (rest) {
      emit(session, {
        type: "log",
        commandId,
        stream: kind,
        text: rest,
        ts: Date.now(),
      })
    }
  } catch {
    // process killed / stream closed
  }
}

function killChild(child: ReturnType<typeof Bun.spawn>) {
  try {
    child.kill()
  } catch {
    // already dead
  }
}

function spawnEnv(appEnv: Record<string, string>): Record<string, string> {
  const env: Record<string, string> = {}
  for (const [key, value] of Object.entries(process.env)) {
    if (typeof value === "string") env[key] = value
  }
  Object.assign(env, appEnv)
  // Reduce stdout buffering for many CLIs
  env.PYTHONUNBUFFERED = "1"
  env.FORCE_COLOR = env.FORCE_COLOR ?? "0"
  return env
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

  const isWin = process.platform === "win32"
  let child: ReturnType<typeof Bun.spawn>
  try {
    child = Bun.spawn({
      cmd: isWin ? ["cmd", "/c", cmd.command] : ["sh", "-c", cmd.command],
      cwd,
      env,
      stdout: "pipe",
      stderr: "pipe",
      stdin: "ignore",
    })
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

  const env = spawnEnv(envVarsRepo.toRecord(session.appId))
  const config = runConfigsRepo.getByApp(session.appId)
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
  const config = runConfigsRepo.getOrCreate(appId)
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

  subscribe(appId: number, listener: (event: RunnerEvent) => void): () => void {
    let set = globalListeners.get(appId)
    if (!set) {
      set = new Set()
      globalListeners.set(appId, set)
    }
    set.add(listener)

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
    for (const processState of session.processes) {
      if (processState.status === "running") {
        processState.status = "killed"
      }
    }
    for (const child of session.children.values()) {
      killChild(child)
    }
    session.children.clear()
    session.running = false

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
