import { appsRepo } from "@db/apps"
import {
  configSetsRepo,
  type ConfigCopyParts,
} from "@db/config-sets"
import { envVarsRepo } from "@db/env-vars"
import { runConfigsRepo } from "@db/run-configs"
import { templatesRepo } from "@db/templates"
import { error, json, notFound, parseId, readJson } from "../lib/http"

function parseListPart<T>(
  value: unknown,
  guard: (item: unknown) => item is T
): boolean | T[] | undefined {
  if (typeof value === "boolean") return value
  if (Array.isArray(value)) {
    const items = value.filter(guard)
    return items.length > 0 ? items : false
  }
  return undefined
}

function parseParts(raw: unknown): ConfigCopyParts | undefined {
  if (!raw || typeof raw !== "object") return undefined
  const p = raw as Record<string, unknown>
  return {
    env: parseListPart(p.env, (v): v is string => typeof v === "string"),
    templates: parseListPart(p.templates, (v): v is string => typeof v === "string"),
    run: parseListPart(
      p.run,
      (v): v is number => typeof v === "number" && Number.isInteger(v)
    ),
  }
}

function isPartEnabled(
  part: boolean | string[] | number[] | undefined
): boolean {
  if (part === undefined || part === true) return true
  if (Array.isArray(part)) return part.length > 0
  return false
}

function hasAnyPart(parts?: ConfigCopyParts): boolean {
  if (!parts) return true
  return (
    isPartEnabled(parts.env) ||
    isPartEnabled(parts.templates) ||
    isPartEnabled(parts.run)
  )
}

export async function handleConfigSets(
  req: Request,
  pathname: string
): Promise<Response | null> {
  const listMatch = pathname.match(/^\/api\/apps\/(\d+)\/config-sets$/)
  if (listMatch) {
    const appId = parseId(listMatch[1])
    if (!appId) return error("Invalid app id")
    if (!appsRepo.get(appId)) return notFound("App not found")

    if (req.method === "GET") {
      return json(configSetsRepo.listByApp(appId))
    }

    if (req.method === "POST") {
      const body = await readJson<{
        name?: string
        copy_from_id?: number
        activate?: boolean
        parts?: ConfigCopyParts
      }>(req)
      if (!body?.name?.trim()) return error("name is required")

      const parts = parseParts(body.parts)
      if (body.copy_from_id != null && !hasAnyPart(parts)) {
        return error("Select at least one part to copy")
      }

      if (body.copy_from_id != null) {
        const source = configSetsRepo.get(body.copy_from_id)
        if (!source || source.app_id !== appId) {
          return error("copy_from_id must be a config set on this app")
        }
      }

      try {
        const set = configSetsRepo.create({
          app_id: appId,
          name: body.name.trim(),
        })
        if (body.copy_from_id != null) {
          configSetsRepo.copyFrom(body.copy_from_id, set.id, parts)
        }
        if (body.activate !== false) {
          appsRepo.setActiveConfigSet(appId, set.id)
        }
        return json(set, { status: 201 })
      } catch {
        return error("A config set with this name already exists", 409)
      }
    }

    return null
  }

  const activateMatch = pathname.match(
    /^\/api\/config-sets\/(\d+)\/activate$/
  )
  if (activateMatch) {
    if (req.method !== "POST") return null
    const id = parseId(activateMatch[1])
    if (!id) return error("Invalid config set id")
    const set = configSetsRepo.get(id)
    if (!set) return notFound("Config set not found")
    const app = appsRepo.setActiveConfigSet(set.app_id, set.id)
    return json({ ...set, app })
  }

  const copyMatch = pathname.match(/^\/api\/config-sets\/(\d+)\/copy-from$/)
  if (copyMatch) {
    if (req.method !== "POST") return null
    const id = parseId(copyMatch[1])
    if (!id) return error("Invalid config set id")
    const target = configSetsRepo.get(id)
    if (!target) return notFound("Config set not found")

    const body = await readJson<{
      source_id?: number
      parts?: ConfigCopyParts
    }>(req)
    if (!body?.source_id) return error("source_id is required")
    const parts = parseParts(body.parts)
    if (!hasAnyPart(parts)) {
      return error("Select at least one part to copy")
    }
    const source = configSetsRepo.get(body.source_id)
    if (!source || source.app_id !== target.app_id) {
      return error("source_id must be a config set on the same app")
    }

    try {
      configSetsRepo.copyFrom(body.source_id, id, parts)
      return json(configSetsRepo.get(id))
    } catch (err) {
      return error(err instanceof Error ? err.message : "Copy failed")
    }
  }

  const itemMatch = pathname.match(/^\/api\/config-sets\/(\d+)$/)
  if (!itemMatch) return null

  const id = parseId(itemMatch[1])
  if (!id) return error("Invalid config set id")

  if (req.method === "GET") {
    const set = configSetsRepo.get(id)
    if (!set) return notFound("Config set not found")
    return json({
      ...set,
      env_vars: envVarsRepo.listByConfigSet(id),
      templates: templatesRepo.listByConfigSet(id),
      run_config: runConfigsRepo.getByConfigSet(id),
    })
  }

  if (req.method === "PATCH") {
    const body = await readJson<{ name?: string }>(req)
    if (!body?.name?.trim()) return error("name is required")
    try {
      const set = configSetsRepo.update(id, { name: body.name.trim() })
      if (!set) return notFound("Config set not found")
      return json(set)
    } catch {
      return error("A config set with this name already exists", 409)
    }
  }

  if (req.method === "DELETE") {
    const set = configSetsRepo.get(id)
    if (!set) return notFound("Config set not found")

    const siblings = configSetsRepo.listByApp(set.app_id)
    if (siblings.length <= 1) {
      return error("Cannot delete the last config set", 400)
    }

    const app = appsRepo.get(set.app_id)
    configSetsRepo.delete(id)

    if (app?.active_config_set_id === id) {
      const next = configSetsRepo.listByApp(set.app_id)[0]
      if (next) appsRepo.setActiveConfigSet(set.app_id, next.id)
    }

    return new Response(null, { status: 204 })
  }

  return null
}
