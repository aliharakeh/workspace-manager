package native

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExistingDir(t *testing.T) {
	dir := t.TempDir()
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		resolved = dir
	}
	if got := existingDir(dir); got != resolved && got != dir {
		t.Fatalf("existingDir(%q)=%q, want %q", dir, got, resolved)
	}
	if got := existingDir(""); got != "" {
		t.Fatalf("empty: got %q", got)
	}
	missing := filepath.Join(dir, "nope")
	if got := existingDir(missing); got != "" {
		t.Fatalf("missing: got %q", got)
	}
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := existingDir(file); got != "" {
		t.Fatalf("file: got %q", got)
	}
}
