import { eq, sql } from "drizzle-orm"
import { db } from "./index"
import { envVars } from "./schema"
import type { EnvVar } from "./types"

export const envVarsRepo = {
  listByConfigSet(configSetId: number): EnvVar[] {
    return db
      .select()
      .from(envVars)
      .where(eq(envVars.config_set_id, configSetId))
      .orderBy(sql`${envVars.key} COLLATE NOCASE ASC`)
      .all()
  },

  get(id: number): EnvVar | null {
    return db.select().from(envVars).where(eq(envVars.id, id)).get() ?? null
  },

  create(input: {
    config_set_id: number
    key: string
    value?: string
  }): EnvVar {
    return db
      .insert(envVars)
      .values({
        config_set_id: input.config_set_id,
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

  toRecord(configSetId: number): Record<string, string> {
    return Object.fromEntries(
      this.listByConfigSet(configSetId).map((v) => [v.key, v.value])
    )
  },
}
