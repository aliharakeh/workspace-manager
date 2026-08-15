import path from "node:path"
import { fileURLToPath } from "node:url"
import type { IncomingMessage, ServerResponse } from "node:http"
import type { Plugin } from "vite"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

const rootDir = path.dirname(fileURLToPath(import.meta.url))
const apiPort = Number(process.env.API_PORT || process.env.VITE_API_PORT) || 3000
const webPort = Number(process.env.WEB_PORT) || 5173
const apiOrigin = `http://localhost:${apiPort}`

/**
 * Vite's http-proxy buffers chunked SSE responses. Intercept EventSource
 * requests and pipe the upstream stream ourselves.
 */
function sseProxyPlugin(target: string): Plugin {
  return {
    name: "sse-proxy",
    configureServer(server) {
      server.middlewares.use((req, res, next) => {
        const isLogs =
          !!req.url && /\/api\/apps\/\d+\/run\/logs(?:\?|$)/.test(req.url)
        const wantsSse = req.headers.accept?.includes("text/event-stream")
        if (!isLogs && !wantsSse) {
          next()
          return
        }
        if (!req.url?.startsWith("/api/")) {
          next()
          return
        }
        void pipeSse(target, req, res)
      })
    },
  }
}

async function pipeSse(
  target: string,
  req: IncomingMessage,
  res: ServerResponse
) {
  const ac = new AbortController()
  const onClose = () => ac.abort()
  req.on("close", onClose)

  try {
    const upstream = await fetch(`${target}${req.url}`, {
      headers: { Accept: "text/event-stream" },
      signal: ac.signal,
    })

    res.statusCode = upstream.status
    res.setHeader("Content-Type", "text/event-stream; charset=utf-8")
    res.setHeader("Cache-Control", "no-cache, no-transform")
    res.setHeader("Connection", "keep-alive")
    res.setHeader("X-Accel-Buffering", "no")
    res.flushHeaders?.()

    if (!upstream.body) {
      res.end()
      return
    }

    const reader = upstream.body.getReader()
    try {
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        if (!res.writableEnded) res.write(Buffer.from(value))
      }
    } finally {
      if (!res.writableEnded) res.end()
    }
  } catch (err) {
    if (ac.signal.aborted) {
      if (!res.writableEnded) res.end()
      return
    }
    if (!res.headersSent) {
      res.statusCode = 502
      res.end(err instanceof Error ? err.message : "SSE proxy failed")
    } else if (!res.writableEnded) {
      res.end()
    }
  } finally {
    req.off("close", onClose)
  }
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss(), sseProxyPlugin(apiOrigin)],
  resolve: {
    alias: {
      "@": path.resolve(rootDir, "./src"),
    },
  },
  server: {
    port: webPort,
    // Prefer the pre-selected port from scripts; still allow Vite to recover
    // if something races us between check and bind.
    strictPort: false,
    proxy: {
      "/api": {
        target: apiOrigin,
        changeOrigin: true,
        timeout: 0,
      },
    },
  },
})
