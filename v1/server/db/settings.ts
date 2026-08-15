import { eq, sql } from "drizzle-orm"
import { db } from "./index"
import { appSettings } from "./schema"

export const settingsRepo = {
  /** All settings as a flat `key -> value` map. */
  getAll(): Record<string, string> {
    const rows = db.select().from(appSettings).all()
    return Object.fromEntries(rows.map((row) => [row.key, row.value]))
  },

  get(key: string): string | null {
    const row = db.select().from(appSettings).where(eq(appSettings.key, key)).get()
    return row?.value ?? null
  },

  set(key: string, value: string): void {
    db.insert(appSettings)
      .values({ key, value })
      .onConflictDoUpdate({
        target: appSettings.key,
        set: { value, updated_at: sql`(datetime('now'))` },
      })
      .run()
  },

  delete(key: string): void {
    db.delete(appSettings).where(eq(appSettings.key, key)).run()
  },
}
