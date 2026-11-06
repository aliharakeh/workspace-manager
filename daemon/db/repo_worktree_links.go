package db

import (
	"context"
)

// ListWorktreeLinksT returns the app's shared worktree paths in order.
func (d *DB) ListWorktreeLinksT(ctx context.Context, appID int64) ([]string, error) {
	return d.ListWorktreeLinks(ctx, appID)
}

// SetWorktreeLinksT replaces the app's shared worktree paths with paths, in order.
func (d *DB) SetWorktreeLinksT(ctx context.Context, appID int64, paths []string) error {
	tx, err := d.SQL.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	q := d.WithTx(tx)
	if err := q.DeleteWorktreeLinks(ctx, appID); err != nil {
		return err
	}
	for i, p := range paths {
		if err := q.CreateWorktreeLink(ctx, CreateWorktreeLinkParams{AppID: appID, Path: p, SortOrder: int64(i)}); err != nil {
			return err
		}
	}
	return tx.Commit()
}
