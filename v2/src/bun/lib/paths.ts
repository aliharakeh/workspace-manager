import { existsSync, mkdirSync } from "node:fs"
import { join } from "node:path"
import { Utils } from "electrobun/bun"

export function dataDir() {
  const root = Utils.paths.userData || join(process.cwd(), "data")
  mkdirSync(root, { recursive: true })
  return root
}

export function migrationsFolder() {
  const candidates = [
    join(import.meta.dir, "../drizzle"),
    join(import.meta.dir, "../../drizzle"),
    join(process.cwd(), "drizzle"),
  ]
  for (const dir of candidates) {
    if (existsSync(join(dir, "meta", "_journal.json"))) return dir
  }
  return candidates[0]!
}
