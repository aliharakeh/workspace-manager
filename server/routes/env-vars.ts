import { appsRepo } from "../db/apps"
import { configSetsRepo } from "../db/config-sets"
import { envVarsRepo } from "../db/env-vars"
import { error, json, notFound, parseId, readJson } from "../lib/http"

export async function handleEnvVars(
  req: Request,
  pathname: string
): Promise<Response | null> {
  const listMatch = pathname.match(/^\/api\/apps\/(\d+)\/env-vars$/)
  if (listMatch) {
    const appId = parseId(listMatch[1])
    if (!appId) return error("Invalid app id")
    if (!appsRepo.get(appId)) return notFound("App not found")

    const set = configSetsRepo.resolveActive(appId)

    if (req.method === "GET") {
      return json(envVarsRepo.listByConfigSet(set.id))
    }

    if (req.method === "POST") {
      const body = await readJson<{ key?: string; value?: string }>(req)
      if (!body?.key?.trim()) return error("key is required")
      try {
        const envVar = envVarsRepo.create({
          config_set_id: set.id,
          key: body.key.trim(),
          value: body.value ?? "",
        })
        return json(envVar, { status: 201 })
      } catch {
        return error("Env var key already exists for this config set", 409)
      }
    }

    return null
  }

  const itemMatch = pathname.match(/^\/api\/env-vars\/(\d+)$/)
  if (!itemMatch) return null

  const id = parseId(itemMatch[1])
  if (!id) return error("Invalid env var id")

  if (req.method === "PATCH") {
    const body = await readJson<{ key?: string; value?: string }>(req)
    if (!body) return error("Invalid JSON body")
    if (body.key !== undefined && !body.key.trim()) {
      return error("key cannot be empty")
    }
    try {
      const envVar = envVarsRepo.update(id, {
        key: body.key?.trim(),
        value: body.value,
      })
      if (!envVar) return notFound("Env var not found")
      return json(envVar)
    } catch {
      return error("Env var key already exists for this config set", 409)
    }
  }

  if (req.method === "DELETE") {
    if (!envVarsRepo.delete(id)) return notFound("Env var not found")
    return new Response(null, { status: 204 })
  }

  return null
}
