import type { GitBranchInfo, GitRemoteInfo, GitRepoGraph } from '@/lib/types'

export type GitGraphSnapshot = {
    projectPath: string
    graph: GitRepoGraph
    catalog: GitBranchInfo[]
    remote: GitRemoteInfo | null
    authors: Set<string>
    visible: Set<string>
    axisRange: [number, number] | null
    loaded: { from: number; to: number; pastDone: boolean; futureDone: boolean }
}

// Per-app git graph snapshots, kept while navigating within a workspace so
// re-entering an app restores the graph instantly. Dropped when the
// workspace is left (see clearGitGraphCache).
const cache = new Map<number, GitGraphSnapshot>()

const MAX_ENTRIES = 8

export function getGitGraphSnapshot(appId: number, projectPath: string): GitGraphSnapshot | null {
    const hit = cache.get(appId)
    return hit && hit.projectPath === projectPath ? hit : null
}

export function setGitGraphSnapshot(appId: number, snapshot: GitGraphSnapshot) {
    cache.set(appId, snapshot)
    while (cache.size > MAX_ENTRIES) {
        const oldest = cache.keys().next().value
        if (oldest === undefined) break
        cache.delete(oldest)
    }
}

export function clearGitGraphCache() {
    cache.clear()
}
