package handlers

import (
	"context"
	"strings"

	"workspace-manager/daemon/ops"
	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"
	"workspace-manager/shared/rpc"
)

// kindInput is a handler that needs both its parameters and the client kind.
func kindTyped[In any, Out any](fn func(kind string, in In) (Out, error)) rpc.Handler {
	return func(ctx context.Context, raw Params) (any, error) {
		in, err := decode[In](raw)
		if err != nil {
			return nil, err
		}
		return fn(clientKind(ctx), in)
	}
}

func (r *Registry) configSets() {
	o := r.Ops

	r.handle(api.MethodConfigSetsList, typed(func(in struct {
		AppID int64 `json:"appID"`
	}) ([]types.ConfigSet, error) {
		return o.ConfigSetsList(in.AppID)
	}))
	r.handle(api.MethodConfigSetsDetail, kindTyped(func(kind string, in struct {
		ID int64 `json:"id"`
	}) (types.ConfigSetDetail, error) {
		d, err := o.ConfigSetsGetDetail(in.ID)
		if err != nil {
			return types.ConfigSetDetail{}, err
		}
		return maskDetail(kind, d), nil
	}))
	r.handle(api.MethodConfigSetsCreate, typed(func(in struct {
		AppID int64                      `json:"appID"`
		Body  types.ConfigSetCreateInput `json:"body"`
	}) (types.ConfigSet, error) {
		return o.ConfigSetsCreate(in.AppID, in.Body)
	}))
	r.handle(api.MethodConfigSetsUpdate, typed(func(in struct {
		ID   int64                      `json:"id"`
		Body types.ConfigSetUpdateInput `json:"body"`
	}) (types.ConfigSet, error) {
		return o.ConfigSetsUpdate(in.ID, in.Body)
	}))
	r.handle(api.MethodConfigSetsDelete, typed(func(in struct {
		ID int64 `json:"id"`
	}) (types.Ok, error) {
		return o.ConfigSetsDelete(in.ID)
	}))
	r.handle(api.MethodConfigSetsActivate, typed(func(in struct {
		ID int64 `json:"id"`
	}) (types.ConfigSetActivateResult, error) {
		return o.ConfigSetsActivate(in.ID)
	}))
	r.handle(api.MethodConfigSetsCopyFrom, typed(func(in struct {
		ID       int64            `json:"id"`
		SourceID int64            `json:"sourceID"`
		Parts    *types.CopyParts `json:"parts"`
	}) (types.ConfigSet, error) {
		return o.ConfigSetsCopyFrom(in.ID, in.SourceID, in.Parts)
	}))
	r.handle(api.MethodConfigSetsApplyChanges, typed(func(in struct {
		AppID       int64            `json:"appID"`
		ConfigSetID int64            `json:"configSetID"`
		Patch       types.AppAIPatch `json:"patch"`
	}) (types.Ok, error) {
		if err := o.ConfigSetsApplyChanges(in.AppID, in.ConfigSetID, in.Patch); err != nil {
			return types.Ok{}, err
		}
		return types.Ok{Ok: true}, nil
	}))
}

func (r *Registry) envVars() {
	o := r.Ops

	r.handle(api.MethodEnvVarsList, kindTyped(func(kind string, in struct {
		AppID       int64 `json:"appID"`
		ConfigSetID int64 `json:"configSetID"`
	}) ([]types.EnvVar, error) {
		vars, err := o.EnvVarsList(in.AppID, in.ConfigSetID)
		if err != nil {
			return nil, err
		}
		return hideEnvVars(kind, vars), nil
	}))
	r.handle(api.MethodEnvVarsCreate, kindTyped(func(kind string, in struct {
		AppID       int64                   `json:"appID"`
		ConfigSetID int64                   `json:"configSetID"`
		Body        types.EnvVarCreateInput `json:"body"`
	}) (types.EnvVar, error) {
		if err := requireVisibleVarKey(kind, o, in.AppID, in.ConfigSetID, in.Body.Key); err != nil {
			return types.EnvVar{}, err
		}
		v, err := o.EnvVarsCreate(in.AppID, in.ConfigSetID, in.Body)
		if err != nil {
			return types.EnvVar{}, err
		}
		return hideEnvVar(kind, v), nil
	}))
	r.handle(api.MethodEnvVarsUpdate, kindTyped(func(kind string, in struct {
		ID   int64                   `json:"id"`
		Body types.EnvVarUpdateInput `json:"body"`
	}) (types.EnvVar, error) {
		if kind == ClientAgent {
			existing, err := o.GetEnvVarByID(in.ID)
			if err != nil {
				return types.EnvVar{}, err
			}
			if !existing.IncludeInAI {
				// Hidden variables allow value replacement, never renaming, unhiding
				// or changing their options and choice.
				if in.Body.Value == nil || in.Body.Options != nil || in.Body.Selected != nil ||
					(in.Body.Key != nil && strings.TrimSpace(*in.Body.Key) != existing.Key) ||
					(in.Body.IncludeInAI != nil && *in.Body.IncludeInAI) {
					return types.EnvVar{}, errHidden()
				}
				in.Body.IncludeInAI = &existing.IncludeInAI
			}
		}
		v, err := o.EnvVarsUpdate(in.ID, in.Body)
		if err != nil {
			return types.EnvVar{}, err
		}
		return hideEnvVar(kind, v), nil
	}))
	r.handle(api.MethodEnvVarsDelete, kindTyped(func(kind string, in struct {
		ID int64 `json:"id"`
	}) (types.Ok, error) {
		// Deleting reveals nothing, so agents may delete hidden variables.
		return o.EnvVarsDelete(in.ID)
	}))
	r.handle(api.MethodEnvVarsImport, kindTyped(func(kind string, in struct {
		AppID       int64  `json:"appID"`
		ConfigSetID int64  `json:"configSetID"`
		Path        string `json:"path"`
	}) (types.ImportEnvResult, error) {
		res, err := o.EnvVarsImport(in.AppID, in.ConfigSetID, in.Path)
		if err != nil {
			return types.ImportEnvResult{}, err
		}
		res.Vars = hideEnvVars(kind, res.Vars)
		return res, nil
	}))
}

func (r *Registry) templates() {
	o := r.Ops

	r.handle(api.MethodTemplatesList, kindTyped(func(kind string, in struct {
		AppID       int64 `json:"appID"`
		ConfigSetID int64 `json:"configSetID"`
	}) ([]types.Template, error) {
		tpls, err := o.TemplatesList(in.AppID, in.ConfigSetID)
		if err != nil {
			return nil, err
		}
		return hideTemplates(kind, tpls), nil
	}))
	r.handle(api.MethodTemplatesCreate, kindTyped(func(kind string, in struct {
		AppID       int64                     `json:"appID"`
		ConfigSetID int64                     `json:"configSetID"`
		Body        types.TemplateCreateInput `json:"body"`
	}) (types.Template, error) {
		t, err := o.TemplatesCreate(in.AppID, in.ConfigSetID, in.Body)
		if err != nil {
			return types.Template{}, err
		}
		return hideTemplate(kind, t), nil
	}))
	r.handle(api.MethodTemplatesUpdate, kindTyped(func(kind string, in struct {
		ID   int64                     `json:"id"`
		Body types.TemplateUpdateInput `json:"body"`
	}) (types.Template, error) {
		if kind == ClientAgent {
			existing, err := o.GetTemplateByID(in.ID)
			if err != nil {
				return types.Template{}, err
			}
			if !existing.IncludeInAI {
				// Hidden templates allow content replacement, never renaming or unhiding.
				if in.Body.Content == nil ||
					(in.Body.FilePath != nil && strings.TrimSpace(*in.Body.FilePath) != existing.FilePath) ||
					(in.Body.IncludeInAI != nil && *in.Body.IncludeInAI) {
					return types.Template{}, errHidden()
				}
				in.Body.IncludeInAI = &existing.IncludeInAI
			}
		}
		t, err := o.TemplatesUpdate(in.ID, in.Body)
		if err != nil {
			return types.Template{}, err
		}
		return hideTemplate(kind, t), nil
	}))
	r.handle(api.MethodTemplatesDelete, kindTyped(func(kind string, in struct {
		ID int64 `json:"id"`
	}) (types.Ok, error) {
		// Deleting reveals nothing, so agents may delete hidden templates.
		return o.TemplatesDelete(in.ID)
	}))
	r.handle(api.MethodTemplatesRender, typed(func(in types.TemplateRenderInput) (types.TemplateRenderResult, error) {
		return o.TemplatesRender(in.ConfigSetID, in.Content)
	}))
}

// maskDetail blanks AI-excluded values and contents for agent clients.
func maskDetail(kind string, d types.ConfigSetDetail) types.ConfigSetDetail {
	if kind != ClientAgent {
		return d
	}
	d.EnvVars = hideEnvVars(kind, d.EnvVars)
	d.Templates = hideTemplates(kind, d.Templates)
	return d
}

// errHidden is what an agent client sees for an AI-excluded item: the same
// "not found" a GUI user would get for a missing one, so an agent cannot
// tell what exists.
func errHidden() error {
	return api.Errf(api.KindNotFound, "not found")
}

func requireVisibleVarKey(kind string, o *ops.Ops, appID, setID int64, key string) error {
	if kind != ClientAgent {
		return nil
	}
	vars, err := o.EnvVarsList(appID, setID)
	if err != nil {
		return err
	}
	for _, v := range vars {
		if v.Key == key && !v.IncludeInAI {
			return errHidden()
		}
	}
	return nil
}

func hideEnvVars(kind string, vars []types.EnvVar) []types.EnvVar {
	if kind != ClientAgent {
		return vars
	}
	out := make([]types.EnvVar, len(vars))
	copy(out, vars)
	for i := range out {
		out[i] = hideEnvVar(kind, out[i])
	}
	return out
}

// hideEnvVar blanks the value of a hidden variable for agents, and the value of
// each of its options (the labels stay so the choice can still be described).
func hideEnvVar(kind string, v types.EnvVar) types.EnvVar {
	if kind == ClientAgent && !v.IncludeInAI {
		v.Value = ""
		opts := make([]types.EnvVarOption, len(v.Options))
		for i, o := range v.Options {
			opts[i] = types.EnvVarOption{Label: o.Label}
		}
		v.Options = opts
	}
	return v
}

func hideTemplates(kind string, tpls []types.Template) []types.Template {
	if kind != ClientAgent {
		return tpls
	}
	out := make([]types.Template, len(tpls))
	copy(out, tpls)
	for i := range out {
		if !out[i].IncludeInAI {
			out[i].Content = ""
		}
	}
	return out
}

func hideTemplate(kind string, t types.Template) types.Template {
	if kind == ClientAgent && !t.IncludeInAI {
		t.Content = ""
	}
	return t
}
