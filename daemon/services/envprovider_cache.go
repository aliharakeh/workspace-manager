package services

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"workspace-manager/daemon/db"
	"workspace-manager/shared/api/types"
)

type providerID struct {
	shared bool
	id     int64
}
type providerResult struct {
	config types.EnvProvider
	values map[string]string
}

// EnvProviders retains private results until daemon shutdown. App starts only read them.
type EnvProviders struct {
	db      *db.DB
	mu      sync.Mutex
	runMu   sync.Mutex
	results map[providerID]providerResult
}

func NewEnvProviders(d *db.DB) *EnvProviders {
	return &EnvProviders{db: d, results: map[providerID]providerResult{}}
}
func (c *EnvProviders) read(_ context.Context, shared bool, id int64, p types.EnvProvider, _ string, _ map[string]string) (map[string]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	result, ok := c.results[providerID{shared, id}]
	if !ok || !reflect.DeepEqual(result.config, p) {
		return nil, fmt.Errorf("Run the dynamic environment script manually before starting this app")
	}
	out := make(map[string]string, len(result.values))
	for k, v := range result.values {
		out[k] = v
	}
	return out, nil
}
func (c *EnvProviders) Resolve(ctx context.Context, setID int64, dir string) (map[string]string, error) {
	return resolveRunEnv(ctx, c.db, setID, dir, c.read)
}

// Run refreshes only the requested script. App scripts consume previously refreshed shared values.
func (c *EnvProviders) Run(ctx context.Context, shared bool, id int64) error {
	if !c.runMu.TryLock() {
		return fmt.Errorf("A dynamic environment script is already running")
	}
	defer c.runMu.Unlock()
	p, err := c.db.EnvProvider(ctx, shared, id)
	if err != nil {
		return err
	}
	if p.Command == "" {
		return fmt.Errorf("No dynamic environment script configured")
	}
	var values map[string]string
	if shared {
		rows, e := c.db.ListWorkspaceEnvVars(ctx, id)
		if e != nil {
			return e
		}
		inputs := map[string]string{}
		for _, v := range rows {
			inputs[v.Key] = v.Value
		}
		values, err = runEnvProvider(ctx, p, p.Directory, inputs)
	} else {
		set, e := c.db.GetConfigSet(ctx, id)
		if e != nil {
			return e
		}
		app, e := c.db.GetAppT(ctx, set.AppID)
		if e != nil {
			return e
		}
		_, err = resolveRunEnv(ctx, c.db, id, app.ProjectPath, func(ctx context.Context, s bool, key int64, config types.EnvProvider, dir string, env map[string]string) (map[string]string, error) {
			if s {
				return c.read(ctx, s, key, config, dir, env)
			}
			if !reflect.DeepEqual(config, p) {
				return nil, fmt.Errorf("Script configuration changed; run it again")
			}
			var e error
			values, e = runEnvProvider(ctx, config, dir, env)
			return values, e
		})
	}
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.results[providerID{shared, id}] = providerResult{config: p, values: values}
	c.mu.Unlock()
	return nil
}
