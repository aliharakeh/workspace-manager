package db

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"wails_backend/internal/types"
)

func workspaceFrom(row Workspace) types.Workspace {
	return types.Workspace{
		ID: row.ID, Name: row.Name, Icon: row.Icon,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func appFromRow(id, workspaceID int64, name, projectPath string, activeID *int64, createdAt, updatedAt string, activeName *string) types.App {
	return types.App{
		ID: id, WorkspaceID: workspaceID, Name: name, ProjectPath: projectPath,
		ActiveConfigSetID: activeID, ActiveConfigSetName: activeName,
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
}

func appFromGet(row GetAppRow) types.App {
	return appFromRow(row.ID, row.WorkspaceID, row.Name, row.ProjectPath, row.ActiveConfigSetID, row.CreatedAt, row.UpdatedAt, row.ActiveConfigSetName)
}

func appFromList(row ListAppsByWorkspaceRow) types.App {
	return appFromRow(row.ID, row.WorkspaceID, row.Name, row.ProjectPath, row.ActiveConfigSetID, row.CreatedAt, row.UpdatedAt, row.ActiveConfigSetName)
}

func configSetFrom(row ConfigSet) types.ConfigSet {
	return types.ConfigSet{
		ID: row.ID, AppID: row.AppID, Name: row.Name,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func EnvVarFrom(row EnvVar) types.EnvVar {
	return types.EnvVar{
		ID: row.ID, ConfigSetID: row.ConfigSetID, Key: row.Key, Value: row.Value,
		IncludeInAI: row.IncludeInAi != 0,
		CreatedAt:   row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func templateFrom(row Template) types.Template {
	return types.Template{
		ID: row.ID, ConfigSetID: row.ConfigSetID, FilePath: row.FilePath, Content: row.Content,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func runCommandFrom(row RunCommand) types.RunCommand {
	return types.RunCommand{
		ID: row.ID, RunConfigID: row.RunConfigID, Label: row.Label, Command: row.Command,
		SortOrder: row.SortOrder, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func readyURLFrom(row ReadyUrlPattern) types.ReadyUrlPattern {
	return types.ReadyUrlPattern{
		ID: row.ID, Key: row.Key, Label: row.Label, Pattern: row.Pattern, Flags: row.Flags,
		SortOrder: row.SortOrder, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func (d *DB) withCommands(ctx context.Context, cfg RunConfig) (types.RunConfig, error) {
	cmds, err := d.ListRunCommands(ctx, cfg.ID)
	if err != nil {
		return types.RunConfig{}, err
	}
	out := types.RunConfig{
		ID: cfg.ID, ConfigSetID: cfg.ConfigSetID, Mode: cfg.Mode,
		CreatedAt: cfg.CreatedAt, UpdatedAt: cfg.UpdatedAt,
		Commands: make([]types.RunCommand, 0, len(cmds)),
	}
	for _, c := range cmds {
		out.Commands = append(out.Commands, runCommandFrom(c))
	}
	return out, nil
}

func (d *DB) ListWorkspacesT(ctx context.Context) ([]types.Workspace, error) {
	rows, err := d.ListWorkspaces(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]types.Workspace, 0, len(rows))
	for _, r := range rows {
		out = append(out, workspaceFrom(r))
	}
	return out, nil
}

func (d *DB) GetWorkspaceT(ctx context.Context, id int64) (types.Workspace, error) {
	row, err := d.GetWorkspace(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return types.Workspace{}, fmt.Errorf("Workspace not found")
		}
		return types.Workspace{}, err
	}
	return workspaceFrom(row), nil
}

func (d *DB) CreateWorkspaceT(ctx context.Context, name string, icon *string) (types.Workspace, error) {
	row, err := d.CreateWorkspace(ctx, CreateWorkspaceParams{Name: name, Icon: icon})
	if err != nil {
		return types.Workspace{}, err
	}
	return workspaceFrom(row), nil
}

func (d *DB) UpdateWorkspaceT(ctx context.Context, id int64, name *string, icon *string, iconSet bool) (types.Workspace, error) {
	existing, err := d.GetWorkspace(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return types.Workspace{}, fmt.Errorf("Workspace not found")
		}
		return types.Workspace{}, err
	}
	nextName := existing.Name
	if name != nil {
		nextName = *name
	}
	nextIcon := existing.Icon
	if iconSet {
		nextIcon = icon
	}
	row, err := d.UpdateWorkspace(ctx, UpdateWorkspaceParams{Name: nextName, Icon: nextIcon, ID: id})
	if err != nil {
		return types.Workspace{}, err
	}
	return workspaceFrom(row), nil
}

func (d *DB) GetAppT(ctx context.Context, id int64) (types.App, error) {
	row, err := d.GetApp(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return types.App{}, fmt.Errorf("App not found")
		}
		return types.App{}, err
	}
	return appFromGet(row), nil
}

func (d *DB) ListAppsByWorkspaceT(ctx context.Context, workspaceID int64) ([]types.App, error) {
	rows, err := d.ListAppsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	out := make([]types.App, 0, len(rows))
	for _, r := range rows {
		out = append(out, appFromList(r))
	}
	return out, nil
}

func (d *DB) CreateAppT(ctx context.Context, workspaceID int64, name, projectPath string) (types.App, error) {
	tx, err := d.SQL.BeginTx(ctx, nil)
	if err != nil {
		return types.App{}, err
	}
	defer func() { _ = tx.Rollback() }()
	q := d.WithTx(tx)
	app, err := q.CreateApp(ctx, CreateAppParams{WorkspaceID: workspaceID, Name: name, ProjectPath: projectPath})
	if err != nil {
		return types.App{}, err
	}
	set, err := q.CreateConfigSet(ctx, CreateConfigSetParams{AppID: app.ID, Name: "Default"})
	if err != nil {
		return types.App{}, err
	}
	if _, err := q.CreateRunConfig(ctx, CreateRunConfigParams{ConfigSetID: set.ID, Mode: "parallel"}); err != nil {
		return types.App{}, err
	}
	if _, err := q.SetActiveConfigSet(ctx, SetActiveConfigSetParams{ActiveConfigSetID: &set.ID, ID: app.ID}); err != nil {
		return types.App{}, err
	}
	if err := tx.Commit(); err != nil {
		return types.App{}, err
	}
	return d.GetAppT(ctx, app.ID)
}

func (d *DB) UpdateAppT(ctx context.Context, id int64, name, projectPath *string) (types.App, error) {
	existing, err := d.GetApp(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return types.App{}, fmt.Errorf("App not found")
		}
		return types.App{}, err
	}
	nextName, nextPath := existing.Name, existing.ProjectPath
	if name != nil {
		nextName = *name
	}
	if projectPath != nil {
		nextPath = *projectPath
	}
	if _, err := d.UpdateApp(ctx, UpdateAppParams{Name: nextName, ProjectPath: nextPath, ID: id}); err != nil {
		return types.App{}, err
	}
	return d.GetAppT(ctx, id)
}

func (d *DB) SetActiveConfigSetT(ctx context.Context, appID, configSetID int64) (types.App, error) {
	if _, err := d.SetActiveConfigSet(ctx, SetActiveConfigSetParams{ActiveConfigSetID: &configSetID, ID: appID}); err != nil {
		if err == sql.ErrNoRows {
			return types.App{}, fmt.Errorf("App not found")
		}
		return types.App{}, err
	}
	return d.GetAppT(ctx, appID)
}

func (d *DB) GetConfigSetT(ctx context.Context, id int64) (types.ConfigSet, error) {
	row, err := d.GetConfigSet(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return types.ConfigSet{}, fmt.Errorf("Config set not found")
		}
		return types.ConfigSet{}, err
	}
	return configSetFrom(row), nil
}

func (d *DB) ListConfigSetsByAppT(ctx context.Context, appID int64) ([]types.ConfigSet, error) {
	rows, err := d.ListConfigSetsByApp(ctx, appID)
	if err != nil {
		return nil, err
	}
	out := make([]types.ConfigSet, 0, len(rows))
	for _, r := range rows {
		out = append(out, configSetFrom(r))
	}
	return out, nil
}

func (d *DB) CreateConfigSetT(ctx context.Context, appID int64, name string) (types.ConfigSet, error) {
	row, err := d.CreateConfigSet(ctx, CreateConfigSetParams{AppID: appID, Name: name})
	if err != nil {
		return types.ConfigSet{}, UniqueErr(err, "A config set with this name already exists")
	}
	if _, err := d.GetOrCreateRunConfig(ctx, row.ID); err != nil {
		return types.ConfigSet{}, err
	}
	return configSetFrom(row), nil
}

func (d *DB) UpdateConfigSetT(ctx context.Context, id int64, name string) (types.ConfigSet, error) {
	row, err := d.UpdateConfigSet(ctx, UpdateConfigSetParams{Name: name, ID: id})
	if err != nil {
		if err == sql.ErrNoRows {
			return types.ConfigSet{}, fmt.Errorf("Config set not found")
		}
		return types.ConfigSet{}, UniqueErr(err, "A config set with this name already exists")
	}
	return configSetFrom(row), nil
}

func (d *DB) ResolveActive(ctx context.Context, appID int64) (types.ConfigSet, error) {
	app, err := d.GetApp(ctx, appID)
	if err != nil {
		if err == sql.ErrNoRows {
			return types.ConfigSet{}, fmt.Errorf("App not found")
		}
		return types.ConfigSet{}, err
	}
	if app.ActiveConfigSetID != nil {
		active, err := d.GetConfigSet(ctx, *app.ActiveConfigSetID)
		if err == nil && active.AppID == appID {
			return configSetFrom(active), nil
		}
	}
	existing, err := d.ListConfigSetsByAppT(ctx, appID)
	if err != nil {
		return types.ConfigSet{}, err
	}
	if len(existing) > 0 {
		_, _ = d.SetActiveConfigSet(ctx, SetActiveConfigSetParams{ActiveConfigSetID: &existing[0].ID, ID: appID})
		return existing[0], nil
	}
	created, err := d.CreateConfigSetT(ctx, appID, "Default")
	if err != nil {
		return types.ConfigSet{}, err
	}
	_, _ = d.SetActiveConfigSet(ctx, SetActiveConfigSetParams{ActiveConfigSetID: &created.ID, ID: appID})
	return created, nil
}

func (d *DB) GetOrCreateRunConfig(ctx context.Context, configSetID int64) (types.RunConfig, error) {
	cfg, err := d.GetRunConfigByConfigSet(ctx, configSetID)
	if err == nil {
		return d.withCommands(ctx, cfg)
	}
	if err != sql.ErrNoRows {
		return types.RunConfig{}, err
	}
	created, err := d.CreateRunConfig(ctx, CreateRunConfigParams{ConfigSetID: configSetID, Mode: "parallel"})
	if err != nil {
		return types.RunConfig{}, err
	}
	return d.withCommands(ctx, created)
}

func (d *DB) GetRunConfigByConfigSetT(ctx context.Context, configSetID int64) (*types.RunConfig, error) {
	cfg, err := d.GetRunConfigByConfigSet(ctx, configSetID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	out, err := d.withCommands(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (d *DB) UpsertRunConfig(ctx context.Context, configSetID int64, mode *string, commands []types.RunCommandInput) (types.RunConfig, error) {
	cfg, err := d.GetRunConfigByConfigSet(ctx, configSetID)
	if err != nil && err != sql.ErrNoRows {
		return types.RunConfig{}, err
	}
	if err == sql.ErrNoRows {
		m := "parallel"
		if mode != nil && *mode != "" {
			m = *mode
		}
		cfg, err = d.CreateRunConfig(ctx, CreateRunConfigParams{ConfigSetID: configSetID, Mode: m})
		if err != nil {
			return types.RunConfig{}, err
		}
	} else if mode != nil && *mode != "" {
		cfg, err = d.UpdateRunConfigMode(ctx, UpdateRunConfigModeParams{Mode: *mode, ID: cfg.ID})
		if err != nil {
			return types.RunConfig{}, err
		}
	}
	if commands != nil {
		if err := d.DeleteRunCommands(ctx, cfg.ID); err != nil {
			return types.RunConfig{}, err
		}
		for i, cmd := range commands {
			if _, err := d.CreateRunCommand(ctx, CreateRunCommandParams{
				RunConfigID: cfg.ID, Label: cmd.Label, Command: cmd.Command, SortOrder: int64(i),
			}); err != nil {
				return types.RunConfig{}, err
			}
		}
		cfg, err = d.TouchRunConfig(ctx, cfg.ID)
		if err != nil {
			return types.RunConfig{}, err
		}
	}
	return d.withCommands(ctx, cfg)
}

func (d *DB) ListEnvVarsT(ctx context.Context, configSetID int64) ([]types.EnvVar, error) {
	rows, err := d.ListEnvVarsByConfigSet(ctx, configSetID)
	if err != nil {
		return nil, err
	}
	out := make([]types.EnvVar, 0, len(rows))
	for _, r := range rows {
		out = append(out, EnvVarFrom(r))
	}
	return out, nil
}

func (d *DB) EnvToRecord(ctx context.Context, configSetID int64) (map[string]string, error) {
	rows, err := d.ListEnvVarsByConfigSet(ctx, configSetID)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, r := range rows {
		out[r.Key] = r.Value
	}
	return out, nil
}

func BoolInt(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

func (d *DB) UpsertEnvVarByKey(ctx context.Context, configSetID int64, key, value string, includeInAI bool) (types.EnvVar, error) {
	existing, err := d.GetEnvVarByKey(ctx, GetEnvVarByKeyParams{ConfigSetID: configSetID, Key: key})
	if err == nil {
		row, err := d.UpdateEnvVar(ctx, UpdateEnvVarParams{
			Key: existing.Key, Value: value, IncludeInAi: BoolInt(includeInAI), ID: existing.ID,
		})
		if err != nil {
			return types.EnvVar{}, err
		}
		return EnvVarFrom(row), nil
	}
	if err != sql.ErrNoRows {
		return types.EnvVar{}, err
	}
	row, err := d.CreateEnvVar(ctx, CreateEnvVarParams{
		ConfigSetID: configSetID, Key: key, Value: value, IncludeInAi: BoolInt(includeInAI),
	})
	if err != nil {
		return types.EnvVar{}, err
	}
	return EnvVarFrom(row), nil
}

func (d *DB) ListTemplatesT(ctx context.Context, configSetID int64) ([]types.Template, error) {
	rows, err := d.ListTemplatesByConfigSet(ctx, configSetID)
	if err != nil {
		return nil, err
	}
	out := make([]types.Template, 0, len(rows))
	for _, r := range rows {
		out = append(out, templateFrom(r))
	}
	return out, nil
}

func isPartEnabled(v any) bool {
	if v == nil {
		return true
	}
	switch t := v.(type) {
	case bool:
		return t
	case []any:
		return len(t) > 0
	case []string:
		return len(t) > 0
	case []int64:
		return len(t) > 0
	case []float64:
		return len(t) > 0
	default:
		return false
	}
}

func HasAnyPart(parts *types.CopyParts) bool {
	if parts == nil {
		return true
	}
	return isPartEnabled(parts.Env) || isPartEnabled(parts.Templates) || isPartEnabled(parts.Run)
}

func stringList(v any) (all bool, items []string, skip bool) {
	if v == nil {
		return true, nil, false
	}
	switch t := v.(type) {
	case bool:
		if !t {
			return false, nil, true
		}
		return true, nil, false
	case []any:
		for _, x := range t {
			items = append(items, fmt.Sprint(x))
		}
		return false, items, false
	case []string:
		return false, t, false
	default:
		return true, nil, false
	}
}

func intList(v any) (all bool, ids []int64, skip bool) {
	if v == nil {
		return true, nil, false
	}
	switch t := v.(type) {
	case bool:
		if !t {
			return false, nil, true
		}
		return true, nil, false
	case []any:
		for _, x := range t {
			switch n := x.(type) {
			case float64:
				ids = append(ids, int64(n))
			case int:
				ids = append(ids, int64(n))
			case int64:
				ids = append(ids, n)
			case string:
				parsed, err := strconv.ParseInt(n, 10, 64)
				if err == nil {
					ids = append(ids, parsed)
				}
			}
		}
		return false, ids, false
	case []int64:
		return false, t, false
	default:
		return true, nil, false
	}
}

func containsStr(items []string, s string) bool {
	for _, x := range items {
		if x == s {
			return true
		}
	}
	return false
}

func containsInt(items []int64, n int64) bool {
	for _, x := range items {
		if x == n {
			return true
		}
	}
	return false
}

func (d *DB) CopyFrom(ctx context.Context, sourceID, targetID int64, parts *types.CopyParts) error {
	source, err := d.GetConfigSetT(ctx, sourceID)
	if err != nil {
		return err
	}
	target, err := d.GetConfigSetT(ctx, targetID)
	if err != nil {
		return err
	}
	if source.AppID != target.AppID {
		return fmt.Errorf("Config sets must belong to the same app")
	}
	if sourceID == targetID {
		return nil
	}
	envAll, envItems, envSkip := stringList(nil)
	tplAll, tplItems, tplSkip := stringList(nil)
	runAll, runIDs, runSkip := intList(nil)
	if parts != nil {
		envAll, envItems, envSkip = stringList(parts.Env)
		tplAll, tplItems, tplSkip = stringList(parts.Templates)
		runAll, runIDs, runSkip = intList(parts.Run)
	}
	if envSkip && tplSkip && runSkip {
		return fmt.Errorf("Select at least one part to copy")
	}

	if !envSkip {
		sourceEnv, err := d.ListEnvVarsT(ctx, sourceID)
		if err != nil {
			return err
		}
		if err := d.DeleteEnvVarsByConfigSet(ctx, targetID); err != nil {
			return err
		}
		for _, v := range sourceEnv {
			if envAll || containsStr(envItems, v.Key) {
				if _, err := d.CreateEnvVar(ctx, CreateEnvVarParams{
					ConfigSetID: targetID, Key: v.Key, Value: v.Value, IncludeInAi: BoolInt(v.IncludeInAI),
				}); err != nil {
					return err
				}
			}
		}
	}
	if !tplSkip {
		sourceTpl, err := d.ListTemplatesT(ctx, sourceID)
		if err != nil {
			return err
		}
		if err := d.DeleteTemplatesByConfigSet(ctx, targetID); err != nil {
			return err
		}
		for _, t := range sourceTpl {
			if tplAll || containsStr(tplItems, t.FilePath) {
				if _, err := d.CreateTemplate(ctx, CreateTemplateParams{ConfigSetID: targetID, FilePath: t.FilePath, Content: t.Content}); err != nil {
					return err
				}
			}
		}
	}
	if !runSkip {
		sourceRun, err := d.GetRunConfigByConfigSetT(ctx, sourceID)
		if err != nil {
			return err
		}
		var cmds []types.RunCommandInput
		mode := "parallel"
		if sourceRun != nil {
			mode = sourceRun.Mode
			for _, c := range sourceRun.Commands {
				if runAll || containsInt(runIDs, c.ID) {
					cmds = append(cmds, types.RunCommandInput{Label: c.Label, Command: c.Command})
				}
			}
		}
		_, err = d.UpsertRunConfig(ctx, targetID, &mode, cmds)
		return err
	}
	return nil
}

func (d *DB) SettingsMap(ctx context.Context) (map[string]string, error) {
	rows, err := d.ListSettings(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, r := range rows {
		out[r.Key] = r.Value
	}
	return out, nil
}

func (d *DB) ListReadyURLPatternsT(ctx context.Context) ([]types.ReadyUrlPattern, error) {
	if err := d.EnsureReadyURLPatternsSeeded(ctx); err != nil {
		return nil, err
	}
	rows, err := d.ListReadyUrlPatterns(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]types.ReadyUrlPattern, 0, len(rows))
	for _, r := range rows {
		out = append(out, readyURLFrom(r))
	}
	return out, nil
}

func NormalizeFlags(flags string) (string, error) {
	next := strings.TrimSpace(flags)
	if next == "" {
		next = "i"
	}
	for _, c := range next {
		if !strings.ContainsRune("gimsuy", c) {
			return "", fmt.Errorf("Invalid regex flags")
		}
	}
	return next, nil
}
