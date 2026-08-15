import { eq, sql } from "drizzle-orm"
import { db } from "./index"
import { apps, configSets, envVars, templates } from "./schema"
import type { ConfigSet } from "./types"
import { runConfigsRepo } from "./run-configs"
import { envVarsRepo } from "./env-vars"
import { templatesRepo } from "./templates"

/**
 * What to copy from a source set.
 * - `true` copies everything, `false` skips.
 * - An array copies only the listed items: env var keys, template file paths,
 *   or source run command ids.
 */
export type ConfigCopyParts = {
  env?: boolean | string[]
  templates?: boolean | string[]
  run?: boolean | number[]
}

type ResolvedPart = { all: boolean; items: string[] } | null
type ResolvedRunPart = { all: boolean; commandIds: number[] } | null

function resolveCategory(part: boolean | string[] | undefined): ResolvedPart {
  if (part === false) return null
  if (part === undefined || part === true) return { all: true, items: [] }
  return { all: false, items: part.filter((k): k is string => typeof k === "string") }
}

function resolveRun(part: boolean | number[] | undefined): ResolvedRunPart {
  if (part === false) return null
  if (part === undefined || part === true) return { all: true, commandIds: [] }
  return {
    all: false,
    commandIds: part.filter(
      (n): n is number => typeof n === "number" && Number.isInteger(n)
    ),
  }
}

function resolveParts(parts?: ConfigCopyParts): {
  env: ResolvedPart
  templates: ResolvedPart
  run: ResolvedRunPart
} {
  return {
    env: resolveCategory(parts?.env),
    templates: resolveCategory(parts?.templates),
    run: resolveRun(parts?.run),
  }
}

export const configSetsRepo = {
  listByApp(appId: number): ConfigSet[] {
    return db
      .select()
      .from(configSets)
      .where(eq(configSets.app_id, appId))
      .orderBy(sql`${configSets.name} COLLATE NOCASE ASC`)
      .all()
  },

  get(id: number): ConfigSet | null {
    return (
      db.select().from(configSets).where(eq(configSets.id, id)).get() ?? null
    )
  },

  create(input: { app_id: number; name: string }): ConfigSet {
    const set = db
      .insert(configSets)
      .values({ app_id: input.app_id, name: input.name })
      .returning()
      .get()
    runConfigsRepo.getOrCreate(set.id)
    return set
  },

  update(id: number, input: { name: string }): ConfigSet | null {
    if (!this.get(id)) return null
    return db
      .update(configSets)
      .set({ name: input.name, updated_at: sql`(datetime('now'))` })
      .where(eq(configSets.id, id))
      .returning()
      .get()
  },

  delete(id: number): boolean {
    return db.delete(configSets).where(eq(configSets.id, id)).run().changes > 0
  },

  /** Resolve active set for an app; create Default if missing. */
  resolveActive(appId: number): ConfigSet {
    const app = db.select().from(apps).where(eq(apps.id, appId)).get()
    if (!app) throw new Error("App not found")

    if (app.active_config_set_id) {
      const active = this.get(app.active_config_set_id)
      if (active && active.app_id === appId) return active
    }

    const existing = this.listByApp(appId)
    if (existing[0]) {
      db.update(apps)
        .set({
          active_config_set_id: existing[0].id,
          updated_at: sql`(datetime('now'))`,
        })
        .where(eq(apps.id, appId))
        .run()
      return existing[0]
    }

    const created = this.create({ app_id: appId, name: "Default" })
    db.update(apps)
      .set({
        active_config_set_id: created.id,
        updated_at: sql`(datetime('now'))`,
      })
      .where(eq(apps.id, appId))
      .run()
    return created
  },

  /** Replace selected parts of target with a copy of source. */
  copyFrom(
    sourceId: number,
    targetId: number,
    parts?: ConfigCopyParts
  ): void {
    const source = this.get(sourceId)
    const target = this.get(targetId)
    if (!source || !target) throw new Error("Config set not found")
    if (source.app_id !== target.app_id) {
      throw new Error("Config sets must belong to the same app")
    }
    if (sourceId === targetId) return

    const selected = resolveParts(parts)
    if (!selected.env && !selected.templates && !selected.run) {
      throw new Error("Select at least one part to copy")
    }

    if (selected.env) {
      const sourceEnv = envVarsRepo.listByConfigSet(sourceId)
      const chosen = selected.env.all
        ? sourceEnv
        : sourceEnv.filter((v) => selected.env.items.includes(v.key))
      db.delete(envVars).where(eq(envVars.config_set_id, targetId)).run()
      if (chosen.length > 0) {
        db.insert(envVars)
          .values(
            chosen.map((v) => ({
              config_set_id: targetId,
              key: v.key,
              value: v.value,
            }))
          )
          .run()
      }
    }

    if (selected.templates) {
      const sourceTemplates = templatesRepo.listByConfigSet(sourceId)
      const chosen = selected.templates.all
        ? sourceTemplates
        : sourceTemplates.filter((t) =>
            selected.templates.items.includes(t.file_path)
          )
      db.delete(templates).where(eq(templates.config_set_id, targetId)).run()
      if (chosen.length > 0) {
        db.insert(templates)
          .values(
            chosen.map((t) => ({
              config_set_id: targetId,
              file_path: t.file_path,
              content: t.content,
            }))
          )
          .run()
      }
    }

    if (selected.run) {
      const sourceRun = runConfigsRepo.getByConfigSet(sourceId)
      const commands = (sourceRun?.commands ?? []).filter(
        (c) => selected.run.all || selected.run.commandIds.includes(c.id)
      )
      runConfigsRepo.upsert(targetId, {
        mode: sourceRun?.mode ?? "parallel",
        commands: commands.map((c) => ({
          label: c.label,
          command: c.command,
        })),
      })
    }
  },
}
