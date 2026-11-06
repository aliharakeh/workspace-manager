package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"time"
	"workspace-manager/daemon/db"
	"workspace-manager/shared/api/types"
	"workspace-manager/shared/native"
)

type providerOutput struct{ bytes.Buffer }

func (b *providerOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 1024*1024 {
		return 0, fmt.Errorf("provider output exceeds limit")
	}
	return b.Buffer.Write(p)
}
func runEnvProvider(ctx context.Context, p types.EnvProvider, dir string, env map[string]string) (map[string]string, error) {
	if p.Command == "" {
		return map[string]string{}, nil
	}
	timeout := p.Timeout
	if timeout == 0 {
		timeout = 30
	}
	if timeout < 1 || timeout > 600 {
		return nil, fmt.Errorf("invalid script timeout")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	shell, args := "sh", []string{"-c", p.Command}
	if runtime.GOOS == "windows" {
		shell = "cmd.exe"
		args = []string{"/d", "/s", "/c", p.Command}
	}
	cmd := exec.CommandContext(ctx, shell, args...)
	hideProviderWindow(cmd)
	cmd.Dir = dir
	cmd.Env = native.MergeTerminalEnv(env)
	cmd.WaitDelay = time.Second
	cmd.Cancel = func() error { return native.KillPid(cmd.Process.Pid) }
	var out providerOutput
	cmd.Stdout = &out
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("script timed out after %d seconds", timeout)
		}
		if ctx.Err() != nil {
			return nil, fmt.Errorf("script stopped")
		}
		return nil, fmt.Errorf("script failed or exceeded output limit")
	}
	dec := json.NewDecoder(&out.Buffer)
	var object map[string]any
	if err := dec.Decode(&object); err != nil || object == nil {
		return nil, fmt.Errorf("script must return a JSON object of strings")
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return nil, fmt.Errorf("unexpected content after script JSON")
	}
	result := make(map[string]string, len(object))
	for key, raw := range object {
		value, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("script must return a JSON object of strings")
		}
		result[key] = value
	}
	return result, nil
}

type providerLookup func(context.Context, bool, int64, types.EnvProvider, string, map[string]string) (map[string]string, error)

func resolveRunEnv(ctx context.Context, d *db.DB, setID int64, dir string, lookup providerLookup) (map[string]string, error) {
	set, err := d.GetConfigSet(ctx, setID)
	if err != nil {
		return nil, err
	}
	env, err := d.EnvToRecord(ctx, setID)
	if err != nil {
		return nil, err
	}
	// Shared output is a runtime overlay for apps linked to this shared set.
	if set.SharedSetID != nil {
		p, e := d.EnvProvider(ctx, true, *set.SharedSetID)
		if e != nil {
			return nil, e
		}
		if p.Command != "" {
			values, e := lookup(ctx, true, *set.SharedSetID, p, p.Directory, nil)
			if e != nil {
				return nil, fmt.Errorf("Shared environment: %w", e)
			}
			for key, value := range values {
				env[key] = value
			}
		}
	}
	p, err := d.EnvProvider(ctx, false, setID)
	if err != nil {
		return nil, err
	}
	if p.Command != "" {
		values, e := lookup(ctx, false, setID, p, dir, env)
		if e != nil {
			return nil, fmt.Errorf("App environment: %w", e)
		}
		for key, value := range values {
			env[key] = value
		}
	}
	return env, nil
}
