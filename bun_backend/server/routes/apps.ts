import { openInEditor } from "@native/editor"
import { appsRepo } from "@db/apps"
import { workspacesRepo } from "@db/workspaces"
import { validateProjectPath } from "../lib/fs"
import { error, json, notFound, parseId, readJson } from "../lib/http"

export async function handleApps(
  req: Request,
  pathname: string
): Promise<Response | null> {
  const workspaceApps = pathname.match(/^\/api\/workspaces\/(\d+)\/apps$/)
  if (workspaceApps) {
    const workspaceId = parseId(workspaceApps[1])
    if (!workspaceId) return error("Invalid workspace id")

    if (!workspacesRepo.get(workspaceId)) {
      return notFound("Workspace not found")
    }

    if (req.method === "GET") {
      return json(appsRepo.listByWorkspace(workspaceId))
    }

    if (req.method === "POST") {
      const body = await readJson<{ name?: string; project_path?: string }>(req)
      if (!body?.name?.trim()) return error("name is required")
      if (!body.project_path?.trim()) return error("project_path is required")

      const pathCheck = validateProjectPath(body.project_path)
      if (!pathCheck.ok) return error(pathCheck.error!)

      const app = appsRepo.create({
        workspace_id: workspaceId,
        name: body.name.trim(),
        project_path: pathCheck.resolved!,
      })
      return json(app, { status: 201 })
    }

    return null
  }

  const openEditor = pathname.match(/^\/api\/apps\/(\d+)\/open-in-editor$/)
  if (openEditor && req.method === "POST") {
    const id = parseId(openEditor[1])
    if (!id) return error("Invalid app id")
    const app = appsRepo.get(id)
    if (!app) return notFound("App not found")

    const pathCheck = validateProjectPath(app.project_path)
    if (!pathCheck.ok) return error(pathCheck.error!)

    openInEditor(pathCheck.resolved!)
    return json({ ok: true })
  }

  const appMatch = pathname.match(/^\/api\/apps\/(\d+)$/)
  if (!appMatch) return null

  const id = parseId(appMatch[1])
  if (!id) return error("Invalid app id")

  if (req.method === "GET") {
    const app = appsRepo.get(id)
    if (!app) return notFound("App not found")
    return json(app)
  }

  if (req.method === "PATCH") {
    const body = await readJson<{ name?: string; project_path?: string }>(req)
    if (!body) return error("Invalid JSON body")
    if (body.name !== undefined && !body.name.trim()) {
      return error("name cannot be empty")
    }

    let projectPath: string | undefined
    if (body.project_path !== undefined) {
      const pathCheck = validateProjectPath(body.project_path)
      if (!pathCheck.ok) return error(pathCheck.error!)
      projectPath = pathCheck.resolved
    }

    const app = appsRepo.update(id, {
      name: body.name?.trim(),
      project_path: projectPath,
    })
    if (!app) return notFound("App not found")
    return json(app)
  }

  if (req.method === "DELETE") {
    if (!appsRepo.delete(id)) return notFound("App not found")
    return new Response(null, { status: 204 })
  }

  return null
}
