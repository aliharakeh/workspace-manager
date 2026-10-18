package handlers

import (
	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"
)

func (r *Registry) workspaces() {
	o := r.Ops

	r.handle(api.MethodWorkspacesList, typed(func(struct{}) ([]types.Workspace, error) {
		return o.WorkspacesList()
	}))
	r.handle(api.MethodWorkspacesCreate, typed(func(in types.WorkspaceCreateInput) (types.Workspace, error) {
		return o.WorkspacesCreate(in)
	}))
	r.handle(api.MethodWorkspacesUpdate, typed(func(in struct {
		ID   int64                      `json:"id"`
		Body types.WorkspaceUpdateInput `json:"body"`
	}) (types.Workspace, error) {
		return o.WorkspacesUpdate(in.ID, in.Body)
	}))
	r.handle(api.MethodWorkspacesReorder, typed(func(in struct {
		IDs []int64 `json:"ids"`
	}) (types.Ok, error) {
		return o.WorkspacesReorder(in.IDs)
	}))
	r.handle(api.MethodWorkspacesSetPinned, typed(func(in struct {
		ID     int64 `json:"id"`
		Pinned bool  `json:"pinned"`
	}) (types.Workspace, error) {
		return o.WorkspacesSetPinned(in.ID, in.Pinned)
	}))
	r.handle(api.MethodWorkspacesDelete, typed(func(in struct {
		ID int64 `json:"id"`
	}) (types.Ok, error) {
		return o.WorkspacesDelete(in.ID)
	}))
}

func (r *Registry) apps() {
	o := r.Ops

	r.handle(api.MethodAppsList, typed(func(in struct {
		WorkspaceID int64 `json:"workspaceID"`
	}) ([]types.App, error) {
		return o.AppsList(in.WorkspaceID)
	}))
	r.handle(api.MethodAppsGet, typed(func(in struct {
		ID int64 `json:"id"`
	}) (types.App, error) {
		return o.AppsGet(in.ID)
	}))
	r.handle(api.MethodAppsCreate, typed(func(in struct {
		WorkspaceID int64                `json:"workspaceID"`
		Body        types.AppCreateInput `json:"body"`
	}) (types.App, error) {
		return o.AppsCreate(in.WorkspaceID, in.Body)
	}))
	r.handle(api.MethodAppsUpdate, typed(func(in struct {
		ID   int64                `json:"id"`
		Body types.AppUpdateInput `json:"body"`
	}) (types.App, error) {
		return o.AppsUpdate(in.ID, in.Body)
	}))
	r.handle(api.MethodAppsReorder, typed(func(in struct {
		WorkspaceID int64   `json:"workspaceID"`
		IDs         []int64 `json:"ids"`
	}) (types.Ok, error) {
		return o.AppsReorder(in.WorkspaceID, in.IDs)
	}))
	r.handle(api.MethodAppsDelete, typed(func(in struct {
		ID int64 `json:"id"`
	}) (types.Ok, error) {
		return o.AppsDelete(in.ID)
	}))
	r.handle(api.MethodAppsOpenFolder, typed(func(in struct {
		ID int64 `json:"id"`
	}) (types.Ok, error) {
		return o.AppsOpenFolder(in.ID)
	}))
	r.handle(api.MethodAppsEditors, typed(func(struct{}) ([]types.Editor, error) {
		return o.AppsEditors(), nil
	}))
	r.handle(api.MethodAppsOpenInEditor, typed(func(in struct {
		ID     int64  `json:"id"`
		Editor string `json:"editor"`
	}) (types.Ok, error) {
		return o.AppsOpenInEditor(in.ID, in.Editor)
	}))
}
