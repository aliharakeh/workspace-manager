-- workspaces (db/workspaces.ts)
-- name: ListWorkspaces :many
SELECT * FROM workspaces ORDER BY name COLLATE NOCASE ASC;

-- name: GetWorkspace :one
SELECT * FROM workspaces WHERE id = ?;

-- name: CreateWorkspace :one
INSERT INTO workspaces (name, icon) VALUES (?, ?) RETURNING *;

-- name: UpdateWorkspace :one
UPDATE workspaces
SET name = ?, icon = ?, updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: DeleteWorkspace :execrows
DELETE FROM workspaces WHERE id = ?;

-- apps (db/apps.ts)
-- name: ListAppsByWorkspace :many
SELECT
  a.id, a.workspace_id, a.name, a.project_path, a.active_config_set_id,
  a.created_at, a.updated_at,
  cs.name AS active_config_set_name
FROM apps a
LEFT JOIN config_sets cs ON cs.id = a.active_config_set_id
WHERE a.workspace_id = ?
ORDER BY a.name COLLATE NOCASE ASC;

-- name: GetApp :one
SELECT
  a.id, a.workspace_id, a.name, a.project_path, a.active_config_set_id,
  a.created_at, a.updated_at,
  cs.name AS active_config_set_name
FROM apps a
LEFT JOIN config_sets cs ON cs.id = a.active_config_set_id
WHERE a.id = ?;

-- name: CreateApp :one
INSERT INTO apps (workspace_id, name, project_path)
VALUES (?, ?, ?)
RETURNING *;

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
INSERT INTO env_vars (config_set_id, key, value) VALUES (?, ?, ?) RETURNING *;

-- name: UpdateEnvVar :one
UPDATE env_vars
SET key = ?, value = ?, updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: DeleteEnvVar :execrows
DELETE FROM env_vars WHERE id = ?;

-- name: DeleteEnvVarsByConfigSet :exec
DELETE FROM env_vars WHERE config_set_id = ?;

-- templates (db/templates.ts)
-- name: ListTemplatesByConfigSet :many
SELECT * FROM templates WHERE config_set_id = ? ORDER BY file_path COLLATE NOCASE ASC;

-- name: GetTemplate :one
SELECT * FROM templates WHERE id = ?;

-- name: GetTemplateByPath :one
SELECT * FROM templates WHERE config_set_id = ? AND file_path = ?;

-- name: CreateTemplate :one
INSERT INTO templates (config_set_id, file_path, content) VALUES (?, ?, ?) RETURNING *;

-- name: UpdateTemplate :one
UPDATE templates
SET file_path = ?, content = ?, updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: DeleteTemplate :execrows
DELETE FROM templates WHERE id = ?;

-- name: DeleteTemplatesByConfigSet :exec
DELETE FROM templates WHERE config_set_id = ?;

-- run_configs (db/run-configs.ts)
-- name: GetRunConfigByConfigSet :one
SELECT * FROM run_configs WHERE config_set_id = ?;

-- name: CreateRunConfig :one
INSERT INTO run_configs (config_set_id, mode) VALUES (?, ?) RETURNING *;

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
