-- workspaces (db/workspaces.ts)
-- name: ListWorkspaces :many
SELECT * FROM workspaces ORDER BY sort_order ASC, id ASC;

-- name: GetWorkspace :one
SELECT * FROM workspaces WHERE id = ?;

-- name: CreateWorkspace :one
INSERT INTO workspaces (name, icon, color, sort_order)
VALUES (
  sqlc.arg(name), sqlc.arg(icon), sqlc.arg(color),
  (SELECT COALESCE(max(sort_order), -1) + 1 FROM workspaces)
)
RETURNING *;

-- name: SetWorkspaceSortOrder :exec
UPDATE workspaces SET sort_order = ? WHERE id = ?;

-- name: SetWorkspacePinned :execrows
UPDATE workspaces SET pinned = ? WHERE id = ?;

-- name: UpdateWorkspace :one
UPDATE workspaces
SET name = ?, icon = ?, color = ?, updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: DeleteWorkspace :execrows
DELETE FROM workspaces WHERE id = ?;

-- apps (db/apps.ts)
-- name: ListAppsByWorkspace :many
SELECT
  a.id, a.workspace_id, a.name, a.project_path, a.active_config_set_id,
  a.created_at, a.updated_at, a.exclude_from_start_all,
  cs.name AS active_config_set_name
FROM apps a
LEFT JOIN config_sets cs ON cs.id = a.active_config_set_id
WHERE a.workspace_id = ?
ORDER BY a.sort_order ASC, a.id ASC;

-- name: GetApp :one
SELECT
  a.id, a.workspace_id, a.name, a.project_path, a.active_config_set_id,
  a.created_at, a.updated_at, a.exclude_from_start_all,
  cs.name AS active_config_set_name
FROM apps a
LEFT JOIN config_sets cs ON cs.id = a.active_config_set_id
WHERE a.id = ?;

-- name: CreateApp :one
INSERT INTO apps (workspace_id, name, project_path, sort_order)
VALUES (
  sqlc.arg(workspace_id), sqlc.arg(name), sqlc.arg(project_path),
  (SELECT COALESCE(max(sort_order), -1) + 1 FROM apps WHERE workspace_id = sqlc.arg(workspace_id))
)
RETURNING *;

-- name: SetAppSortOrder :exec
UPDATE apps SET sort_order = ? WHERE id = ? AND workspace_id = ?;

-- name: UpdateApp :one
UPDATE apps
SET name = ?, project_path = ?, updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: SetActiveConfigSet :one
UPDATE apps
SET active_config_set_id = ?, updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: SetAppExcludeFromStartAll :exec
UPDATE apps SET exclude_from_start_all = ?, updated_at = datetime('now') WHERE id = ?;

-- name: DeleteApp :execrows
DELETE FROM apps WHERE id = ?;

-- config_sets (db/config-sets.ts)
-- name: ListConfigSetsByApp :many
SELECT * FROM config_sets WHERE app_id = ? ORDER BY name COLLATE NOCASE ASC;

-- name: GetConfigSet :one
SELECT * FROM config_sets WHERE id = ?;

-- name: CreateConfigSet :one
INSERT INTO config_sets (app_id, name) VALUES (?, ?) RETURNING *;

-- name: UpdateConfigSet :one
UPDATE config_sets
SET name = ?, updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: DeleteConfigSet :execrows
DELETE FROM config_sets WHERE id = ?;

-- env_vars (db/env-vars.ts)
-- name: ListEnvVarsByConfigSet :many
SELECT * FROM env_vars WHERE config_set_id = ? ORDER BY key COLLATE NOCASE ASC;

-- name: GetEnvVar :one
SELECT * FROM env_vars WHERE id = ?;

-- name: GetEnvVarByKey :one
SELECT * FROM env_vars WHERE config_set_id = ? AND key = ?;

-- name: CreateEnvVar :one
INSERT INTO env_vars (config_set_id, key, value, private, options, selected) VALUES (?, ?, ?, ?, ?, ?) RETURNING *;

-- name: UpdateEnvVar :one
UPDATE env_vars
SET key = ?, value = ?, private = ?, options = ?, selected = ?, updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: DeleteEnvVar :execrows
DELETE FROM env_vars WHERE id = ?;

-- name: DeleteOwnEnvVarsByConfigSet :exec
DELETE FROM env_vars WHERE config_set_id = ? AND shared_var_id IS NULL;

-- templates (db/templates.ts)
-- name: ListTemplatesByConfigSet :many
SELECT * FROM templates WHERE config_set_id = ? ORDER BY file_path COLLATE NOCASE ASC;

-- name: GetTemplate :one
SELECT * FROM templates WHERE id = ?;

-- name: GetTemplateByPath :one
SELECT * FROM templates WHERE config_set_id = ? AND file_path = ?;

-- name: CreateTemplate :one
INSERT INTO templates (config_set_id, file_path, content, private) VALUES (?, ?, ?, ?) RETURNING *;

-- name: UpdateTemplate :one
UPDATE templates
SET file_path = ?, content = ?, private = ?, updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: DeleteTemplate :execrows
DELETE FROM templates WHERE id = ?;

-- name: DeleteTemplatesByConfigSet :exec
DELETE FROM templates WHERE config_set_id = ?;

-- run_configs (db/run-configs.ts)
-- name: GetRunConfigByConfigSet :one
SELECT * FROM run_configs WHERE config_set_id = ? AND kind = ?;

-- name: CreateRunConfig :one
INSERT INTO run_configs (config_set_id, kind, mode) VALUES (?, ?, ?) RETURNING *;

-- name: UpdateRunConfigMode :one
UPDATE run_configs
SET mode = ?, updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: TouchRunConfig :one
UPDATE run_configs
SET updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: ListRunCommands :many
SELECT * FROM run_commands
WHERE run_config_id = ?
ORDER BY sort_order ASC, id ASC;

-- name: DeleteRunCommands :exec
DELETE FROM run_commands WHERE run_config_id = ?;

-- name: CreateRunCommand :one
INSERT INTO run_commands (run_config_id, label, command, sort_order)
VALUES (?, ?, ?, ?)
RETURNING *;

-- settings (db/settings.ts)
-- name: ListSettings :many
SELECT * FROM app_settings;

-- name: GetSetting :one
SELECT * FROM app_settings WHERE key = ?;

-- name: UpsertSetting :exec
INSERT INTO app_settings (key, value)
VALUES (?, ?)
ON CONFLICT (key) DO UPDATE SET value = excluded.value, updated_at = datetime('now');

-- name: DeleteSetting :exec
DELETE FROM app_settings WHERE key = ?;

-- ready_url_patterns (db/ready-url-patterns.ts)
-- name: ListReadyUrlPatterns :many
SELECT * FROM ready_url_patterns ORDER BY sort_order ASC, id ASC;

-- name: GetReadyUrlPattern :one
SELECT * FROM ready_url_patterns WHERE id = ?;

-- name: MaxReadyUrlSortOrder :one
SELECT COALESCE(max(sort_order), -1) AS max_order FROM ready_url_patterns;

-- name: CreateReadyUrlPattern :one
INSERT INTO ready_url_patterns (key, label, pattern, flags, sort_order)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateReadyUrlPattern :one
UPDATE ready_url_patterns
SET label = ?, pattern = ?, flags = ?, updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: UpdateReadyUrlPatternSeed :exec
UPDATE ready_url_patterns
SET key = ?, label = ?, flags = ?, sort_order = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteReadyUrlPattern :execrows
DELETE FROM ready_url_patterns WHERE id = ?;

-- blueprints
-- name: ListBlueprints :many
SELECT * FROM blueprints ORDER BY name COLLATE NOCASE ASC;

-- name: GetBlueprint :one
SELECT * FROM blueprints WHERE id = ?;

-- name: CreateBlueprint :one
INSERT INTO blueprints (name, description, sample_name, create_folder, commands)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateBlueprint :one
UPDATE blueprints
SET name = ?, description = ?, sample_name = ?, create_folder = ?, commands = ?, updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: DeleteBlueprint :execrows
DELETE FROM blueprints WHERE id = ?;

-- worktree_links
-- name: ListWorktreeLinks :many
SELECT path FROM worktree_links WHERE app_id = ? ORDER BY sort_order ASC, id ASC;

-- name: DeleteWorktreeLinks :exec
DELETE FROM worktree_links WHERE app_id = ?;

-- name: CreateWorktreeLink :exec
INSERT INTO worktree_links (app_id, path, sort_order) VALUES (?, ?, ?);

-- workspace shared config
-- name: ListWorkspaceConfigSets :many
SELECT * FROM workspace_config_sets WHERE workspace_id = ? ORDER BY name COLLATE NOCASE ASC;

-- name: GetWorkspaceConfigSet :one
SELECT * FROM workspace_config_sets WHERE id = ?;

-- name: CreateWorkspaceConfigSet :one
INSERT INTO workspace_config_sets (workspace_id, name) VALUES (?, ?) RETURNING *;

-- name: UpdateWorkspaceConfigSet :one
UPDATE workspace_config_sets SET name = ?, updated_at = datetime('now') WHERE id = ? RETURNING *;

-- name: DeleteWorkspaceConfigSet :execrows
DELETE FROM workspace_config_sets WHERE id = ?;

-- name: ListWorkspaceEnvVars :many
SELECT * FROM workspace_env_vars WHERE workspace_config_set_id = ? ORDER BY key COLLATE NOCASE ASC;

-- name: GetWorkspaceEnvVar :one
SELECT * FROM workspace_env_vars WHERE id = ?;

-- name: CreateWorkspaceEnvVar :one
INSERT INTO workspace_env_vars (workspace_config_set_id, key, value, private, options, selected) VALUES (?, ?, ?, ?, ?, ?) RETURNING *;

-- name: UpdateWorkspaceEnvVar :one
UPDATE workspace_env_vars
SET key = ?, value = ?, private = ?, options = ?, selected = ?, updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: DeleteWorkspaceEnvVar :execrows
DELETE FROM workspace_env_vars WHERE id = ?;

-- name: SetConfigSetSharedSet :exec
UPDATE config_sets SET shared_set_id = ?, updated_at = datetime('now') WHERE id = ?;

-- name: CreateSharedEnvVar :one
INSERT INTO env_vars (config_set_id, key, value, private, options, selected, shared_var_id) VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING *;

-- name: PropagateSharedEnvVar :exec
UPDATE env_vars
SET key = ?, value = ?, private = ?, options = ?, selected = ?, updated_at = datetime('now')
WHERE shared_var_id = ?;

-- name: DeleteSharedEnvVarsByConfigSet :exec
DELETE FROM env_vars WHERE config_set_id = ? AND shared_var_id IS NOT NULL;

-- name: DetachSharedEnvVarsByConfigSet :exec
UPDATE env_vars SET shared_var_id = NULL WHERE config_set_id = ? AND shared_var_id IS NOT NULL;

-- name: ListSharedUsageBySet :many
SELECT ev.shared_var_id AS shared_var_id, cs.id AS config_set_id, cs.name AS config_set_name, a.id AS app_id, a.name AS app_name
FROM env_vars ev
JOIN config_sets cs ON cs.id = ev.config_set_id
JOIN apps a ON a.id = cs.app_id
WHERE ev.shared_var_id IN (SELECT id FROM workspace_env_vars WHERE workspace_config_set_id = ?)
ORDER BY a.name COLLATE NOCASE ASC, cs.name COLLATE NOCASE ASC;

-- name: ListAppsLinkedToSharedSet :many
SELECT DISTINCT app_id FROM config_sets WHERE shared_set_id = ?;

-- name: ListSharedRenameConflicts :many
SELECT a.name AS app_name, cs.name AS config_set_name
FROM env_vars lv
JOIN config_sets cs ON cs.id = lv.config_set_id
JOIN apps a ON a.id = cs.app_id
WHERE lv.shared_var_id = ?
  AND EXISTS (SELECT 1 FROM env_vars o WHERE o.config_set_id = lv.config_set_id AND o.key = ? AND o.id <> lv.id);

-- name: GetAppEnvProvider :one
SELECT config FROM env_providers WHERE app_set_id = ?;
-- name: GetSharedEnvProvider :one
SELECT config FROM env_providers WHERE shared_set_id = ?;
-- name: SaveAppEnvProvider :exec
INSERT INTO env_providers(app_set_id, config) VALUES (?, ?) ON CONFLICT(app_set_id) DO UPDATE SET config=excluded.config;
-- name: SaveSharedEnvProvider :exec
INSERT INTO env_providers(shared_set_id, config) VALUES (?, ?) ON CONFLICT(shared_set_id) DO UPDATE SET config=excluded.config;
