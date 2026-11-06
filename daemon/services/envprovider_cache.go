package services

import (
	"context"
	"fmt"
	"path/filepath"
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
	cancel  context.CancelFunc // stops the script that is running, if any
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

// Values returns a copy of the last successful result of a script.
func (c *EnvProviders) Values(shared bool, id int64) map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := map[string]string{}
	for k, v := range c.results[providerID{shared, id}].values {
		out[k] = v
	}
	return out
}
func (c *EnvProviders) Resolve(ctx context.Context, setID int64, dir string) (map[string]string, error) {
	return resolveRunEnv(ctx, c.db, setID, dir, c.read)
}

// Stop ends the script that is running (and its child processes). It reports
// whether there was one. The last successful result stays as it was.
func (c *EnvProviders) Stop() bool {
	c.mu.Lock()
	cancel := c.cancel
	c.mu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

// sharedDir is where a shared script runs. An absolute directory is used as
// is. A relative or empty one is resolved against the project folder of the
// first app (in the workspace's order) linked to the shared set, or else of the
// workspace's first app.
func (c *EnvProviders) sharedDir(ctx context.Context, setID int64, dir string) (string, error) {
	if filepath.IsAbs(dir) {
		return dir, nil
	}
	set, err := c.db.GetWorkspaceConfigSet(ctx, setID)
	if err != nil {
		return "", err
	}
	apps, err := c.db.ListAppsByWorkspace(ctx, set.WorkspaceID)
	if err != nil {
		return "", err
	}
	linked, err := c.db.ListAppsLinkedToSharedSet(ctx, &setID)
	if err != nil {
		return "", err
	}
	isLinked := map[int64]bool{}
	for _, id := range linked {
		isLinked[id] = true
	}
	base := ""
	for _, a := range apps {
		if a.ProjectPath == "" {
			continue
		}
		if isLinked[a.ID] {
			base = a.ProjectPath
			break
		}
		if base == "" {
			base = a.ProjectPath
		}
	}
	if base == "" {
		return "", fmt.Errorf("Set an absolute working directory: the workspace has no app folder to resolve it from")
	}
	return filepath.Join(base, dir), nil
}

// Run refreshes only the requested script. App scripts consume previously refreshed shared values.
func (c *EnvProviders) Run(ctx context.Context, shared bool, id int64) error {
	if !c.runMu.TryLock() {
		return fmt.Errorf("A dynamic environment script is already running")
	}
	defer c.runMu.Unlock()
	ctx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.cancel = cancel
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.cancel = nil
		c.mu.Unlock()
		cancel()
	}()
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
		dir, e := c.sharedDir(ctx, id, p.Directory)
		if e != nil {
			return e
		}
		values, err = runEnvProvider(ctx, p, dir, inputs)
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
