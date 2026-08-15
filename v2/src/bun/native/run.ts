/**
 * Shared helper for capturing stdout/stderr from a short-lived OS command.
 */

export type RunResult = {
  stdout: string
  stderr: string
  code: number
}

/** Spawn `cmd` and wait for exit, returning captured output. */
export async function run(
  cmd: string[],
  opts?: { rejectOnError?: boolean }
): Promise<RunResult> {
  const proc = Bun.spawn(cmd, { stdout: "pipe", stderr: "pipe" })
  const [stdout, stderr, code] = await Promise.all([
    new Response(proc.stdout).text(),
    new Response(proc.stderr).text(),
    proc.exited,
  ])
  if (opts?.rejectOnError && code !== 0) {
    throw new Error(
      `${cmd.join(" ")} exited ${code}${stderr.trim() ? `: ${stderr.trim()}` : ""}`
    )
  }
  return { stdout, stderr, code }
}
