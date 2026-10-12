package lib

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGlobToRegexp(t *testing.T) {
	re, err := globToRegexp("**/*.env")
	if err != nil {
		t.Fatal(err)
	}
	if !re.MatchString(".env") || !re.MatchString("a/.env") {
		t.Fatalf("**/*.env should match .env and a/.env")
	}
	if re.MatchString("a/env") {
		t.Fatalf("should not match a/env")
	}
}

func TestSearchProjectFilesGitignore(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("secret.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "keep.ts"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("no"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "app.ts"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", dir, "init").Run(); err != nil {
		t.Skip("git not available")
	}
	files, _, errMsg := SearchProjectFiles(dir, "**/*.ts")
	if errMsg != "" {
		t.Fatal(errMsg)
	}
	got := map[string]bool{}
	for _, f := range files {
		got[f] = true
	}
	if !got["keep.ts"] || !got["src/app.ts"] {
		t.Fatalf("got %v", files)
	}
	if got["secret.txt"] {
		t.Fatal("secret.txt should be ignored")
	}
}

func TestSanitizeGlobPattern(t *testing.T) {
	if _, msg := SanitizeGlobPattern("../x"); msg == "" {
		t.Fatal("expected error")
	}
}
