import { useEffect, useState } from "react"
import type { Workspace } from "@/lib/types"
import { cachedLogo, loadWorkspaceLogo, logoKey } from "@/lib/workspace-logo"

/**
 * The workspace's logo as an <img> source: null while it loads, when it has
 * none, or when the file cannot be read.
 */
export function useWorkspaceLogo(
  workspace: Pick<Workspace, "id" | "logo_path"> | null | undefined
): string | null {
  const key = logoKey(workspace)
  const id = workspace?.id ?? 0
  const [result, setResult] = useState<{ key: string; src: string | null }>({
    key: "",
    src: null,
  })

  useEffect(() => {
    if (!key || cachedLogo(key) !== undefined) return
    let live = true
    void loadWorkspaceLogo(id, key).then((src) => {
      if (live) setResult({ key, src })
    })
    return () => {
      live = false
    }
  }, [id, key])

  if (!key) return null
  const cached = cachedLogo(key)
  if (cached !== undefined) return cached
  return result.key === key ? result.src : null
}
