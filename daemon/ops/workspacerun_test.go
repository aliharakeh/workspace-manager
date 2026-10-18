package ops_test

import (
	"os"
	"runtime"
	"testing"
	"time"

	"workspace-manager/shared/api/types"
)

func TestStartAndStopWorkspace(t *testing.T) {
	o, rec, firstID := setup(t)
	first, err := o.AppsGet(firstID)
	if err != nil {
		t.Fatal(err)
	}
	long := "sleep 30"
	if runtime.GOOS == "windows" {
		long = "ping -n 30 127.0.0.1"
	}
	mode := "parallel"
	// Killed process trees can hold their folder for a moment on Windows, so the
	// apps run in folders the test does not have to remove.
	scratch := func() string {
		dir, err := os.MkdirTemp("", "wm-startall")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.RemoveAll(dir) })
		return dir
	}
	firstDir := scratch()
	if _, err := o.AppsUpdate(firstID, types.AppUpdateInput{ProjectPath: &firstDir}); err != nil {
		t.Fatal(err)
	}
	mk := func(name string, withCommand bool) int64 {
		app, err := o.AppsCreate(first.WorkspaceID, types.AppCreateInput{Name: name, ProjectPath: scratch()})
		if err != nil {
			t.Fatal(err)
		}
		if withCommand {
			if _, err := o.RunConfigSave(app.ID, types.KindDev, types.RunConfigSaveInput{
				Mode: &mode, Commands: []types.RunCommandInput{{Command: long}},
			}); err != nil {
				t.Fatal(err)
			}
		}
		return app.ID
	}
	if _, err := o.RunConfigSave(firstID, types.KindDev, types.RunConfigSaveInput{
		Mode: &mode, Commands: []types.RunCommandInput{{Command: long}},
	}); err != nil {
		t.Fatal(err)
	}
	flagged := mk("flagged", true)
	picked := mk("picked", true)
	t.Cleanup(func() {
		apps, _ := o.AppsList(first.WorkspaceID)
		for _, a := range apps {
			_, _ = o.RunnerStop(a.ID)
		}
		time.Sleep(time.Second) // let the processes release their folders
	})

	// The flag persists, is announced, and is read back with the app.
	rec.changed = nil
	yes := true
	if _, err := o.AppsUpdate(flagged, types.AppUpdateInput{ExcludeFromStartAll: &yes}); err != nil {
		t.Fatal(err)
	}
	if len(rec.changed) == 0 {
		t.Fatal("setting the exclusion was not announced")
	}
	if got, _ := o.AppsGet(flagged); !got.ExcludeFromStartAll {
		t.Fatal("exclusion was not stored")
	}

	statuses := func(res types.WorkspaceRunResult) map[int64]string {
		m := map[int64]string{}
		for _, r := range res.Results {
			m[r.AppID] = r.Status
		}
		return m
	}
	check := func(res types.WorkspaceRunResult, want map[int64]string) {
		t.Helper()
		got := statuses(res)
		for id, status := range want {
			if got[id] != status {
				t.Fatalf("app %d: got %q, want %q (all: %v)", id, got[id], status, res.Results)
			}
		}
	}

	for _, sequential := range []bool{false, true} {
		// "picked" is excluded for this start only; "flagged" by its flag.
		res, err := o.RunnerStartWorkspace(first.WorkspaceID, []int64{picked}, sequential)
		if err != nil {
			t.Fatal(err)
		}
		check(res, map[int64]string{firstID: "started", flagged: "excluded", picked: "excluded"})

		// A second start skips what already runs and reports it as such, not as an error.
		res, err = o.RunnerStartWorkspace(first.WorkspaceID, []int64{picked}, sequential)
		if err != nil {
			t.Fatal(err)
		}
		check(res, map[int64]string{firstID: "skipped", flagged: "excluded", picked: "excluded"})

		// Stop all stops what runs, skips the rest, and honours the exclude list.
		if _, err := o.RunnerRun(picked); err != nil {
			t.Fatal(err)
		}
		res, err = o.RunnerStopWorkspace(first.WorkspaceID, []int64{picked})
		if err != nil {
			t.Fatal(err)
		}
		check(res, map[int64]string{firstID: "stopped", flagged: "skipped", picked: "excluded"})
		if _, err := o.RunnerStop(picked); err != nil {
			t.Fatal(err)
		}
	}

	// An app that cannot start is an error entry; the others still start.
	broken := mk("broken", false)
	res, err := o.RunnerStartWorkspace(first.WorkspaceID, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	check(res, map[int64]string{firstID: "started", picked: "started", flagged: "excluded"})
	if got := statuses(res)[broken]; got != "started" && got != "error" {
		t.Fatalf("broken app: unexpected status %q", got)
	}

	if _, err := o.RunnerStartWorkspace(99999, nil, false); err == nil {
		t.Fatal("an unknown workspace must fail")
	}
}
