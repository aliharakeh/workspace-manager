package handlers

import (
	"strings"

	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"
)

// hideSharedVar blanks a private shared variable's values for agents, exactly as
// hideEnvVar does for an app variable (the option labels stay).
func hideSharedVar(kind string, v types.SharedEnvVar) types.SharedEnvVar {
	if kind != ClientAgent || !v.Private {
		return v
	}
	v.Value = ""
	opts := make([]types.EnvVarOption, len(v.Options))
	for i, o := range v.Options {
		opts[i] = types.EnvVarOption{Label: o.Label}
	}
	v.Options = opts
	return v
}

func hideSharedVars(kind string, vars []types.SharedEnvVar) []types.SharedEnvVar {
	if kind != ClientAgent {
		return vars
	}
	out := make([]types.SharedEnvVar, len(vars))
	for i, v := range vars {
		out[i] = hideSharedVar(kind, v)
	}
	return out
}

func (r *Registry) sharedConfig() {
	o := r.Ops

	r.handle(api.MethodSharedSetsList, typed(func(in struct {
		WorkspaceID int64 `json:"workspaceID"`
	}) ([]types.SharedSet, error) {
		return o.SharedSetsList(in.WorkspaceID)
	}))
	r.handle(api.MethodSharedSetsGet, kindTyped(func(kind string, in struct {
		ID int64 `json:"id"`
	}) (types.SharedSetDetail, error) {
		d, err := o.SharedSetsGet(in.ID)
		if err != nil {
			return types.SharedSetDetail{}, err
		}
		d.Vars = hideSharedVars(kind, d.Vars)
		return d, nil
	}))
	r.handle(api.MethodSharedSetsCreate, typed(func(in struct {
		WorkspaceID int64  `json:"workspaceID"`
		Name        string `json:"name"`
	}) (types.SharedSet, error) {
		return o.SharedSetsCreate(in.WorkspaceID, in.Name)
	}))
	r.handle(api.MethodSharedSetsRename, typed(func(in struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}) (types.SharedSet, error) {
		return o.SharedSetsRename(in.ID, in.Name)
	}))
	r.handle(api.MethodSharedSetsDelete, typed(func(in struct {
		ID int64 `json:"id"`
	}) (types.Ok, error) {
		return o.SharedSetsDelete(in.ID)
	}))

	r.handle(api.MethodSharedVarsCreate, kindTyped(func(kind string, in struct {
		SetID int64                   `json:"setID"`
		Body  types.EnvVarCreateInput `json:"body"`
	}) (types.SharedEnvVar, error) {
		if kind == ClientAgent {
			// Creating over a private key would overwrite what the agent cannot see.
			d, err := o.SharedSetsGet(in.SetID)
			if err != nil {
				return types.SharedEnvVar{}, err
			}
			for _, v := range d.Vars {
				if v.Key == strings.TrimSpace(in.Body.Key) && v.Private {
					return types.SharedEnvVar{}, errHidden()
				}
			}
		}
		v, err := o.SharedVarsCreate(in.SetID, in.Body)
		if err != nil {
			return types.SharedEnvVar{}, err
		}
		return hideSharedVar(kind, v), nil
	}))
	r.handle(api.MethodSharedVarsUpdate, kindTyped(func(kind string, in struct {
		ID   int64                   `json:"id"`
		Body types.EnvVarUpdateInput `json:"body"`
	}) (types.SharedEnvVar, error) {
		if kind == ClientAgent {
			existing, err := o.SharedVarsGet(in.ID)
			if err != nil {
				return types.SharedEnvVar{}, err
			}
			if existing.Private {
				// Private variables allow value replacement, never renaming, making public
				// or changing their options and choice.
				if in.Body.Value == nil || in.Body.Options != nil || in.Body.Selected != nil ||
					(in.Body.Key != nil && strings.TrimSpace(*in.Body.Key) != existing.Key) ||
					(in.Body.Private != nil && !*in.Body.Private) {
					return types.SharedEnvVar{}, errHidden()
				}
				in.Body.Private = &existing.Private
			}
		}
		v, err := o.SharedVarsUpdate(in.ID, in.Body)
		if err != nil {
			return types.SharedEnvVar{}, err
		}
		return hideSharedVar(kind, v), nil
	}))
	r.handle(api.MethodSharedVarsDelete, typed(func(in struct {
		ID int64 `json:"id"`
	}) (types.Ok, error) {
		// Deleting reveals nothing, so agents may delete private variables.
		return o.SharedVarsDelete(in.ID)
	}))

	r.handle(api.MethodConfigSetsLinkShared, typed(func(in struct {
		ID   int64                 `json:"id"`
		Body types.SharedLinkInput `json:"body"`
	}) (types.ConfigSet, error) {
		return o.ConfigSetsLinkShared(in.ID, in.Body)
	}))
	r.handle(api.MethodConfigSetsUnlinkShared, typed(func(in struct {
		ID   int64 `json:"id"`
		Keep bool  `json:"keep"`
	}) (types.ConfigSet, error) {
		return o.ConfigSetsUnlinkShared(in.ID, in.Keep)
	}))
}
