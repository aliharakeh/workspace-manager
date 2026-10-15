package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"workspace-manager/api"
)

// setupApp builds a data folder with one app whose dev config has a command,
// a visible and a hidden env var, and a .env template, and opens the store.
func setupApp(t *testing.T) (appID int64, dir string) {
	t.Helper()
	t.Setenv("WM_DATA_DIR", t.TempDir())
	if err := openStore(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	dir = t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := daoCall(api.MethodWorkspacesCreate, map[string]any{"name": "w"})
	if err != nil {
		t.Fatal(err)
	}
	app, err := daoCall(api.MethodAppsCreate, map[string]any{
		"workspaceID": ws.(map[string]any)["id"],
		"body":        map[string]any{"name": "app", "project_path": dir},
	})
	if err != nil {
		t.Fatal(err)
	}
	appID = appIDOf(app.(map[string]any))
	for _, v := range []map[string]any{
		{"key": "PUBLIC", "value": "visible", "include_in_ai": true},
		{"key": "SECRET", "value": "hidden-value", "include_in_ai": false},
	} {
		if _, err := daoCall(api.MethodEnvVarsCreate, map[string]any{"appID": appID, "body": v}); err != nil {
			t.Fatal(err)
		}
	}
	// Templates and run configs are written by the GUI in real use; the agent
	// edits them here through the same tools.
	if _, err := daoCall(api.MethodTemplatesCreate, map[string]any{"appID": appID, "body": map[string]any{
		"file_path": ".env", "content": "S={{SECRET}}", "include_in_ai": true,
	}}); err != nil {
		t.Fatal(err)
	}
	if _, err := daoCall(api.MethodRunConfigSave, map[string]any{"appID": appID, "kind": "dev", "body": map[string]any{
		"mode": "parallel", "commands": []map[string]any{{"command": "echo hi"}},
	}}); err != nil {
		t.Fatal(err)
	}
	return appID, dir
}

func TestPlanNamesHiddenValuesButNeverShowsThem(t *testing.T) {
	appID, _ := setupApp(t)
	app, err := resolveApp("app")
	if err != nil || appIDOf(app) != appID {
		t.Fatalf("resolve: %v", err)
	}
	p, err := buildRunPlan(app, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.HiddenEnv) != 1 || p.HiddenEnv[0] != "SECRET" {
		t.Fatalf("hidden keys %v", p.HiddenEnv)
	}
	for _, e := range p.Env {
		if strings.Contains(e.Value, "hidden-value") {
			t.Fatal("the plan leaked a hidden value")
		}
	}
	if len(p.Commands) != 1 || p.Commands[0].Command != "echo hi" {
		t.Fatalf("commands %+v", p.Commands)
	}
}

func TestPrepareAndFinish(t *testing.T) {
	appID, dir := setupApp(t)
	app, _ := resolveApp("app")

	res, err := prepareRun(app, map[string]any{"kind": "dev"})
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, ".env")); string(b) != "S=hidden-value" {
		t.Fatalf("template not rendered: %q", b)
	}
	if strings.Contains(toText(res, nil), "hidden-value") {
		t.Fatal("prepare returned a hidden value")
	}
	// A second prepare is refused while the first session holds the app.
	if _, err := prepareRun(app, map[string]any{"kind": "dev"}); api.KindOf(err) != api.KindConflict {
		t.Fatalf("second prepare: want a conflict, got %v", err)
	}
	if st, _ := sessionOf(appID); st["running"] != true || st["owner"] != "agent" {
		t.Fatalf("status %v", st)
	}

	if _, err := finishRun(appID); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, ".env")); string(b) != "original" {
		t.Fatalf("original file not restored: %q", b)
	}
	if st, _ := sessionOf(appID); st["running"] != false {
		t.Fatalf("status after finish %v", st)
	}
}

func TestExitReleasesOwnSessions(t *testing.T) {
	appID, dir := setupApp(t)
	app, _ := resolveApp("app")
	if _, err := prepareRun(app, map[string]any{}); err != nil {
		t.Fatal(err)
	}
	releaseOwnSessions()
	if b, _ := os.ReadFile(filepath.Join(dir, ".env")); string(b) != "original" {
		t.Fatalf("original file not restored: %q", b)
	}
	if st, _ := sessionOf(appID); st["running"] != false {
		t.Fatalf("session survived: %v", st)
	}
}

func TestGuideIsServed(t *testing.T) {
	if !strings.Contains(guideText, "wm_run") || !strings.Contains(guideText, "runs nothing") {
		t.Fatal("the guide is missing its core rules")
	}
	for _, tool := range allTools() {
		if tool.Name == "wm_call" {
			t.Fatal("the raw escape hatch is back")
		}
	}
}
