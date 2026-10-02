package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"workspace-manager/types"
)

func blueprintFrom(row Blueprint) (types.Blueprint, error) {
	cmds := []types.BlueprintCommand{}
	if err := json.Unmarshal([]byte(row.Commands), &cmds); err != nil {
		return types.Blueprint{}, fmt.Errorf("Blueprint commands are corrupt: %w", err)
	}
	return types.Blueprint{
		ID: row.ID, Name: row.Name, Description: row.Description, SampleName: row.SampleName,
		CreateFolder: row.CreateFolder != 0, Commands: cmds,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

// cleanBlueprint trims fields and drops blank commands.
func cleanBlueprint(in types.BlueprintInput) (types.BlueprintInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return in, fmt.Errorf("name is required")
	}
	in.Description = strings.TrimSpace(in.Description)
	in.SampleName = strings.TrimSpace(in.SampleName)
	cmds := make([]types.BlueprintCommand, 0, len(in.Commands))
	for _, c := range in.Commands {
		c.Command = strings.TrimSpace(c.Command)
		if c.Command == "" {
			continue
		}
		if c.Label != nil {
			label := strings.TrimSpace(*c.Label)
			c.Label = nil
			if label != "" {
				c.Label = &label
			}
		}
		cmds = append(cmds, c)
	}
	in.Commands = cmds
	return in, nil
}

func (d *DB) ListBlueprintsT(ctx context.Context) ([]types.Blueprint, error) {
	rows, err := d.ListBlueprints(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]types.Blueprint, 0, len(rows))
	for _, r := range rows {
		bp, err := blueprintFrom(r)
		if err != nil {
			return nil, err
		}
		out = append(out, bp)
	}
	return out, nil
}

func (d *DB) GetBlueprintT(ctx context.Context, id int64) (types.Blueprint, error) {
	row, err := d.GetBlueprint(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return types.Blueprint{}, fmt.Errorf("Blueprint not found")
		}
		return types.Blueprint{}, err
	}
	return blueprintFrom(row)
}

func (d *DB) CreateBlueprintT(ctx context.Context, in types.BlueprintInput) (types.Blueprint, error) {
	in, err := cleanBlueprint(in)
	if err != nil {
		return types.Blueprint{}, err
	}
	cmds, _ := json.Marshal(in.Commands)
	row, err := d.CreateBlueprint(ctx, CreateBlueprintParams{
		Name: in.Name, Description: in.Description, SampleName: in.SampleName,
		CreateFolder: BoolInt(in.CreateFolder), Commands: string(cmds),
	})
	if err != nil {
		return types.Blueprint{}, UniqueErr(err, "A blueprint with this name already exists")
	}
	return blueprintFrom(row)
}

func (d *DB) UpdateBlueprintT(ctx context.Context, id int64, in types.BlueprintInput) (types.Blueprint, error) {
	in, err := cleanBlueprint(in)
	if err != nil {
		return types.Blueprint{}, err
	}
	cmds, _ := json.Marshal(in.Commands)
	row, err := d.UpdateBlueprint(ctx, UpdateBlueprintParams{
		Name: in.Name, Description: in.Description, SampleName: in.SampleName,
		CreateFolder: BoolInt(in.CreateFolder), Commands: string(cmds), ID: id,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return types.Blueprint{}, fmt.Errorf("Blueprint not found")
		}
		return types.Blueprint{}, UniqueErr(err, "A blueprint with this name already exists")
	}
	return blueprintFrom(row)
}
