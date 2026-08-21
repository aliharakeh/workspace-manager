package db

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
)

const appDir = "workspace-manager"
const dbFile = "workspace-manager.sqlite"

func DataDir() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(base, appDir)
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", appDir)
	default:
		base := os.Getenv("XDG_DATA_HOME")
		if base == "" {
			base = filepath.Join(home, ".local", "share")
		}
		return filepath.Join(base, appDir)
	}
}

func DBPath() string {
	return filepath.Join(DataDir(), dbFile)
}

func electrobunAppData() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			return v
		}
		return filepath.Join(home, "AppData", "Local")
	case "darwin":
		return filepath.Join(home, "Library", "Application Support")
	default:
		if v := os.Getenv("XDG_DATA_HOME"); v != "" {
			return v
		}
		return filepath.Join(home, ".local", "share")
	}
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func copySqlite(src, dest string) error {
	if err := copyFile(src, dest); err != nil {
		return err
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		side := src + suffix
		if _, err := os.Stat(side); err == nil {
			_ = copyFile(side, dest+suffix)
		}
	}
	return nil
}

// MigrateLegacyDb one-shot copy from old bun_backend cwd/data or electrobun userData.
func MigrateLegacyDb(destDir string) {
	dest := filepath.Join(destDir, dbFile)
	if _, err := os.Stat(dest); err == nil {
		return
	}
	cwd, _ := os.Getwd()
	candidates := []string{filepath.Join(cwd, "data", dbFile)}
	for _, channel := range []string{"dev", "canary", "stable"} {
		candidates = append(candidates, filepath.Join(electrobunAppData(), "workspace-manager.app", channel, dbFile))
	}
	for _, src := range candidates {
		if _, err := os.Stat(src); err != nil {
			continue
		}
		_ = os.MkdirAll(destDir, 0o755)
		_ = copySqlite(src, dest)
		return
	}
}
