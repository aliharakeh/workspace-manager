import { appsRepo } from "@db/apps"
import { validateProjectPath } from "../lib/fs"
import {
  fetchRemote,
  listBranches,
  listRemote,
  loadGraphAt,
  parseISO,
} from "../lib/git-graph"
import { error, json, notFound, parseId, readJson } from "../lib/http"

function appPath(id: number) {
  const app = appsRepo.get(id)
  if (!app) return { error: notFound("App not found") as Response }
  const pathCheck = validateProjectPath(app.project_path)
  if (!pathCheck.ok) return { error: error(pathCheck.error!) }
  return { path: pathCheck.resolved! }
}

export async function handleGit(
  req: Request,
  pathname: string
): Promise<Response | null> {
  const branches = pathname.match(/^\/api\/apps\/(\d+)\/git\/branches$/)
  if (branches && req.method === "GET") {
    const id = parseId(branches[1])
    if (!id) return error("Invalid app id")
    const resolved = appPath(id)
    if ("error" in resolved) return resolved.error
    try {
      return json(await listBranches(resolved.path))
    } catch (err) {
      return error(err instanceof Error ? err.message : "Failed to list branches")
    }
  }

  const remote = pathname.match(/^\/api\/apps\/(\d+)\/git\/remote$/)
  if (remote && req.method === "GET") {
    const id = parseId(remote[1])
    if (!id) return error("Invalid app id")
    const resolved = appPath(id)
    if ("error" in resolved) return resolved.error
    try {
      return json(await listRemote(resolved.path))
    } catch (err) {
      return error(err instanceof Error ? err.message : "Failed to read remote", 404)
    }
  }

  const fetchMatch = pathname.match(/^\/api\/apps\/(\d+)\/git\/fetch$/)
  if (fetchMatch && req.method === "POST") {
    const id = parseId(fetchMatch[1])
    if (!id) return error("Invalid app id")
    const resolved = appPath(id)
    if ("error" in resolved) return resolved.error
    try {
      await fetchRemote(resolved.path)
      return json({ ok: true })
    } catch (err) {
      return error(err instanceof Error ? err.message : "Fetch failed")
    }
  }

  const graph = pathname.match(/^\/api\/apps\/(\d+)\/git\/graph$/)
  if (graph && req.method === "POST") {
    const id = parseId(graph[1])
    if (!id) return error("Invalid app id")
    const resolved = appPath(id)
    if ("error" in resolved) return resolved.error
    const body = await readJson<{
      branches?: string[] | null
      since?: string
      until?: string
    }>(req)
    try {
      const since = parseISO(body?.since ?? "")
      const until = parseISO(body?.until ?? "")
      const only = body?.branches ?? null
      return json(await loadGraphAt(resolved.path, only, since, until))
    } catch (err) {
      return error(err instanceof Error ? err.message : "Failed to load graph")
    }
  }

  return null
}
