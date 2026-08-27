import type { AIProviderConfig } from "@/lib/types"
import type { AppAIStreamEvent } from "@/lib/app-ai"
import {
  activateAIConnection,
  deleteAIConnection,
  loadAIStore,
  saveAIStore,
  slugify,
  upsertAIConnection,
} from "../lib/ai-config"
import { aiInfo, aiChat, aiAppChat, aiTest } from "../services/ai"
import { error, json, readJson } from "../lib/http"

const sseHeaders = {
  "Content-Type": "text/event-stream; charset=utf-8",
  "Cache-Control": "no-cache, no-transform",
  Connection: "keep-alive",
  "X-Accel-Buffering": "no",
}

export async function handleAI(
  req: Request,
  pathname: string
): Promise<Response | null> {
  if (pathname === "/api/ai/config") {
    if (req.method === "GET") return json(aiInfo())

    // Upserts one connection (keyed by its normalized name).
    if (req.method === "PUT") {
      const body = await readJson<AIProviderConfig>(req)
      if (!body) return error("Invalid JSON body")
      if (!body.name?.trim()) return error("connection name is required")
      if (!body.provider?.trim()) return error("provider is required")
      const store = loadAIStore()
      try {
        upsertAIConnection(store, body)
      } catch (err) {
        return error(err instanceof Error ? err.message : "Invalid config", 400)
      }
      saveAIStore(store)
      return json(aiInfo())
    }
  }

  if (pathname === "/api/ai/config/activate" && req.method === "POST") {
    const body = await readJson<{ name?: string }>(req)
    const store = loadAIStore()
    try {
      activateAIConnection(store, slugify(body?.name))
    } catch (err) {
      return error(err instanceof Error ? err.message : "Unknown connection", 404)
    }
    saveAIStore(store)
    return json(aiInfo())
  }

  const connMatch = pathname.match(/^\/api\/ai\/config\/([^/]+)$/)
  if (connMatch && req.method === "DELETE") {
    const store = loadAIStore()
    if (!deleteAIConnection(store, decodeURIComponent(connMatch[1]!)))
      return error("Connection not found", 404)
    saveAIStore(store)
    return json(aiInfo())
  }

  // Tests an unsaved connection payload: applies defaults, runs one minimal
  // generation, and reports the reply. Nothing is persisted.
  if (pathname === "/api/ai/test" && req.method === "POST") {
    const body = await readJson<AIProviderConfig>(req)
    if (!body) return error("Invalid JSON body")
    if (!body.provider?.trim()) return error("provider is required")
    try {
      const text = await aiTest(body)
      return json({ ok: true, text })
    } catch (err) {
      return error(err instanceof Error ? err.message : String(err), 502)
    }
  }

  if (pathname === "/api/ai/chat" && req.method === "POST") {
    const body = await readJson<{ system?: string; prompt?: string }>(req)
    try {
      const text = await aiChat(body?.system, body?.prompt)
      return json({ text })
    } catch (err) {
      return error(err instanceof Error ? err.message : String(err), 500)
    }
  }

  if (pathname === "/api/ai/app-chat" && req.method === "POST") {
    const body = await readJson<{
      appId?: number
      configSetId?: number
      history?: {
        role?: string
        text?: string
        tools?: { name?: string; input?: unknown; output?: unknown }[]
      }[]
      instruction?: string
    }>(req)
    if (!body?.appId || !body.configSetId) {
      return error("appId and configSetId are required")
    }
    if (!body.instruction?.trim()) return error("instruction is required")
    const appId = body.appId
    const configSetId = body.configSetId
    const instruction = body.instruction
    const history = (body.history ?? [])
      .filter(
        (t): t is {
          role: "user" | "assistant"
          text: string
          tools?: { name: string; input: unknown; output: unknown }[]
        } =>
          (t.role === "user" || t.role === "assistant") &&
          typeof t.text === "string"
      )
      .map((t) => ({
        role: t.role,
        text: t.text,
        tools: (t.tools ?? [])
          .filter(
            (c): c is { name: string; input: unknown; output: unknown } =>
              typeof c?.name === "string" && c.name.length > 0
          )
          .map((c) => ({ name: c.name, input: c.input, output: c.output })),
      }))
    try {
      const wantsStream = req.headers
        .get("accept")
        ?.includes("text/event-stream")
      if (!wantsStream) {
        return json(
          await aiAppChat({
            appId,
            configSetId,
            history,
            instruction,
          })
        )
      }

      const stream = new ReadableStream({
        async start(controller) {
          const encoder = new TextEncoder()
          let closed = false
          const send = (ev: AppAIStreamEvent) => {
            if (closed) return
            try {
              controller.enqueue(
                encoder.encode(`data: ${JSON.stringify(ev)}\n\n`)
              )
            } catch {
              closed = true
            }
          }
          try {
            const result = await aiAppChat(
              {
                appId,
                configSetId,
                history,
                instruction,
              },
              send
            )
            send({ type: "done", ...result })
          } catch (err) {
            send({
              type: "error",
              error: err instanceof Error ? err.message : String(err),
            })
          } finally {
            if (!closed) {
              try {
                controller.close()
              } catch {
                // already closed
              }
            }
          }
        },
      })
      return new Response(stream, { headers: sseHeaders })
    } catch (err) {
      return error(err instanceof Error ? err.message : String(err), 500)
    }
  }

  return null
}
