package lib

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadPackageScripts(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"dev":"vite","build":"tsc"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	got := ReadPackageScripts(dir)
	if !got.HasPackageJSON || got.PackageManager != "pnpm" || len(got.Scripts) != 2 {
		t.Fatalf("%+v", got)
	}
	if got.Scripts[0].Name != "build" || got.Scripts[0].Script != "tsc" || got.Scripts[0].Command != "pnpm run build" {
		t.Fatalf("%+v", got.Scripts[0])
	}
	if got.Scripts[1].Name != "dev" || got.Scripts[1].Command != "pnpm run dev" {
		t.Fatalf("%+v", got.Scripts[1])
	}
}

func TestReadPackageScriptsWithoutPackageJSON(t *testing.T) {
	got := ReadPackageScripts(t.TempDir())
	if got.HasPackageJSON || len(got.Scripts) != 0 || got.PackageManager != "npm" {
		t.Fatalf("%+v", got)
	}
}
