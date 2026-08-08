import { settingsRepo } from "../db/settings"
import { error, json, readJson } from "../lib/http"

export async function handleSettings(
  req: Request,
  pathname: string
): Promise<Response | null> {
  if (pathname === "/api/settings" && req.method === "GET") {
    return json(settingsRepo.getAll())
  }

  if (pathname === "/api/settings" && req.method === "PUT") {
    const body = await readJson<{ key?: string; value?: string }>(req)
    if (!body?.key?.trim()) return error("key is required")
    settingsRepo.set(body.key.trim(), body.value ?? "")
    return json(settingsRepo.getAll())
  }

  return null
}
