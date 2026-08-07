import { eq, sql } from "drizzle-orm"
import { db } from "./index"
import { apps } from "./schema"
import type { App } from "./types"

export const appsRepo = {
  listByWorkspace(workspaceId: number): App[] {
    return db
      .select()
      .from(apps)
      .where(eq(apps.workspace_id, workspaceId))
      .orderBy(sql`${apps.name} COLLATE NOCASE ASC`)
      .all()
  },

  get(id: number): App | null {
    return db.select().from(apps).where(eq(apps.id, id)).get() ?? null
  },

  create(input: {
    workspace_id: number
    name: string
    project_path: string
  }): App {
    return db.insert(apps).values(input).returning().get()
  },

  update(
    id: number,
    input: { name?: string; project_path?: string }
  ): App | null {
    if (!this.get(id)) return null
    return db
      .update(apps)
      .set({
        ...(input.name !== undefined ? { name: input.name } : {}),
        ...(input.project_path !== undefined
          ? { project_path: input.project_path }
          : {}),
        updated_at: sql`(datetime('now'))`,
      })
      .where(eq(apps.id, id))
      .returning()
      .get()
  },

  delete(id: number): boolean {
    return db.delete(apps).where(eq(apps.id, id)).run().changes > 0
  },
}
