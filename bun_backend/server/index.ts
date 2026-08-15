import "@db"
import { readyUrlPatternsRepo } from "@db/ready-url-patterns"
import { findAvailablePort } from "../native/ports"

// Persist built-in log URL patterns so they show up in Settings by default.
readyUrlPatternsRepo.ensureSeeded()
import { health } from "./routes/health"
import { handleWorkspaces } from "./routes/workspaces"
import { handleApps } from "./routes/apps"
import { handleFs } from "./routes/fs"
import { handleConfigSets } from "./routes/config-sets"
import { handleEnvVars } from "./routes/env-vars"
import { handleTemplates } from "./routes/templates"
import { handleRunConfigs } from "./routes/run-configs"
import { handleRunner } from "./routes/runner"
import { handlePorts } from "./routes/ports"
import { handleReadyUrlPatterns } from "./routes/ready-url-patterns"
import { handleSettings } from "./routes/settings"
import { isStandaloneBinary, openBrowser } from "../native/browser"
import { hasFrontendBuild, serveStatic } from "./static"

const preferredPort = Number(process.env.PORT || process.env.API_PORT) || 3000
// Prefer PORT / API_PORT when set, otherwise 3000. Always walk upward if busy
// so startup works when another app already holds the preferred port.
const port = await findAvailablePort(preferredPort)

if (port !== preferredPort) {
  console.log(`Port ${preferredPort} is in use — falling back to ${port}`)
}

const servingFrontend = hasFrontendBuild()

const corsHeaders = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Methods": "GET,POST,PUT,PATCH,DELETE,OPTIONS",
  "Access-Control-Allow-Headers": "Content-Type, Last-Event-ID",
}

function withCors(res: Response): Response {
  const headers = new Headers(res.headers)
  for (const [key, value] of Object.entries(corsHeaders)) {
    if (!headers.has(key)) headers.set(key, value)
  }
  return new Response(res.body, {
    status: res.status,
    statusText: res.statusText,
    headers,
  })
}

const server = Bun.serve({
  port,
  idleTimeout: 255,
  async fetch(req) {
    const { pathname } = new URL(req.url)

    if (req.method === "OPTIONS") {
      return new Response(null, { headers: corsHeaders })
    }

    if (req.method === "GET" && pathname === "/api/health") {
      return withCors(health())
    }

    const handlers = [
      handleWorkspaces,
      handleApps,
      handleFs,
      handleConfigSets,
      handleEnvVars,
      handleTemplates,
      handleRunConfigs,
      handleRunner,
      handlePorts,
      handleReadyUrlPatterns,
      handleSettings,
    ]

    for (const handler of handlers) {
      const res = await handler(req, pathname)
      if (res) return withCors(res)
    }

    if (pathname.startsWith("/api/")) {
      return withCors(Response.json({ error: "Not found" }, { status: 404 }))
    }

    if (req.method === "GET" || req.method === "HEAD") {
      const staticRes = await serveStatic(pathname)
      if (staticRes) return staticRes
    }

    return withCors(
      Response.json(
        {
          error: servingFrontend
            ? "Not found"
            : "Frontend not built. Run `bun run bun_backend:build` or use `bun run bun_backend:start`.",
        },
        { status: 404 }
      )
    )
  },
})

const url = `http://localhost:${server.port}`

if (servingFrontend) {
  console.log(`App listening on ${url}`)
  if (isStandaloneBinary()) {
    openBrowser(url)
  }
} else {
  console.log(`API listening on ${url}`)
  console.log(
    `(No dist/ build found — API only. Use bun run bun_backend:start for production.)`
  )
}
