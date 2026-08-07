/**
 * dev-web.ts — Vite frontend only (development).
 *
 * Entry: `bun run dev:web`
 *
 * What it does:
 * - Finds a free Vite port (prefers 5173, or `WEB_PORT` if set)
 * - Starts the Vite dev server with HMR
 * - Proxies `/api` to the Bun API at `API_PORT` (default 3000)
 *
 * Env (optional):
 * - `WEB_PORT` — preferred Vite port
 * - `API_PORT` — where the Bun API is listening (must match `dev:server`)
 *
 * Run `dev:server` in another terminal first (or use `dev.ts` for both).
 */
import { findAvailablePort } from "../native/ports"

const preferredWeb = Number(process.env.WEB_PORT) || 5173
const apiPort = Number(process.env.API_PORT) || 3000
const webPort = await findAvailablePort(preferredWeb)

if (webPort !== preferredWeb) {
  console.log(`Web port ${preferredWeb} busy — using ${webPort}`)
}

console.log(`Proxy /api → http://localhost:${apiPort}`)

const child = Bun.spawn(
  ["bunx", "vite", "--port", String(webPort), "--strictPort"],
  {
    stdout: "inherit",
    stderr: "inherit",
    stdin: "inherit",
    env: {
      ...process.env,
      WEB_PORT: String(webPort),
      API_PORT: String(apiPort),
      VITE_API_PORT: String(apiPort),
    },
  }
)

function shutdown() {
  child.kill()
  process.exit(0)
}

process.on("SIGINT", shutdown)
process.on("SIGTERM", shutdown)

process.exit(await child.exited)
