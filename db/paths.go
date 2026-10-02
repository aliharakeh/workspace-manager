package db

import (
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

