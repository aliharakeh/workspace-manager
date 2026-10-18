package gui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// maxBundleBytes caps the bundle file the GUI will read.
const maxBundleBytes = 32 << 20

// WorkspacesExport returns a workspace's bundle without writing a file.
func (a *App) WorkspacesExport(id int64, options types.WorkspaceExportOptions) (types.WorkspaceBundle, error) {
	return call[types.WorkspaceBundle](a.link, api.MethodWorkspacesExport, m{"id": id, "options": options})
}

// WorkspacesImport creates a workspace from a bundle (or previews it with
// options.dry_run).
func (a *App) WorkspacesImport(bundle types.WorkspaceBundle, options types.WorkspaceImportOptions) (types.WorkspaceImportResult, error) {
	return call[types.WorkspaceImportResult](a.link, api.MethodWorkspacesImport, m{"bundle": bundle, "options": options})
}

// WorkspaceExportToFile asks the daemon for the bundle, shows a save dialog and
// writes the file. The file dialog stays in this process.
func (a *App) WorkspaceExportToFile(id int64, options types.WorkspaceExportOptions) (types.WorkspaceExportFileResult, error) {
	bundle, err := a.WorkspacesExport(id, options)
	if err != nil {
		return types.WorkspaceExportFileResult{}, err
	}
	name := strings.Map(func(r rune) rune {
		if strings.ContainsRune(`<>:"/\|?*`, r) || r < 32 {
			return '-'
		}
		return r
	}, strings.TrimSpace(bundle.Workspace.Name))
	if name == "" {
		name = "workspace"
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export workspace",
		DefaultFilename: name + ".workspace.json",
		Filters:         []runtime.FileFilter{{DisplayName: "Workspace bundle (*.json)", Pattern: "*.json"}},
	})
	if err != nil {
		return types.WorkspaceExportFileResult{}, err
	}
	if path == "" {
		return types.WorkspaceExportFileResult{Cancelled: true}, nil
	}
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return types.WorkspaceExportFileResult{}, err
	}
	// 0600: a bundle may hold secrets.
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return types.WorkspaceExportFileResult{}, err
	}
	return types.WorkspaceExportFileResult{Path: path, Apps: len(bundle.Apps), Secrets: bundle.Secrets}, nil
}

// WorkspaceImportOpenFile shows an open dialog and reads a bundle from the
// chosen file; the daemon validates it when it is imported.
func (a *App) WorkspaceImportOpenFile() (types.WorkspaceBundleFileResult, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Import workspace",
		Filters: []runtime.FileFilter{{DisplayName: "Workspace bundle (*.json)", Pattern: "*.json"}},
	})
	if err != nil {
		return types.WorkspaceBundleFileResult{}, err
	}
	if path == "" {
		return types.WorkspaceBundleFileResult{Cancelled: true}, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return types.WorkspaceBundleFileResult{}, err
	}
	if info.Size() > maxBundleBytes {
		return types.WorkspaceBundleFileResult{}, fmt.Errorf("%s is too large to be a workspace bundle", filepath.Base(path))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return types.WorkspaceBundleFileResult{}, err
	}
	var bundle types.WorkspaceBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return types.WorkspaceBundleFileResult{}, fmt.Errorf("%s is not a workspace bundle: %v", filepath.Base(path), err)
	}
	if bundle.Format != types.WorkspaceBundleFormat {
		return types.WorkspaceBundleFileResult{}, fmt.Errorf("%s is not a workspace bundle", filepath.Base(path))
	}
	return types.WorkspaceBundleFileResult{Path: path, Bundle: &bundle}, nil
}
