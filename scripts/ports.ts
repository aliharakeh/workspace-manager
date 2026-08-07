/**
 * ports.ts — shared port helper for the other launch scripts.
 *
 * Finds the first free TCP port starting at a preferred value (e.g. 3000),
 * then walking upward if that port is already in use. Used so API / Vite
 * startup does not crash with EADDRINUSE.
 *
 * Not run directly; imported by `dev.ts`, `dev-server.ts`, `dev-web.ts`,
 * and `start.ts`.
 */
import { createServer } from "node:net"

/** Returns the first free TCP port at or after `preferred`. */
export function findAvailablePort(
  preferred: number,
  host = "127.0.0.1",
  maxAttempts = 40
): Promise<number> {
  return new Promise((resolve, reject) => {
    const tryListen = (port: number, remaining: number) => {
      const server = createServer()
      server.unref()

      server.once("error", (err: NodeJS.ErrnoException) => {
        if ((err.code === "EADDRINUSE" || err.code === "EACCES") && remaining > 0) {
          tryListen(port + 1, remaining - 1)
          return
        }
        reject(err)
      })

      server.listen(port, host, () => {
        const address = server.address()
        const chosen =
          typeof address === "object" && address ? address.port : port
        server.close((closeErr) => {
          if (closeErr) {
            reject(closeErr)
            return
          }
          resolve(chosen)
        })
      })
    }

    tryListen(preferred, maxAttempts)
  })
}
