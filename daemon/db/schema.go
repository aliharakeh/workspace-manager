package db

import (
	"database/sql"
	_ "embed"
)

// schemaSQL is the same file sqlc reads. Every statement is idempotent, so it
// runs on every start: it creates a fresh database and leaves an existing one
// untouched.
//
//go:embed schema.sql
var schemaSQL string

func applySchema(sqlDB *sql.DB) error {
	if err := detachBlueprints(sqlDB); err != nil {
		return err
	}
	if err := addRunConfigKind(sqlDB); err != nil {
		return err
	}
	if err := addAppSortOrder(sqlDB); err != nil {
		return err
	}
	if err := addWorkspaceOrderAndPin(sqlDB); err != nil {
		return err
	}
	if err := addWorkspaceColor(sqlDB); err != nil {
		return err
	}
	if err := addTemplatePrivate(sqlDB); err != nil {
		return err
	}
	if err := renameAIFlagToPrivate(sqlDB); err != nil {
		return err
	}
	if err := addAppExcludeFromStartAll(sqlDB); err != nil {
		return err
	}
	if err := addEnvVarOptions(sqlDB); err != nil {
		return err
	}
	if err := addSharedLinkColumns(sqlDB); err != nil {
		return err
	}
	if _, err := sqlDB.Exec(schemaSQL); err != nil {
		return err
	}
	if err := splitRunConfig(sqlDB); err != nil {
		return err
	}
	if err := restoreBlueprints(sqlDB); err != nil {
		return err
	}
	return addBlueprintSampleName(sqlDB)
}

// splitRunConfig preserves existing commands and ids as development commands.
func splitRunConfig(sqlDB *sql.DB) error {
	tx, err := sqlDB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`
		UPDATE run_configs SET kind = 'dev' WHERE kind = 'run';
		INSERT INTO run_configs (config_set_id, kind, mode)
		SELECT id, 'prod', 'parallel' FROM config_sets WHERE true
		ON CONFLICT (config_set_id, kind) DO NOTHING;`); err != nil {
		return err
	}
	return tx.Commit()
}

// addRunConfigKind upgrades a run_configs table created before build configs
// existed: its rows become kind 'run' and the one-per-config-set unique index
// is dropped, since schemaSQL now makes it one per config set and kind. It runs
// before schemaSQL, which would otherwise index a column the old table lacks.
func addRunConfigKind(sqlDB *sql.DB) error {
	cols, err := tableColumns(sqlDB, "run_configs")
	if err != nil || len(cols) == 0 {
		return err // fresh database: schemaSQL creates the table with kind
	}
	has, err := hasColumn(sqlDB, "run_configs", "kind")
	if err != nil || has {
		return err
	}
	_, err = sqlDB.Exec(`
		ALTER TABLE run_configs ADD COLUMN kind text DEFAULT 'run' NOT NULL;
		DROP INDEX IF EXISTS run_configs_config_set_id_unique;`)
	return err
}

// addAppSortOrder upgrades an apps table created before apps could be
// reordered. Apps used to be listed by name, so each workspace's apps are
// numbered in that order to keep the list looking the same. A fresh table
// already has the column from schemaSQL.
func addAppSortOrder(sqlDB *sql.DB) error {
	cols, err := tableColumns(sqlDB, "apps")
	if err != nil || len(cols) == 0 {
		return err // fresh database: schemaSQL creates the table with sort_order
	}
	has, err := hasColumn(sqlDB, "apps", "sort_order")
	if err != nil || has {
		return err
	}
	_, err = sqlDB.Exec(`
		ALTER TABLE apps ADD COLUMN sort_order integer DEFAULT 0 NOT NULL;
		UPDATE apps SET sort_order = (
		  SELECT count(*) FROM apps b
		  WHERE b.workspace_id = apps.workspace_id
		    AND (b.name COLLATE NOCASE < apps.name COLLATE NOCASE
		         OR (b.name COLLATE NOCASE = apps.name COLLATE NOCASE AND b.id < apps.id))
		);`)
	return err
}

// addWorkspaceColor upgrades a workspaces table created before workspaces could
// be colored. NULL means no color. A fresh table already has the column from
// schemaSQL.
func addWorkspaceColor(sqlDB *sql.DB) error {
	cols, err := tableColumns(sqlDB, "workspaces")
	if err != nil || len(cols) == 0 {
		return err // fresh database: schemaSQL creates the table with color
	}
	has, err := hasColumn(sqlDB, "workspaces", "color")
	if err != nil || has {
		return err
	}
	_, err = sqlDB.Exec(`ALTER TABLE workspaces ADD COLUMN color text`)
	return err
}

// legacyAIFlag is the column the private flag replaced. It meant the opposite:
// true was visible to agents.
const legacyAIFlag = "include_in_ai"

// addTemplatePrivate upgrades a templates table created before templates could
// be private. Existing templates stay visible to agents. A fresh table already
// has the column from schemaSQL. A table that still has the older include_in_ai
// column is handled by renameAIFlagToPrivate.
func addTemplatePrivate(sqlDB *sql.DB) error {
	cols, err := tableColumns(sqlDB, "templates")
	if err != nil || len(cols) == 0 {
		return err // fresh database: schemaSQL creates the table with private
	}
	for _, c := range cols {
		if c == "private" || c == legacyAIFlag {
			return nil
		}
	}
	_, err = sqlDB.Exec(`ALTER TABLE templates ADD COLUMN private integer DEFAULT false NOT NULL`)
	return err
}

// renameAIFlagToPrivate upgrades the tables that still carry the earlier
// include_in_ai flag (true = visible to agents) to private (true = hidden from
// them). The meaning is the same, so the value is inverted. It runs before
// schemaSQL and is a no-op once the column is gone.
func renameAIFlagToPrivate(sqlDB *sql.DB) error {
	for _, table := range []string{"env_vars", "templates", "workspace_env_vars"} {
		has, err := hasColumn(sqlDB, table, legacyAIFlag)
		if err != nil {
			return err
		}
		if !has {
			continue
		}
		if _, err := sqlDB.Exec(`
			ALTER TABLE ` + table + ` ADD COLUMN private integer DEFAULT false NOT NULL;
			UPDATE ` + table + ` SET private = CASE WHEN ` + legacyAIFlag + ` = 0 THEN 1 ELSE 0 END;
			ALTER TABLE ` + table + ` DROP COLUMN ` + legacyAIFlag + `;`); err != nil {
			return err
		}
	}
	return nil
}

// addEnvVarOptions upgrades an env_vars table created before variables could
// have options. Existing variables stay plain values. A fresh table already has
// the columns from schemaSQL.
func addEnvVarOptions(sqlDB *sql.DB) error {
	cols, err := tableColumns(sqlDB, "env_vars")
	if err != nil || len(cols) == 0 {
		return err
	}
	has, err := hasColumn(sqlDB, "env_vars", "options")
	if err != nil || has {
		return err
	}
	_, err = sqlDB.Exec(`
		ALTER TABLE env_vars ADD COLUMN options text DEFAULT '[]' NOT NULL;
		ALTER TABLE env_vars ADD COLUMN selected text DEFAULT '' NOT NULL;`)
	return err
}

// addSharedLinkColumns upgrades config_sets and env_vars tables created before
// workspaces had shared config. It runs before schemaSQL, which indexes
// env_vars.shared_var_id. A column can only reference a table that exists, so
// the two workspace tables are created here first (schemaSQL's own statements
// for them are then no-ops). SQLite allows adding a REFERENCES column when its
// default is NULL.
func addSharedLinkColumns(sqlDB *sql.DB) error {
	envCols, err := tableColumns(sqlDB, "env_vars")
	if err != nil || len(envCols) == 0 {
		return err // fresh database: schemaSQL creates everything
	}
	hasVar, err := hasColumn(sqlDB, "env_vars", "shared_var_id")
	if err != nil {
		return err
	}
	hasSet, err := hasColumn(sqlDB, "config_sets", "shared_set_id")
	if err != nil {
		return err
	}
	if hasVar && hasSet {
		return nil
	}
	if _, err := sqlDB.Exec(`
		CREATE TABLE IF NOT EXISTS workspace_config_sets (
		  id integer PRIMARY KEY AUTOINCREMENT NOT NULL,
		  workspace_id integer NOT NULL,
		  name text NOT NULL,
		  created_at text DEFAULT (datetime('now')) NOT NULL,
		  updated_at text DEFAULT (datetime('now')) NOT NULL,
		  FOREIGN KEY (workspace_id) REFERENCES workspaces (id) ON DELETE cascade
		);
		CREATE TABLE IF NOT EXISTS workspace_env_vars (
		  id integer PRIMARY KEY AUTOINCREMENT NOT NULL,
		  workspace_config_set_id integer NOT NULL,
		  key text NOT NULL,
		  value text DEFAULT '' NOT NULL,
		  private integer DEFAULT false NOT NULL,
		  options text DEFAULT '[]' NOT NULL,
		  selected text DEFAULT '' NOT NULL,
		  created_at text DEFAULT (datetime('now')) NOT NULL,
		  updated_at text DEFAULT (datetime('now')) NOT NULL,
		  FOREIGN KEY (workspace_config_set_id) REFERENCES workspace_config_sets (id) ON DELETE cascade
		);`); err != nil {
		return err
	}
	if !hasSet {
		if _, err := sqlDB.Exec(`ALTER TABLE config_sets ADD COLUMN shared_set_id integer REFERENCES workspace_config_sets (id) ON DELETE set null`); err != nil {
			return err
		}
	}
	if !hasVar {
		if _, err := sqlDB.Exec(`ALTER TABLE env_vars ADD COLUMN shared_var_id integer REFERENCES workspace_env_vars (id) ON DELETE cascade`); err != nil {
			return err
		}
	}
	return nil
}

// addBlueprintSampleName upgrades a blueprints table created before the
// sample_name column existed. A fresh table already has it from schemaSQL.
func addBlueprintSampleName(sqlDB *sql.DB) error {
	has, err := hasColumn(sqlDB, "blueprints", "sample_name")
	if err != nil || has {
		return err
	}
	_, err = sqlDB.Exec(`ALTER TABLE blueprints ADD COLUMN sample_name text DEFAULT '' NOT NULL`)
	return err
}

func tableColumns(sqlDB *sql.DB, table string) ([]string, error) {
	rows, err := sqlDB.Query("SELECT name FROM pragma_table_info(?)", table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		cols = append(cols, name)
	}
	return cols, rows.Err()
}

func hasColumn(sqlDB *sql.DB, table, column string) (bool, error) {
	cols, err := tableColumns(sqlDB, table)
	if err != nil {
		return false, err
	}
	for _, c := range cols {
		if c == column {
			return true, nil
		}
	}
	return false, nil
}

// Blueprints used to belong to a workspace. SQLite cannot drop a NOT NULL
// column in place, so an old table is renamed out of the way before schemaSQL
// creates the new one, then restoreBlueprints copies the rows across.
func detachBlueprints(sqlDB *sql.DB) error {
	legacy, err := hasColumn(sqlDB, "blueprints", "workspace_id")
	if err != nil || !legacy {
		return err
	}
	if cols, err := tableColumns(sqlDB, "blueprints_old"); err != nil || len(cols) > 0 {
		return err // a previous attempt was interrupted; restoreBlueprints finishes it
	}
	_, err = sqlDB.Exec(`
		ALTER TABLE blueprints RENAME TO blueprints_old;
		DROP INDEX IF EXISTS idx_blueprints_workspace_id;
		DROP INDEX IF EXISTS blueprints_workspace_id_name_unique;`)
	return err
}

// restoreBlueprints copies rows from blueprints_old into the new table. Names
// are now unique across all blueprints, so later duplicates get their id appended.
func restoreBlueprints(sqlDB *sql.DB) error {
	cols, err := tableColumns(sqlDB, "blueprints_old")
	if err != nil || len(cols) == 0 {
		return err
	}
	tx, err := sqlDB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`
		INSERT INTO blueprints (name, description, create_folder, commands, created_at, updated_at)
		SELECT
		  CASE WHEN o.id = (SELECT MIN(o2.id) FROM blueprints_old o2 WHERE o2.name = o.name)
		       THEN o.name ELSE o.name || ' (' || o.id || ')' END,
		  o.description, o.create_folder, o.commands, o.created_at, o.updated_at
		FROM blueprints_old o ORDER BY o.id`); err != nil {
		return err
	}
	if _, err := tx.Exec("DROP TABLE blueprints_old"); err != nil {
		return err
	}
	return tx.Commit()
}

// addWorkspaceOrderAndPin upgrades a workspaces table created before
// workspaces could be reordered or pinned. Workspaces used to be listed by name,
// so they are numbered in that order to keep the list looking the same; none
// starts pinned (the old pins lived in the GUI's local storage and are pushed up
// by the GUI). A fresh table already has both columns from schemaSQL.
func addWorkspaceOrderAndPin(sqlDB *sql.DB) error {
	cols, err := tableColumns(sqlDB, "workspaces")
	if err != nil || len(cols) == 0 {
		return err // fresh database: schemaSQL creates the table with both columns
	}
	hasOrder, err := hasColumn(sqlDB, "workspaces", "sort_order")
	if err != nil {
		return err
	}
	if !hasOrder {
		if _, err := sqlDB.Exec(`
			ALTER TABLE workspaces ADD COLUMN sort_order integer DEFAULT 0 NOT NULL;
			UPDATE workspaces SET sort_order = (
			  SELECT count(*) FROM workspaces b
			  WHERE b.name COLLATE NOCASE < workspaces.name COLLATE NOCASE
			     OR (b.name COLLATE NOCASE = workspaces.name COLLATE NOCASE AND b.id < workspaces.id)
			);`); err != nil {
			return err
		}
	}
	hasPinned, err := hasColumn(sqlDB, "workspaces", "pinned")
	if err != nil || hasPinned {
		return err
	}
	_, err = sqlDB.Exec(`ALTER TABLE workspaces ADD COLUMN pinned integer DEFAULT false NOT NULL`)
	return err
}

// addAppExcludeFromStartAll upgrades an apps table created before apps could be
// left out of a workspace's "Start all". Existing apps stay included. A fresh
// table already has the column from schemaSQL.
func addAppExcludeFromStartAll(sqlDB *sql.DB) error {
	cols, err := tableColumns(sqlDB, "apps")
	if err != nil || len(cols) == 0 {
		return err // fresh database: schemaSQL creates the table with the column
	}
	has, err := hasColumn(sqlDB, "apps", "exclude_from_start_all")
	if err != nil || has {
		return err
	}
	_, err = sqlDB.Exec(`ALTER TABLE apps ADD COLUMN exclude_from_start_all integer DEFAULT 0 NOT NULL`)
	return err
}
