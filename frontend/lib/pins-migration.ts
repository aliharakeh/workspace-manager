import { api } from "@/host"

// Pins used to live in this browser profile's local storage. They now live in
// the daemon, so the GUI and the VS Code extension agree. Once, on startup, a
// leftover local list is pushed up when the daemon has no pins yet, and the
// local copy is deleted either way so it is never read again.
const LEGACY_PINNED_KEY = "workspace-manager.sidebar.pinned"

/** Returns true when pins were pushed to the daemon (the caller reloads). */
export async function migrateLocalPins(
  workspaces: { id: number; pinned: boolean }[]
): Promise<boolean> {
  let raw: string | null
  try {
    raw = localStorage.getItem(LEGACY_PINNED_KEY)
  } catch {
    return false
  }
  if (raw === null) return false
  let ids: number[] = []
  try {
    const parsed: unknown = JSON.parse(raw)
    if (Array.isArray(parsed)) ids = parsed.filter(Number.isInteger)
  } catch {
    // Unreadable: treat as none.
  }
  const known = new Set(workspaces.map((w) => w.id))
  const toPin = ids.filter((id) => known.has(id))
  let pushed = false
  if (toPin.length > 0 && !workspaces.some((w) => w.pinned)) {
    try {
      await Promise.all(toPin.map((id) => api.workspaces.setPinned(id, true)))
      pushed = true
    } catch {
      return false // keep the local copy and try again next start
    }
  }
  try {
    localStorage.removeItem(LEGACY_PINNED_KEY)
  } catch {
    // Nothing more to do.
  }
  return pushed
}
