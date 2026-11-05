package gui

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"
)

// TestAppTalksToTheDaemon builds wmd, lets the App start it on demand in a fresh
// data folder, and drives the bound methods through it: data calls, a GUI-only
// call, and git, which runs in this process but reads the app from the daemon.
func TestAppTalksToTheDaemon(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "wmd")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", exe, "../cmd/wmd").CombinedOutput(); err != nil {
		t.Fatalf("build wmd: %v\n%s", err, out)
	}
	t.Setenv("WM_DAEMON", exe)
	t.Setenv("WM_DATA_DIR", t.TempDir())

	app := NewApp(nil)
	t.Cleanup(func() {
		_ = app.link.Call(api.MethodStop, nil, nil)
		app.link.Close()
	})

	ws, err := app.WorkspacesCreate(types.WorkspaceCreateInput{Name: "w"})
	if err != nil {
		t.Fatalf("WorkspacesCreate (starts the daemon): %v", err)
	}
	proj := t.TempDir()
	created, err := app.AppsCreate(ws.ID, types.AppCreateInput{Name: "a", ProjectPath: proj})
	if err != nil {
		t.Fatalf("AppsCreate: %v", err)
	}
	apps, err := app.AppsList(ws.ID)
	if err != nil || len(apps) != 1 || apps[0].ID != created.ID {
		t.Fatalf("AppsList = %v, %v", apps, err)
	}

	// GUI-only: refused to agents, allowed here.
	if _, err := app.SettingsGet(); err != nil {
		t.Errorf("SettingsGet: %v", err)
	}
	if st := app.RunnerStatus(created.ID); st.Running {
		t.Errorf("a fresh app is running: %+v", st)
	}

	// Every bound start command forwards a per-run set without activating it.
	selected, err := app.ConfigSetsCreate(created.ID, types.ConfigSetCreateInput{Name: "staging"})
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{types.KindDev, types.KindProd, types.KindBuild, types.KindSetup} {
		if _, err := app.RunConfigSave(created.ID, kind, types.RunConfigSaveInput{
			Commands: []types.RunCommandInput{{Command: "echo selected-" + kind}},
		}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := app.ConfigSetsActivate(*created.ActiveConfigSetID); err != nil {
		t.Fatal(err)
	}
	for _, start := range []struct {
		name string
		fn   func(int64, int64) (types.StatusEvent, error)
	}{
		{"dev", app.RunnerDev}, {"prod", app.RunnerProd},
		{"build", app.RunnerBuild}, {"setup", app.RunnerSetup}, {"run", app.RunnerRun},
	} {
		st, err := start.fn(created.ID, selected.ID)
		if err != nil || st.ConfigSetID != selected.ID {
			t.Fatalf("%s: status=%+v, err=%v", start.name, st, err)
		}
		deadline := time.Now().Add(20 * time.Second)
		for app.RunnerStatus(created.ID).Running {
			if time.Now().After(deadline) {
				t.Fatalf("%s did not finish", start.name)
			}
			time.Sleep(20 * time.Millisecond)
		}
		current, err := app.AppsGet(created.ID)
		if err != nil || current.ActiveConfigSetID == nil || *current.ActiveConfigSetID != *created.ActiveConfigSetID {
			t.Fatalf("%s changed the active config set: %+v, %v", start.name, current, err)
		}
	}

	// Git runs here; a folder that is not a repo is an error, not a panic.
	if _, err := app.GitInfo(created.ID); err == nil {
		t.Logf("GitInfo on a plain folder returned no error")
	}
	if links, err := app.WorktreeLinksSave(created.ID, []string{"node_modules"}); err != nil || len(links) != 1 {
		t.Errorf("WorktreeLinksSave = %v, %v", links, err)
	}
}
