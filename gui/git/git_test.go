package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"workspace-manager/shared/api/types"
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
	if !SamePath(path, filepath.Join(filepath.Dir(dir), "repo-feat-x")) {
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
	if len(res.Conflicts) != 1 || !SamePath(res.Conflicts[0], filepath.Join(b, "f.txt")) {
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
