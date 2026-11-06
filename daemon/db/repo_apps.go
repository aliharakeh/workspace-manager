package db

import (
	"context"
	"database/sql"
	"fmt"

	"workspace-manager/shared/api/types"
)

func appFromRow(id, workspaceID int64, name, projectPath string, activeID *int64, createdAt, updatedAt string, activeName *string) types.App {
	return types.App{
		ID: id, WorkspaceID: workspaceID, Name: name, ProjectPath: projectPath,
		ActiveConfigSetID: activeID, ActiveConfigSetName: activeName,
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
}

func appFromGet(row GetAppRow) types.App {
	app := appFromRow(row.ID, row.WorkspaceID, row.Name, row.ProjectPath, row.ActiveConfigSetID, row.CreatedAt, row.UpdatedAt, row.ActiveConfigSetName)
	app.ExcludeFromStartAll = row.ExcludeFromStartAll != 0
	return app
}

func appFromList(row ListAppsByWorkspaceRow) types.App {
	app := appFromRow(row.ID, row.WorkspaceID, row.Name, row.ProjectPath, row.ActiveConfigSetID, row.CreatedAt, row.UpdatedAt, row.ActiveConfigSetName)
	app.ExcludeFromStartAll = row.ExcludeFromStartAll != 0
	return app
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

// ReorderAppsT stores the order of a workspace's apps: ids lists every app in
// the workspace, first to last.
func (d *DB) ReorderAppsT(ctx context.Context, workspaceID int64, ids []int64) error {
	current, err := d.ListAppsByWorkspace(ctx, workspaceID)
	if err != nil {
		return err
	}
	if len(ids) != len(current) {
		return fmt.Errorf("order must list every app in the workspace")
	}
	known := make(map[int64]bool, len(current))
	for _, a := range current {
		known[a.ID] = true
	}
	for _, id := range ids {
		if !known[id] {
			return fmt.Errorf("App %d is not in this workspace", id)
		}
		delete(known, id)
	}
	if len(known) > 0 {
		return fmt.Errorf("order lists an app twice")
	}
	tx, err := d.SQL.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := d.WithTx(tx)
	for i, id := range ids {
		if err := q.SetAppSortOrder(ctx, SetAppSortOrderParams{SortOrder: int64(i), ID: id, WorkspaceID: workspaceID}); err != nil {
			return err
		}
	}
	return tx.Commit()
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
	for _, kind := range []string{types.KindDev, types.KindProd, types.KindBuild, types.KindSetup} {
		if _, err := q.CreateRunConfig(ctx, CreateRunConfigParams{ConfigSetID: set.ID, Kind: kind, Mode: DefaultMode(kind)}); err != nil {
			return types.App{}, err
		}
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

// SetAppExcludeFromStartAllT stores whether the workspace's "Start all" skips the app.
func (d *DB) SetAppExcludeFromStartAllT(ctx context.Context, id int64, exclude bool) error {
	var v int64
	if exclude {
		v = 1
	}
	return d.SetAppExcludeFromStartAll(ctx, SetAppExcludeFromStartAllParams{ExcludeFromStartAll: v, ID: id})
}
