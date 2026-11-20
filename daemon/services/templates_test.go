package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"workspace-manager/daemon/db"
)

func TestRenderHandlebars(t *testing.T) {
	t.Run("basic variable substitution", func(t *testing.T) {
		tpl := "PORT={{PORT}}\nHOST={{HOST}}"
		env := map[string]string{
			"PORT": "3000",
			"HOST": "localhost",
		}

		got, err := renderHandlebars(tpl, env)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := "PORT=3000\nHOST=localhost"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("no html escaping for special characters", func(t *testing.T) {
		tpl := "DATABASE_URL={{DATABASE_URL}}"
		env := map[string]string{
			"DATABASE_URL": "postgres://user:p&ss\"w'd@localhost:5432/db?sslmode=disable&foo=bar<baz>",
		}

		got, err := renderHandlebars(tpl, env)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := "DATABASE_URL=postgres://user:p&ss\"w'd@localhost:5432/db?sslmode=disable&foo=bar<baz>"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("conditionals and blocks", func(t *testing.T) {
		tpl := "{{#if ENABLE_FEATURE}}feature is on{{else}}feature is off{{/if}}"
		env := map[string]string{
			"ENABLE_FEATURE": "true",
		}

		got, err := renderHandlebars(tpl, env)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "feature is on" {
			t.Fatalf("got %q, want %q", got, "feature is on")
		}

		// When omitted / empty
		gotOff, err := renderHandlebars(tpl, map[string]string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotOff != "feature is off" {
			t.Fatalf("got %q, want %q", gotOff, "feature is off")
		}
	})

	t.Run("syntax error returns error", func(t *testing.T) {
		tpl := "{{#if UNCLOSED"
		_, err := renderHandlebars(tpl, map[string]string{})
		if err == nil {
			t.Fatal("expected syntax error, got nil")
		}
	})
}

func TestApplyTemplatesCreatesMissingFiles(t *testing.T) {
	t.Setenv("WM_DATA_DIR", t.TempDir())
	ctx := context.Background()
	d := newBlueprintTestDB(t)
	ws, _ := d.CreateWorkspaceT(ctx, "ws", nil, nil)
	dir := t.TempDir()
	app, err := d.CreateAppT(ctx, ws.ID, "demo", dir)
	if err != nil {
		t.Fatal(err)
	}
	set, err := d.CreateConfigSetT(ctx, app.ID, "local")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.UpsertEnvVarByKey(ctx, set.ID, "PORT", "4000", true); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(dir, "src", "app.yml")
	if err := os.WriteFile(existing, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tpl := range []db.CreateTemplateParams{
		{ConfigSetID: set.ID, FilePath: "src/app.yml", Content: "port: {{PORT}}"},
		{ConfigSetID: set.ID, FilePath: "src/application-local.yml", Content: "local: {{PORT}}"},
		{ConfigSetID: set.ID, FilePath: "config/deep/extra.env", Content: "PORT={{PORT}}"},
	} {
		if _, err := d.CreateTemplate(ctx, tpl); err != nil {
			t.Fatal(err)
		}
	}

	if err := ApplyTemplates(ctx, d, app.ID, set.ID, "", "s1"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	for path, want := range map[string]string{
		existing: "port: 4000",
		filepath.Join(dir, "src", "application-local.yml"): "local: 4000",
		filepath.Join(dir, "config", "deep", "extra.env"):  "PORT=4000",
	} {
		if got, err := os.ReadFile(path); err != nil || string(got) != want {
			t.Fatalf("%s: got %q (%v), want %q", path, got, err, want)
		}
	}

	if err := RestoreTemplates(ctx, d, app.ID, set.ID, "", "s1"); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if got, _ := os.ReadFile(existing); string(got) != "original" {
		t.Fatalf("existing file not restored: %q", got)
	}
	for _, gone := range []string{
		filepath.Join(dir, "src", "application-local.yml"),
		filepath.Join(dir, "config"),
	} {
		if _, err := os.Stat(gone); !os.IsNotExist(err) {
			t.Fatalf("%s should have been removed (%v)", gone, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "src")); err != nil {
		t.Fatalf("pre-existing folder removed: %v", err)
	}
}
