package db

import (
	"context"
	"database/sql"

	"workspace-manager/shared/api/types"
)

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
		ID: cfg.ID, ConfigSetID: cfg.ConfigSetID, Kind: cfg.Kind, Mode: cfg.Mode,
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
