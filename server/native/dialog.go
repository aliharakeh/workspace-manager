package native

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type PickResult struct {
	Path      string
	Cancelled bool
}

func PickFolder(ctx context.Context, startDir string) (PickResult, error) {
	opts := runtime.OpenDialogOptions{
		Title:                "Select project folder",
		CanCreateDirectories: true,
		DefaultDirectory:     existingDir(startDir),
	}
	path, err := runtime.OpenDirectoryDialog(ctx, opts)
	if err != nil {
		return PickResult{}, err
	}
	if path == "" {
		return PickResult{Cancelled: true}, nil
	}
	return PickResult{Path: path}, nil
}

func PickFile(ctx context.Context, startDir string) (PickResult, error) {
	opts := runtime.OpenDialogOptions{
		Title:            "Select template file",
		DefaultDirectory: existingDir(startDir),
	}
	path, err := runtime.OpenFileDialog(ctx, opts)
	if err != nil {
		return PickResult{}, err
	}
	if path == "" {
		return PickResult{Cancelled: true}, nil
	}
	return PickResult{Path: path}, nil
}

// Wails refuses to open the picker if DefaultDirectory is missing or a symlink.
func existingDir(startDir string) string {
	startDir = strings.TrimSpace(startDir)
	if startDir == "" {
		return ""
	}
	abs, err := filepath.Abs(startDir)
	if err != nil {
		return ""
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		resolved = abs
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return ""
	}
	return resolved
}
