/**
 * Low-level helpers for spawning and stopping shell commands.
 */

export type SpawnedProcess = ReturnType<typeof Bun.spawn>

/** Best-effort kill; ignores already-exited processes. */
export function killProcess(child: SpawnedProcess) {
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
