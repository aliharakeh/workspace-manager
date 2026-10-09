package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"workspace-manager/types"
)

func TestNormalizeWorktreeLinks(t *testing.T) {
	got, err := NormalizeWorktreeLinks([]string{" node_modules/ ", "", "web\\node_modules", "./.env", "."})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"node_modules", "web/node_modules", ".env"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("got %v, want %v", got, want)
	}
	for _, bad := range [][]string{
		{"../x"},
		{"/abs"},
		{"C:/abs"},
		{".git/hooks"},
		{"a", "a/"},
		{"a", "a/b"},
		{"a/b", "a"},
	} {
		if _, err := NormalizeWorktreeLinks(bad); err == nil {
			t.Errorf("%v: expected an error", bad)
		}
	}
}

func stateMap(states []types.WorktreeLinkState) map[string]string {
	m := map[string]string{}
	for _, s := range states {
		m[s.Path] = s.State
	}
	return m
}

func TestLinkWorktree(t *testing.T) {
	dir := initRepo(t)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0o755))
	must(os.WriteFile(filepath.Join(dir, "node_modules", "pkg", "index.js"), []byte("x"), 0o644))
	must(os.WriteFile(filepath.Join(dir, ".env"), []byte("A=1"), 0o644))
	must(os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/\n.env\n"), 0o644))
	if _, err := git(dir, "add", ".gitignore"); err != nil {
		t.Fatal(err)
	}
	if _, err := git(dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-m", "ignore"); err != nil {
		t.Fatal(err)
	}
	wt, err := GitWorktreeAdd(dir, types.GitWorktreeAddInput{Branch: "feat", NewBranch: true})
	must(err)
	must(os.MkdirAll(filepath.Join(wt, "dist"), 0o755))

	links := []string{"node_modules", ".env", "dist", "missing"}
	before := stateMap(WorktreeLinkStates(dir, wt, links))
	if before["node_modules"] != LinkMissing || before[".env"] != LinkMissing || before["missing"] != LinkNoSource {
		t.Fatalf("before = %v", before)
	}

	must(os.MkdirAll(filepath.Join(dir, "dist"), 0o755))
	states := LinkWorktree(dir, wt, wt, links)
	got := stateMap(states)
	if got["node_modules"] != LinkLinked || got[".env"] != LinkLinked {
		t.Fatalf("after = %+v", states)
	}
	if got["dist"] != LinkExists {
		t.Errorf("a real folder in the worktree must be left alone, got %v", got["dist"])
	}
	if b, err := os.ReadFile(filepath.Join(wt, "node_modules", "pkg", "index.js")); err != nil || string(b) != "x" {
		t.Errorf("linked file = %q, %v", b, err)
	}
	// Linking again is a no-op.
	if again := stateMap(LinkWorktree(dir, wt, wt, links)); again["node_modules"] != LinkLinked {
		t.Errorf("relink = %v", again)
	}
	// git must not list the links as untracked.
	if out, err := git(wt, "status", "--porcelain"); err != nil || out != "" {
		t.Errorf("git status in worktree = %q, %v", out, err)
	}

	// Removing the worktree must not touch the app's own copies.
	must(UnlinkWorktree(wt, links))
	if _, err := os.Lstat(filepath.Join(wt, "node_modules")); !os.IsNotExist(err) {
		t.Errorf("link still there: %v", err)
	}
	if _, err := os.Stat(filepath.Join(wt, "dist")); err != nil {
		t.Errorf("real folder was removed: %v", err)
	}
	must(GitWorktreeRemove(dir, wt, true))
	if _, err := os.Stat(filepath.Join(dir, "node_modules", "pkg", "index.js")); err != nil {
		t.Errorf("app's node_modules damaged: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".env")); err != nil {
		t.Errorf("app's .env damaged: %v", err)
	}
}

func TestWorktreeLinksOverride(t *testing.T) {
	dir := initRepo(t)
	if _, ok := WorktreeLinksOverride(dir); ok {
		t.Error("the main worktree has no own list")
	}
	if _, err := GitWorktreeAdd(dir, types.GitWorktreeAddInput{Branch: "bad", NewBranch: true, Links: []string{"../x"}}); err == nil {
		t.Error("expected an error for a path outside the app folder")
	}
	plain, err := GitWorktreeAdd(dir, types.GitWorktreeAddInput{Branch: "plain", NewBranch: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := WorktreeLinksOverride(plain); ok {
		t.Error("a worktree added without links uses the app's list")
	}
	wt, err := GitWorktreeAdd(dir, types.GitWorktreeAddInput{Branch: "own", NewBranch: true, Links: []string{"node_modules/", "web\\.env"}})
	if err != nil {
		t.Fatal(err)
	}
	got, ok := WorktreeLinksOverride(wt)
	if !ok || strings.Join(got, ",") != "node_modules,web/.env" {
		t.Errorf("override = %v, %v", got, ok)
	}
	info, err := GitInfo(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range info.Worktrees {
		if samePath(w.Path, wt) && len(w.SharedLinks) != 2 {
			t.Errorf("GitInfo shared_links = %v", w.SharedLinks)
		}
	}

	// An extra path from the override is linked and unlinked like the app's.
	if err := os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if s := stateMap(LinkWorktree(dir, wt, wt, got)); s["node_modules"] != LinkLinked {
		t.Fatalf("link = %v", s)
	}
	if err := UnlinkWorktree(wt, got); err != nil {
		t.Fatal(err)
	}
	if err := GitWorktreeRemove(dir, wt, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "node_modules", "pkg")); err != nil {
		t.Errorf("app's node_modules damaged: %v", err)
	}
}

func TestCreateLinkJunctionIsDetected(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "sub", "link")
	how, err := createLink(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("created a %s", how)
	if !isLink(dst) {
		t.Errorf("%s link not detected as a link", how)
	}
	if state, _ := linkState(src, dst); state != LinkLinked {
		t.Errorf("state = %s", state)
	}
}
