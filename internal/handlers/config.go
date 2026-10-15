package handlers

import (

	"workspace-manager/internal/api"
	"workspace-manager/internal/api/types"
)

func (r *Registry) configSets() {
	o := r.Ops

	r.handle(api.MethodConfigSetsList, typed(func(in struct {
		AppID int64 `json:"appID"`
	}) ([]types.ConfigSet, error) {
		return o.ConfigSetsList(in.AppID)
	}))
	r.handle(api.MethodConfigSetsDetail, typed(func(in struct {
		ID int64 `json:"id"`
	}) (types.ConfigSetDetail, error) {
		d, err := o.ConfigSetsGetDetail(in.ID)
		if err != nil {
			return types.ConfigSetDetail{}, err
		}
		return d, nil
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

	r.handle(api.MethodEnvVarsList, typed(func(in struct {
		AppID int64 `json:"appID"`
	}) ([]types.EnvVar, error) {
		vars, err := o.EnvVarsList(in.AppID)
		if err != nil {
			return nil, err
		}
		return vars, nil
	}))
	r.handle(api.MethodEnvVarsCreate, typed(func(in struct {
		AppID int64                   `json:"appID"`
		Body  types.EnvVarCreateInput `json:"body"`
	}) (types.EnvVar, error) {
		v, err := o.EnvVarsCreate(in.AppID, in.Body)
		if err != nil {
			return types.EnvVar{}, err
		}
		return v, nil
	}))
	r.handle(api.MethodEnvVarsUpdate, typed(func(in struct {
		ID   int64                   `json:"id"`
		Body types.EnvVarUpdateInput `json:"body"`
	}) (types.EnvVar, error) {
		v, err := o.EnvVarsUpdate(in.ID, in.Body)
		if err != nil {
			return types.EnvVar{}, err
		}
		return v, nil
	}))
	r.handle(api.MethodEnvVarsDelete, typed(func(in struct {
		ID int64 `json:"id"`
	}) (types.Ok, error) {
		return o.EnvVarsDelete(in.ID)
	}))
	r.handle(api.MethodEnvVarsImport, typed(func(in struct {
		AppID int64  `json:"appID"`
		Path  string `json:"path"`
	}) (types.ImportEnvResult, error) {
		res, err := o.EnvVarsImport(in.AppID, in.Path)
		if err != nil {
			return types.ImportEnvResult{}, err
		}
		return res, nil
	}))
}

func (r *Registry) templates() {
	o := r.Ops

	r.handle(api.MethodTemplatesList, typed(func(in struct {
		AppID int64 `json:"appID"`
	}) ([]types.Template, error) {
		tpls, err := o.TemplatesList(in.AppID)
		if err != nil {
			return nil, err
		}
		return tpls, nil
	}))
	r.handle(api.MethodTemplatesCreate, typed(func(in struct {
		AppID int64                     `json:"appID"`
		Body  types.TemplateCreateInput `json:"body"`
	}) (types.Template, error) {
		t, err := o.TemplatesCreate(in.AppID, in.Body)
		if err != nil {
			return types.Template{}, err
		}
		return t, nil
	}))
	r.handle(api.MethodTemplatesUpdate, typed(func(in struct {
		ID   int64                     `json:"id"`
		Body types.TemplateUpdateInput `json:"body"`
	}) (types.Template, error) {
		t, err := o.TemplatesUpdate(in.ID, in.Body)
		if err != nil {
			return types.Template{}, err
		}
		return t, nil
	}))
	r.handle(api.MethodTemplatesDelete, typed(func(in struct {
		ID int64 `json:"id"`
	}) (types.Ok, error) {
		return o.TemplatesDelete(in.ID)
	}))
}
