import { eq, sql } from "drizzle-orm"
import { db } from "./index"
import { templates } from "./schema"
import type { Template } from "./types"

export const templatesRepo = {
  listByConfigSet(configSetId: number): Template[] {
    return db
      .select()
      .from(templates)
      .where(eq(templates.config_set_id, configSetId))
      .orderBy(sql`${templates.file_path} COLLATE NOCASE ASC`)
      .all()
  },

  get(id: number): Template | null {
    return (
      db.select().from(templates).where(eq(templates.id, id)).get() ?? null
    )
  },

  create(input: {
    config_set_id: number
    file_path: string
    content?: string
  }): Template {
    return db
      .insert(templates)
      .values({
        config_set_id: input.config_set_id,
        file_path: input.file_path,
        content: input.content ?? "",
      })
      .returning()
      .get()
  },

  update(
    id: number,
    input: { file_path?: string; content?: string }
  ): Template | null {
    if (!this.get(id)) return null
    return db
      .update(templates)
      .set({
        ...(input.file_path !== undefined ? { file_path: input.file_path } : {}),
        ...(input.content !== undefined ? { content: input.content } : {}),
        updated_at: sql`(datetime('now'))`,
      })
      .where(eq(templates.id, id))
      .returning()
      .get()
  },

  delete(id: number): boolean {
    return db.delete(templates).where(eq(templates.id, id)).run().changes > 0
  },
}
