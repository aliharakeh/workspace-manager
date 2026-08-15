import { eq, sql } from "drizzle-orm"
import { db } from "./index"
import { workspaces } from "./schema"
import type { Workspace } from "./types"

export const workspacesRepo = {
  list(): Workspace[] {
    return db
      .select()
      .from(workspaces)
      .orderBy(sql`${workspaces.name} COLLATE NOCASE ASC`)
      .all()
  },

  get(id: number): Workspace | null {
    return (
      db.select().from(workspaces).where(eq(workspaces.id, id)).get() ?? null
    )
  },

  create(input: { name: string; icon?: string | null }): Workspace {
    return db
      .insert(workspaces)
      .values({ name: input.name, icon: input.icon ?? null })
      .returning()
      .get()
  },

  update(
    id: number,
    input: { name?: string; icon?: string | null }
  ): Workspace | null {
    const existing = this.get(id)
    if (!existing) return null

    return db
      .update(workspaces)
      .set({
        name: input.name ?? existing.name,
        icon: input.icon !== undefined ? input.icon : existing.icon,
        updated_at: sql`(datetime('now'))`,
      })
      .where(eq(workspaces.id, id))
      .returning()
      .get()
  },

  delete(id: number): boolean {
    return db.delete(workspaces).where(eq(workspaces.id, id)).run().changes > 0
  },
}
