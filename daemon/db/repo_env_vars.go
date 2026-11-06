package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"workspace-manager/shared/api/types"
)

func EnvVarFrom(row EnvVar) types.EnvVar {
	return types.EnvVar{
		ID: row.ID, ConfigSetID: row.ConfigSetID, Key: row.Key, Value: row.Value,
		Private: row.Private != 0,
		Options: decodeEnvOptions(row.Options), Selected: row.Selected,
		SharedVarID: row.SharedVarID,
		CreatedAt:   row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func decodeEnvOptions(s string) []types.EnvVarOption {
	var out []types.EnvVarOption
	if s != "" {
		_ = json.Unmarshal([]byte(s), &out)
	}
	if out == nil {
		out = []types.EnvVarOption{}
	}
	return out
}

// NormalizeEnvOptions trims labels and rejects empty or repeated ones.
func NormalizeEnvOptions(opts []types.EnvVarOption) ([]types.EnvVarOption, error) {
	out := make([]types.EnvVarOption, 0, len(opts))
	seen := map[string]bool{}
	for _, o := range opts {
		label := strings.TrimSpace(o.Label)
		if label == "" {
			return nil, fmt.Errorf("option label is required")
		}
		if seen[label] {
			return nil, fmt.Errorf("option label %q is repeated", label)
		}
		seen[label] = true
		out = append(out, types.EnvVarOption{Label: label, Value: o.Value})
	}
	return out, nil
}

// EnvStore returns the value, options and selected columns to store. With
// options, the selected label falls back to the first option and the value
// mirrors the selected option; without, the plain value is kept.
func EnvStore(value string, opts []types.EnvVarOption, selected string) (string, string, string) {
	if len(opts) == 0 {
		return value, "[]", ""
	}
	idx := 0
	for i, o := range opts {
		if o.Label == selected {
			idx = i
			break
		}
	}
	data, _ := json.Marshal(opts)
	return opts[idx].Value, string(data), opts[idx].Label
}

// EnvStoreValue is EnvStore after editing the value of v: on a variable with
// options the edit lands on the selected option.
func EnvStoreValue(v types.EnvVar, value string) (string, string, string) {
	opts := append([]types.EnvVarOption(nil), v.Options...)
	for i := range opts {
		if opts[i].Label == v.Selected || v.Selected == "" && i == 0 {
			opts[i].Value = value
			break
		}
	}
	return EnvStore(value, opts, v.Selected)
}

func (d *DB) ListEnvVarsT(ctx context.Context, configSetID int64) ([]types.EnvVar, error) {
	rows, err := d.ListEnvVarsByConfigSet(ctx, configSetID)
	if err != nil {
		return nil, err
	}
	out := make([]types.EnvVar, 0, len(rows))
	for _, r := range rows {
		out = append(out, EnvVarFrom(r))
	}
	return out, nil
}

func (d *DB) EnvToRecord(ctx context.Context, configSetID int64) (map[string]string, error) {
	rows, err := d.ListEnvVarsByConfigSet(ctx, configSetID)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, r := range rows {
		out[r.Key] = r.Value
	}
	return out, nil
}

func (d *DB) UpsertEnvVarByKey(ctx context.Context, configSetID int64, key, value string, private bool) (types.EnvVar, error) {
	existing, err := d.GetEnvVarByKey(ctx, GetEnvVarByKeyParams{ConfigSetID: configSetID, Key: key})
	if err == nil {
		if existing.SharedVarID != nil {
			return EnvVarFrom(existing), nil // shared rows belong to the workspace
		}
		val, opts, sel := EnvStoreValue(EnvVarFrom(existing), value)
		row, err := d.UpdateEnvVar(ctx, UpdateEnvVarParams{
			Key: existing.Key, Value: val, Private: BoolInt(private), Options: opts, Selected: sel, ID: existing.ID,
		})
		if err != nil {
			return types.EnvVar{}, err
		}
		return EnvVarFrom(row), nil
	}
	if err != sql.ErrNoRows {
		return types.EnvVar{}, err
	}
	row, err := d.CreateEnvVar(ctx, CreateEnvVarParams{
		ConfigSetID: configSetID, Key: key, Value: value, Private: BoolInt(private), Options: "[]",
	})
	if err != nil {
		return types.EnvVar{}, err
	}
	return EnvVarFrom(row), nil
}
