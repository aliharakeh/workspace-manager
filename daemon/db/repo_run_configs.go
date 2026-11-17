package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"workspace-manager/shared/api/types"
)

func DefaultRunName(kind string) string {
	switch kind {
	case types.KindDev:
		return "Dev"
	case types.KindProd:
		return "Prod"
	case types.KindBuild:
		return "Build"
	case types.KindSetup:
		return "Setup"
	}
	return kind
}

func (d *DB) ListRunConfigsT(ctx context.Context, setID int64) ([]types.RunConfig, error) {
	rows, err := d.ListRunConfigs(ctx, setID)
	if err != nil {
		return nil, err
	}
	out := []types.RunConfig{}
	for _, row := range rows {
		cfg, err := d.withCommands(ctx, row)
		if err != nil {
			return nil, err
		}
		if cfg.Name == "" && len(cfg.Commands) == 0 {
			continue
		}
		if cfg.Name == "" {
			cfg.Name = DefaultRunName(cfg.Kind)
		}
		out = append(out, cfg)
	}
	return out, nil
}

// SaveNamedRun commits metadata and commands together, including on legacy calls.
func (d *DB) SaveNamedRun(ctx context.Context, setID int64, kind string, body types.RunConfigSaveInput) (types.RunConfig, error) {
	tx, err := d.SQL.BeginTx(ctx, nil)
	if err != nil {
		return types.RunConfig{}, err
	}
	defer tx.Rollback()
	td := &DB{SQL: d.SQL, Queries: d.WithTx(tx)}
	cfg, err := td.UpsertRunConfig(ctx, setID, kind, body.Mode, body.Commands)
	if err != nil {
		return types.RunConfig{}, err
	}
	name, ready := cfg.Name, cfg.DetectReady
	if name == "" {
		name = DefaultRunName(kind)
	}
	if body.Name != nil {
		name = strings.TrimSpace(*body.Name)
	}
	if body.DetectReady != nil {
		ready = *body.DetectReady
	}
	var duplicate int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM run_configs WHERE config_set_id = ? AND id <> ? AND lower(name) = lower(?)`, setID, cfg.ID, name).Scan(&duplicate); err != nil {
		return types.RunConfig{}, err
	}
	if duplicate > 0 {
		return types.RunConfig{}, fmt.Errorf("A run configuration with this name already exists")
	}
	row, err := td.UpdateRunConfigDetails(ctx, UpdateRunConfigDetailsParams{ID: cfg.ID, Name: name, DetectReady: BoolInt(ready)})
	if err != nil {
		return types.RunConfig{}, err
	}
	out, err := td.withCommands(ctx, row)
	if err != nil {
		return types.RunConfig{}, err
	}
	if err := tx.Commit(); err != nil {
		return types.RunConfig{}, err
	}
	return out, nil
}

func runCommandFrom(row RunCommand) types.RunCommand {
	return types.RunCommand{
		ID: row.ID, RunConfigID: row.RunConfigID, Label: row.Label, Command: row.Command,
		SortOrder: row.SortOrder, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func (d *DB) withCommands(ctx context.Context, cfg RunConfig) (types.RunConfig, error) {
	cmds, err := d.ListRunCommands(ctx, cfg.ID)
	if err != nil {
		return types.RunConfig{}, err
	}
	out := types.RunConfig{
		ID: cfg.ID, ConfigSetID: cfg.ConfigSetID, Kind: cfg.Kind, Mode: cfg.Mode, Name: cfg.Name, DetectReady: cfg.DetectReady != 0,
		CreatedAt: cfg.CreatedAt, UpdatedAt: cfg.UpdatedAt,
		Commands: make([]types.RunCommand, 0, len(cmds)),
	}
	for _, c := range cmds {
		out.Commands = append(out.Commands, runCommandFrom(c))
	}
	return out, nil
}

// DefaultMode is the execution mode a new config of the kind starts with: a
// dev and prod start processes together; build and setup run steps in order.
func DefaultMode(kind string) string {
	if kind == types.KindBuild || kind == types.KindSetup {
		return "sequential"
	}
	return "parallel"
}

func (d *DB) GetOrCreateRunConfig(ctx context.Context, configSetID int64, kind string) (types.RunConfig, error) {
	cfg, err := d.GetRunConfigByConfigSet(ctx, GetRunConfigByConfigSetParams{ConfigSetID: configSetID, Kind: kind})
	if err == nil {
		return d.withCommands(ctx, cfg)
	}
	if err != sql.ErrNoRows {
		return types.RunConfig{}, err
	}
	created, err := d.CreateRunConfig(ctx, CreateRunConfigParams{ConfigSetID: configSetID, Kind: kind, Mode: DefaultMode(kind)})
	if err != nil {
		return types.RunConfig{}, err
	}
	created, err = d.UpdateRunConfigDetails(ctx, UpdateRunConfigDetailsParams{ID: created.ID, Name: DefaultRunName(kind), DetectReady: BoolInt(types.IsServing(kind))})
	if err != nil {
		return types.RunConfig{}, err
	}
	return d.withCommands(ctx, created)
}

func (d *DB) GetRunConfigByConfigSetT(ctx context.Context, configSetID int64, kind string) (*types.RunConfig, error) {
	cfg, err := d.GetRunConfigByConfigSet(ctx, GetRunConfigByConfigSetParams{ConfigSetID: configSetID, Kind: kind})
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

func (d *DB) UpsertRunConfig(ctx context.Context, configSetID int64, kind string, mode *string, commands []types.RunCommandInput) (types.RunConfig, error) {
	cfg, err := d.GetRunConfigByConfigSet(ctx, GetRunConfigByConfigSetParams{ConfigSetID: configSetID, Kind: kind})
	if err != nil && err != sql.ErrNoRows {
		return types.RunConfig{}, err
	}
	if err == sql.ErrNoRows {
		m := DefaultMode(kind)
		if mode != nil && *mode != "" {
			m = *mode
		}
		cfg, err = d.CreateRunConfig(ctx, CreateRunConfigParams{ConfigSetID: configSetID, Kind: kind, Mode: m})
		if err != nil {
			return types.RunConfig{}, err
		}
		cfg, err = d.UpdateRunConfigDetails(ctx, UpdateRunConfigDetailsParams{ID: cfg.ID, Name: DefaultRunName(kind), DetectReady: BoolInt(types.IsServing(kind))})
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
