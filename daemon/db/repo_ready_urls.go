package db

import (
	"context"
	"fmt"
	"strings"

	"workspace-manager/shared/api/types"
)

func readyURLFrom(row ReadyUrlPattern) types.ReadyUrlPattern {
	return types.ReadyUrlPattern{
		ID: row.ID, Key: row.Key, Label: row.Label, Pattern: row.Pattern, Flags: row.Flags,
		SortOrder: row.SortOrder, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
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
