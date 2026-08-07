import { eq, sql } from "drizzle-orm"
import { db } from "./index"
import { envVars } from "./schema"
import type { EnvVar } from "./types"

export const envVarsRepo = {
  listByApp(appId: number): EnvVar[] {
    return db
      .select()
      .from(envVars)
      .where(eq(envVars.app_id, appId))
      .orderBy(sql`${envVars.key} COLLATE NOCASE ASC`)
      .all()
  },

  get(id: number): EnvVar | null {
    return db.select().from(envVars).where(eq(envVars.id, id)).get() ?? null
  },

  create(input: { app_id: number; key: string; value?: string }): EnvVar {
    return db
      .insert(envVars)
      .values({
        app_id: input.app_id,
        key: input.key,
        value: input.value ?? "",
      })
      .returning()
      .get()
  },

  update(
    id: number,
    input: { key?: string; value?: string }
  ): EnvVar | null {
    if (!this.get(id)) return null
    return db
      .update(envVars)
      .set({
        ...(input.key !== undefined ? { key: input.key } : {}),
        ...(input.value !== undefined ? { value: input.value } : {}),
        updated_at: sql`(datetime('now'))`,
      })
      .where(eq(envVars.id, id))
      .returning()
      .get()
  },

  delete(id: number): boolean {
    return db.delete(envVars).where(eq(envVars.id, id)).run().changes > 0
  },

  toRecord(appId: number): Record<string, string> {
    return Object.fromEntries(
      this.listByApp(appId).map((v) => [v.key, v.value])
    )
  },
}
