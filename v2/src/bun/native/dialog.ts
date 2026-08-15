import { Utils } from "electrobun/bun"

export type NativePickResult =
  | { ok: true; path: string }
  | { ok: false; cancelled: true }
  | { ok: false; error: string }

function firstPath(paths: string[] | string | null | undefined): string | null {
  const list = Array.isArray(paths) ? paths : paths ? [paths] : []
  const path = list.map((p) => p.trim()).find(Boolean)
  return path || null
}

async function pick(opts: {
  startDir?: string
  files: boolean
  directories: boolean
}): Promise<NativePickResult> {
  try {
    const paths = await Utils.openFileDialog({
      startingFolder: opts.startDir || Utils.paths.documents || Utils.paths.home,
      allowedFileTypes: "*",
      canChooseFiles: opts.files,
      canChooseDirectory: opts.directories,
      allowsMultipleSelection: false,
    })
    const path = firstPath(paths)
    if (!path) return { ok: false, cancelled: true }
    return { ok: true, path }
  } catch (err) {
    return {
      ok: false,
      error: err instanceof Error ? err.message : "Dialog failed",
    }
  }
}

export function pickNativeFile(startDir?: string) {
  return pick({ startDir, files: true, directories: false })
}

export function pickNativeFolder(startDir?: string) {
  return pick({ startDir, files: false, directories: true })
}
