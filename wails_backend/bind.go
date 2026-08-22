package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"wails_backend/internal/db"
	"wails_backend/internal/lib"
	"wails_backend/internal/native"
	"wails_backend/internal/services"
	"wails_backend/internal/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) WorkspacesList() ([]types.Workspace, error) {
	return a.db.ListWorkspacesT(a.ctx)
}

func (a *App) WorkspacesCreate(body types.WorkspaceCreateInput) (types.Workspace, error) {
	name := strings.TrimSpace(body.Name)
	if name == "" {
		return types.Workspace{}, fmt.Errorf("name is required")
	}
	return a.db.CreateWorkspaceT(a.ctx, name, body.Icon)
}

func (a *App) WorkspacesUpdate(id int64, body types.WorkspaceUpdateInput) (types.Workspace, error) {
	if body.Name != nil && strings.TrimSpace(*body.Name) == "" {
		return types.Workspace{}, fmt.Errorf("name cannot be empty")
	}
	var name *string
	if body.Name != nil {
		n := strings.TrimSpace(*body.Name)
		name = &n
	}
	return a.db.UpdateWorkspaceT(a.ctx, id, name, body.Icon, body.Icon != nil)
}

func (a *App) WorkspacesDelete(id int64) (types.Ok, error) {
	n, err := a.db.DeleteWorkspace(a.ctx, id)
	if err != nil {
		return types.Ok{}, err
	}
	if n == 0 {
		return types.Ok{}, fmt.Errorf("Workspace not found")
	}
	return types.Ok{Ok: true}, nil
}

func (a *App) AppsList(workspaceID int64) ([]types.App, error) {
	if _, err := a.db.GetWorkspaceT(a.ctx, workspaceID); err != nil {
		return nil, err
	}
	return a.db.ListAppsByWorkspaceT(a.ctx, workspaceID)
}

func (a *App) AppsGet(id int64) (types.App, error) {
	return a.db.GetAppT(a.ctx, id)
}

func (a *App) AppsCreate(workspaceID int64, body types.AppCreateInput) (types.App, error) {
	if _, err := a.db.GetWorkspaceT(a.ctx, workspaceID); err != nil {
		return types.App{}, err
	}
	if strings.TrimSpace(body.Name) == "" {
		return types.App{}, fmt.Errorf("name is required")
	}
	if strings.TrimSpace(body.ProjectPath) == "" {
		return types.App{}, fmt.Errorf("project_path is required")
	}
	ok, resolved, errMsg := lib.ValidateProjectPath(body.ProjectPath)
	if !ok {
		return types.App{}, fmt.Errorf("%s", errMsg)
	}
	return a.db.CreateAppT(a.ctx, workspaceID, strings.TrimSpace(body.Name), resolved)
}

func (a *App) AppsUpdate(id int64, body types.AppUpdateInput) (types.App, error) {
	if body.Name != nil && strings.TrimSpace(*body.Name) == "" {
		return types.App{}, fmt.Errorf("name cannot be empty")
	}
	var name, projectPath *string
	if body.Name != nil {
		n := strings.TrimSpace(*body.Name)
		name = &n
	}
	if body.ProjectPath != nil {
		ok, resolved, errMsg := lib.ValidateProjectPath(*body.ProjectPath)
		if !ok {
			return types.App{}, fmt.Errorf("%s", errMsg)
		}
		projectPath = &resolved
	}
	return a.db.UpdateAppT(a.ctx, id, name, projectPath)
}

func (a *App) AppsDelete(id int64) (types.Ok, error) {
	n, err := a.db.DeleteApp(a.ctx, id)
	if err != nil {
		return types.Ok{}, err
	}
	if n == 0 {
		return types.Ok{}, fmt.Errorf("App not found")
	}
	return types.Ok{Ok: true}, nil
}

func (a *App) AppsOpenInEditor(id int64) (types.Ok, error) {
	app, err := a.db.GetAppT(a.ctx, id)
	if err != nil {
		return types.Ok{}, err
	}
	ok, resolved, errMsg := lib.ValidateProjectPath(app.ProjectPath)
	if !ok {
		return types.Ok{}, fmt.Errorf("%s", errMsg)
	}
	if err := native.OpenInEditor(resolved); err != nil {
		return types.Ok{}, err
	}
	return types.Ok{Ok: true}, nil
}

func (a *App) ConfigSetsList(appID int64) ([]types.ConfigSet, error) {
	if _, err := a.db.GetAppT(a.ctx, appID); err != nil {
		return nil, err
	}
	return a.db.ListConfigSetsByAppT(a.ctx, appID)
}

func (a *App) ConfigSetsGetDetail(id int64) (types.ConfigSetDetail, error) {
	set, err := a.db.GetConfigSetT(a.ctx, id)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	envVars, err := a.db.ListEnvVarsT(a.ctx, id)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	templates, err := a.db.ListTemplatesT(a.ctx, id)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	runCfg, err := a.db.GetRunConfigByConfigSetT(a.ctx, id)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	return types.ConfigSetDetail{ConfigSet: set, EnvVars: envVars, Templates: templates, RunConfig: runCfg}, nil
}

func (a *App) ConfigSetsCreate(appID int64, body types.ConfigSetCreateInput) (types.ConfigSet, error) {
	if _, err := a.db.GetAppT(a.ctx, appID); err != nil {
		return types.ConfigSet{}, err
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		return types.ConfigSet{}, fmt.Errorf("name is required")
	}
	if body.CopyFromID != nil && !db.HasAnyPart(body.Parts) {
		return types.ConfigSet{}, fmt.Errorf("Select at least one part to copy")
	}
	if body.CopyFromID != nil {
		source, err := a.db.GetConfigSetT(a.ctx, *body.CopyFromID)
		if err != nil || source.AppID != appID {
			return types.ConfigSet{}, fmt.Errorf("copy_from_id must be a config set on this app")
		}
	}
	set, err := a.db.CreateConfigSetT(a.ctx, appID, name)
	if err != nil {
		return types.ConfigSet{}, err
	}
	if body.CopyFromID != nil {
		if err := a.db.CopyFrom(a.ctx, *body.CopyFromID, set.ID, body.Parts); err != nil {
			return types.ConfigSet{}, err
		}
	}
	if body.Activate == nil || *body.Activate {
		_, _ = a.db.SetActiveConfigSetT(a.ctx, appID, set.ID)
	}
	return set, nil
}

func (a *App) ConfigSetsUpdate(id int64, body types.ConfigSetUpdateInput) (types.ConfigSet, error) {
	name := strings.TrimSpace(body.Name)
	if name == "" {
		return types.ConfigSet{}, fmt.Errorf("name is required")
	}
	return a.db.UpdateConfigSetT(a.ctx, id, name)
}

func (a *App) ConfigSetsDelete(id int64) (types.Ok, error) {
	set, err := a.db.GetConfigSetT(a.ctx, id)
	if err != nil {
		return types.Ok{}, err
	}
	siblings, err := a.db.ListConfigSetsByAppT(a.ctx, set.AppID)
	if err != nil {
		return types.Ok{}, err
	}
	if len(siblings) <= 1 {
		return types.Ok{}, fmt.Errorf("Cannot delete the last config set")
	}
	app, _ := a.db.GetAppT(a.ctx, set.AppID)
	if _, err := a.db.DeleteConfigSet(a.ctx, id); err != nil {
		return types.Ok{}, err
	}
	if app.ActiveConfigSetID != nil && *app.ActiveConfigSetID == id {
		next, err := a.db.ListConfigSetsByAppT(a.ctx, set.AppID)
		if err == nil && len(next) > 0 {
			_, _ = a.db.SetActiveConfigSetT(a.ctx, set.AppID, next[0].ID)
		}
	}
	return types.Ok{Ok: true}, nil
}

func (a *App) ConfigSetsActivate(id int64) (types.ConfigSetActivateResult, error) {
	set, err := a.db.GetConfigSetT(a.ctx, id)
	if err != nil {
		return types.ConfigSetActivateResult{}, err
	}
	app, err := a.db.SetActiveConfigSetT(a.ctx, set.AppID, set.ID)
	if err != nil {
		return types.ConfigSetActivateResult{}, err
	}
	return types.ConfigSetActivateResult{ID: set.ID, AppID: set.AppID, Name: set.Name, App: app}, nil
}

func (a *App) ConfigSetsCopyFrom(id, sourceID int64, parts *types.CopyParts) (types.ConfigSet, error) {
	target, err := a.db.GetConfigSetT(a.ctx, id)
	if err != nil {
		return types.ConfigSet{}, err
	}
	if !db.HasAnyPart(parts) {
		return types.ConfigSet{}, fmt.Errorf("Select at least one part to copy")
	}
	source, err := a.db.GetConfigSetT(a.ctx, sourceID)
	if err != nil || source.AppID != target.AppID {
		return types.ConfigSet{}, fmt.Errorf("source_id must be a config set on the same app")
	}
	if err := a.db.CopyFrom(a.ctx, sourceID, id, parts); err != nil {
		return types.ConfigSet{}, err
	}
	return a.db.GetConfigSetT(a.ctx, id)
}

func (a *App) EnvVarsList(appID int64) ([]types.EnvVar, error) {
	if _, err := a.db.GetAppT(a.ctx, appID); err != nil {
		return nil, err
	}
	set, err := a.db.ResolveActive(a.ctx, appID)
	if err != nil {
		return nil, err
	}
	return a.db.ListEnvVarsT(a.ctx, set.ID)
}

func (a *App) EnvVarsCreate(appID int64, body types.EnvVarCreateInput) (types.EnvVar, error) {
	if _, err := a.db.GetAppT(a.ctx, appID); err != nil {
		return types.EnvVar{}, err
	}
	key := strings.TrimSpace(body.Key)
	if key == "" {
		return types.EnvVar{}, fmt.Errorf("key is required")
	}
	set, err := a.db.ResolveActive(a.ctx, appID)
	if err != nil {
		return types.EnvVar{}, err
	}
	value := ""
	if body.Value != nil {
		value = *body.Value
	}
	row, err := a.db.CreateEnvVar(a.ctx, db.CreateEnvVarParams{ConfigSetID: set.ID, Key: key, Value: value})
	if err != nil {
		return types.EnvVar{}, db.UniqueErr(err, "Env var key already exists for this config set")
	}
	return types.EnvVar{ID: row.ID, ConfigSetID: row.ConfigSetID, Key: row.Key, Value: row.Value, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (a *App) EnvVarsUpdate(id int64, body types.EnvVarUpdateInput) (types.EnvVar, error) {
	existing, err := a.db.GetEnvVar(a.ctx, id)
	if err != nil {
		return types.EnvVar{}, fmt.Errorf("Env var not found")
	}
	key, value := existing.Key, existing.Value
	if body.Key != nil {
		key = strings.TrimSpace(*body.Key)
		if key == "" {
			return types.EnvVar{}, fmt.Errorf("key cannot be empty")
		}
	}
	if body.Value != nil {
		value = *body.Value
	}
	row, err := a.db.UpdateEnvVar(a.ctx, db.UpdateEnvVarParams{Key: key, Value: value, ID: id})
	if err != nil {
		return types.EnvVar{}, db.UniqueErr(err, "Env var key already exists for this config set")
	}
	return types.EnvVar{ID: row.ID, ConfigSetID: row.ConfigSetID, Key: row.Key, Value: row.Value, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (a *App) EnvVarsDelete(id int64) (types.Ok, error) {
	n, err := a.db.DeleteEnvVar(a.ctx, id)
	if err != nil {
		return types.Ok{}, err
	}
	if n == 0 {
		return types.Ok{}, fmt.Errorf("Env var not found")
	}
	return types.Ok{Ok: true}, nil
}

func (a *App) EnvVarsImport(appID int64) (types.ImportEnvResult, error) {
	app, err := a.db.GetAppT(a.ctx, appID)
	if err != nil {
		return types.ImportEnvResult{}, err
	}
	picked, err := native.PickFile(a.ctx, app.ProjectPath)
	if err != nil {
		return types.ImportEnvResult{}, err
	}
	if picked.Cancelled {
		return types.ImportEnvResult{Cancelled: true}, nil
	}
	content, err := os.ReadFile(picked.Path)
	if err != nil {
		return types.ImportEnvResult{}, fmt.Errorf("Failed to read selected file")
	}
	format := lib.DetectImportFormat(picked.Path)
	if format == nil {
		return types.ImportEnvResult{}, fmt.Errorf("Unsupported file format: %s", filepath.Base(picked.Path))
	}
	entries, err := format.Parse(string(content))
	if err != nil {
		return types.ImportEnvResult{}, err
	}
	set, err := a.db.ResolveActive(a.ctx, appID)
	if err != nil {
		return types.ImportEnvResult{}, err
	}
	for _, e := range entries {
		_, _ = a.db.UpsertEnvVarByKey(a.ctx, set.ID, e.Key, e.Value)
	}
	result := types.ImportEnvResult{Path: picked.Path, Format: format.Label, Imported: len(entries)}
	vars, err := a.db.ListEnvVarsT(a.ctx, set.ID)
	if err != nil {
		return types.ImportEnvResult{}, err
	}
	result.Vars = vars
	rel, ok := lib.ToProjectRelative(app.ProjectPath, picked.Path)
	if !ok {
		rel = filepath.ToSlash(filepath.Base(picked.Path))
	}
	tplContent, err := format.ToTemplate(string(content))
	if err == nil {
		existing, findErr := a.db.GetTemplateByPath(a.ctx, db.GetTemplateByPathParams{ConfigSetID: set.ID, FilePath: rel})
		if findErr == nil {
			row, _ := a.db.UpdateTemplate(a.ctx, db.UpdateTemplateParams{FilePath: existing.FilePath, Content: tplContent, ID: existing.ID})
			result.Template = &types.ImportTemplateResult{ID: row.ID, FilePath: rel, Created: false}
		} else {
			row, createErr := a.db.CreateTemplate(a.ctx, db.CreateTemplateParams{ConfigSetID: set.ID, FilePath: rel, Content: tplContent})
			if createErr == nil {
				result.Template = &types.ImportTemplateResult{ID: row.ID, FilePath: rel, Created: true}
			}
		}
	}
	return result, nil
}

func (a *App) TemplatesList(appID int64) ([]types.Template, error) {
	if _, err := a.db.GetAppT(a.ctx, appID); err != nil {
		return nil, err
	}
	set, err := a.db.ResolveActive(a.ctx, appID)
	if err != nil {
		return nil, err
	}
	return a.db.ListTemplatesT(a.ctx, set.ID)
}

func (a *App) TemplatesCreate(appID int64, body types.TemplateCreateInput) (types.Template, error) {
	if _, err := a.db.GetAppT(a.ctx, appID); err != nil {
		return types.Template{}, err
	}
	path := strings.ReplaceAll(strings.TrimSpace(body.FilePath), "\\", "/")
	if path == "" {
		return types.Template{}, fmt.Errorf("file_path is required")
	}
	set, err := a.db.ResolveActive(a.ctx, appID)
	if err != nil {
		return types.Template{}, err
	}
	content := ""
	if body.Content != nil {
		content = *body.Content
	}
	row, err := a.db.CreateTemplate(a.ctx, db.CreateTemplateParams{ConfigSetID: set.ID, FilePath: path, Content: content})
	if err != nil {
		return types.Template{}, db.UniqueErr(err, "Template for this file already exists")
	}
	return types.Template{ID: row.ID, ConfigSetID: row.ConfigSetID, FilePath: row.FilePath, Content: row.Content, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (a *App) TemplatesUpdate(id int64, body types.TemplateUpdateInput) (types.Template, error) {
	existing, err := a.db.GetTemplate(a.ctx, id)
	if err != nil {
		return types.Template{}, fmt.Errorf("Template not found")
	}
	path, content := existing.FilePath, existing.Content
	if body.FilePath != nil {
		path = strings.ReplaceAll(strings.TrimSpace(*body.FilePath), "\\", "/")
		if path == "" {
			return types.Template{}, fmt.Errorf("file_path cannot be empty")
		}
	}
	if body.Content != nil {
		content = *body.Content
	}
	row, err := a.db.UpdateTemplate(a.ctx, db.UpdateTemplateParams{FilePath: path, Content: content, ID: id})
	if err != nil {
		return types.Template{}, db.UniqueErr(err, "Template for this file already exists")
	}
	return types.Template{ID: row.ID, ConfigSetID: row.ConfigSetID, FilePath: row.FilePath, Content: row.Content, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (a *App) TemplatesDelete(id int64) (types.Ok, error) {
	n, err := a.db.DeleteTemplate(a.ctx, id)
	if err != nil {
		return types.Ok{}, err
	}
	if n == 0 {
		return types.Ok{}, fmt.Errorf("Template not found")
	}
	return types.Ok{Ok: true}, nil
}

func (a *App) RunConfigGet(appID int64) (types.RunConfig, error) {
	if _, err := a.db.GetAppT(a.ctx, appID); err != nil {
		return types.RunConfig{}, err
	}
	set, err := a.db.ResolveActive(a.ctx, appID)
	if err != nil {
		return types.RunConfig{}, err
	}
	return a.db.GetOrCreateRunConfig(a.ctx, set.ID)
}

func (a *App) RunConfigSave(appID int64, body types.RunConfigSaveInput) (types.RunConfig, error) {
	if _, err := a.db.GetAppT(a.ctx, appID); err != nil {
		return types.RunConfig{}, err
	}
	if body.Mode != nil && *body.Mode != "sequential" && *body.Mode != "parallel" {
		return types.RunConfig{}, fmt.Errorf("mode must be sequential or parallel")
	}
	var cmds []types.RunCommandInput
	if body.Commands != nil {
		cmds = make([]types.RunCommandInput, 0, len(body.Commands))
		for _, c := range body.Commands {
			if strings.TrimSpace(c.Command) == "" {
				return types.RunConfig{}, fmt.Errorf("Each command must have a non-empty command string")
			}
			cmds = append(cmds, types.RunCommandInput{Label: c.Label, Command: strings.TrimSpace(c.Command)})
		}
	}
	set, err := a.db.ResolveActive(a.ctx, appID)
	if err != nil {
		return types.RunConfig{}, err
	}
	return a.db.UpsertRunConfig(a.ctx, set.ID, body.Mode, cmds)
}

func (a *App) RunnerStatus(appID int64) types.StatusEvent {
	return a.runner.GetStatus(appID)
}

func (a *App) RunnerWorkspaceStatus(workspaceID int64) ([]types.StatusEvent, error) {
	apps, err := a.db.ListAppsByWorkspaceT(a.ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	out := make([]types.StatusEvent, 0, len(apps))
	for _, app := range apps {
		out = append(out, a.runner.GetStatus(app.ID))
	}
	return out, nil
}

func (a *App) RunnerLogs(appID int64) types.RunnerLogsSnapshot {
	return a.runner.GetSnapshot(appID)
}

func (a *App) RunnerRun(appID int64) (types.StatusEvent, error) {
	if _, err := a.db.GetAppT(a.ctx, appID); err != nil {
		return types.StatusEvent{}, err
	}
	return a.runner.Start(a.ctx, appID)
}

func (a *App) RunnerStop(appID int64) (types.StatusEvent, error) {
	return a.runner.Stop(a.ctx, appID)
}

func (a *App) RunnerReload(appID int64) (types.StatusEvent, error) {
	if _, err := a.db.GetAppT(a.ctx, appID); err != nil {
		return types.StatusEvent{}, err
	}
	return a.runner.Reload(a.ctx, appID)
}

func (a *App) PortsList() (types.PortsListResult, error) {
	procs, err := native.ListListeningProcesses()
	if err != nil {
		return types.PortsListResult{}, err
	}
	out := make([]types.ListeningProcess, 0, len(procs))
	for _, p := range procs {
		out = append(out, types.ListeningProcess{Port: p.Port, PID: p.PID, Name: p.Name})
	}
	return types.PortsListResult{Min: native.UserPortMin, Max: native.UserPortMax, Processes: out}, nil
}

func (a *App) PortsKill(pid int64) (types.PortsKillResult, error) {
	if err := native.KillPid(int(pid)); err != nil {
		return types.PortsKillResult{}, err
	}
	return types.PortsKillResult{Ok: true, PID: pid}, nil
}

func (a *App) ReadyUrlPatternsList() ([]types.ReadyUrlPattern, error) {
	return a.db.ListReadyURLPatternsT(a.ctx)
}

func (a *App) ReadyUrlPatternsCreate(body types.ReadyUrlPatternCreateInput) (types.ReadyUrlPattern, error) {
	label := strings.TrimSpace(body.Label)
	pattern := strings.TrimSpace(body.Pattern)
	if label == "" {
		return types.ReadyUrlPattern{}, fmt.Errorf("label is required")
	}
	if pattern == "" {
		return types.ReadyUrlPattern{}, fmt.Errorf("pattern is required")
	}
	flags := "i"
	if body.Flags != nil {
		flags = *body.Flags
	}
	normalized, err := db.NormalizeFlags(flags)
	if err != nil {
		return types.ReadyUrlPattern{}, err
	}
	if err := services.ValidateReadyURLPattern(pattern, normalized); err != nil {
		return types.ReadyUrlPattern{}, err
	}
	row, err := a.db.CreateReadyURLPatternUser(a.ctx, label, pattern, normalized)
	if err != nil {
		return types.ReadyUrlPattern{}, err
	}
	services.InvalidateReadyURLPatternsCache()
	return types.ReadyUrlPattern{ID: row.ID, Key: row.Key, Label: row.Label, Pattern: row.Pattern, Flags: row.Flags, SortOrder: row.SortOrder, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (a *App) ReadyUrlPatternsUpdate(id int64, body types.ReadyUrlPatternUpdateInput) (types.ReadyUrlPattern, error) {
	existing, err := a.db.GetReadyUrlPattern(a.ctx, id)
	if err != nil {
		return types.ReadyUrlPattern{}, fmt.Errorf("Pattern not found")
	}
	label, pattern, flags := existing.Label, existing.Pattern, existing.Flags
	if body.Label != nil {
		label = strings.TrimSpace(*body.Label)
	}
	if body.Pattern != nil {
		pattern = strings.TrimSpace(*body.Pattern)
	}
	if body.Flags != nil {
		flags, err = db.NormalizeFlags(*body.Flags)
		if err != nil {
			return types.ReadyUrlPattern{}, err
		}
	}
	if label == "" {
		return types.ReadyUrlPattern{}, fmt.Errorf("label cannot be empty")
	}
	if pattern == "" {
		return types.ReadyUrlPattern{}, fmt.Errorf("pattern cannot be empty")
	}
	if err := services.ValidateReadyURLPattern(pattern, flags); err != nil {
		return types.ReadyUrlPattern{}, err
	}
	row, err := a.db.UpdateReadyUrlPattern(a.ctx, db.UpdateReadyUrlPatternParams{Label: label, Pattern: pattern, Flags: flags, ID: id})
	if err != nil {
		return types.ReadyUrlPattern{}, err
	}
	services.InvalidateReadyURLPatternsCache()
	return types.ReadyUrlPattern{ID: row.ID, Key: row.Key, Label: row.Label, Pattern: row.Pattern, Flags: row.Flags, SortOrder: row.SortOrder, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (a *App) ReadyUrlPatternsDelete(id int64) (types.Ok, error) {
	n, err := a.db.DeleteReadyUrlPattern(a.ctx, id)
	if err != nil {
		return types.Ok{}, err
	}
	if n == 0 {
		return types.Ok{}, fmt.Errorf("Pattern not found")
	}
	services.InvalidateReadyURLPatternsCache()
	return types.Ok{Ok: true}, nil
}

func (a *App) SettingsGet() (map[string]string, error) {
	return a.db.SettingsMap(a.ctx)
}

func (a *App) SettingsSet(key string, value string) (map[string]string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}
	if err := a.db.UpsertSetting(a.ctx, db.UpsertSettingParams{Key: key, Value: value}); err != nil {
		return nil, err
	}
	return a.db.SettingsMap(a.ctx)
}

// aiConnectionInfo maps a stored connection to what the UI sees (no API key).
func aiConnectionInfo(name string, cfg services.AIProviderConfig) types.AIConnectionInfo {
	return types.AIConnectionInfo{
		Name:        name,
		Provider:    cfg.Provider,
		BaseURL:     cfg.BaseURL,
		Model:       cfg.Model,
		HasAPIKey:   cfg.APIKey != "",
		Temperature: cfg.Temperature,
	}
}

func aiConfigInfo(store services.AIStore) types.AIConfigInfo {
	info := types.AIConfigInfo{
		Providers: make([]types.AIConnectionInfo, 0, len(store.Providers)),
		Active:    store.Active,
	}
	names := make([]string, 0, len(store.Providers))
	for name := range store.Providers {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		info.Providers = append(info.Providers, aiConnectionInfo(name, store.Providers[name]))
	}
	return info
}

func (a *App) AIConfigGet() (types.AIConfigInfo, error) {
	store, err := services.LoadAIStore()
	if err != nil {
		return types.AIConfigInfo{}, err
	}
	return aiConfigInfo(store), nil
}

// AIConfigSave upserts one connection keyed by its normalized name.
func (a *App) AIConfigSave(body types.AIProviderConfigInput) (types.AIConfigInfo, error) {
	store, err := services.LoadAIStore()
	if err != nil {
		return types.AIConfigInfo{}, err
	}
	if err := store.UpsertAIConnection(body.Name, body.Provider, services.AIProviderConfig{
		BaseURL:     body.BaseURL,
		APIKey:      body.APIKey,
		Model:       body.Model,
		Temperature: body.Temperature,
	}, body.ClearAPIKey); err != nil {
		return types.AIConfigInfo{}, err
	}
	if err := services.SaveAIStore(store); err != nil {
		return types.AIConfigInfo{}, err
	}
	return aiConfigInfo(store), nil
}

func (a *App) AIConfigDelete(name string) (types.AIConfigInfo, error) {
	store, err := services.LoadAIStore()
	if err != nil {
		return types.AIConfigInfo{}, err
	}
	if !store.DeleteAIConnection(name) {
		return types.AIConfigInfo{}, fmt.Errorf("Connection not found")
	}
	if err := services.SaveAIStore(store); err != nil {
		return types.AIConfigInfo{}, err
	}
	return aiConfigInfo(store), nil
}

func (a *App) AIConfigActivate(body types.AIActivateInput) (types.AIConfigInfo, error) {
	store, err := services.LoadAIStore()
	if err != nil {
		return types.AIConfigInfo{}, err
	}
	if err := store.ActivateAIConnection(body.Name); err != nil {
		return types.AIConfigInfo{}, err
	}
	if err := services.SaveAIStore(store); err != nil {
		return types.AIConfigInfo{}, err
	}
	return aiConfigInfo(store), nil
}

func (a *App) AIChat(body types.AIChatInput) (types.AIChatResult, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	text, err := services.ChatAI(ctx, body.System, body.Prompt)
	if err != nil {
		return types.AIChatResult{}, err
	}
	return types.AIChatResult{Text: text}, nil
}

// AITest runs one minimal generation against an unsaved connection payload so
// the UI can verify credentials before saving. Nothing is persisted.
func (a *App) AITest(body types.AITestInput) (types.AITestResult, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	slug := services.NormalizeProvider(body.Provider)
	text, err := services.TestAI(ctx, slug, services.AIProviderConfig{
		BaseURL:     body.BaseURL,
		APIKey:      body.APIKey,
		Model:       body.Model,
		Temperature: body.Temperature,
	})
	if err != nil {
		return types.AITestResult{}, err
	}
	return types.AITestResult{Ok: true, Text: text}, nil
}

func (a *App) FsValidatePath(path string) (types.ValidatePathResult, error) {
	if path == "" {
		return types.ValidatePathResult{}, fmt.Errorf("path is required")
	}
	ok, resolved, errMsg := lib.ValidateProjectPath(path)
	if !ok {
		return types.ValidatePathResult{Ok: false, Error: errMsg}, nil
	}
	return types.ValidatePathResult{Ok: true, Path: resolved}, nil
}

func (a *App) FsPickFolder(body types.FsPickFolderInput) (types.PickFolderResult, error) {
	start := ""
	if body.StartDir != nil {
		start = strings.TrimSpace(*body.StartDir)
	}
	picked, err := native.PickFolder(a.ctx, start)
	if err != nil {
		return types.PickFolderResult{}, err
	}
	if picked.Cancelled {
		return types.PickFolderResult{Cancelled: true}, nil
	}
	return types.PickFolderResult{Path: picked.Path}, nil
}

func (a *App) FsPickFile(body types.FsPickFileInput) (types.PickFileResult, error) {
	dir := ""
	if body.StartDir != nil {
		dir = strings.TrimSpace(*body.StartDir)
	}
	var projectRoot string
	if body.AppID != nil {
		app, err := a.db.GetAppT(a.ctx, *body.AppID)
		if err != nil {
			return types.PickFileResult{}, err
		}
		projectRoot = app.ProjectPath
		if dir == "" {
			dir = app.ProjectPath
		}
	}
	picked, err := native.PickFile(a.ctx, dir)
	if err != nil {
		return types.PickFileResult{}, err
	}
	if picked.Cancelled {
		return types.PickFileResult{Cancelled: true}, nil
	}
	data, err := os.ReadFile(picked.Path)
	if err != nil {
		return types.PickFileResult{}, fmt.Errorf("Failed to read selected file")
	}
	if projectRoot != "" {
		rel, ok := lib.ToProjectRelative(projectRoot, picked.Path)
		if !ok {
			return types.PickFileResult{}, fmt.Errorf("Selected file must be inside the app project directory")
		}
		return types.PickFileResult{Path: picked.Path, RelativePath: rel, Content: string(data)}, nil
	}
	return types.PickFileResult{Path: picked.Path, Content: string(data)}, nil
}

func (a *App) FsPickAppFile(appID int64) (types.PickFileResult, error) {
	app, err := a.db.GetAppT(a.ctx, appID)
	if err != nil {
		return types.PickFileResult{}, err
	}
	picked, err := native.PickFile(a.ctx, app.ProjectPath)
	if err != nil {
		return types.PickFileResult{}, err
	}
	if picked.Cancelled {
		return types.PickFileResult{Cancelled: true}, nil
	}
	rel, ok := lib.ToProjectRelative(app.ProjectPath, picked.Path)
	if !ok {
		return types.PickFileResult{}, fmt.Errorf("Selected file must be inside the app project directory")
	}
	data, err := os.ReadFile(picked.Path)
	if err != nil {
		return types.PickFileResult{}, err
	}
	return types.PickFileResult{Path: picked.Path, RelativePath: rel, Content: string(data)}, nil
}

func (a *App) FsReadAppFile(appID int64, path string) (types.ReadAppFileResult, error) {
	app, err := a.db.GetAppT(a.ctx, appID)
	if err != nil {
		return types.ReadAppFileResult{}, err
	}
	if strings.TrimSpace(path) == "" {
		return types.ReadAppFileResult{}, fmt.Errorf("path is required")
	}
	content, rel, errMsg := lib.ReadProjectFile(app.ProjectPath, path)
	if errMsg != "" {
		return types.ReadAppFileResult{}, fmt.Errorf("%s", errMsg)
	}
	return types.ReadAppFileResult{Ok: true, Content: content, RelativePath: rel}, nil
}

func (a *App) OpenExternal(url string) (types.Ok, error) {
	if strings.TrimSpace(url) == "" {
		return types.Ok{}, fmt.Errorf("url is required")
	}
	runtime.BrowserOpenURL(a.ctx, strings.TrimSpace(url))
	return types.Ok{Ok: true}, nil
}
