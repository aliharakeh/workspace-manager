package handlers

import (
	"context"

	"workspace-manager/api"
	"workspace-manager/api/types"
	"workspace-manager/core/ops"
)

// kindInput is a handler that needs both its parameters and the client kind.
func kindTyped[In any, Out any](fn func(kind string, in In) (Out, error)) Handler {
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
		AppID int64 `json:"appID"`
	}) ([]types.EnvVar, error) {
		vars, err := o.EnvVarsList(in.AppID)
		if err != nil {
			return nil, err
		}
		return hideEnvVars(kind, vars), nil
	}))
	r.handle(api.MethodEnvVarsCreate, kindTyped(func(kind string, in struct {
		AppID int64                   `json:"appID"`
		Body  types.EnvVarCreateInput `json:"body"`
	}) (types.EnvVar, error) {
		if err := requireVisibleVarKey(kind, o, in.AppID, in.Body.Key); err != nil {
			return types.EnvVar{}, err
		}
		v, err := o.EnvVarsCreate(in.AppID, in.Body)
		if err != nil {
			return types.EnvVar{}, err
		}
		return hideEnvVar(kind, v), nil
	}))
	r.handle(api.MethodEnvVarsUpdate, kindTyped(func(kind string, in struct {
		ID   int64                   `json:"id"`
		Body types.EnvVarUpdateInput `json:"body"`
	}) (types.EnvVar, error) {
		if err := requireVisibleVarID(kind, o, in.ID); err != nil {
			return types.EnvVar{}, err
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
		if err := requireVisibleVarID(kind, o, in.ID); err != nil {
			return types.Ok{}, err
		}
		return o.EnvVarsDelete(in.ID)
	}))
	r.handle(api.MethodEnvVarsImport, kindTyped(func(kind string, in struct {
		AppID int64  `json:"appID"`
		Path  string `json:"path"`
	}) (types.ImportEnvResult, error) {
		res, err := o.EnvVarsImport(in.AppID, in.Path)
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
		AppID int64 `json:"appID"`
	}) ([]types.Template, error) {
		tpls, err := o.TemplatesList(in.AppID)
		if err != nil {
			return nil, err
		}
		return hideTemplates(kind, tpls), nil
	}))
	r.handle(api.MethodTemplatesCreate, kindTyped(func(kind string, in struct {
		AppID int64                     `json:"appID"`
		Body  types.TemplateCreateInput `json:"body"`
	}) (types.Template, error) {
		t, err := o.TemplatesCreate(in.AppID, in.Body)
		if err != nil {
			return types.Template{}, err
		}
		return hideTemplate(kind, t), nil
	}))
	r.handle(api.MethodTemplatesUpdate, kindTyped(func(kind string, in struct {
		ID   int64                     `json:"id"`
		Body types.TemplateUpdateInput `json:"body"`
	}) (types.Template, error) {
		if err := requireVisibleTemplateID(kind, o, in.ID); err != nil {
			return types.Template{}, err
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
		if err := requireVisibleTemplateID(kind, o, in.ID); err != nil {
			return types.Ok{}, err
		}
		return o.TemplatesDelete(in.ID)
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

func requireVisibleVarKey(kind string, o *ops.Ops, appID int64, key string) error {
	if kind != ClientAgent {
		return nil
	}
	vars, err := o.EnvVarsList(appID)
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

func requireVisibleVarID(kind string, o *ops.Ops, id int64) error {
	if kind != ClientAgent {
		return nil
	}
	v, err := o.GetEnvVarByID(id)
	if err != nil {
		return err
	}
	if !v.IncludeInAI {
		return errHidden()
	}
	return nil
}

func requireVisibleTemplateID(kind string, o *ops.Ops, id int64) error {
	if kind != ClientAgent {
		return nil
	}
	t, err := o.GetTemplateByID(id)
	if err != nil {
		return err
	}
	if !t.IncludeInAI {
		return errHidden()
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
		if !out[i].IncludeInAI {
			out[i].Value = ""
		}
	}
	return out
}

func hideEnvVar(kind string, v types.EnvVar) types.EnvVar {
	if kind == ClientAgent && !v.IncludeInAI {
		v.Value = ""
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
