import { asc, eq, sql } from "drizzle-orm"
import { db } from "./index"
import { runCommands, runConfigs } from "./schema"
import type { RunConfig, RunConfigWithCommands, RunMode } from "./types"

function withCommands(config: RunConfig): RunConfigWithCommands {
  return {
    ...config,
    commands: db
      .select()
      .from(runCommands)
      .where(eq(runCommands.run_config_id, config.id))
      .orderBy(asc(runCommands.sort_order), asc(runCommands.id))
      .all(),
  }
}

export const runConfigsRepo = {
  getByConfigSet(configSetId: number): RunConfigWithCommands | null {
    const config = db
      .select()
      .from(runConfigs)
      .where(eq(runConfigs.config_set_id, configSetId))
      .get()
    if (!config) return null
    return withCommands(config)
  },

  getOrCreate(configSetId: number): RunConfigWithCommands {
    const existing = this.getByConfigSet(configSetId)
    if (existing) return existing

    const created = db
      .insert(runConfigs)
      .values({ config_set_id: configSetId, mode: "parallel" })
      .returning()
      .get()
    return withCommands(created)
  },

  upsert(
    configSetId: number,
    input: {
      mode?: RunMode
      commands?: Array<{ label?: string | null; command: string }>
    }
  ): RunConfigWithCommands {
    let config = db
      .select()
      .from(runConfigs)
      .where(eq(runConfigs.config_set_id, configSetId))
      .get()

    if (!config) {
      config = db
        .insert(runConfigs)
        .values({
          config_set_id: configSetId,
          mode: input.mode ?? "parallel",
        })
        .returning()
        .get()
    } else if (input.mode) {
      config = db
        .update(runConfigs)
        .set({ mode: input.mode, updated_at: sql`(datetime('now'))` })
        .where(eq(runConfigs.id, config.id))
        .returning()
        .get()
    }

    if (input.commands) {
      db.delete(runCommands)
        .where(eq(runCommands.run_config_id, config.id))
        .run()

      if (input.commands.length > 0) {
        db.insert(runCommands)
          .values(
            input.commands.map((cmd, index) => ({
              run_config_id: config.id,
              label: cmd.label ?? null,
              command: cmd.command,
              sort_order: index,
            }))
          )
          .run()
      }

      config = db
        .update(runConfigs)
        .set({ updated_at: sql`(datetime('now'))` })
        .where(eq(runConfigs.id, config.id))
        .returning()
        .get()
    }

    return withCommands(config)
  },
}
