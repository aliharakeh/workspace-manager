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

	"workspace-manager/db"
	"workspace-manager/types"
)

func TestParseWorktrees(t *testing.T) {
	out := "worktree C:/repo\nHEAD abc\nbranch refs/heads/main\n\nworktree C:/repo-feat\nHEAD def\ndetached\nlocked\n\n"
	wts := parseWorktrees(out)
	if len(wts) != 2 {
		t.Fatalf("got %d worktrees", len(wts))
	}
	if wts[0].Branch != "main" || wts[0].Head != "abc" {
		t.Errorf("first = %+v", wts[0])
	}
	if !wts[1].Detached || !wts[1].Locked || wts[1].Branch != "" {
		t.Errorf("second = %+v", wts[1])
	}
}

func TestDefaultWorktreePath(t *testing.T) {
	got := DefaultWorktreePath(filepath.Join("x", "repo"), "feat/login")
	if want := filepath.Join("x", "repo-feat-login"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	dir := filepath.Join(root, "repo")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-b", "main"},
		{"-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init"},
	} {
		if _, err := git(dir, args...); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestGitWorktreeAddRemove(t *testing.T) {
	dir := initRepo(t)

	info, err := GitInfo(dir)
	if err != nil || !info.IsRepo || len(info.Worktrees) != 1 || !info.Worktrees[0].Main || !info.Worktrees[0].Current {
		t.Fatalf("info = %+v, err = %v", info, err)
	}

	path, err := GitWorktreeAdd(dir, types.GitWorktreeAddInput{Branch: "feat/x", NewBranch: true})
	if err != nil {
		t.Fatal(err)
	}
	if !samePath(path, filepath.Join(filepath.Dir(dir), "repo-feat-x")) {
		t.Errorf("path = %q", path)
	}
	if _, err := GitWorktreeAdd(dir, types.GitWorktreeAddInput{Branch: "feat/x", Path: "../other"}); err == nil {
		t.Error("checking out a branch that is already checked out should fail")
	}

	wts, _ := GitWorktrees(dir)
	if len(wts) != 2 || wts[1].Branch != "feat/x" {
		t.Fatalf("worktrees = %+v", wts)
	}
	if err := GitWorktreeRemove(dir, dir, false); err == nil {
		t.Error("removing the main worktree should fail")
	}
	if err := GitWorktreeRemove(dir, path, false); err != nil {
		t.Fatal(err)
	}
	if wts, _ := GitWorktrees(dir); len(wts) != 1 {
		t.Errorf("worktrees after remove = %+v", wts)
	}
	branches, _ := GitBranches(dir)
	if len(branches) != 2 {
		t.Errorf("branches = %v (branch should be kept)", branches)
	}
}

func TestGitInfoNotARepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	info, err := GitInfo(t.TempDir())
	if err != nil || info.IsRepo {
		t.Errorf("info = %+v, err = %v", info, err)
	}
}

func TestAppAIWorktreeStaging(t *testing.T) {
	dir := initRepo(t)
	s := newAppAIState(types.ConfigSetDetail{}, dir)

	if out := s.addWorktree(types.GitWorktreeAddInput{Branch: "a", NewBranch: true}); out["error"] != nil {
		t.Fatal(out["error"])
	}
	if out := s.addWorktree(types.GitWorktreeAddInput{Branch: "a", NewBranch: true}); out["error"] == nil {
		t.Error("second add at the same default path should fail")
	}
	if out := s.removeWorktree(types.AppAIWorktreeRemove{Path: dir}); out["error"] == nil {
		t.Error("removing the main worktree should fail")
	}
	p := s.patch()
	if p.Worktrees == nil || len(p.Worktrees.Add) != 1 || !patchHasEdits(p) {
		t.Fatalf("patch = %+v", p.Worktrees)
	}
	// Removing a staged add unstages it; nothing ran yet.
	if out := s.removeWorktree(types.AppAIWorktreeRemove{Path: p.Worktrees.Add[0].Path}); out["error"] != nil {
		t.Fatal(out["error"])
	}
	if s.patch().Worktrees != nil {
		t.Error("expected no staged worktree changes")
	}
	if wts, _ := GitWorktrees(dir); len(wts) != 1 {
		t.Errorf("staging must not touch the repo: %+v", wts)
	}
}

func TestGitWorktreePrune(t *testing.T) {
	dir := initRepo(t)
	path, err := GitWorktreeAdd(dir, types.GitWorktreeAddInput{Branch: "gone", NewBranch: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
	if wts, _ := GitWorktrees(dir); len(wts) != 2 || !wts[1].Prunable {
		t.Fatalf("expected a prunable worktree: %+v", wts)
	}
	if _, err := GitWorktreePrune(dir); err != nil {
		t.Fatal(err)
	}
	if wts, _ := GitWorktrees(dir); len(wts) != 1 {
		t.Errorf("worktrees after prune = %+v", wts)
	}
}

// An app in a subfolder of the repo runs in the same subfolder of a worktree,
// with the app's templates rendered there and restored afterwards.
func TestRunnerStartInWorktree(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	ctx := context.Background()
	repo := initRepo(t)
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
		if _, err := git(repo, args...); err != nil {
			t.Fatal(err)
		}
	}
	wt, err := GitWorktreeAdd(web, types.GitWorktreeAddInput{Branch: "feat", NewBranch: true})
	if err != nil {
		t.Fatal(err)
	}
	appDir, root, err := WorktreeAppDir(web, wt)
	if err != nil {
		t.Fatal(err)
	}
	if !samePath(root, wt) || !samePath(appDir, filepath.Join(wt, "web")) {
		t.Fatalf("appDir=%q root=%q", appDir, root)
	}

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
	if !samePath(status.Worktree, wt) {
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
