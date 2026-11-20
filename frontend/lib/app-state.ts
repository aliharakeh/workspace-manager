/** True for a session that ends by itself (build, setup) rather than a run. */
export function isTaskKind(kind: string | undefined) {
  return kind === "build" || kind === "setup"
}

/** The state shown for an app: its session kind while it is going, else idle. */
export function appStateLabel(running: boolean, kind?: string) {
  if (!running) return "Idle"
  if (kind === "build") return "Building"
  if (kind === "setup") return "Setting up"
  if (kind === "dev") return "Running dev"
  if (kind === "prod") return "Running prod"
  return "Running"
}

/**
 * The visual state of an app: running (a run), task (build/setup going),
 * failed (stopped with an error from its last session) or idle.
 */
export type AppVisualState = "running" | "task" | "failed" | "idle"

export function appVisualState(
  running: boolean,
  kind?: string,
  error?: string
): AppVisualState {
  if (running) return isTaskKind(kind) ? "task" : "running"
  return error ? "failed" : "idle"
}
