import { readFileSync } from "node:fs"
import { appsRepo } from "@db/apps"
import {
  pickNativeFile,
  pickNativeFolder,
} from "@native/dialog"
import {
  readProjectFile,
  toProjectRelative,
  validateProjectPath,
} from "../lib/fs"
import { error, json, notFound, parseId, readJson } from "../lib/http"

function readPickedFileContent(absolutePath: string): string {
  try {
    return readFileSync(absolutePath, "utf8")
  } catch (err) {
    throw new Error(
      err instanceof Error ? err.message : "Failed to read selected file"
    )
  }
}

export async function handleFs(
  req: Request,
  pathname: string
): Promise<Response | null> {
  if (pathname === "/api/fs/validate-path" && req.method === "POST") {
    const body = await readJson<{ path?: string }>(req)
    if (!body?.path) return error("path is required")
    const result = validateProjectPath(body.path)
    if (!result.ok) {
      return json({ ok: false, error: result.error }, { status: 400 })
    }
    return json({ ok: true, path: result.resolved })
  }

  if (pathname === "/api/fs/pick-folder" && req.method === "POST") {
    const body = await readJson<{ startDir?: string }>(req)
    const startDir = body?.startDir?.trim() || undefined
    const picked = await pickNativeFolder(startDir)
    if (!picked.ok) {
      if ("cancelled" in picked && picked.cancelled) {
        return json({ cancelled: true })
      }
      return error(
        "error" in picked ? picked.error : "Folder dialog failed",
        500
      )
    }
    return json({ cancelled: false, path: picked.path })
  }

  if (pathname === "/api/fs/pick-file" && req.method === "POST") {
    const body = await readJson<{
      startDir?: string
      appId?: number
    }>(req)

    let startDir = body?.startDir?.trim() || undefined
    let projectRoot: string | undefined

    if (body?.appId != null) {
      const appId = Number(body.appId)
      if (!Number.isInteger(appId) || appId <= 0) return error("Invalid app id")
      const app = appsRepo.get(appId)
      if (!app) return notFound("App not found")
      projectRoot = app.project_path
      startDir = startDir || app.project_path
    }

    const picked = await pickNativeFile(startDir)
    if (!picked.ok) {
      if ("cancelled" in picked && picked.cancelled) {
        return json({ cancelled: true })
      }
      return error("error" in picked ? picked.error : "File dialog failed", 500)
    }

    let content = ""
    try {
      content = readPickedFileContent(picked.path)
    } catch (err) {
      return error(err instanceof Error ? err.message : "Failed to read file")
    }

    if (projectRoot) {
      const relative = toProjectRelative(projectRoot, picked.path)
      if (!relative) {
        return error(
          "Selected file must be inside the app project directory",
          400
        )
      }
      return json({
        cancelled: false,
        path: picked.path,
        relative_path: relative,
        content,
      })
    }

    return json({ cancelled: false, path: picked.path, content })
  }

  const appPick = pathname.match(/^\/api\/apps\/(\d+)\/pick-file$/)
  if (appPick && req.method === "POST") {
    const appId = parseId(appPick[1])
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

    const relative = toProjectRelative(app.project_path, picked.path)
    if (!relative) {
      return error(
        "Selected file must be inside the app project directory",
        400
      )
    }

    let content = ""
    try {
      content = readPickedFileContent(picked.path)
    } catch (err) {
      return error(err instanceof Error ? err.message : "Failed to read file")
    }

    return json({
      cancelled: false,
      path: picked.path,
      relative_path: relative,
      content,
    })
  }

  const appRead = pathname.match(/^\/api\/apps\/(\d+)\/read-file$/)
  if (appRead && req.method === "POST") {
    const appId = parseId(appRead[1])
    if (!appId) return error("Invalid app id")
    const app = appsRepo.get(appId)
    if (!app) return notFound("App not found")

    const body = await readJson<{ path?: string }>(req)
    if (!body?.path?.trim()) return error("path is required")

    const result = readProjectFile(app.project_path, body.path)
    if (!result.ok) return error(result.error, 400)
    return json(result)
  }

  return null
}
