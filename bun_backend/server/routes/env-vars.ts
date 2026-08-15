import { readFileSync } from "node:fs"
import { basename } from "node:path"
import { pickNativeFile } from "@native/dialog"
import { appsRepo } from "@db/apps"
import { configSetsRepo } from "@db/config-sets"
import { envVarsRepo } from "@db/env-vars"
import { templatesRepo } from "@db/templates"
import { toProjectRelative } from "../lib/fs"
import { error, json, notFound, parseId, readJson } from "../lib/http"
import { detectImportFormat, type ImportEntry } from "../lib/import-formats"

export async function handleEnvVars(
  req: Request,
  pathname: string
): Promise<Response | null> {
  const importMatch = pathname.match(/^\/api\/apps\/(\d+)\/env-vars\/import$/)
  if (importMatch && req.method === "POST") {
    const appId = parseId(importMatch[1])
    if (!appId) return error("Invalid app id")
    const app = appsRepo.get(appId)
    if (!app) return notFound("App not found")

    const picked = await pickNativeFile(app.project_path)
    if (!picked.ok) {
      if ("cancelled" in picked && picked.cancelled) {
        return json({ cancelled: true })
      }
      return error("error" in picked ? picked.error : "File dialog failed", 500)
    }

    let content: string
    try {
      content = readFileSync(picked.path, "utf8")
    } catch (err) {
      return error(
        err instanceof Error ? err.message : "Failed to read selected file"
      )
    }

    const format = detectImportFormat(picked.path)
    if (!format) {
      return error(`Unsupported file format: ${basename(picked.path)}`, 400)
    }

    let entries: ImportEntry[]
    try {
      entries = format.parse(content)
    } catch (err) {
      return error(
        err instanceof Error
          ? err.message
          : `Failed to parse the selected ${format.label} file`,
        400
      )
    }

    const set = configSetsRepo.resolveActive(appId)
    const imported = entries.map((e) =>
      envVarsRepo.upsertByKey(set.id, e.key, e.value)
    )

    // Auto-create (or refresh) a template for the picked file so each value
    // resolves from the app's env vars at render time, e.g. `KEY={{KEY}}`.
    // Always create it: fall back to the file name if it's outside the project.
    let template: {
      id: number
      file_path: string
      created: boolean
    } | null = null
    try {
      const relPath =
        toProjectRelative(app.project_path, picked.path) ??
        basename(picked.path.replace(/\\/g, "/"))
      const templateContent = format.toTemplate(content)
      const existing = templatesRepo
        .listByConfigSet(set.id)
        .find((t) => t.file_path === relPath)
      if (existing) {
        templatesRepo.update(existing.id, { content: templateContent })
        template = { id: existing.id, file_path: relPath, created: false }
      } else {
        const created = templatesRepo.create({
          config_set_id: set.id,
          file_path: relPath,
          content: templateContent,
        })
        template = { id: created.id, file_path: relPath, created: true }
      }
      console.log(
        `[env-vars/import] app=${appId} file="${picked.path}" rel="${relPath}" template=${template.created ? "created" : "updated"}`
      )
    } catch (err) {
      console.error(
        `[env-vars/import] template creation failed for "${picked.path}":`,
        err
      )
    }

    return json({
      cancelled: false,
      path: picked.path,
      format: format.label,
      imported: imported.length,
      vars: envVarsRepo.listByConfigSet(set.id),
      template,
    })
  }

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
