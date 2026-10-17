package services

import (
	"context"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"workspace-manager/daemon/db"
	"workspace-manager/shared/api/types"
)

// wtGit runs git in dir for the worktree test's fixtures.
func wtGit(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func wtSamePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func wtInitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := filepath.Join(t.TempDir(), "repo")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-b", "main"},
		{"-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init"},
	} {
		if out, err := wtGit(dir, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

// An app in a subfolder of the repo runs in the same subfolder of a worktree,
// with the app's templates rendered there and restored afterwards.
func TestRunnerStartInWorktree(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	ctx := context.Background()
	repo := wtInitRepo(t)
	web := filepath.Join(repo, "web")
	if err := os.Mkdir(web, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(web, "config.txt"), []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"add", "."},
		{"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-m", "web"},
	} {
		if _, err := wtGit(repo, args...); err != nil {
			t.Fatal(err)
		}
	}
	wt := repo + "-feat"
	if out, err := wtGit(repo, "worktree", "add", "-b", "feat", wt); err != nil {
		t.Fatalf("git worktree add: %v\n%s", err, out)
	}
	// The app lives in repo/web, so it runs in the same folder of the worktree.
	appDir, root := filepath.Join(wt, "web"), wt

	d := newBlueprintTestDB(t)
	ws, _ := d.CreateWorkspaceT(ctx, "ws", nil, nil)
	app, err := d.CreateAppT(ctx, ws.ID, "web", web)
	if err != nil {
		t.Fatal(err)
	}
	set, _ := d.ResolveActive(ctx, app.ID)
	if _, err := d.UpsertEnvVarByKey(ctx, set.ID, "PORT", "4000", true); err != nil {
		t.Fatal(err)
	}
	if _, err := d.CreateTemplate(ctx, db.CreateTemplateParams{ConfigSetID: set.ID, FilePath: "config.txt", Content: "port={{PORT}}"}); err != nil {
		t.Fatal(err)
	}
	command := "type config.txt"
	if runtime.GOOS != "windows" {
		command = "cat config.txt"
	}
	mode := "sequential"
	if _, err := d.UpsertRunConfig(ctx, set.ID, types.KindBuild, &mode, []types.RunCommandInput{{Command: command}}); err != nil {
		t.Fatal(err)
	}

	runner := NewRunner(d, func(int64, any) {}, nil)
	status, err := runner.StartIn(ctx, app.ID, types.KindBuild, appDir, root, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !wtSamePath(status.Worktree, wt) {
		t.Fatalf("status.Worktree = %q", status.Worktree)
	}
	deadline := time.Now().Add(20 * time.Second)
	for runner.GetStatus(app.ID).Running {
		if time.Now().After(deadline) {
			t.Fatal("build did not finish")
		}
		time.Sleep(50 * time.Millisecond)
	}
	out := runner.GetOutput(app.ID, status.Processes[0].CommandID)
	raw, _ := base64.StdEncoding.DecodeString(out.Data)
	if !strings.Contains(string(raw), "port=4000") {
		t.Fatalf("template was not applied in the worktree:\n%q", raw)
	}
	for _, f := range []string{filepath.Join(web, "config.txt"), filepath.Join(appDir, "config.txt")} {
		if data, _ := os.ReadFile(f); string(data) != "original" {
			t.Errorf("%s = %q, want restored original", f, data)
		}
	}

	// Another config set of the app, not the active one, can run in the worktree.
	staging, err := d.CreateConfigSetT(ctx, app.ID, "staging")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.UpsertEnvVarByKey(ctx, staging.ID, "PORT", "5000", true); err != nil {
		t.Fatal(err)
	}
	if _, err := d.CreateTemplate(ctx, db.CreateTemplateParams{ConfigSetID: staging.ID, FilePath: "config.txt", Content: "port={{PORT}}"}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.UpsertRunConfig(ctx, staging.ID, types.KindBuild, &mode, []types.RunCommandInput{{Command: command}}); err != nil {
		t.Fatal(err)
	}
	status, err = runner.StartIn(ctx, app.ID, types.KindBuild, appDir, root, staging.ID)
	if err != nil {
		t.Fatal(err)
	}
	if status.ConfigSetID != staging.ID {
		t.Fatalf("status.ConfigSetID = %d, want %d", status.ConfigSetID, staging.ID)
	}
	for runner.GetStatus(app.ID).Running {
		if time.Now().After(deadline) {
			t.Fatal("build did not finish")
		}
		time.Sleep(50 * time.Millisecond)
	}
	out = runner.GetOutput(app.ID, status.Processes[0].CommandID)
	raw, _ = base64.StdEncoding.DecodeString(out.Data)
	if !strings.Contains(string(raw), "port=5000") {
		t.Fatalf("staging set was not used:\n%q", raw)
	}

	// A config set of another app is refused.
	other, _ := d.CreateAppT(ctx, ws.ID, "other", t.TempDir())
	otherSet, _ := d.ResolveActive(ctx, other.ID)
	if _, err := runner.StartIn(ctx, app.ID, types.KindBuild, appDir, root, otherSet.ID); err == nil {
		t.Error("starting with another app's config set should fail")
	}
}
