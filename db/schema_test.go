package db

import (
	"database/sql"
	"testing"
)

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
