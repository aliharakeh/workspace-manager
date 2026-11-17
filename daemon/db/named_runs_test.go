package db

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"workspace-manager/shared/api/types"
)

func TestNamedRunsMigrationBackup(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		t.Run(map[bool]string{false: "preserves data", true: "backup failure aborts"}[blocked], func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("WM_DATA_DIR", dir)
			d, err := Open()
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			ws, _ := d.CreateWorkspaceT(ctx, "workspace", nil, nil)
			app, _ := d.CreateAppT(ctx, ws.ID, "app", t.TempDir())
			set, _ := d.ResolveActive(ctx, app.ID)
			mode := "sequential"
			cfg, err := d.UpsertRunConfig(ctx, set.ID, "dev", &mode, []types.RunCommandInput{{Command: "echo first"}, {Command: "echo second"}})
			if err != nil {
				t.Fatal(err)
			}
			// Recreate the schema immediately before named configurations.
			if _, err := d.SQL.Exec(`ALTER TABLE run_configs DROP COLUMN name; ALTER TABLE run_configs DROP COLUMN detect_ready;`); err != nil {
				t.Fatal(err)
			}
			if blocked {
				if err := os.WriteFile(filepath.Join(dir, "backups"), []byte("blocked"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := addNamedRuns(d.SQL); err == nil {
					t.Fatal("expected backup failure")
				}
				if has, _ := hasColumn(d.SQL, "run_configs", "name"); has {
					t.Fatal("schema changed without backup")
				}
				var command string
				if err := d.SQL.QueryRow(`SELECT command FROM run_commands WHERE id = ?`, cfg.Commands[0].ID).Scan(&command); err != nil || command != "echo first" {
					t.Fatalf("commands changed: %q %v", command, err)
				}
				d.Close()
				return
			}
			d.Close()
			d, err = Open()
			if err != nil {
				t.Fatal(err)
			}
			defer d.Close()
			configs, err := d.ListRunConfigsT(ctx, set.ID)
			if err != nil || len(configs) != 1 {
				t.Fatalf("configs: %+v %v", configs, err)
			}
			got := configs[0]
			if got.ID != cfg.ID || got.Name != "Dev" || !got.DetectReady || got.Mode != mode || len(got.Commands) != 2 || got.Commands[0].ID != cfg.Commands[0].ID || got.Commands[1].Command != "echo second" {
				t.Fatalf("migration lost data: %+v", got)
			}
			files, _ := filepath.Glob(filepath.Join(dir, "backups", "run-configs-before-named-*.json"))
			if len(files) != 1 {
				t.Fatalf("backups: %v", files)
			}
			raw, err := os.ReadFile(files[0])
			if err != nil {
				t.Fatal(err)
			}
			var backup struct {
				Version  int                `json:"version"`
				Configs  []types.RunConfig  `json:"run_configs"`
				Commands []types.RunCommand `json:"run_commands"`
			}
			if err := json.Unmarshal(raw, &backup); err != nil {
				t.Fatal(err)
			}
			if backup.Version != 1 || len(backup.Configs) != 1 || backup.Configs[0].ID != cfg.ID || len(backup.Commands) != 2 || backup.Commands[0].Command != "echo first" {
				t.Fatalf("incomplete backup: %s", raw)
			}
			if err := applySchema(d.SQL); err != nil {
				t.Fatal(err)
			}
			after, _ := filepath.Glob(filepath.Join(dir, "backups", "*.json"))
			if len(after) != 1 {
				t.Fatal("migration ran again")
			}
		})
	}
}

func TestNamedRunsEmptyAppsRenameCopyAndRollback(t *testing.T) {
	t.Setenv("WM_DATA_DIR", t.TempDir())
	d, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	ctx := context.Background()
	ws, _ := d.CreateWorkspaceT(ctx, "ws", nil, nil)
	app, _ := d.CreateAppT(ctx, ws.ID, "app", t.TempDir())
	set, _ := d.ResolveActive(ctx, app.ID)
	if rows, err := d.ListRunConfigs(ctx, set.ID); err != nil || len(rows) != 0 {
		t.Fatalf("new app has configurations: %+v %v", rows, err)
	}
	name, ready := "My server", true
	first, err := d.SaveNamedRun(ctx, set.ID, "custom-one", types.RunConfigSaveInput{Name: &name, DetectReady: &ready, Commands: []types.RunCommandInput{{Command: "echo hello"}}})
	if err != nil {
		t.Fatal(err)
	}
	name = "Renamed server"
	renamed, err := d.SaveNamedRun(ctx, set.ID, first.Kind, types.RunConfigSaveInput{Name: &name})
	if err != nil || renamed.ID != first.ID || renamed.Commands[0].ID != first.Commands[0].ID || !renamed.DetectReady {
		t.Fatalf("rename lost identity: %+v %v", renamed, err)
	}
	otherName := "Tests"
	second, err := d.SaveNamedRun(ctx, set.ID, "custom-two", types.RunConfigSaveInput{Name: &otherName, Commands: []types.RunCommandInput{{Command: "echo tests"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.SaveNamedRun(ctx, set.ID, second.Kind, types.RunConfigSaveInput{Name: &name, Commands: []types.RunCommandInput{{Command: "should roll back"}}}); err == nil {
		t.Fatal("duplicate name accepted")
	}
	secondAfter, _ := d.GetRunConfigByConfigSetT(ctx, set.ID, second.Kind)
	if secondAfter.Name != otherName || secondAfter.Commands[0].ID != second.Commands[0].ID {
		t.Fatalf("failed save changed data: %+v", secondAfter)
	}
	target, _ := d.CreateConfigSetT(ctx, app.ID, "copy")
	if err := d.CopyFrom(ctx, set.ID, target.ID, &types.CopyParts{Env: false, Templates: false, RunConfigs: []string{first.Kind}}); err != nil {
		t.Fatal(err)
	}
	copied, _ := d.ListRunConfigsT(ctx, target.ID)
	if len(copied) != 1 || copied[0].Name != name || !copied[0].DetectReady || copied[0].Commands[0].Command != "echo hello" {
		t.Fatalf("copy lost data: %+v", copied)
	}
}
