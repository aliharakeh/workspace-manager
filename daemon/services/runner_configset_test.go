package services

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"workspace-manager/daemon/db"
	"workspace-manager/shared/api/types"
)

func TestRunnerExplicitConfigSet(t *testing.T) {
	t.Setenv("WM_DATA_DIR", t.TempDir())
	ctx := context.Background()
	d := newBlueprintTestDB(t)
	ws, _ := d.CreateWorkspaceT(ctx, "ws", nil, nil)
	dir := t.TempDir()
	app, err := d.CreateAppT(ctx, ws.ID, "demo", dir)
	if err != nil {
		t.Fatal(err)
	}
	active, err := d.ResolveActive(ctx, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := d.CreateConfigSetT(ctx, app.ID, "staging")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.UpsertEnvVarByKey(ctx, selected.ID, "RUN_VALUE", "selected-env", true); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.txt")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := d.CreateTemplate(ctx, db.CreateTemplateParams{ConfigSetID: selected.ID, FilePath: "config.txt", Content: "template={{RUN_VALUE}}"}); err != nil {
		t.Fatal(err)
	}
	command := `echo $RUN_VALUE; cat config.txt`
	if runtime.GOOS == "windows" {
		command = `echo %RUN_VALUE% & type config.txt`
	}
	if _, err := d.UpsertRunConfig(ctx, selected.ID, types.KindDev, nil, []types.RunCommandInput{{Command: command}}); err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(d, nil, nil)
	wait := func() {
		t.Helper()
		deadline := time.Now().Add(20 * time.Second)
		for runner.GetStatus(app.ID).Running {
			if time.Now().After(deadline) {
				t.Fatal("run did not finish")
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	status, err := runner.StartIn(ctx, app.ID, types.KindDev, "", "", selected.ID)
	if err != nil {
		t.Fatal(err)
	}
	wait()
	if status.ConfigSetID != selected.ID {
		t.Fatalf("config set: got %d, want %d", status.ConfigSetID, selected.ID)
	}
	out, _ := base64.StdEncoding.DecodeString(runner.GetOutput(app.ID, status.Processes[0].CommandID).Data)
	if !strings.Contains(string(out), "template=selected-env") {
		t.Fatalf("selected environment and templates were not applied: %q", out)
	}
	if original, _ := os.ReadFile(path); string(original) != "original" {
		t.Fatalf("template was not restored: %q", original)
	}
	if current, _ := d.ResolveActive(ctx, app.ID); current.ID != active.ID {
		t.Fatal("starting with a config set changed the app's active set")
	}
	// Reload must not fall back to the active set, which has no dev commands.
	status, err = runner.Reload(ctx, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	wait()
	if status.ConfigSetID != selected.ID {
		t.Fatal("reload lost the explicitly selected config set")
	}
	// GUI reload uses the active set even when MCP started the previous run
	// with an explicit override.
	if _, err := d.UpsertRunConfig(ctx, active.ID, types.KindDev, nil, []types.RunCommandInput{{Command: "echo active"}}); err != nil {
		t.Fatal(err)
	}
	status, err = runner.ReloadActive(ctx, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	wait()
	if status.ConfigSetID != active.ID {
		t.Fatal("GUI reload retained the MCP override")
	}
	other, err := d.CreateAppT(ctx, ws.ID, "other", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	otherSet, _ := d.ResolveActive(ctx, other.ID)
	if _, err := runner.StartIn(ctx, app.ID, types.KindDev, "", "", otherSet.ID); err == nil {
		t.Fatal("accepted another app's config set")
	}
}
