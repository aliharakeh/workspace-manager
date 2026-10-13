package ops

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"workspace-manager/api/lib"
	"workspace-manager/api/types"
	"workspace-manager/server/db"
)

// EnvVarsImport imports a .env or YAML file into the app's active config set,
// and stores a matching template. path must be inside the app's project folder:
// the GUI gets it from its file picker and every caller is validated here.
func (o *Ops) EnvVarsImport(appID int64, path string) (types.ImportEnvResult, error) {
	app, err := o.DB.GetAppT(o.Ctx(), appID)
	if err != nil {
		return types.ImportEnvResult{}, err
	}
	ok, resolved, errMsg := lib.ValidateProjectPath(app.ProjectPath)
	if !ok {
		return types.ImportEnvResult{}, fmt.Errorf("%s", errMsg)
	}
	abs, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return types.ImportEnvResult{}, err
	}
	if _, inside := lib.ToProjectRelative(resolved, abs); !inside {
		return types.ImportEnvResult{}, fmt.Errorf("Selected file must be inside the app project directory")
	}
	content, err := os.ReadFile(abs)
	if err != nil {
		return types.ImportEnvResult{}, fmt.Errorf("Failed to read selected file")
	}
	format := lib.DetectImportFormat(abs)
	if format == nil {
		return types.ImportEnvResult{}, fmt.Errorf("Unsupported file format: %s", filepath.Base(abs))
	}
	entries, err := format.Parse(string(content))
	if err != nil {
		return types.ImportEnvResult{}, err
	}
	set, err := o.DB.ResolveActive(o.Ctx(), appID)
	if err != nil {
		return types.ImportEnvResult{}, err
	}
	for _, e := range entries {
		_, _ = o.DB.UpsertEnvVarByKey(o.Ctx(), set.ID, e.Key, e.Value, false)
	}
	result := types.ImportEnvResult{Path: abs, Format: format.Label, Imported: len(entries)}
	vars, err := o.DB.ListEnvVarsT(o.Ctx(), set.ID)
	if err != nil {
		return types.ImportEnvResult{}, err
	}
	result.Vars = vars
	rel, ok := lib.ToProjectRelative(resolved, abs)
	if !ok {
		rel = filepath.ToSlash(filepath.Base(abs))
	}
	tplContent, err := format.ToTemplate(string(content))
	if err == nil {
		existing, findErr := o.DB.GetTemplateByPath(o.Ctx(), db.GetTemplateByPathParams{ConfigSetID: set.ID, FilePath: rel})
		if findErr == nil {
			row, _ := o.DB.UpdateTemplate(o.Ctx(), db.UpdateTemplateParams{FilePath: existing.FilePath, Content: tplContent, IncludeInAi: existing.IncludeInAi, ID: existing.ID})
			result.Template = &types.ImportTemplateResult{ID: row.ID, FilePath: rel, Created: false}
		} else {
			row, createErr := o.DB.CreateTemplate(o.Ctx(), db.CreateTemplateParams{ConfigSetID: set.ID, FilePath: rel, Content: tplContent, IncludeInAi: db.BoolInt(true)})
			if createErr == nil {
				result.Template = &types.ImportTemplateResult{ID: row.ID, FilePath: rel, Created: true}
			}
		}
	}
	o.changed("envVar", appID)
	o.changed("template", appID)
	return result, nil
}
