package db

import (
	"context"
)

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
