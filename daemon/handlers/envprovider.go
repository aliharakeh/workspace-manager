package handlers

import (
	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"
)

func (r *Registry) envProvider() {
	r.handle(api.MethodEnvProviderRun, typed(func(in struct {
		Shared bool  `json:"shared"`
		ID     int64 `json:"id"`
	}) (types.Ok, error) { return r.Ops.EnvProviderRun(in.Shared, in.ID) }))
	r.handle(api.MethodEnvProviderStop, typed(func(in struct{}) (types.Ok, error) {
		return r.Ops.EnvProviderStop()
	}))
	r.handle(api.MethodEnvProviderGet, typed(func(in struct {
		Shared bool  `json:"shared"`
		ID     int64 `json:"id"`
	}) (types.EnvProvider, error) {
		return r.Ops.EnvProviderGet(in.Shared, in.ID)
	}))
	r.handle(api.MethodEnvProviderSave, typed(func(in struct {
		Shared bool              `json:"shared"`
		ID     int64             `json:"id"`
		Body   types.EnvProvider `json:"body"`
	}) (types.Ok, error) {
		return r.Ops.EnvProviderSave(in.Shared, in.ID, in.Body)
	}))
}
