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
    // ponytail: no FK — avoids circular reference with config_sets.app_id
    active_config_set_id: integer("active_config_set_id"),
    ...timestamps,
  },
  (t) => [index("idx_apps_workspace_id").on(t.workspace_id)]
)

export const configSets = sqliteTable(
  "config_sets",
  {
    id: integer("id").primaryKey({ autoIncrement: true }),
    app_id: integer("app_id")
      .notNull()
      .references(() => apps.id, { onDelete: "cascade" }),
    name: text("name").notNull(),
    ...timestamps,
  },
  (t) => [
    unique().on(t.app_id, t.name),
    index("idx_config_sets_app_id").on(t.app_id),
  ]
)

export const envVars = sqliteTable(
  "env_vars",
  {
    id: integer("id").primaryKey({ autoIncrement: true }),
    config_set_id: integer("config_set_id")
      .notNull()
      .references(() => configSets.id, { onDelete: "cascade" }),
    key: text("key").notNull(),
    value: text("value").notNull().default(""),
    include_in_ai: integer("include_in_ai", { mode: "boolean" })
      .notNull()
      .default(true),
    ...timestamps,
  },
  (t) => [
    unique().on(t.config_set_id, t.key),
    index("idx_env_vars_config_set_id").on(t.config_set_id),
  ]
)

export const templates = sqliteTable(
  "templates",
  {
    id: integer("id").primaryKey({ autoIncrement: true }),
    config_set_id: integer("config_set_id")
      .notNull()
      .references(() => configSets.id, { onDelete: "cascade" }),
    file_path: text("file_path").notNull(),
    content: text("content").notNull().default(""),
    ...timestamps,
  },
  (t) => [
    unique().on(t.config_set_id, t.file_path),
    index("idx_templates_config_set_id").on(t.config_set_id),
  ]
)

export const runConfigs = sqliteTable("run_configs", {
  id: integer("id").primaryKey({ autoIncrement: true }),
  config_set_id: integer("config_set_id")
    .notNull()
    .unique()
    .references(() => configSets.id, { onDelete: "cascade" }),
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

/**
 * Flat key/value store for app-wide settings (e.g. keyboard shortcuts).
 * Values are stored as plain strings; consumers parse them.
 */
export const appSettings = sqliteTable("app_settings", {
  key: text("key").primaryKey(),
  value: text("value").notNull(),
  updated_at: text("updated_at")
    .notNull()
    .default(sql`(datetime('now'))`),
})

/** Regex patterns used to detect ready URLs from process logs. */
export const readyUrlPatterns = sqliteTable("ready_url_patterns", {
  id: integer("id").primaryKey({ autoIncrement: true }),
  /** Stable id for built-in defaults; null for user-created patterns. */
  key: text("key").unique(),
  label: text("label").notNull(),
  /** RegExp source; must include named group `url` and/or `port`. */
  pattern: text("pattern").notNull(),
  /** RegExp flags (default `i`). */
  flags: text("flags").notNull().default("i"),
  sort_order: integer("sort_order").notNull().default(0),
  ...timestamps,
})
