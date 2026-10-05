package db

import (
	"context"
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

func TestApplySchemaAddsRunConfigKind(t *testing.T) {
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1)
	// Shape of run_configs before build configs existed.
	if _, err := sqlDB.Exec(`
		CREATE TABLE run_configs (
		  id integer PRIMARY KEY AUTOINCREMENT NOT NULL, config_set_id integer NOT NULL,
		  mode text DEFAULT 'parallel' NOT NULL, created_at text DEFAULT (datetime('now')) NOT NULL,
		  updated_at text DEFAULT (datetime('now')) NOT NULL);
		CREATE UNIQUE INDEX run_configs_config_set_id_unique ON run_configs (config_set_id);
		INSERT INTO run_configs (config_set_id, mode) VALUES (1, 'sequential');`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := applySchema(sqlDB); err != nil {
			t.Fatalf("apply #%d: %v", i+1, err)
		}
	}
	var kind, mode string
	if err := sqlDB.QueryRow(`SELECT kind, mode FROM run_configs`).Scan(&kind, &mode); err != nil {
		t.Fatal(err)
	}
	if kind != "run" || mode != "sequential" {
		t.Fatalf("row not preserved: %q %q", kind, mode)
	}
	// A config set can now hold a build config next to its run config, but only one of each.
	if _, err := sqlDB.Exec(`INSERT INTO run_configs (config_set_id, kind) VALUES (1, 'build')`); err != nil {
		t.Fatalf("build config next to run config: %v", err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO run_configs (config_set_id, kind) VALUES (1, 'build')`); err == nil {
		t.Fatal("a second build config for one config set should be rejected")
	}
}

func TestApplySchemaAddsBlueprintSampleName(t *testing.T) {
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1)
	// Shape of the global blueprints table before sample_name existed.
	if _, err := sqlDB.Exec(`
		CREATE TABLE blueprints (
		  id integer PRIMARY KEY AUTOINCREMENT NOT NULL, name text NOT NULL,
		  description text DEFAULT '' NOT NULL, create_folder integer DEFAULT true NOT NULL,
		  commands text DEFAULT '[]' NOT NULL, created_at text DEFAULT (datetime('now')) NOT NULL,
		  updated_at text DEFAULT (datetime('now')) NOT NULL);
		CREATE UNIQUE INDEX blueprints_name_unique ON blueprints (name);
		INSERT INTO blueprints (name, commands) VALUES ('Vite', '[{"label":null,"command":"bun install"}]');`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := applySchema(sqlDB); err != nil {
			t.Fatalf("apply #%d: %v", i+1, err)
		}
	}
	var name, sample, commands string
	if err := sqlDB.QueryRow(`SELECT name, sample_name, commands FROM blueprints`).Scan(&name, &sample, &commands); err != nil {
		t.Fatal(err)
	}
	if name != "Vite" || sample != "" || commands != `[{"label":null,"command":"bun install"}]` {
		t.Fatalf("row not preserved: %q %q %q", name, sample, commands)
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

func TestApplySchemaAddsAppSortOrder(t *testing.T) {
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1)
	// Shape of apps before they could be reordered, when they were listed by name.
	if _, err := sqlDB.Exec(`
		CREATE TABLE apps (
		  id integer PRIMARY KEY AUTOINCREMENT NOT NULL, workspace_id integer NOT NULL, name text NOT NULL,
		  project_path text NOT NULL, active_config_set_id integer,
		  created_at text DEFAULT (datetime('now')) NOT NULL, updated_at text DEFAULT (datetime('now')) NOT NULL);
		INSERT INTO apps (workspace_id, name, project_path) VALUES
		  (1, 'web', '.'), (1, 'Api', '.'), (2, 'zeta', '.'), (1, 'docs', '.'), (2, 'alpha', '.');`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := applySchema(sqlDB); err != nil {
			t.Fatalf("apply #%d: %v", i+1, err)
		}
	}
	rows, err := sqlDB.Query(`SELECT workspace_id, name FROM apps ORDER BY workspace_id, sort_order`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var ws int
		var name string
		if err := rows.Scan(&ws, &name); err != nil {
			t.Fatal(err)
		}
		got = append(got, string(rune('0'+ws))+":"+name)
	}
	want := []string{"1:Api", "1:docs", "1:web", "2:alpha", "2:zeta"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestReorderApps(t *testing.T) {
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1)
	if err := applySchema(sqlDB); err != nil {
		t.Fatal(err)
	}
	d := &DB{SQL: sqlDB, Queries: New(sqlDB)}
	ctx := context.Background()
	ws, err := d.CreateWorkspaceT(ctx, "w", nil)
	if err != nil {
		t.Fatal(err)
	}
	other, err := d.CreateWorkspaceT(ctx, "other", nil)
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for _, name := range []string{"b", "c", "a"} {
		app, err := d.CreateAppT(ctx, ws.ID, name, ".")
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, app.ID)
	}
	foreign, err := d.CreateAppT(ctx, other.ID, "x", ".")
	if err != nil {
		t.Fatal(err)
	}
	names := func() string {
		apps, err := d.ListAppsByWorkspaceT(ctx, ws.ID)
		if err != nil {
			t.Fatal(err)
		}
		s := ""
		for _, a := range apps {
			s += a.Name
		}
		return s
	}
	// New apps go to the end, not into name order.
	if got := names(); got != "bca" {
		t.Fatalf("created order: %s", got)
	}
	if err := d.ReorderAppsT(ctx, ws.ID, []int64{ids[2], ids[0], ids[1]}); err != nil {
		t.Fatal(err)
	}
	if got := names(); got != "abc" {
		t.Fatalf("reordered: %s", got)
	}
	for _, bad := range [][]int64{{ids[0]}, {ids[0], ids[0], ids[1]}, {ids[0], ids[1], foreign.ID}} {
		if err := d.ReorderAppsT(ctx, ws.ID, bad); err == nil {
			t.Fatalf("order %v should be rejected", bad)
		}
	}
	if got := names(); got != "abc" {
		t.Fatalf("a rejected order changed the list: %s", got)
	}
}
