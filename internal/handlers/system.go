package handlers

import (
	"context"
	"strings"

	"workspace-manager/internal/api"
	"workspace-manager/internal/api/types"
	"workspace-manager/internal/native"
)

func (r *Registry) system() {
	o := r.Ops

	r.handle(api.MethodPortsList, typed(func(struct{}) (types.PortsListResult, error) {
		return o.PortsList()
	}))
	r.handle(api.MethodPortsKill, typed(func(in struct {
		PID int64 `json:"pid"`
	}) (types.PortsKillResult, error) {
		return o.PortsKill(in.PID)
	}))

	r.handle(api.MethodReadyUrlPatternsList, typed(func(struct{}) ([]types.ReadyUrlPattern, error) {
		return o.ReadyUrlPatternsList()
	}))
	r.handle(api.MethodReadyUrlPatternsCreate, typed(func(in types.ReadyUrlPatternCreateInput) (types.ReadyUrlPattern, error) {
		return o.ReadyUrlPatternsCreate(in)
	}))
	r.handle(api.MethodReadyUrlPatternsUpdate, typed(func(in struct {
		ID   int64                          `json:"id"`
		Body types.ReadyUrlPatternUpdateInput `json:"body"`
	}) (types.ReadyUrlPattern, error) {
		return o.ReadyUrlPatternsUpdate(in.ID, in.Body)
	}))
	r.handle(api.MethodReadyUrlPatternsDelete, typed(func(in struct {
		ID int64 `json:"id"`
	}) (types.Ok, error) {
		return o.ReadyUrlPatternsDelete(in.ID)
	}))

	r.handle(api.MethodSettingsGet, typed(func(struct{}) (map[string]string, error) {
		return o.SettingsGet()
	}))
	r.handle(api.MethodSettingsSet, typed(func(in types.SettingsSetInput) (map[string]string, error) {
		return o.SettingsSet(in.Key, in.Value)
	}))

	r.handle(api.MethodFsValidatePath, typed(func(in struct {
		Path string `json:"path"`
	}) (types.ValidatePathResult, error) {
		return o.FsValidatePath(in.Path)
	}))
	r.handle(api.MethodFsReadAppFile, typed(func(in struct {
		AppID int64  `json:"appID"`
		Path  string `json:"path"`
	}) (types.ReadAppFileResult, error) {
		return o.FsReadAppFile(in.AppID, in.Path)
	}))

	r.handle(api.MethodOpenExternal, typed(func(in struct {
		URL string `json:"url"`
	}) (types.Ok, error) {
		url := strings.TrimSpace(in.URL)
		if url == "" {
			return types.Ok{}, api.Errf(api.KindInvalid, "url is required")
		}
		if err := native.OpenURL(url); err != nil {
			return types.Ok{}, err
		}
		return types.Ok{Ok: true}, nil
	}))
}

func (r *Registry) blueprints() {
	o := r.Ops

	r.handle(api.MethodBlueprintsList, typed(func(struct{}) ([]types.Blueprint, error) {
		return o.BlueprintsList()
	}))
	r.handle(api.MethodBlueprintsCreate, typed(func(in types.BlueprintInput) (types.Blueprint, error) {
		return o.BlueprintsCreate(in)
	}))
	r.handle(api.MethodBlueprintsUpdate, typed(func(in struct {
		ID   int64                `json:"id"`
		Body types.BlueprintInput `json:"body"`
	}) (types.Blueprint, error) {
		return o.BlueprintsUpdate(in.ID, in.Body)
	}))
	r.handle(api.MethodBlueprintsDelete, typed(func(in struct {
		ID int64 `json:"id"`
	}) (types.Ok, error) {
		return o.BlueprintsDelete(in.ID)
	}))
	r.handle(api.MethodBlueprintsCreateApp, typedCtx(func(ctx context.Context, in types.BlueprintRunInput) (types.BlueprintRunResult, error) {
		_ = ctx
		return o.BlueprintsCreateApp(in)
	}))
	r.handle(api.MethodBlueprintsResolve, typed(func(in struct {
		RunID  string `json:"runID"`
		Action string `json:"action"`
	}) (types.Ok, error) {
		return o.BlueprintsResolve(in.RunID, in.Action), nil
	}))
	r.handle(api.MethodBlueprintsSendInput, typed(func(in struct {
		RunID string `json:"runID"`
		Data  string `json:"data"`
	}) (types.Ok, error) {
		return o.BlueprintsSendInput(in.RunID, in.Data)
	}))
	r.handle(api.MethodBlueprintsResize, typed(func(in struct {
		RunID string `json:"runID"`
		Cols  int    `json:"cols"`
		Rows  int    `json:"rows"`
	}) (types.Ok, error) {
		return o.BlueprintsResize(in.RunID, in.Cols, in.Rows), nil
	}))
	r.handle(api.MethodBlueprintsCancel, typed(func(in struct {
		RunID string `json:"runID"`
	}) (types.Ok, error) {
		return o.BlueprintsCancel(in.RunID), nil
	}))
}
