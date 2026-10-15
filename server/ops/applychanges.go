package ops

import (
	"fmt"
	"strings"

	"workspace-manager/api/types"
	"workspace-manager/server/db"
)

// ConfigSetsApplyChanges writes a reviewed set of staged edits to a config set
// in one call, so they cannot half-land while a review UI is still running:
// env upserts and deletes, template content, and the dev, prod and build
// command lists. Values excluded from AI are never touched.
func (o *Ops) ConfigSetsApplyChanges(appID, configSetID int64, patch types.AppAIPatch) error {
	ctx := o.Ctx()
	set, err := o.DB.GetConfigSetT(ctx, configSetID)
	if err != nil {
		return err
	}
	if set.AppID != appID {
		return fmt.Errorf("Config set not found")
	}
	vars, err := o.DB.ListEnvVarsT(ctx, configSetID)
	if err != nil {
		return err
	}
	byKey := map[string]types.EnvVar{}
	for _, v := range vars {
		byKey[v.Key] = v
	}

	var envChanged, templatesChanged, runChanged bool
	if patch.Env != nil {
		for _, key := range patch.Env.Delete {
			v, ok := byKey[key]
			if !ok || !v.IncludeInAI {
				continue
			}
			if _, err := o.DB.DeleteEnvVar(ctx, v.ID); err != nil {
				return err
			}
			delete(byKey, key)
			envChanged = true
		}
		for _, up := range patch.Env.Upsert {
			v, ok := byKey[up.Key]
			if ok && !v.IncludeInAI {
				continue
			}
			if ok {
				if v.Value == up.Value {
					continue
				}
				if _, err := o.DB.UpdateEnvVar(ctx, db.UpdateEnvVarParams{
					Key: v.Key, Value: up.Value, IncludeInAi: db.BoolInt(v.IncludeInAI), ID: v.ID,
				}); err != nil {
					return err
				}
				envChanged = true
				continue
			}
			if _, err := o.DB.CreateEnvVar(ctx, db.CreateEnvVarParams{
				ConfigSetID: configSetID, Key: up.Key, Value: up.Value, IncludeInAi: db.BoolInt(true),
			}); err != nil {
				return err
			}
			envChanged = true
		}
	}

	tpls, err := o.DB.ListTemplatesT(ctx, configSetID)
	if err != nil {
		return err
	}
	byPath := map[string]types.Template{}
	for _, t := range tpls {
		byPath[t.FilePath] = t
	}
	for _, tp := range patch.Templates {
		existing, ok := byPath[strings.ReplaceAll(tp.FilePath, "\\", "/")]
		if !ok || !existing.IncludeInAI || existing.Content == tp.Content {
			continue
		}
		if _, err := o.DB.UpdateTemplate(ctx, db.UpdateTemplateParams{
			FilePath: existing.FilePath, Content: tp.Content, IncludeInAi: db.BoolInt(true), ID: existing.ID,
		}); err != nil {
			return err
		}
		templatesChanged = true
	}

	for _, list := range []struct {
		kind  string
		patch *types.AppAIRunPatch
	}{
		{types.KindDev, patch.Dev},
		{types.KindProd, patch.Prod},
		{types.KindBuild, patch.Build},
	} {
		if list.patch == nil {
			continue
		}
		if err := o.saveCommandList(configSetID, list.kind, list.patch); err != nil {
			return err
		}
		runChanged = true
	}

	if envChanged {
		o.changed("envVar", appID)
	}
	if templatesChanged {
		o.changed("template", appID)
	}
	if runChanged {
		o.changed("runConfig", appID)
	}
	return nil
}

// saveCommandList writes a staged command list, keeping the current commands
// when the patch only changed the mode.
func (o *Ops) saveCommandList(configSetID int64, kind string, list *types.AppAIRunPatch) error {
	ctx := o.Ctx()
	cmds := []types.RunCommandInput{}
	if list.Commands != nil {
		for _, c := range list.Commands {
			cmds = append(cmds, types.RunCommandInput{Label: c.Label, Command: c.Command})
		}
	} else {
		cfg, err := o.DB.GetOrCreateRunConfig(ctx, configSetID, kind)
		if err != nil {
			return err
		}
		for _, c := range cfg.Commands {
			cmds = append(cmds, types.RunCommandInput{Label: c.Label, Command: c.Command})
		}
	}
	_, err := o.DB.UpsertRunConfig(ctx, configSetID, kind, list.Mode, cmds)
	return err
}
