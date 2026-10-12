/** True for a session that ends by itself (build, setup) rather than a run. */
export function isTaskKind(kind: string | undefined) {
  return kind === "build" || kind === "setup"
}

/** The state shown for an app: its session kind while it is going, else idle. */
export function appStateLabel(running: boolean, kind?: string) {
  if (!running) return "Idle"
  if (kind === "build") return "Building"
  if (kind === "setup") return "Setting up"
  return "Running"
}
