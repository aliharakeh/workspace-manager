import { Database } from "bun:sqlite"
import { drizzle } from "drizzle-orm/bun-sqlite"
import { migrate } from "drizzle-orm/bun-sqlite/migrator"
import { copyFileSync, existsSync, mkdirSync } from "node:fs"
import { tmpdir } from "node:os"
import { dirname, join } from "node:path"
import { migrationFiles } from "./embedded-migrations"
import { dataDir, dbPath, migrateLegacyDb } from "./paths"
import * as schema from "./schema"

function unpackEmbeds(): string | null {
  const entries = Object.entries(migrationFiles)
  if (entries.length === 0) return null

  const folder = join(tmpdir(), "workspace-manager-migrations")
  for (const [rel, embeddedPath] of entries) {
    const dest = join(folder, rel)
    mkdirSync(dirname(dest), { recursive: true })
    copyFileSync(embeddedPath, dest)
  }
  return folder
}

function resolveMigrationsFolder(): string {
  const unpacked = unpackEmbeds()
  if (unpacked) return unpacked

  const candidates = [
    join(import.meta.dir, "../drizzle"),
    join(process.cwd(), "drizzle"),
    join(process.cwd(), "../drizzle"),
  ]
  for (const dir of candidates) {
    if (existsSync(join(dir, "meta", "_journal.json"))) return dir
  }
  return candidates[0]!
}

const DATA_DIR = dataDir()
mkdirSync(DATA_DIR, { recursive: true })
migrateLegacyDb(DATA_DIR)

const sqlite = new Database(dbPath(), { create: true })
sqlite.exec("PRAGMA journal_mode = WAL;")
sqlite.exec("PRAGMA foreign_keys = ON;")

export const db = drizzle(sqlite, { schema })

migrate(db, { migrationsFolder: resolveMigrationsFolder() })
