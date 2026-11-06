import { api } from "@/lib/api"
import type { WorktreeLinkState } from "@/lib/types"
import { toast } from "sonner"

const LINK_STATE_LABELS: Record<WorktreeLinkState["state"], string> = {
  linked: "linked",
  missing: "not linked",
  exists: "skipped, real copy present",
  other_link: "skipped, links elsewhere",
  no_source: "skipped, not in the app folder",
  error: "failed",
}

/** One line per shared path, for the "Last git output" box. */
function formatLinkStates(states: WorktreeLinkState[]): string {
  return states
    .map(
      (s) =>
        `${s.path}: ${LINK_STATE_LABELS[s.state]}${s.message ? ` (${s.message})` : ""}`
    )
    .join("\n")
}

/** Links the shared paths into the worktree at path and reports the outcome. */
export async function linkSharedPaths(
  appId: number,
  path: string,
  setOutput: (output: string) => void
) {
  reportLinkStates(await api.git.worktreeLinks(appId, path, true), setOutput)
}

/** Shows the outcome of linking shared paths in the output box and a toast. */
export function reportLinkStates(
  states: WorktreeLinkState[],
  setOutput: (output: string) => void
) {
  setOutput(formatLinkStates(states))
  const linked = states.filter((s) => s.state === "linked").length
  const failed = states.filter((s) => s.state === "error").length
  const message = `${linked} of ${states.length} shared paths linked`
  if (failed) toast.error(`${message}, ${failed} failed`)
  else if (linked < states.length) toast.warning(`${message}, see the output`)
  else toast.success(message)
}
