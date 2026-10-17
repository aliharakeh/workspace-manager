package gui

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

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

	app := NewApp()
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

	// Git runs here; a folder that is not a repo is an error, not a panic.
	if _, err := app.GitInfo(created.ID); err == nil {
		t.Logf("GitInfo on a plain folder returned no error")
	}
	if links, err := app.WorktreeLinksSave(created.ID, []string{"node_modules"}); err != nil || len(links) != 1 {
		t.Errorf("WorktreeLinksSave = %v, %v", links, err)
	}
}
