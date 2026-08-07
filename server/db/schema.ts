import { sql } from "drizzle-orm"
import {
  index,
  integer,
  sqliteTable,
  text,
  unique,
} from "drizzle-orm/sqlite-core"

const timestamps = {
  created_at: text("created_at")
    .notNull()
    .default(sql`(datetime('now'))`),
  updated_at: text("updated_at")
    .notNull()
    .default(sql`(datetime('now'))`),
}

export const workspaces = sqliteTable("workspaces", {
  id: integer("id").primaryKey({ autoIncrement: true }),
  name: text("name").notNull(),
  icon: text("icon"),
  ...timestamps,
})

export const apps = sqliteTable(
  "apps",
  {
    id: integer("id").primaryKey({ autoIncrement: true }),
    workspace_id: integer("workspace_id")
      .notNull()
      .references(() => workspaces.id, { onDelete: "cascade" }),
    name: text("name").notNull(),
    project_path: text("project_path").notNull(),
    ...timestamps,
  },
  (t) => [index("idx_apps_workspace_id").on(t.workspace_id)]
)

export const envVars = sqliteTable(
  "env_vars",
  {
    id: integer("id").primaryKey({ autoIncrement: true }),
    app_id: integer("app_id")
      .notNull()
      .references(() => apps.id, { onDelete: "cascade" }),
    key: text("key").notNull(),
    value: text("value").notNull().default(""),
    ...timestamps,
  },
  (t) => [
    unique().on(t.app_id, t.key),
    index("idx_env_vars_app_id").on(t.app_id),
  ]
)

export const templates = sqliteTable(
  "templates",
  {
    id: integer("id").primaryKey({ autoIncrement: true }),
    app_id: integer("app_id")
      .notNull()
      .references(() => apps.id, { onDelete: "cascade" }),
    file_path: text("file_path").notNull(),
    content: text("content").notNull().default(""),
    ...timestamps,
  },
  (t) => [
    unique().on(t.app_id, t.file_path),
    index("idx_templates_app_id").on(t.app_id),
  ]
)

export const runConfigs = sqliteTable("run_configs", {
  id: integer("id").primaryKey({ autoIncrement: true }),
  app_id: integer("app_id")
    .notNull()
    .unique()
    .references(() => apps.id, { onDelete: "cascade" }),
  mode: text("mode", { enum: ["sequential", "parallel"] })
    .notNull()
    .default("parallel"),
  ...timestamps,
})

export const runCommands = sqliteTable(
  "run_commands",
  {
    id: integer("id").primaryKey({ autoIncrement: true }),
    run_config_id: integer("run_config_id")
      .notNull()
      .references(() => runConfigs.id, { onDelete: "cascade" }),
    label: text("label"),
    command: text("command").notNull(),
    sort_order: integer("sort_order").notNull().default(0),
    ...timestamps,
  },
  (t) => [index("idx_run_commands_config_id").on(t.run_config_id)]
)
