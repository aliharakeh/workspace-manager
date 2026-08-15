import { Database } from "bun:sqlite"
import { drizzle } from "drizzle-orm/bun-sqlite"
import { migrate } from "drizzle-orm/bun-sqlite/migrator"
import { mkdirSync, readFileSync, writeFileSync } from "node:fs"
import { dirname, join } from "node:path"
import { tmpdir } from "node:os"
import { migrationFiles } from "../embedded-assets"
import * as schema from "./schema"

const DATA_DIR = join(process.cwd(), "data")
const DB_PATH = join(DATA_DIR, "app-runner.sqlite")

function resolveMigrationsFolder(): string {
  const entries = Object.entries(migrationFiles)
  if (entries.length === 0) {
    return join(import.meta.dir, "../../drizzle")
  }

  // Drizzle's migrator needs a real directory; unpack embeds once per process.
  const folder = join(tmpdir(), "app-runner-v2-migrations")
  for (const [rel, embeddedPath] of entries) {
    const dest = join(folder, rel)
    mkdirSync(dirname(dest), { recursive: true })
    writeFileSync(dest, readFileSync(embeddedPath))
  }
  return folder
}

mkdirSync(DATA_DIR, { recursive: true })

const sqlite = new Database(DB_PATH, { create: true })
sqlite.exec("PRAGMA journal_mode = WAL;")
sqlite.exec("PRAGMA foreign_keys = ON;")

export const db = drizzle(sqlite, { schema })

migrate(db, { migrationsFolder: resolveMigrationsFolder() })
