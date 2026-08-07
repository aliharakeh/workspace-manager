/**
 * dev.ts — local development (API + Vite together).
 *
 * Entry: `bun run dev`
 *
 * What it does:
 * - Picks free ports for the Bun API (prefers 3000) and Vite (prefers 5173)
 * - Starts `server/index.ts` with `--hot` on the API port
 * - Starts Vite on the web port with `/api` proxied to the API
 * - Forwards SIGINT/SIGTERM to both child processes on shutdown
 *
 * Env (optional):
 * - `API_PORT` / `PORT` — preferred API port
 * - `WEB_PORT` — preferred Vite port
 *
 * Use this for day-to-day UI work with HMR. For production, use `start.ts`.
 */
import { findAvailablePort } from "../native/ports"

const preferredApi = Number(process.env.API_PORT || process.env.PORT) || 3000
const preferredWeb = Number(process.env.WEB_PORT) || 5173

const apiPort = await findAvailablePort(preferredApi)
const webPort = await findAvailablePort(
  preferredWeb === apiPort ? preferredWeb + 1 : preferredWeb
)

if (apiPort !== preferredApi) {
  console.log(`API port ${preferredApi} busy — using ${apiPort}`)
}
if (webPort !== preferredWeb) {
  console.log(`Web port ${preferredWeb} busy — using ${webPort}`)
}

console.log(`Starting API on http://localhost:${apiPort}`)
console.log(`Starting web on http://localhost:${webPort}`)
console.log(`Proxy /api → http://localhost:${apiPort}`)

const children = [
  Bun.spawn(["bun", "run", "--hot", "server/index.ts"], {
    stdout: "inherit",
    stderr: "inherit",
    stdin: "inherit",
    env: {
      ...process.env,
      PORT: String(apiPort),
      API_PORT: String(apiPort),
    },
  }),
  Bun.spawn(["bunx", "vite", "--port", String(webPort), "--strictPort"], {
    stdout: "inherit",
    stderr: "inherit",
    stdin: "inherit",
    env: {
      ...process.env,
      PORT: String(webPort),
      WEB_PORT: String(webPort),
      API_PORT: String(apiPort),
      VITE_API_PORT: String(apiPort),
    },
  }),
]

function shutdown() {
  for (const child of children) {
    child.kill()
  }
  process.exit(0)
}

process.on("SIGINT", shutdown)
process.on("SIGTERM", shutdown)

const codes = await Promise.all(children.map((child) => child.exited))
process.exit(codes.find((code) => code !== 0) ?? 0)
