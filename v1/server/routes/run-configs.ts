import { appsRepo } from "../db/apps"
import { configSetsRepo } from "../db/config-sets"
import { runConfigsRepo } from "../db/run-configs"
import type { RunMode } from "../db/types"
import { error, json, notFound, parseId, readJson } from "../lib/http"

export async function handleRunConfigs(
  req: Request,
  pathname: string
): Promise<Response | null> {
  const match = pathname.match(/^\/api\/apps\/(\d+)\/run-config$/)
  if (!match) return null

  const appId = parseId(match[1])
  if (!appId) return error("Invalid app id")
  if (!appsRepo.get(appId)) return notFound("App not found")

  const set = configSetsRepo.resolveActive(appId)

  if (req.method === "GET") {
    return json(runConfigsRepo.getOrCreate(set.id))
  }

  if (req.method === "PUT") {
    const body = await readJson<{
      mode?: RunMode
      commands?: Array<{ label?: string | null; command: string }>
    }>(req)
    if (!body) return error("Invalid JSON body")
    if (body.mode && body.mode !== "sequential" && body.mode !== "parallel") {
      return error("mode must be sequential or parallel")
    }
    if (body.commands) {
      for (const cmd of body.commands) {
        if (!cmd.command?.trim()) {
          return error("Each command must have a non-empty command string")
        }
      }
    }

    const config = runConfigsRepo.upsert(set.id, {
      mode: body.mode,
      commands: body.commands?.map((c) => ({
        label: c.label ?? null,
        command: c.command.trim(),
      })),
    })
    return json(config)
  }

  return null
}
