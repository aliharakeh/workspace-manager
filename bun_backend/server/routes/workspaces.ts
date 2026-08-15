import { workspacesRepo } from "@db/workspaces"
import { error, json, notFound, parseId, readJson } from "../lib/http"

export async function handleWorkspaces(
  req: Request,
  pathname: string
): Promise<Response | null> {
  if (pathname === "/api/workspaces" && req.method === "GET") {
    return json(workspacesRepo.list())
  }

  if (pathname === "/api/workspaces" && req.method === "POST") {
    const body = await readJson<{ name?: string; icon?: string | null }>(req)
    if (!body?.name?.trim()) return error("name is required")
    const workspace = workspacesRepo.create({
      name: body.name.trim(),
      icon: body.icon ?? null,
    })
    return json(workspace, { status: 201 })
  }

  const match = pathname.match(/^\/api\/workspaces\/(\d+)$/)
  if (!match) return null

  const id = parseId(match[1])
  if (!id) return error("Invalid workspace id")

  if (req.method === "GET") {
    const workspace = workspacesRepo.get(id)
    if (!workspace) return notFound("Workspace not found")
    return json(workspace)
  }

  if (req.method === "PATCH") {
    const body = await readJson<{ name?: string; icon?: string | null }>(req)
    if (!body) return error("Invalid JSON body")
    if (body.name !== undefined && !body.name.trim()) {
      return error("name cannot be empty")
    }
    const workspace = workspacesRepo.update(id, {
      name: body.name?.trim(),
      icon: body.icon,
    })
    if (!workspace) return notFound("Workspace not found")
    return json(workspace)
  }

  if (req.method === "DELETE") {
    if (!workspacesRepo.delete(id)) return notFound("Workspace not found")
    return new Response(null, { status: 204 })
  }

  return null
}
