import { api } from "@/lib/api"

/** The side, in pixels, an uploaded logo is scaled to before it is saved. */
const LOGO_SIZE = 128

// Logos are files the daemon keeps; the GUI reads one through
// workspaces.logo. A new upload gets a new file name (logo_path), so a
// workspace id plus its logo_path identifies an image for good and can be
// cached for the session.
const loaded = new Map<string, string | null>()
const pending = new Map<string, Promise<string | null>>()

/** The cache key for a workspace's current logo, or "" when it has none. */
export function logoKey(workspace: { id: number; logo_path: string } | null | undefined): string {
  return workspace?.logo_path ? `${workspace.id}:${workspace.logo_path}` : ""
}

/** The cached image for a key: a data URL, null (unreadable) or undefined (not loaded yet). */
export function cachedLogo(key: string): string | null | undefined {
  return loaded.get(key)
}

/** Loads a workspace's logo as a data URL once per key; null when it cannot be read. */
export function loadWorkspaceLogo(id: number, key: string): Promise<string | null> {
  const done = loaded.get(key)
  if (done !== undefined) return Promise.resolve(done)
  let p = pending.get(key)
  if (!p) {
    p = api.workspaces
      .logo(id)
      .then((src) => src || null)
      .catch(() => null)
      .then((src) => {
        loaded.set(key, src)
        pending.delete(key)
        return src
      })
    pending.set(key, p)
  }
  return p
}

/**
 * Reads an image file and returns it as a small square PNG data URL: the
 * image is scaled to fit LOGO_SIZE×LOGO_SIZE, centred, on a transparent
 * background. Any format the webview can draw works (SVG included); the
 * result is always a raster PNG, which the daemon saves as the logo file.
 */
export async function readLogoFile(file: File): Promise<string> {
  if (!file.type.startsWith("image/")) {
    throw new Error("Choose an image file")
  }
  const url = URL.createObjectURL(file)
  try {
    const img = new Image()
    img.decoding = "async"
    img.src = url
    try {
      await img.decode()
    } catch {
      throw new Error("That image could not be read")
    }
    // An SVG without width/height reports 0; draw it at the target size.
    const w = img.naturalWidth || LOGO_SIZE
    const h = img.naturalHeight || LOGO_SIZE
    const scale = Math.min(LOGO_SIZE / w, LOGO_SIZE / h)
    const dw = Math.max(1, Math.round(w * scale))
    const dh = Math.max(1, Math.round(h * scale))
    const canvas = document.createElement("canvas")
    canvas.width = LOGO_SIZE
    canvas.height = LOGO_SIZE
    const ctx = canvas.getContext("2d")
    if (!ctx) throw new Error("That image could not be read")
    ctx.imageSmoothingQuality = "high"
    ctx.drawImage(img, (LOGO_SIZE - dw) / 2, (LOGO_SIZE - dh) / 2, dw, dh)
    return canvas.toDataURL("image/png")
  } finally {
    URL.revokeObjectURL(url)
  }
}
