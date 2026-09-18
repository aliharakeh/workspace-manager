import { appsRepo } from "@db/apps";
import { error, json, notFound, parseId } from "../lib/http";
import { readPackageScripts } from "../lib/package-scripts";

export async function handlePackageScripts(
  req: Request,
  pathname: string
): Promise<Response | null> {
  const match = pathname.match(/^\/api\/apps\/(\d+)\/package-scripts$/)
  if (!match) return null
  if (req.method !== "GET") return null

  const appId = parseId(match[1])
  if (!appId) return error("Invalid app id")
  const app = appsRepo.get(appId)
  if (!app) return notFound("App not found")

  return json(readPackageScripts(app.project_path))
}
