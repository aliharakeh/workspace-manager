import { appsRepo } from "../db/apps"
import { error, json, notFound, parseId } from "../lib/http"
import { runner, type RunnerEvent } from "../services/runner"

const sseHeaders = {
  "Content-Type": "text/event-stream; charset=utf-8",
  "Cache-Control": "no-cache, no-transform",
  Connection: "keep-alive",
  "X-Accel-Buffering": "no",
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Headers": "Content-Type, Last-Event-ID",
}

export async function handleRunner(
  req: Request,
  pathname: string
): Promise<Response | null> {
  const runMatch = pathname.match(/^\/api\/apps\/(\d+)\/run$/)
  if (runMatch && req.method === "POST") {
    const appId = parseId(runMatch[1])
    if (!appId) return error("Invalid app id")
    if (!appsRepo.get(appId)) return notFound("App not found")
    try {
      const status = await runner.start(appId)
      return json(status)
    } catch (err) {
      return error(err instanceof Error ? err.message : String(err))
    }
  }

  const stopMatch = pathname.match(/^\/api\/apps\/(\d+)\/stop$/)
  if (stopMatch && req.method === "POST") {
    const appId = parseId(stopMatch[1])
    if (!appId) return error("Invalid app id")
    try {
      const status = await runner.stop(appId)
      return json(status)
    } catch (err) {
      return error(err instanceof Error ? err.message : String(err))
    }
  }

  const reloadMatch = pathname.match(/^\/api\/apps\/(\d+)\/reload$/)
  if (reloadMatch && req.method === "POST") {
    const appId = parseId(reloadMatch[1])
    if (!appId) return error("Invalid app id")
    if (!appsRepo.get(appId)) return notFound("App not found")
    try {
      const status = await runner.reload(appId)
      return json(status)
    } catch (err) {
      return error(err instanceof Error ? err.message : String(err))
    }
  }

  const statusMatch = pathname.match(/^\/api\/apps\/(\d+)\/run\/status$/)
  if (statusMatch && req.method === "GET") {
    const appId = parseId(statusMatch[1])
    if (!appId) return error("Invalid app id")
    const status = runner.getStatus(appId)
    return json(status ?? { running: false, appId, processes: [] })
  }

  const workspaceStatusMatch = pathname.match(
    /^\/api\/workspaces\/(\d+)\/run-status$/
  )
  if (workspaceStatusMatch && req.method === "GET") {
    const workspaceId = parseId(workspaceStatusMatch[1])
    if (!workspaceId) return error("Invalid workspace id")
    const apps = appsRepo.listByWorkspace(workspaceId)
    return json(
      apps.map((app) => {
        const status = runner.getStatus(app.id)
        return status ?? { running: false, appId: app.id, processes: [] }
      })
    )
  }

  const logsMatch = pathname.match(/^\/api\/apps\/(\d+)\/run\/logs$/)
  if (logsMatch && req.method === "GET") {
    const appId = parseId(logsMatch[1])
    if (!appId) return error("Invalid app id")

    let unsubscribe = () => {}
    let heartbeat: ReturnType<typeof setInterval> | undefined
    let closed = false

    const stream = new ReadableStream({
      start(controller) {
        const encoder = new TextEncoder()

        const safeEnqueue = (chunk: string) => {
          if (closed) return
          try {
            controller.enqueue(encoder.encode(chunk))
          } catch {
            closed = true
          }
        }

        const send = (event: RunnerEvent) => {
          safeEnqueue(`data: ${JSON.stringify(event)}\n\n`)
        }

        // Initial comment so the client knows the stream is alive
        safeEnqueue(`: connected\n\n`)

        unsubscribe = runner.subscribe(appId, send)
        heartbeat = setInterval(() => {
          safeEnqueue(`: ping\n\n`)
        }, 10000)

        const close = () => {
          if (closed) return
          closed = true
          if (heartbeat) clearInterval(heartbeat)
          unsubscribe()
          try {
            controller.close()
          } catch {
            // already closed
          }
        }

        req.signal.addEventListener("abort", close)
      },
      cancel() {
        closed = true
        if (heartbeat) clearInterval(heartbeat)
        unsubscribe()
      },
    })

    return new Response(stream, { headers: sseHeaders })
  }

  return null
}
