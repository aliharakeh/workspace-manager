/**
 * start.ts — production launcher (build + single Bun server).
 *
 * Entry: `bun run start`
 *
 * What it does:
 * 1. Runs `bun run build` (TypeScript check + Vite production bundle → `dist/`)
 * 2. Picks a free listen port (prefers 3000, or `PORT` / `API_PORT`)
 * 3. Starts `server/index.ts` in production mode
 *
 * The Bun server serves both `/api/*` and the built SPA from `dist/` on one
 * origin. Vite is not started.
 *
 * Env (optional):
 * - `PORT` / `API_PORT` — preferred listen port
 *
 * If you already built (`bun run build`), use `bun run start:server` to skip
 * the rebuild and only run the server.
 */
import { findAvailablePort } from "../native/ports"

console.log("Building frontend…")
const build = Bun.spawn(["bun", "run", "build"], {
  stdout: "inherit",
  stderr: "inherit",
  stdin: "inherit",
})
const buildCode = await build.exited
if (buildCode !== 0) {
  console.error("Frontend build failed")
  process.exit(buildCode)
}

const preferred = Number(process.env.PORT || process.env.API_PORT) || 3000
const port = await findAvailablePort(preferred)

if (port !== preferred) {
  console.log(`Port ${preferred} busy — using ${port}`)
}

console.log(`Starting production server on http://localhost:${port}`)

const child = Bun.spawn(["bun", "run", "server/index.ts"], {
  stdout: "inherit",
  stderr: "inherit",
  stdin: "inherit",
  env: {
    ...process.env,
    NODE_ENV: "production",
    PORT: String(port),
    API_PORT: String(port),
  },
})

function shutdown() {
  child.kill()
  process.exit(0)
}

process.on("SIGINT", shutdown)
process.on("SIGTERM", shutdown)

process.exit(await child.exited)
