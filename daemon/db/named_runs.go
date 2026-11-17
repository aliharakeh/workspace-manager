package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Upgrade in place so command/configuration ids and execution order survive.
// The backup is durable before any rows are changed and is never overwritten.
func addNamedRuns(sqlDB *sql.DB) error {
	cols, err := tableColumns(sqlDB, "run_configs")
	if err != nil || len(cols) == 0 {
		return err
	}
	has, err := hasColumn(sqlDB, "run_configs", "name")
	if err != nil || has {
		return err
	}
	if err := backupRunConfigs(sqlDB); err != nil {
		return fmt.Errorf("back up run configurations: %w", err)
	}
	tx, err := sqlDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`
		ALTER TABLE run_configs ADD COLUMN name text DEFAULT '' NOT NULL;
		ALTER TABLE run_configs ADD COLUMN detect_ready integer DEFAULT false NOT NULL;
		UPDATE run_configs SET detect_ready = kind IN ('dev', 'prod', 'run');
		UPDATE run_configs SET name = CASE kind
		  WHEN 'run' THEN 'Dev' WHEN 'dev' THEN 'Dev' WHEN 'prod' THEN 'Prod'
		  WHEN 'build' THEN 'Build' WHEN 'setup' THEN 'Setup' ELSE kind END
		WHERE EXISTS (SELECT 1 FROM run_commands WHERE run_config_id = run_configs.id);`)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func backupRunConfigs(sqlDB *sql.DB) error {
	var filename string
	if err := sqlDB.QueryRow(`SELECT file FROM pragma_database_list WHERE name = 'main'`).Scan(&filename); err != nil {
		return err
	}
	// In-memory databases are used by migration tests and have no durable data.
	if filename == "" {
		return nil
	}
	backup := map[string]any{"version": 1, "created_at": time.Now().UTC().Format(time.RFC3339)}
	for key, query := range map[string]string{
		"run_configs":  "SELECT * FROM run_configs ORDER BY id",
		"run_commands": "SELECT * FROM run_commands ORDER BY run_config_id, sort_order, id",
		"config_sets":  "SELECT id, app_id, name FROM config_sets ORDER BY id",
		"apps":         "SELECT id, workspace_id, name, project_path FROM apps ORDER BY id",
	} {
		rows, err := sqlDB.Query(query)
		if err != nil {
			return err
		}
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			return err
		}
		items := []map[string]any{}
		for rows.Next() {
			values := make([]any, len(columns))
			ptrs := make([]any, len(columns))
			for i := range values {
				ptrs[i] = &values[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				rows.Close()
				return err
			}
			item := map[string]any{}
			for i, column := range columns {
				if b, ok := values[i].([]byte); ok {
					values[i] = string(b)
				}
				item[column] = values[i]
			}
			items = append(items, item)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		backup[key] = items
	}
	dir := filepath.Join(filepath.Dir(filename), "backups")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, "run-configs-before-named-*.json")
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(backup); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	return f.Close()
}
