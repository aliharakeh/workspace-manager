package handlers

import (
	"workspace-manager/internal/api"
	"workspace-manager/internal/api/types"
)

func (r *Registry) scripts() {
	o := r.Ops

	r.handle(api.MethodPackageScriptsList, typed(func(in struct {
		AppID int64 `json:"appID"`
	}) (types.PackageScripts, error) {
		return o.PackageScriptsList(in.AppID)
	}))
	r.handle(api.MethodRunConfigGet, typed(func(in struct {
		AppID int64  `json:"appID"`
		Kind  string `json:"kind"`
	}) (types.RunConfig, error) {
		return o.RunConfigGet(in.AppID, in.Kind)
	}))
	r.handle(api.MethodRunConfigSave, typed(func(in struct {
		AppID int64                    `json:"appID"`
		Kind  string                   `json:"kind"`
		Body  types.RunConfigSaveInput `json:"body"`
	}) (types.RunConfig, error) {
		return o.RunConfigSave(in.AppID, in.Kind, in.Body)
	}))
}

func (r *Registry) runner() {
	o := r.Ops

	r.handle(api.MethodRunnerStatus, typed(func(in struct {
		AppID int64 `json:"appID"`
	}) (types.StatusEvent, error) {
		return o.RunnerStatus(in.AppID), nil
	}))
	r.handle(api.MethodRunnerWorkspaceStatus, typed(func(in struct {
		WorkspaceID int64 `json:"workspaceID"`
	}) ([]types.StatusEvent, error) {
		return o.RunnerWorkspaceStatus(in.WorkspaceID)
	}))
	r.handle(api.MethodRunnerOutput, typed(func(in struct {
		AppID     int64 `json:"appID"`
		CommandID int64 `json:"commandID"`
	}) (types.RunnerOutput, error) {
		return o.RunnerOutput(in.AppID, in.CommandID), nil
	}))
	r.handle(api.MethodRunnerResize, typed(func(in struct {
		AppID     int64 `json:"appID"`
		CommandID int64 `json:"commandID"`
		Cols      int   `json:"cols"`
		Rows      int   `json:"rows"`
	}) (types.Ok, error) {
		return o.RunnerResize(in.AppID, in.CommandID, in.Cols, in.Rows), nil
	}))
	r.handle(api.MethodRunnerRun, typed(func(in struct {
		AppID int64 `json:"appID"`
	}) (types.StatusEvent, error) {
		return o.RunnerRun(in.AppID)
	}))
	r.handle(api.MethodRunnerBuild, typed(func(in struct {
		AppID int64 `json:"appID"`
	}) (types.StatusEvent, error) {
		return o.RunnerBuild(in.AppID)
	}))
	r.handle(api.MethodRunnerDev, typed(func(in struct {
		AppID int64 `json:"appID"`
	}) (types.StatusEvent, error) {
		return o.RunnerRun(in.AppID)
	}))
	r.handle(api.MethodRunnerProd, typed(func(in struct {
		AppID int64 `json:"appID"`
	}) (types.StatusEvent, error) {
		return o.RunnerProd(in.AppID)
	}))
	r.handle(api.MethodRunnerSetup, typed(func(in struct {
		AppID int64 `json:"appID"`
	}) (types.StatusEvent, error) {
		return o.RunnerSetup(in.AppID)
	}))
	r.handle(api.MethodRunnerStop, typed(func(in struct {
		AppID int64 `json:"appID"`
	}) (types.StatusEvent, error) {
		return o.RunnerStop(in.AppID)
	}))
	r.handle(api.MethodRunnerReload, typed(func(in struct {
		AppID int64 `json:"appID"`
	}) (types.StatusEvent, error) {
		return o.RunnerReload(in.AppID)
	}))
}
