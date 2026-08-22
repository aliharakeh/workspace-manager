import path from "node:path"
import { fileURLToPath } from "node:url"
import type { IncomingMessage, ServerResponse } from "node:http"
import type { Plugin } from "vite"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

const rootDir = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.resolve(rootDir, "..")
const frontendDir = path.resolve(repoRoot, "frontend")
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
    const method = (req.method ?? "GET").toUpperCase()
    const chunks: Buffer[] = []
    if (method !== "GET" && method !== "HEAD") {
      for await (const chunk of req) {
        chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk))
      }
    }
    const headers: Record<string, string> = {
      Accept: "text/event-stream",
    }
    const contentType = req.headers["content-type"]
    if (typeof contentType === "string") headers["Content-Type"] = contentType

    const upstream = await fetch(`${target}${req.url}`, {
      method,
      headers,
      body: chunks.length ? Buffer.concat(chunks) : undefined,
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

function resolveDepsFrom(hostImporter: string): Plugin {
  return {
    name: "resolve-deps-from-host",
    enforce: "pre",
    async resolveId(id, importer, options) {
      if (!id || id.startsWith("\0")) return null
      if (id.startsWith(".") || path.isAbsolute(id)) return null
      if (id === "@host" || id.startsWith("@/")) return null
      if (!importer) return null
      const fromFrontend =
        path.normalize(importer).startsWith(path.normalize(frontendDir))
      if (!fromFrontend) return null
      return this.resolve(id, hostImporter, {
        ...options,
        skipSelf: true,
      })
    },
  }
}

export default defineConfig({
  root: rootDir,
  plugins: [
    resolveDepsFrom(path.resolve(rootDir, "host.ts")),
    react(),
    tailwindcss(),
    sseProxyPlugin(apiOrigin),
  ],
  resolve: {
    alias: {
      "@": frontendDir,
      "@host": path.resolve(rootDir, "host.ts"),
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
  server: {
    port: webPort,
    strictPort: false,
    fs: {
      allow: [repoRoot, rootDir, frontendDir],
    },
    proxy: {
      "/api": {
        target: apiOrigin,
        changeOrigin: true,
        timeout: 0,
      },
    },
  },
})
