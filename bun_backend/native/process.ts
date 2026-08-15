/**
 * Low-level helpers for spawning and stopping shell commands.
 */

import { run } from "./run"

export type SpawnedProcess = ReturnType<typeof Bun.spawn>

function isProcessGone(detail: string): boolean {
  const lower = detail.toLowerCase()
  return (
    lower.includes("not found") ||
    lower.includes("not running") ||
    lower.includes("no such process") ||
    lower.includes("no matching")
  )
}

/**
 * Recursively kill a Unix process and its descendants.
 * Avoids process-group kills (negative PIDs), which can take down App Runner itself
 * when the child shares the parent's group.
 */
async function killUnixTree(pid: number): Promise<void> {
  const children = await run(["pgrep", "-P", String(pid)])
  if (children.code === 0) {
    for (const line of children.stdout.split("\n")) {
      const childPid = Number(line.trim())
      if (Number.isInteger(childPid) && childPid > 0) {
        try {
          await killUnixTree(childPid)
        } catch {
          // child may have exited while we walked the tree
        }
      }
    }
  }
  const result = await run(["kill", "-9", String(pid)])
  if (result.code !== 0) {
    const detail = result.stderr.trim() || result.stdout.trim()
    if (!isProcessGone(detail)) {
      throw new Error(detail || `kill failed for pid ${pid}`)
    }
  }
}

/**
 * Force-kill an OS process by PID (and its descendants).
 * Used by app stop and the ports dialog.
 */
export async function killPid(pid: number): Promise<void> {
  if (!Number.isInteger(pid) || pid <= 0) {
    throw new Error("Invalid pid")
  }
  if (pid === process.pid) {
    throw new Error("Refusing to kill the Workspace Manager process")
  }

  if (process.platform === "win32") {
    const result = await run(["taskkill", "/PID", String(pid), "/T", "/F"])
    if (result.code !== 0) {
      const detail = result.stderr.trim() || result.stdout.trim()
      if (isProcessGone(detail)) return
      throw new Error(detail || `taskkill failed for pid ${pid}`)
    }
    return
  }

  await killUnixTree(pid)
}

/**
 * Best-effort kill of a spawned shell and its process tree.
 * Plain `child.kill()` only terminates cmd/sh, leaving npm/node/etc. running.
 */
export async function killProcess(child: SpawnedProcess) {
  const pid = child.pid
  if (typeof pid === "number" && pid > 0) {
    try {
      await killPid(pid)
      return
    } catch {
      // fall through — process may already be exiting
    }
  }
  try {
    child.kill()
  } catch {
    // already dead
  }
}

/**
 * Build a child env from the host process env plus app overrides.
 * Sets PYTHONUNBUFFERED and a default FORCE_COLOR for CLI friendliness.
 */
export function mergeSpawnEnv(
  appEnv: Record<string, string>
): Record<string, string> {
  const env: Record<string, string> = {}
  for (const [key, value] of Object.entries(process.env)) {
    if (typeof value === "string") env[key] = value
  }
  Object.assign(env, appEnv)
  env.PYTHONUNBUFFERED = "1"
  env.FORCE_COLOR = env.FORCE_COLOR ?? "0"
  env.NO_COLOR = env.NO_COLOR ?? "1"
  return env
}

export type SpawnShellOptions = {
  command: string
  cwd: string
  env: Record<string, string>
}

/**
 * Spawn a user/shell command via `cmd /c` (Windows) or `sh -c` (Unix).
 * stdout/stderr are piped; stdin is ignored.
 */
export function spawnShell(options: SpawnShellOptions): SpawnedProcess {
  const isWin = process.platform === "win32"
  return Bun.spawn({
    cmd: isWin
      ? ["cmd", "/c", options.command]
      : ["sh", "-c", options.command],
    cwd: options.cwd,
    env: options.env,
    stdout: "pipe",
    stderr: "pipe",
    stdin: "ignore",
  })
}
