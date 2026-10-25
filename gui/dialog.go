package gui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type PickResult struct {
	Path      string
	Cancelled bool
}

func PickFolder(app *application.App, startDir string) (PickResult, error) {
	opts := application.OpenFileDialogOptions{
		Title:                "Select project folder",
		CanCreateDirectories: true,
		CanChooseDirectories: true,
		Directory:            existingDir(startDir),
	}
	path, err := app.Dialog.OpenFileWithOptions(&opts).PromptForSingleSelection()
	if err != nil {
		return PickResult{}, err
	}
	if path == "" {
		return PickResult{Cancelled: true}, nil
	}
	return PickResult{Path: path}, nil
}

func PickFile(app *application.App, startDir string) (PickResult, error) {
	opts := application.OpenFileDialogOptions{
		Title:     "Select template file",
		Directory: existingDir(startDir),
	}
	path, err := app.Dialog.OpenFileWithOptions(&opts).PromptForSingleSelection()
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
