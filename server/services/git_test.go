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

	"workspace-manager/server/db"
	"workspace-manager/api/types"
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

func TestParseLocalBranch(t *testing.T) {
	b := parseLocalBranch("refs/heads/feat/x\tabc1234\torigin/feat/x\t[ahead 2, behind 1]\tadd x")
	if b.Name != "feat/x" || b.Head != "abc1234" || b.Upstream != "origin/feat/x" || b.Ahead != 2 || b.Behind != 1 || b.Subject != "add x" || b.Gone {
		t.Errorf("got %+v", b)
	}
	b = parseLocalBranch("refs/heads/old\tdef\torigin/old\t[gone]\tmsg")
	if !b.Gone || b.Ahead != 0 {
		t.Errorf("got %+v", b)
	}
	b = parseLocalBranch("refs/heads/plain\tdef\t\t\tmsg")
	if b.Upstream != "" || b.Subject != "msg" {
		t.Errorf("got %+v", b)
	}
}

func TestGitBranchCreateRenameDelete(t *testing.T) {
	dir := initRepo(t)
	if err := GitBranchCreate(dir, "feat/a", ""); err != nil {
		t.Fatal(err)
	}
	if err := GitBranchCreate(dir, "feat/a", ""); err == nil {
		t.Error("creating an existing branch should fail")
	}
	if err := GitBranchCreate(dir, "bad name", ""); err == nil {
		t.Error("an invalid name should be refused")
	}
	if err := GitBranchRename(dir, "feat/a", "feat/b"); err != nil {
		t.Fatal(err)
	}
	wts, _ := GitWorktrees(dir)
	branches, err := GitLocalBranches(dir, wts)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]types.GitLocalBranch{}
	for _, b := range branches {
		names[b.Name] = b
	}
	if _, ok := names["feat/a"]; ok {
		t.Error("old name still present")
	}
	if names["feat/b"].Name == "" || names["main"].Worktree == "" {
		t.Errorf("branches = %+v", branches)
	}
	if err := GitBranchDelete(dir, "main", false); err == nil {
		t.Error("deleting a checked-out branch should fail")
	}
	if err := GitBranchDelete(dir, "feat/b", false); err != nil {
		t.Fatal(err)
	}
}

func TestGitBranchCheckout(t *testing.T) {
	dir := initRepo(t)
	if err := GitBranchCreate(dir, "feat/c", ""); err != nil {
		t.Fatal(err)
	}
	if err := GitBranchCheckout(dir, "feat/c"); err != nil {
		t.Fatal(err)
	}
	if cur, _ := git(dir, "branch", "--show-current"); cur != "feat/c" {
		t.Errorf("current = %q", cur)
	}
	if err := GitBranchCheckout(dir, "nope"); err == nil {
		t.Error("checking out a missing branch should fail")
	}
	if err := GitBranchCheckout(dir, "-x"); err == nil {
		t.Error("a name starting with '-' should be refused")
	}
}

func TestGitBranchPull(t *testing.T) {
	a := initRepo(t)
	root := filepath.Dir(a)
	bare := filepath.Join(root, "remote.git")
	b := filepath.Join(root, "clone")
	run := func(dir string, args ...string) {
		t.Helper()
		if _, err := git(dir, args...); err != nil {
			t.Fatal(err)
		}
	}
	commit := func(dir, content, msg string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		run(dir, "add", "f.txt")
		run(dir, "commit", "-m", msg)
	}
	run(root, "init", "--bare", "-b", "main", bare)
	run(a, "config", "user.email", "t@t")
	run(a, "config", "user.name", "t")
	run(a, "remote", "add", "origin", bare)
	commit(a, "base\n", "base")
	run(a, "push", "-u", "origin", "main")
	run(root, "clone", bare, b)
	run(b, "config", "user.email", "t@t")
	run(b, "config", "user.name", "t")

	// Clean pull: the remote moved ahead, the clone has nothing of its own.
	commit(a, "base\nremote\n", "remote 1")
	run(a, "push")
	run(b, "fetch")
	res, err := GitBranchPull(b, "main")
	if err != nil || len(res.Conflicts) != 0 {
		t.Fatalf("clean pull: %+v, %v", res, err)
	}

	// Conflict: both sides change the same line.
	commit(a, "remote edit\n", "remote 2")
	run(a, "push")
	commit(b, "local edit\n", "local")
	res, err = GitBranchPull(b, "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Conflicts) != 1 || !samePath(res.Conflicts[0], filepath.Join(b, "f.txt")) {
		t.Fatalf("conflicts = %+v", res.Conflicts)
	}
	if file, err := FindWorktreeFile(b, res.Conflicts[0]); err != nil || file == "" {
		t.Errorf("conflicted file should be inside a worktree: %v", err)
	}
	if _, err := FindWorktreeFile(b, filepath.Join(root, "elsewhere.txt")); err == nil {
		t.Error("a file outside the worktrees should be refused")
	}
	if _, err := GitBranchPull(b, "nope"); err == nil {
		t.Error("pulling a missing branch should fail")
	}
}

func TestGitBranchPush(t *testing.T) {
	a := initRepo(t)
	root := filepath.Dir(a)
	bare := filepath.Join(root, "remote.git")
	if _, err := git(root, "init", "--bare", "-b", "main", bare); err != nil {
		t.Fatal(err)
	}
	if _, err := GitBranchPush(a, "main", false); err == nil {
		t.Error("pushing without a remote should fail")
	}
	if _, err := git(a, "remote", "add", "origin", bare); err != nil {
		t.Fatal(err)
	}
	// No upstream: the push publishes the branch and sets its upstream.
	if _, err := GitBranchPush(a, "main", false); err != nil {
		t.Fatal(err)
	}
	if up, _ := git(a, "rev-parse", "--abbrev-ref", "main@{upstream}"); up != "origin/main" {
		t.Errorf("upstream = %q", up)
	}
	// A branch that is not checked out can be published too.
	if err := GitBranchCreate(a, "feat/p", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := GitBranchPush(a, "feat/p", false); err != nil {
		t.Fatal(err)
	}
	if out, _ := git(bare, "branch", "--list", "feat/p"); !strings.Contains(out, "feat/p") {
		t.Errorf("remote branches = %q", out)
	}
	// Rejected when the remote moved, forced with lease after a fetch.
	b := filepath.Join(root, "clone")
	for _, args := range [][]string{{"clone", bare, b}, {"-C", b, "config", "user.email", "t@t"}, {"-C", b, "config", "user.name", "t"}} {
		if _, err := git(root, args...); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := git(b, "commit", "--allow-empty", "-m", "remote"); err != nil {
		t.Fatal(err)
	}
	if _, err := git(b, "push"); err != nil {
		t.Fatal(err)
	}
	if _, err := git(a, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "local"); err != nil {
		t.Fatal(err)
	}
	_, err := GitBranchPush(a, "main", false)
	if err == nil || !strings.Contains(err.Error(), "rejected") {
		t.Fatalf("want a rejected push, got %v", err)
	}
	// The lease fails too until a fetch shows the remote's new commit.
	if _, err := GitBranchPush(a, "main", true); err != nil {
		t.Logf("force-with-lease before a fetch: %v", err)
	}
	if _, err := git(a, "fetch"); err != nil {
		t.Fatal(err)
	}
	if _, err := GitBranchPush(a, "main", true); err != nil {
		t.Fatalf("forced push after a fetch: %v", err)
	}
}
