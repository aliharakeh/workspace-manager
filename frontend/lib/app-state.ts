/** The state shown for an app: its session kind while it is going, else idle. */
export function appStateLabel(running: boolean, building: boolean) {
  if (!running) return "Idle"
  return building ? "Building" : "Running"
}
