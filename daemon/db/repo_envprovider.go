package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"workspace-manager/shared/api/types"
)

func (d *DB) EnvProvider(ctx context.Context, shared bool, id int64) (types.EnvProvider, error) {
	var raw string
	var err error
	if shared {
		raw, err = d.GetSharedEnvProvider(ctx, &id)
	} else {
		raw, err = d.GetAppEnvProvider(ctx, &id)
	}
	p := types.EnvProvider{Timeout: 30}
	if errors.Is(err, sql.ErrNoRows) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	err = json.Unmarshal([]byte(raw), &p)
	return p, err
}
func (d *DB) SaveEnvProvider(ctx context.Context, shared bool, id int64, p types.EnvProvider) error {
	raw, err := json.Marshal(p)
	if err != nil {
		return err
	}
	if shared {
		return d.SaveSharedEnvProvider(ctx, SaveSharedEnvProviderParams{SharedSetID: &id, Config: string(raw)})
	}
	return d.SaveAppEnvProvider(ctx, SaveAppEnvProviderParams{AppSetID: &id, Config: string(raw)})
}
