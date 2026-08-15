import {
  listListeningProcesses,
  USER_PORT_MAX,
  USER_PORT_MIN,
} from "../../native/ports"
import { killPid } from "../../native/process"
import { error, json, matchRoute, parseId } from "../lib/http"

export async function handlePorts(
  req: Request,
  pathname: string
): Promise<Response | null> {
  if (pathname === "/api/ports" && req.method === "GET") {
    try {
      const processes = await listListeningProcesses()
      return json({
        min: USER_PORT_MIN,
        max: USER_PORT_MAX,
        processes,
      })
    } catch (err) {
      return error(
        err instanceof Error ? err.message : "Failed to list listening ports",
        500
      )
    }
  }

  const killMatch = matchRoute("/api/ports/:pid/kill", pathname)
  if (killMatch && req.method === "POST") {
    const pid = parseId(killMatch.params.pid)
    if (pid == null) return error("Invalid pid")
    try {
      await killPid(pid)
      return json({ ok: true, pid })
    } catch (err) {
      return error(
        err instanceof Error ? err.message : `Failed to kill pid ${pid}`,
        500
      )
    }
  }

  return null
}
