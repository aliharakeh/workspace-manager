package db

import (
	"database/sql"
	"testing"
)

func TestApplySchemaMakesBlueprintsGlobal(t *testing.T) {
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1)
	// Shape of the table when blueprints still belonged to a workspace.
	if _, err := sqlDB.Exec(`
		CREATE TABLE workspaces (id integer PRIMARY KEY AUTOINCREMENT NOT NULL, name text NOT NULL, icon text,
		  created_at text DEFAULT (datetime('now')) NOT NULL, updated_at text DEFAULT (datetime('now')) NOT NULL);
		CREATE TABLE blueprints (
		  id integer PRIMARY KEY AUTOINCREMENT NOT NULL, workspace_id integer NOT NULL, name text NOT NULL,
		  description text DEFAULT '' NOT NULL, create_folder integer DEFAULT true NOT NULL,
		  commands text DEFAULT '[]' NOT NULL, created_at text DEFAULT (datetime('now')) NOT NULL,
		  updated_at text DEFAULT (datetime('now')) NOT NULL,
		  FOREIGN KEY (workspace_id) REFERENCES workspaces (id) ON DELETE cascade);
		CREATE INDEX idx_blueprints_workspace_id ON blueprints (workspace_id);
		CREATE UNIQUE INDEX blueprints_workspace_id_name_unique ON blueprints (workspace_id, name);
		INSERT INTO workspaces (name) VALUES ('a'), ('b');
		INSERT INTO blueprints (workspace_id, name, commands) VALUES
		  (1, 'Vite', '[{"label":null,"command":"bun install"}]'), (2, 'Vite', '[]'), (2, 'Api', '[]');`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := applySchema(sqlDB); err != nil {
			t.Fatalf("apply #%d: %v", i+1, err)
		}
	}
	if has, err := hasColumn(sqlDB, "blueprints", "workspace_id"); err != nil || has {
		t.Fatalf("workspace_id should be gone: %v, %v", has, err)
	}
	if cols, _ := tableColumns(sqlDB, "blueprints_old"); len(cols) != 0 {
		t.Fatal("blueprints_old should be dropped")
	}
	rows, err := sqlDB.Query(`SELECT name, commands FROM blueprints ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var names []string
	var firstCommands string
	for rows.Next() {
		var name, commands string
		if err := rows.Scan(&name, &commands); err != nil {
			t.Fatal(err)
		}
		if len(names) == 0 {
			firstCommands = commands
		}
		names = append(names, name)
	}
	want := []string{"Vite", "Vite (2)", "Api"}
	if len(names) != 3 || names[0] != want[0] || names[1] != want[1] || names[2] != want[2] {
		t.Fatalf("got %v, want %v", names, want)
	}
	if firstCommands != `[{"label":null,"command":"bun install"}]` {
		t.Fatalf("commands not carried over: %s", firstCommands)
	}
	// New inserts work without a workspace.
	if _, err := sqlDB.Exec(`INSERT INTO blueprints (name) VALUES ('Fresh')`); err != nil {
		t.Fatal(err)
	}
}

func TestApplySchemaIsIdempotent(t *testing.T) {
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1)

	for i := 0; i < 2; i++ {
		if err := applySchema(sqlDB); err != nil {
			t.Fatalf("apply #%d: %v", i+1, err)
		}
	}

	var n int
	if err := sqlDB.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name IN
		('workspaces','apps','config_sets','env_vars','templates','run_configs','run_commands','app_settings','ready_url_patterns')`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 9 {
		t.Fatalf("expected 9 tables, got %d", n)
	}
}
