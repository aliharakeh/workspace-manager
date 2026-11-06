package db

import (
	"context"

	"workspace-manager/shared/api/types"
)

func templateFrom(row Template) types.Template {
	return types.Template{
		ID: row.ID, ConfigSetID: row.ConfigSetID, FilePath: row.FilePath, Content: row.Content,
		Private:   row.Private != 0,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
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
