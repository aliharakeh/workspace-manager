package main

import (
	"regexp"

	"workspace-manager/api/types"
)

// blueprintVarRe matches the placeholders a blueprint command may use. It
// mirrors the runner's replacement so a preview cannot drift from what runs.
var blueprintVarRe = regexp.MustCompile(`\{\{\s*(app_name|folder_name|app_dir)\s*\}\}`)

// renderBlueprintCommand replaces the placeholders in a blueprint command.
func renderBlueprintCommand(command string, vars map[string]string) string {
	return blueprintVarRe.ReplaceAllStringFunc(command, func(m string) string {
		return vars[blueprintVarRe.FindStringSubmatch(m)[1]]
	})
}

// renderedBlueprint is what a blueprint would do for a given app name.
type renderedBlueprint struct {
	Cwd      string   `json:"cwd"`
	Folder   string   `json:"folder"`
	Commands []string `json:"commands"`
}

// renderBlueprint renders a blueprint's commands the way the runner will.
func renderBlueprint(bp types.Blueprint, name, folder, parent string) renderedBlueprint {
	appDir := joinPath(parent, folder)
	cwd := parent
	if bp.CreateFolder {
		cwd = appDir
	}
	commands := make([]string, 0, len(bp.Commands))
	for _, c := range bp.Commands {
		commands = append(commands, renderBlueprintCommand(c.Command, map[string]string{
			"app_name":    name,
			"folder_name": folder,
			"app_dir":     appDir,
		}))
	}
	return renderedBlueprint{Cwd: cwd, Folder: folder, Commands: commands}
}

// joinPath joins a parent and a child with the separator the parent uses.
func joinPath(parent, folder string) string {
	sep := "/"
	if contains(parent, "\\") {
		sep = "\\"
	}
	for len(parent) > 0 && (parent[len(parent)-1] == '/' || parent[len(parent)-1] == '\\') {
		parent = parent[:len(parent)-1]
	}
	return parent + sep + folder
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
