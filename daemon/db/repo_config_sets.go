package db

import (
	"context"
	"database/sql"
	"fmt"

	"workspace-manager/shared/api/types"
)

func configSetFrom(row ConfigSet) types.ConfigSet {
	return types.ConfigSet{
		ID: row.ID, AppID: row.AppID, Name: row.Name, SharedSetID: row.SharedSetID,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

// namedConfigSet converts a config set row and adds the name of the shared set
// it is linked to.
func (d *DB) namedConfigSet(ctx context.Context, row ConfigSet) types.ConfigSet {
	out := configSetFrom(row)
	if row.SharedSetID != nil {
		if s, err := d.GetWorkspaceConfigSet(ctx, *row.SharedSetID); err == nil {
			out.SharedSetName = &s.Name
		}
	}
	return out
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
	return d.namedConfigSet(ctx, row), nil
}

func (d *DB) ListConfigSetsByAppT(ctx context.Context, appID int64) ([]types.ConfigSet, error) {
	rows, err := d.ListConfigSetsByApp(ctx, appID)
	if err != nil {
		return nil, err
	}
	out := make([]types.ConfigSet, 0, len(rows))
	for _, r := range rows {
		out = append(out, d.namedConfigSet(ctx, r))
	}
	return out, nil
}

func (d *DB) CreateConfigSetT(ctx context.Context, appID int64, name string) (types.ConfigSet, error) {
	row, err := d.CreateConfigSet(ctx, CreateConfigSetParams{AppID: appID, Name: name})
	if err != nil {
		return types.ConfigSet{}, UniqueErr(err, "A config set with this name already exists")
	}
	for _, kind := range []string{types.KindDev, types.KindProd, types.KindBuild, types.KindSetup} {
		if _, err := d.GetOrCreateRunConfig(ctx, row.ID, kind); err != nil {
			return types.ConfigSet{}, err
		}
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
	return d.namedConfigSet(ctx, row), nil
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

// ConfigSetDetailT loads a config set with its env vars, templates, and its
// dev, prod, build and setup configs (nil where none was ever created).
func (d *DB) ConfigSetDetailT(ctx context.Context, set types.ConfigSet) (types.ConfigSetDetail, error) {
	envVars, err := d.ListEnvVarsT(ctx, set.ID)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	templates, err := d.ListTemplatesT(ctx, set.ID)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	devCfg, err := d.GetRunConfigByConfigSetT(ctx, set.ID, types.KindDev)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	prodCfg, err := d.GetRunConfigByConfigSetT(ctx, set.ID, types.KindProd)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	buildCfg, err := d.GetRunConfigByConfigSetT(ctx, set.ID, types.KindBuild)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	setupCfg, err := d.GetRunConfigByConfigSetT(ctx, set.ID, types.KindSetup)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	return types.ConfigSetDetail{ConfigSet: set, EnvVars: envVars, Templates: templates, DevConfig: devCfg, ProdConfig: prodCfg, BuildConfig: buildCfg, SetupConfig: setupCfg}, nil
}
