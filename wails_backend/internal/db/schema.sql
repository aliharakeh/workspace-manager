-- Current schema (post drizzle 0000–0004). sqlc only; runtime applies drizzle SQL.
CREATE TABLE workspaces (
  id integer PRIMARY KEY AUTOINCREMENT NOT NULL,
  name text NOT NULL,
  icon text,
  created_at text DEFAULT (datetime('now')) NOT NULL,
  updated_at text DEFAULT (datetime('now')) NOT NULL
);

CREATE TABLE apps (
  id integer PRIMARY KEY AUTOINCREMENT NOT NULL,
  workspace_id integer NOT NULL,
  name text NOT NULL,
  project_path text NOT NULL,
  active_config_set_id integer,
  created_at text DEFAULT (datetime('now')) NOT NULL,
  updated_at text DEFAULT (datetime('now')) NOT NULL,
  FOREIGN KEY (workspace_id) REFERENCES workspaces (id) ON DELETE cascade
);

CREATE INDEX idx_apps_workspace_id ON apps (workspace_id);

CREATE TABLE config_sets (
  id integer PRIMARY KEY AUTOINCREMENT NOT NULL,
  app_id integer NOT NULL,
  name text NOT NULL,
  created_at text DEFAULT (datetime('now')) NOT NULL,
  updated_at text DEFAULT (datetime('now')) NOT NULL,
  FOREIGN KEY (app_id) REFERENCES apps (id) ON DELETE cascade
);

CREATE INDEX idx_config_sets_app_id ON config_sets (app_id);
CREATE UNIQUE INDEX config_sets_app_id_name_unique ON config_sets (app_id, name);

CREATE TABLE env_vars (
  id integer PRIMARY KEY AUTOINCREMENT NOT NULL,
  config_set_id integer NOT NULL,
  key text NOT NULL,
  value text DEFAULT '' NOT NULL,
  include_in_ai integer DEFAULT true NOT NULL,
  created_at text DEFAULT (datetime('now')) NOT NULL,
  updated_at text DEFAULT (datetime('now')) NOT NULL,
  FOREIGN KEY (config_set_id) REFERENCES config_sets (id) ON DELETE cascade
);

CREATE INDEX idx_env_vars_config_set_id ON env_vars (config_set_id);
CREATE UNIQUE INDEX env_vars_config_set_id_key_unique ON env_vars (config_set_id, key);

CREATE TABLE templates (
  id integer PRIMARY KEY AUTOINCREMENT NOT NULL,
  config_set_id integer NOT NULL,
  file_path text NOT NULL,
  content text DEFAULT '' NOT NULL,
  created_at text DEFAULT (datetime('now')) NOT NULL,
  updated_at text DEFAULT (datetime('now')) NOT NULL,
  FOREIGN KEY (config_set_id) REFERENCES config_sets (id) ON DELETE cascade
);

CREATE INDEX idx_templates_config_set_id ON templates (config_set_id);
CREATE UNIQUE INDEX templates_config_set_id_file_path_unique ON templates (config_set_id, file_path);

CREATE TABLE run_configs (
  id integer PRIMARY KEY AUTOINCREMENT NOT NULL,
  config_set_id integer NOT NULL,
  mode text DEFAULT 'parallel' NOT NULL,
  created_at text DEFAULT (datetime('now')) NOT NULL,
  updated_at text DEFAULT (datetime('now')) NOT NULL,
  FOREIGN KEY (config_set_id) REFERENCES config_sets (id) ON DELETE cascade
);

CREATE UNIQUE INDEX run_configs_config_set_id_unique ON run_configs (config_set_id);

CREATE TABLE run_commands (
  id integer PRIMARY KEY AUTOINCREMENT NOT NULL,
  run_config_id integer NOT NULL,
  label text,
  command text NOT NULL,
  sort_order integer DEFAULT 0 NOT NULL,
  created_at text DEFAULT (datetime('now')) NOT NULL,
  updated_at text DEFAULT (datetime('now')) NOT NULL,
  FOREIGN KEY (run_config_id) REFERENCES run_configs (id) ON DELETE cascade
);

CREATE INDEX idx_run_commands_config_id ON run_commands (run_config_id);

CREATE TABLE app_settings (
  key text PRIMARY KEY NOT NULL,
  value text NOT NULL,
  updated_at text DEFAULT (datetime('now')) NOT NULL
);

CREATE TABLE ready_url_patterns (
  id integer PRIMARY KEY AUTOINCREMENT NOT NULL,
  key text,
  label text NOT NULL,
  pattern text NOT NULL,
  flags text DEFAULT 'i' NOT NULL,
  sort_order integer DEFAULT 0 NOT NULL,
  created_at text DEFAULT (datetime('now')) NOT NULL,
  updated_at text DEFAULT (datetime('now')) NOT NULL
);

CREATE UNIQUE INDEX ready_url_patterns_key_unique ON ready_url_patterns (key);
