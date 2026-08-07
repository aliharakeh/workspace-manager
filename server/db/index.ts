import { Database } from "bun:sqlite"
import { drizzle } from "drizzle-orm/bun-sqlite"
import { migrate } from "drizzle-orm/bun-sqlite/migrator"
import { mkdirSync } from "node:fs"
import { join } from "node:path"
import * as schema from "./schema"

const DATA_DIR = join(process.cwd(), "data")
const DB_PATH = join(DATA_DIR, "app-runner.sqlite")
const MIGRATIONS_FOLDER = join(import.meta.dir, "../../drizzle")

mkdirSync(DATA_DIR, { recursive: true })

const sqlite = new Database(DB_PATH, { create: true })
sqlite.exec("PRAGMA journal_mode = WAL;")
sqlite.exec("PRAGMA foreign_keys = ON;")

export const db = drizzle(sqlite, { schema })

migrate(db, { migrationsFolder: MIGRATIONS_FOLDER })
