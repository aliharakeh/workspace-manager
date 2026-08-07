import { eq, sql } from "drizzle-orm"
import { db } from "./index"
import { apps, configSets, runConfigs } from "./schema"
import type { App } from "./types"

export type AppWithConfigSet = App & {
  active_config_set_name: string | null
}

function withConfigSetName(app: App): AppWithConfigSet {
  let name: string | null = null
  if (app.active_config_set_id != null) {
    name =
      db
        .select({ name: configSets.name })
        .from(configSets)
        .where(eq(configSets.id, app.active_config_set_id))
        .get()?.name ?? null
  }
  return { ...app, active_config_set_name: name }
}

export const appsRepo = {
  listByWorkspace(workspaceId: number): AppWithConfigSet[] {
    return db
      .select()
      .from(apps)
      .where(eq(apps.workspace_id, workspaceId))
      .orderBy(sql`${apps.name} COLLATE NOCASE ASC`)
      .all()
      .map(withConfigSetName)
  },

  get(id: number): AppWithConfigSet | null {
    const app = db.select().from(apps).where(eq(apps.id, id)).get()
    return app ? withConfigSetName(app) : null
  },

  create(input: {
    workspace_id: number
    name: string
    project_path: string
  }): AppWithConfigSet {
    const app = db.insert(apps).values(input).returning().get()
    const set = db
      .insert(configSets)
      .values({ app_id: app.id, name: "Default" })
      .returning()
      .get()
    db.insert(runConfigs)
      .values({ config_set_id: set.id, mode: "parallel" })
      .run()
    return (
      this.setActiveConfigSet(app.id, set.id) ??
      withConfigSetName({ ...app, active_config_set_id: set.id })
    )
  },

  update(
    id: number,
    input: { name?: string; project_path?: string }
  ): AppWithConfigSet | null {
    if (!this.get(id)) return null
    const app = db
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
    return app ? withConfigSetName(app) : null
  },

  setActiveConfigSet(
    appId: number,
    configSetId: number
  ): AppWithConfigSet | null {
    const existing = db.select().from(apps).where(eq(apps.id, appId)).get()
    if (!existing) return null
    const app = db
      .update(apps)
      .set({
        active_config_set_id: configSetId,
        updated_at: sql`(datetime('now'))`,
      })
      .where(eq(apps.id, appId))
      .returning()
      .get()
    return app ? withConfigSetName(app) : null
  },

  delete(id: number): boolean {
    return db.delete(apps).where(eq(apps.id, id)).run().changes > 0
  },
}
