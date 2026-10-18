package handlers

import (
	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"
)

// workspaceRun wires the workspace-wide start and stop, which fan out to the
// per-app runner sessions.
func (r *Registry) workspaceRun() {
	o := r.Ops

	r.handle(api.MethodRunnerStartWorkspace, typed(func(in struct {
		WorkspaceID   int64   `json:"workspaceID"`
		ExcludeAppIDs []int64 `json:"excludeAppIds"`
		Sequential    bool    `json:"sequential"`
	}) (types.WorkspaceRunResult, error) {
		return o.RunnerStartWorkspace(in.WorkspaceID, in.ExcludeAppIDs, in.Sequential)
	}))
	r.handle(api.MethodRunnerStopWorkspace, typed(func(in struct {
		WorkspaceID   int64   `json:"workspaceID"`
		ExcludeAppIDs []int64 `json:"excludeAppIds"`
	}) (types.WorkspaceRunResult, error) {
		return o.RunnerStopWorkspace(in.WorkspaceID, in.ExcludeAppIDs)
	}))
}
