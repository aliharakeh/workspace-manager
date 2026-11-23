package db

import (
	"context"
	"database/sql"
	"fmt"

	"workspace-manager/shared/api/types"
)

func workspaceFrom(row Workspace) types.Workspace {
	return types.Workspace{
		ID: row.ID, Name: row.Name, Color: row.Color, LogoPath: LogoPath(row.Icon), Pinned: row.Pinned != 0,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
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

// CreateWorkspaceT creates a workspace; logo is a stored logo's file name or nil.
func (d *DB) CreateWorkspaceT(ctx context.Context, name string, logo, color *string) (types.Workspace, error) {
	row, err := d.CreateWorkspace(ctx, CreateWorkspaceParams{Name: name, Icon: logo, Color: color})
	if err != nil {
		return types.Workspace{}, err
	}
	return workspaceFrom(row), nil
}

// UpdateWorkspaceT changes the fields that are set; logo is a stored logo's
// file name or nil. It returns the logo file name the workspace had before.
func (d *DB) UpdateWorkspaceT(ctx context.Context, id int64, name *string, logo *string, logoSet bool, color *string, colorSet bool) (types.Workspace, *string, error) {
	existing, err := d.GetWorkspace(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return types.Workspace{}, nil, fmt.Errorf("Workspace not found")
		}
		return types.Workspace{}, nil, err
	}
	nextName := existing.Name
	if name != nil {
		nextName = *name
	}
	nextIcon := existing.Icon
	if logoSet {
		nextIcon = logo
	}
	nextColor := existing.Color
	if colorSet {
		nextColor = color
	}
	row, err := d.UpdateWorkspace(ctx, UpdateWorkspaceParams{Name: nextName, Icon: nextIcon, Color: nextColor, ID: id})
	if err != nil {
		return types.Workspace{}, nil, err
	}
	return workspaceFrom(row), existing.Icon, nil
}

// ReorderWorkspacesT stores the order of the workspaces: ids lists every
// workspace, first to last.
func (d *DB) ReorderWorkspacesT(ctx context.Context, ids []int64) error {
	current, err := d.ListWorkspaces(ctx)
	if err != nil {
		return err
	}
	if len(ids) != len(current) {
		return fmt.Errorf("order must list every workspace")
	}
	known := make(map[int64]bool, len(current))
	for _, w := range current {
		known[w.ID] = true
	}
	for _, id := range ids {
		if !known[id] {
			return fmt.Errorf("Workspace %d does not exist or is listed twice", id)
		}
		delete(known, id)
	}
	tx, err := d.SQL.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := d.WithTx(tx)
	for i, id := range ids {
		if err := q.SetWorkspaceSortOrder(ctx, SetWorkspaceSortOrderParams{SortOrder: int64(i), ID: id}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SetWorkspacePinnedT pins or unpins a workspace and returns it.
func (d *DB) SetWorkspacePinnedT(ctx context.Context, id int64, pinned bool) (types.Workspace, error) {
	pinnedInt := int64(0)
	if pinned {
		pinnedInt = 1
	}
	n, err := d.SetWorkspacePinned(ctx, SetWorkspacePinnedParams{Pinned: pinnedInt, ID: id})
	if err != nil {
		return types.Workspace{}, err
	}
	if n == 0 {
		return types.Workspace{}, fmt.Errorf("Workspace not found")
	}
	return d.GetWorkspaceT(ctx, id)
}

// WorkspaceLogoName returns the file name of the workspace's stored logo, or
// nil when it has none.
func (d *DB) WorkspaceLogoName(ctx context.Context, id int64) (*string, error) {
	row, err := d.GetWorkspace(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("Workspace not found")
		}
		return nil, err
	}
	if row.Icon == nil || !IsLogoName(*row.Icon) {
		return nil, nil
	}
	return row.Icon, nil
}
