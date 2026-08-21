import { copyFileSync, existsSync, mkdirSync } from "node:fs"
import { homedir } from "node:os"
import { join } from "node:path"

const APP_DIR = "workspace-manager"
const DB_FILE = "workspace-manager.sqlite"

export function dataDir() {
  const home = homedir()
  switch (process.platform) {
    case "win32":
      return join(
        process.env.LOCALAPPDATA || join(home, "AppData", "Local"),
        APP_DIR
      )
    case "darwin":
      return join(home, "Library", "Application Support", APP_DIR)
    default:
      return join(
        process.env.XDG_DATA_HOME || join(home, ".local", "share"),
        APP_DIR
      )
  }
}

export function dbPath() {
  return join(dataDir(), DB_FILE)
}

function copySqlite(src: string, dest: string) {
  copyFileSync(src, dest)
  for (const suffix of ["-wal", "-shm"]) {
    const side = src + suffix
    if (existsSync(side)) copyFileSync(side, dest + suffix)
  }
}

/** One-shot copy from old bun_backend cwd/data. */
export function migrateLegacyDb(destDir: string) {
  const dest = join(destDir, DB_FILE)
  if (existsSync(dest)) return

  const src = join(process.cwd(), "data", DB_FILE)
  if (!existsSync(src)) return
  mkdirSync(destDir, { recursive: true })
  copySqlite(src, dest)
}
