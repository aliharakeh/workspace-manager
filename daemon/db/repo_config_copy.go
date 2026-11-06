package db

import (
	"context"
	"fmt"
	"strconv"

	"workspace-manager/shared/api/types"
)

func isPartEnabled(v any) bool {
	if v == nil {
		return true
	}
	switch t := v.(type) {
	case bool:
		return t
	case []any:
		return len(t) > 0
	case []string:
		return len(t) > 0
	case []int64:
		return len(t) > 0
	case []float64:
		return len(t) > 0
	default:
		return false
	}
}

func HasAnyPart(parts *types.CopyParts) bool {
	if parts == nil {
		return true
	}
	return isPartEnabled(parts.Env) || isPartEnabled(parts.Templates) || isPartEnabled(parts.Dev) || isPartEnabled(parts.Prod) || isPartEnabled(parts.Build) || isPartEnabled(parts.Setup)
}

func stringList(v any) (all bool, items []string, skip bool) {
	if v == nil {
		return true, nil, false
	}
	switch t := v.(type) {
	case bool:
		if !t {
			return false, nil, true
		}
		return true, nil, false
	case []any:
		for _, x := range t {
			items = append(items, fmt.Sprint(x))
		}
		return false, items, false
	case []string:
		return false, t, false
	default:
		return true, nil, false
	}
}

func intList(v any) (all bool, ids []int64, skip bool) {
	if v == nil {
		return true, nil, false
	}
	switch t := v.(type) {
	case bool:
		if !t {
			return false, nil, true
		}
		return true, nil, false
	case []any:
		for _, x := range t {
			switch n := x.(type) {
			case float64:
				ids = append(ids, int64(n))
			case int:
				ids = append(ids, int64(n))
			case int64:
				ids = append(ids, n)
			case string:
				parsed, err := strconv.ParseInt(n, 10, 64)
				if err == nil {
					ids = append(ids, parsed)
				}
			}
		}
		return false, ids, false
	case []int64:
		return false, t, false
	default:
		return true, nil, false
	}
}

func containsStr(items []string, s string) bool {
	for _, x := range items {
		if x == s {
			return true
		}
	}
	return false
}

func containsInt(items []int64, n int64) bool {
	for _, x := range items {
		if x == n {
			return true
		}
	}
	return false
}

func (d *DB) CopyFrom(ctx context.Context, sourceID, targetID int64, parts *types.CopyParts) error {
	source, err := d.GetConfigSetT(ctx, sourceID)
	if err != nil {
		return err
	}
	target, err := d.GetConfigSetT(ctx, targetID)
	if err != nil {
		return err
	}
	if source.AppID != target.AppID {
		return fmt.Errorf("Config sets must belong to the same app")
	}
	if sourceID == targetID {
		return nil
	}
	envAll, envItems, envSkip := stringList(nil)
	tplAll, tplItems, tplSkip := stringList(nil)
	// What to copy of the command lists, by kind (nil: all).
	cmdParts := map[string]any{}
	if parts != nil {
		envAll, envItems, envSkip = stringList(parts.Env)
		tplAll, tplItems, tplSkip = stringList(parts.Templates)
		cmdParts[types.KindDev], cmdParts[types.KindProd], cmdParts[types.KindBuild], cmdParts[types.KindSetup] = parts.Dev, parts.Prod, parts.Build, parts.Setup
	}
	cmdKinds := []string{types.KindDev, types.KindProd, types.KindBuild, types.KindSetup}
	cmdSkip := true
	for _, kind := range cmdKinds {
		if _, _, skip := intList(cmdParts[kind]); !skip {
			cmdSkip = false
		}
	}
	if envSkip && tplSkip && cmdSkip {
		return fmt.Errorf("Select at least one part to copy")
	}

	if !envSkip {
		sourceEnv, err := d.ListEnvVarsT(ctx, sourceID)
		if err != nil {
			return err
		}
		if err := d.DeleteOwnEnvVarsByConfigSet(ctx, targetID); err != nil {
			return err
		}
		// Shared rows of the target stay, so a copied key that one already uses is skipped.
		shared := map[string]bool{}
		if kept, err := d.ListEnvVarsT(ctx, targetID); err == nil {
			for _, k := range kept {
				if k.SharedVarID != nil {
					shared[k.Key] = true
				}
			}
		}
		for _, v := range sourceEnv {
			if envAll || containsStr(envItems, v.Key) {
				if shared[v.Key] {
					continue
				}
				val, opts, sel := EnvStore(v.Value, v.Options, v.Selected)
				if _, err := d.CreateEnvVar(ctx, CreateEnvVarParams{
					ConfigSetID: targetID, Key: v.Key, Value: val, Private: BoolInt(v.Private), Options: opts, Selected: sel,
				}); err != nil {
					return err
				}
			}
		}
	}
	if !envSkip {
		p, err := d.EnvProvider(ctx, false, sourceID)
		if err != nil {
			return err
		}
		if !envAll {
			p = types.EnvProvider{Timeout: 30}
		}
		if err = d.SaveEnvProvider(ctx, false, targetID, p); err != nil {
			return err
		}
	}
	if !tplSkip {
		sourceTpl, err := d.ListTemplatesT(ctx, sourceID)
		if err != nil {
			return err
		}
		if err := d.DeleteTemplatesByConfigSet(ctx, targetID); err != nil {
			return err
		}
		for _, t := range sourceTpl {
			if tplAll || containsStr(tplItems, t.FilePath) {
				if _, err := d.CreateTemplate(ctx, CreateTemplateParams{ConfigSetID: targetID, FilePath: t.FilePath, Content: t.Content, Private: BoolInt(t.Private)}); err != nil {
					return err
				}
			}
		}
	}
	for _, kind := range cmdKinds {
		all, ids, skip := intList(cmdParts[kind])
		if skip {
			continue
		}
		source, err := d.GetRunConfigByConfigSetT(ctx, sourceID, kind)
		if err != nil {
			return err
		}
		var cmds []types.RunCommandInput
		mode := DefaultMode(kind)
		if source != nil {
			mode = source.Mode
			for _, c := range source.Commands {
				if all || containsInt(ids, c.ID) {
					cmds = append(cmds, types.RunCommandInput{Label: c.Label, Command: c.Command})
				}
			}
		}
		if _, err := d.UpsertRunConfig(ctx, targetID, kind, &mode, cmds); err != nil {
			return err
		}
	}
	return nil
}
