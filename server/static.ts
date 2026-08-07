import { existsSync } from "node:fs"
import { join, normalize, sep } from "node:path"

const DIST_DIR = join(process.cwd(), "dist")

export function hasFrontendBuild() {
  return existsSync(join(DIST_DIR, "index.html"))
}

function resolveSafe(pathname: string): string | null {
  const relative =
    pathname === "/" || pathname === ""
      ? "index.html"
      : pathname.replace(/^\/+/, "")

  const full = normalize(join(DIST_DIR, decodeURIComponent(relative)))
  const root = normalize(DIST_DIR)

  if (full !== root && !full.startsWith(root + sep)) {
    return null
  }

  return full
}

/** Serve Vite production assets from `dist/`, with SPA fallback to index.html. */
export async function serveStatic(pathname: string): Promise<Response | null> {
  if (!hasFrontendBuild()) return null

  const filePath = resolveSafe(pathname)
  if (!filePath) {
    return new Response("Forbidden", { status: 403 })
  }

  const file = Bun.file(filePath)
  if (await file.exists()) {
    return new Response(file)
  }

  // Client-side routing fallback
  const index = Bun.file(join(DIST_DIR, "index.html"))
  if (await index.exists()) {
    return new Response(index, {
      headers: { "Content-Type": "text/html; charset=utf-8" },
    })
  }

  return null
}
