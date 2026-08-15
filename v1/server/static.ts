import { existsSync } from "node:fs"
import { join, normalize, sep } from "node:path"
import { staticFiles } from "./embedded-assets"

const DIST_DIR = join(process.cwd(), "dist")
const hasEmbeds = Object.keys(staticFiles).length > 0

export function hasFrontendBuild() {
  if (hasEmbeds) return "index.html" in staticFiles
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

function embedKey(pathname: string): string {
  if (pathname === "/" || pathname === "") return "index.html"
  return decodeURIComponent(pathname.replace(/^\/+/, ""))
}

/** Serve Vite production assets (embedded or from `dist/`), with SPA fallback. */
export async function serveStatic(pathname: string): Promise<Response | null> {
  if (!hasFrontendBuild()) return null

  if (hasEmbeds) {
    const key = embedKey(pathname)
    const embedded = staticFiles[key]
    if (embedded) return new Response(Bun.file(embedded))

    const index = staticFiles["index.html"]
    if (index) {
      return new Response(Bun.file(index), {
        headers: { "Content-Type": "text/html; charset=utf-8" },
      })
    }
    return null
  }

  const filePath = resolveSafe(pathname)
  if (!filePath) {
    return new Response("Forbidden", { status: 403 })
  }

  const file = Bun.file(filePath)
  if (await file.exists()) {
    return new Response(file)
  }

  const index = Bun.file(join(DIST_DIR, "index.html"))
  if (await index.exists()) {
    return new Response(index, {
      headers: { "Content-Type": "text/html; charset=utf-8" },
    })
  }

  return null
}
