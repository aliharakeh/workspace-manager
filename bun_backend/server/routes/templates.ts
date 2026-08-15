import { appsRepo } from "@db/apps"
import { configSetsRepo } from "@db/config-sets"
import { templatesRepo } from "@db/templates"
import { error, json, notFound, parseId, readJson } from "../lib/http"

export async function handleTemplates(
  req: Request,
  pathname: string
): Promise<Response | null> {
  const listMatch = pathname.match(/^\/api\/apps\/(\d+)\/templates$/)
  if (listMatch) {
    const appId = parseId(listMatch[1])
    if (!appId) return error("Invalid app id")
    if (!appsRepo.get(appId)) return notFound("App not found")

    const set = configSetsRepo.resolveActive(appId)

    if (req.method === "GET") {
      return json(templatesRepo.listByConfigSet(set.id))
    }

    if (req.method === "POST") {
      const body = await readJson<{ file_path?: string; content?: string }>(req)
      if (!body?.file_path?.trim()) return error("file_path is required")
      try {
        const template = templatesRepo.create({
          config_set_id: set.id,
          file_path: body.file_path.trim().replace(/\\/g, "/"),
          content: body.content ?? "",
        })
        return json(template, { status: 201 })
      } catch {
        return error("Template for this file already exists", 409)
      }
    }

    return null
  }

  const itemMatch = pathname.match(/^\/api\/templates\/(\d+)$/)
  if (!itemMatch) return null

  const id = parseId(itemMatch[1])
  if (!id) return error("Invalid template id")

  if (req.method === "PATCH") {
    const body = await readJson<{ file_path?: string; content?: string }>(req)
    if (!body) return error("Invalid JSON body")
    if (body.file_path !== undefined && !body.file_path.trim()) {
      return error("file_path cannot be empty")
    }
    try {
      const template = templatesRepo.update(id, {
        file_path: body.file_path?.trim().replace(/\\/g, "/"),
        content: body.content,
      })
      if (!template) return notFound("Template not found")
      return json(template)
    } catch {
      return error("Template for this file already exists", 409)
    }
  }

  if (req.method === "DELETE") {
    if (!templatesRepo.delete(id)) return notFound("Template not found")
    return new Response(null, { status: 204 })
  }

  return null
}
