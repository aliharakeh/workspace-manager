package handlers

import (
	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"
)

// configSetCompare registers configSets.compare. For an agent client the op
// returns AI-excluded env vars and templates as a status only (no values).
// The GUI gets the values, as it does from configSets.detail.
func (r *Registry) configSetCompare() {
	o := r.Ops
	r.handle(api.MethodConfigSetsCompare, kindTyped(func(kind string, in struct {
		LeftID  int64 `json:"leftID"`
		RightID int64 `json:"rightID"`
	}) (types.ConfigSetCompare, error) {
		return o.ConfigSetsCompare(in.LeftID, in.RightID, kind == ClientAgent)
	}))
}
