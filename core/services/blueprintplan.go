package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"workspace-manager/api/lib"
	"workspace-manager/api/types"
)

// Plan resolves a blueprint run without running it: the same validation and
// placeholder rendering as Run, returned as data for a caller that runs the
// commands itself. RunID and AskOnError are ignored.
func (b *BlueprintRunner) Plan(ctx context.Context, in types.BlueprintRunInput) (types.BlueprintPlan, error) {
	bp, err := b.d.GetBlueprintT(ctx, in.BlueprintID)
	if err != nil {
		return types.BlueprintPlan{}, err
	}
	if _, err := b.d.GetWorkspaceT(ctx, in.WorkspaceID); err != nil {
		return types.BlueprintPlan{}, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return types.BlueprintPlan{}, fmt.Errorf("name is required")
	}
	folder, err := validateFolderName(in.FolderName)
	if err != nil {
		return types.BlueprintPlan{}, err
	}
	ok, parent, errMsg := lib.ValidateProjectPath(in.ParentPath)
	if !ok {
		return types.BlueprintPlan{}, fmt.Errorf("%s", errMsg)
	}
	appDir := filepath.Join(parent, folder)
	if entries, err := os.ReadDir(appDir); err == nil {
		if len(entries) > 0 {
			return types.BlueprintPlan{}, fmt.Errorf("Folder already exists and is not empty: %s", appDir)
		}
	} else if !os.IsNotExist(err) {
		return types.BlueprintPlan{}, fmt.Errorf("Cannot use folder %s: %v", appDir, err)
	}

	vars := map[string]string{"app_name": name, "folder_name": folder, "app_dir": appDir}
	commands := make([]types.BlueprintCommand, len(bp.Commands))
	for i, c := range bp.Commands {
		commands[i] = types.BlueprintCommand{Label: c.Label, Command: renderBlueprintCommand(c.Command, vars)}
	}
	cwd := parent
	if in.CreateFolder {
		cwd = appDir
	}
	return types.BlueprintPlan{
		Blueprint: bp.Name, AppDir: appDir, WorkingDir: cwd, CreateFolder: in.CreateFolder,
		Commands: commands, GitInit: true,
	}, nil
}
