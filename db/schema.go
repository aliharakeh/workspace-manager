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
	if _, err := sqlDB.Exec(schemaSQL); err != nil {
		return err
	}
	if err := restoreBlueprints(sqlDB); err != nil {
		return err
	}
	return addBlueprintSampleName(sqlDB)
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
