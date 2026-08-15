/**
 * dev-server.ts — Bun API only (development).
 *
 * Entry: `bun run dev:server`
 *
 * What it does:
 * - Finds a free port (prefers 3000, or `API_PORT` / `PORT` if set)
 * - Starts `server/index.ts` with `--hot` for live reload of API code
 * - Does not start Vite; pair with `dev-web.ts` if you need the UI
 *
 * Env (optional):
 * - `API_PORT` / `PORT` — preferred listen port
 *
 * Without a `dist/` build this is API-only. With `dist/` present the server
 * can also serve the production frontend from the same origin.
 */
import { findAvailablePort } from "../native/ports"

const preferred = Number(process.env.API_PORT || process.env.PORT) || 3000
const port = await findAvailablePort(preferred)

if (port !== preferred) {
  console.log(`API port ${preferred} busy — using ${port}`)
}

const child = Bun.spawn(["bun", "run", "--hot", "server/index.ts"], {
  stdout: "inherit",
  stderr: "inherit",
  stdin: "inherit",
  env: {
    ...process.env,
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
