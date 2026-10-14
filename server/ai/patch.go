package ai

import (
	"context"
	"strings"

	"workspace-manager/api/types"
	"workspace-manager/server/db"
)

// PatchWriter writes the parts of an agent's staged patch that live in the
// database. Worktree changes go through Ops, which owns git, so the AI package
// stays free of it.
type PatchWriter interface {
	WriteConfig(configSetID int64, patch types.AppAIPatch) error
}

// Worktrees applies the git part of a staged patch. The daemon passes the ops
// layer, which owns git, so this package stays free of it.
type Worktrees interface {
	GitWorktreeAdd(appID int64, body types.GitWorktreeAddInput) (types.GitWorktreeAddResult, error)
	GitWorktreeRemove(appID int64, path string, force bool) (types.Ok, error)
}

// ApplyPatch writes an agent's staged patch to a config set. It mirrors what
// the GUI used to do call by call: env upserts and deletes, template content,
// the run and build command lists, and git worktree changes. Values excluded
// from AI are never touched.
func (s *Service) ApplyPatch(appID, configSetID int64, patch types.AppAIPatch) error {
	ctx := context.Background()
	vars, err := s.DB.ListEnvVarsT(ctx, configSetID)
	if err != nil {
		return err
	}
	byKey := map[string]types.EnvVar{}
	for _, v := range vars {
		byKey[v.Key] = v
	}

	for _, key := range patch.Env.Delete {
		v, ok := byKey[key]
		if !ok || !v.IncludeInAI {
			continue
		}
		if _, err := s.DB.DeleteEnvVar(ctx, v.ID); err != nil {
			return err
		}
		delete(byKey, key)
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
			if _, err := s.DB.UpdateEnvVar(ctx, db.UpdateEnvVarParams{
				Key: v.Key, Value: up.Value, IncludeInAi: db.BoolInt(v.IncludeInAI), ID: v.ID,
			}); err != nil {
				return err
			}
			continue
		}
		if _, err := s.DB.CreateEnvVar(ctx, db.CreateEnvVarParams{
			ConfigSetID: configSetID, Key: up.Key, Value: up.Value, IncludeInAi: db.BoolInt(true),
		}); err != nil {
			return err
		}
	}

	tpls, err := s.DB.ListTemplatesT(ctx, configSetID)
	if err != nil {
		return err
	}
	byPath := map[string]types.Template{}
	for _, t := range tpls {
		byPath[t.FilePath] = t
	}
	for _, tp := range patch.Templates {
		existing, ok := byPath[strings.ReplaceAll(tp.FilePath, "\\", "/")]
		if !ok || !existing.IncludeInAI {
			continue
		}
		if existing.Content == tp.Content {
			continue
		}
		if _, err := s.DB.UpdateTemplate(ctx, db.UpdateTemplateParams{
			FilePath: existing.FilePath, Content: tp.Content, IncludeInAi: db.BoolInt(true), ID: existing.ID,
		}); err != nil {
			return err
		}
	}

	if patch.Dev != nil {
		if err := s.saveCommandList(configSetID, types.KindDev, patch.Dev); err != nil {
			return err
		}
	}
	if patch.Prod != nil {
		if err := s.saveCommandList(configSetID, types.KindProd, patch.Prod); err != nil {
			return err
		}
	}
	if patch.Build != nil {
		if err := s.saveCommandList(configSetID, types.KindBuild, patch.Build); err != nil {
			return err
		}
	}
	if patch.Worktrees != nil && s.Worktrees != nil {
		for _, r := range patch.Worktrees.Remove {
			if _, err := s.Worktrees.GitWorktreeRemove(appID, r.Path, r.Force); err != nil {
				return err
			}
		}
		for _, a := range patch.Worktrees.Add {
			if _, err := s.Worktrees.GitWorktreeAdd(appID, a); err != nil {
				return err
			}
		}
	}
	return nil
}

// saveCommandList writes a staged run or build list, keeping the current
// commands when the patch only changed the mode.
func (s *Service) saveCommandList(configSetID int64, kind string, list *types.AppAIRunPatch) error {
	ctx := context.Background()
	cmds := []types.RunCommandInput{}
	if list.Commands != nil {
		for _, c := range list.Commands {
			cmds = append(cmds, types.RunCommandInput{Label: c.Label, Command: c.Command})
		}
	} else {
		cfg, err := s.DB.GetOrCreateRunConfig(ctx, configSetID, kind)
		if err != nil {
			return err
		}
		for _, c := range cfg.Commands {
			cmds = append(cmds, types.RunCommandInput{Label: c.Label, Command: c.Command})
		}
	}
	_, err := s.DB.UpsertRunConfig(ctx, configSetID, kind, list.Mode, cmds)
	return err
}
