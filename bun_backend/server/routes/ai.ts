import type { AIProviderConfig } from "@/lib/types"
import {
  activateAIConnection,
  deleteAIConnection,
  loadAIStore,
  saveAIStore,
  slugify,
  upsertAIConnection,
} from "../lib/ai-config"
import { aiInfo, aiChat, aiTest } from "../services/ai"
import { error, json, readJson } from "../lib/http"

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

  return null
}
