package lib

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"wails_backend/internal/types"
)

// lockfileManagers maps a lockfile to its package manager, checked in order.
var lockfileManagers = []struct {
	file    string
	manager string
}{
	{"bun.lock", "bun"},
	{"bun.lockb", "bun"},
	{"pnpm-lock.yaml", "pnpm"},
	{"yarn.lock", "yarn"},
	{"package-lock.json", "npm"},
}

func DetectPackageManager(projectPath string) string {
	for _, entry := range lockfileManagers {
		if _, err := os.Stat(filepath.Join(projectPath, entry.file)); err == nil {
			return entry.manager
		}
	}
	return "npm"
}

// ReadPackageScripts reads the `scripts` of a project's package.json, if it has one.
func ReadPackageScripts(projectPath string) types.PackageScripts {
	out := types.PackageScripts{
		HasPackageJSON: false,
		PackageManager: DetectPackageManager(projectPath),
		Scripts:        []types.PackageScript{},
	}

	data, err := os.ReadFile(filepath.Join(projectPath, "package.json"))
	if err != nil {
		return out
	}
	out.HasPackageJSON = true

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return out
	}

	names := make([]string, 0, len(pkg.Scripts))
	for name := range pkg.Scripts {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		out.Scripts = append(out.Scripts, types.PackageScript{
			Name:    name,
			Script:  pkg.Scripts[name],
			Command: out.PackageManager + " run " + name,
		})
	}
	return out
}
