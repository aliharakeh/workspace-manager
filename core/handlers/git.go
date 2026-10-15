package handlers

import (
	"workspace-manager/api"
	"workspace-manager/api/types"
)

func (r *Registry) git() {
	o := r.Ops

	r.handle(api.MethodGitInfo, typed(func(in struct {
		AppID int64 `json:"appID"`
	}) (types.GitInfo, error) {
		return o.GitInfo(in.AppID)
	}))
	r.handle(api.MethodGitFetchAll, typed(func(in struct {
		AppID int64 `json:"appID"`
	}) (types.GitOutput, error) {
		return o.GitFetchAll(in.AppID)
	}))
	r.handle(api.MethodGitBranchCreate, typed(func(in struct {
		AppID int64  `json:"appID"`
		Name  string `json:"name"`
		Base  string `json:"base"`
	}) (types.Ok, error) {
		return o.GitBranchCreate(in.AppID, in.Name, in.Base)
	}))
	r.handle(api.MethodGitBranchRename, typed(func(in struct {
		AppID   int64  `json:"appID"`
		OldName string `json:"oldName"`
		NewName string `json:"newName"`
	}) (types.Ok, error) {
		return o.GitBranchRename(in.AppID, in.OldName, in.NewName)
	}))
	r.handle(api.MethodGitBranchCheckout, typed(func(in struct {
		AppID int64  `json:"appID"`
		Name  string `json:"name"`
	}) (types.Ok, error) {
		return o.GitBranchCheckout(in.AppID, in.Name)
	}))
	r.handle(api.MethodGitBranchPull, typed(func(in struct {
		AppID int64  `json:"appID"`
		Name  string `json:"name"`
	}) (types.GitPullResult, error) {
		return o.GitBranchPull(in.AppID, in.Name)
	}))
	r.handle(api.MethodGitBranchPush, typed(func(in struct {
		AppID int64  `json:"appID"`
		Name  string `json:"name"`
		Force bool   `json:"force"`
	}) (types.GitOutput, error) {
		return o.GitBranchPush(in.AppID, in.Name, in.Force)
	}))
	r.handle(api.MethodGitBranchDelete, typed(func(in struct {
		AppID int64  `json:"appID"`
		Name  string `json:"name"`
		Force bool   `json:"force"`
	}) (types.Ok, error) {
		return o.GitBranchDelete(in.AppID, in.Name, in.Force)
	}))
	r.handle(api.MethodGitFileOpenInEditor, typed(func(in struct {
		AppID  int64  `json:"appID"`
		File   string `json:"file"`
		Editor string `json:"editor"`
	}) (types.Ok, error) {
		return o.GitFileOpenInEditor(in.AppID, in.File, in.Editor)
	}))
	r.handle(api.MethodGitWorktreeAdd, typed(func(in struct {
		AppID int64                     `json:"appID"`
		Body  types.GitWorktreeAddInput `json:"body"`
	}) (types.GitWorktreeAddResult, error) {
		return o.GitWorktreeAdd(in.AppID, in.Body)
	}))
	r.handle(api.MethodGitWorktreeRemove, typed(func(in struct {
		AppID int64  `json:"appID"`
		Path  string `json:"path"`
		Force bool   `json:"force"`
	}) (types.Ok, error) {
		return o.GitWorktreeRemove(in.AppID, in.Path, in.Force)
	}))
	r.handle(api.MethodGitWorktreePrune, typed(func(in struct {
		AppID int64 `json:"appID"`
	}) (types.GitOutput, error) {
		return o.GitWorktreePrune(in.AppID)
	}))
	r.handle(api.MethodGitWorktreeStart, typed(func(in struct {
		AppID       int64  `json:"appID"`
		Path        string `json:"path"`
		Kind        string `json:"kind"`
		ConfigSetID int64  `json:"configSetID"`
	}) (types.StatusEvent, error) {
		return o.GitWorktreeStart(in.AppID, in.Path, in.Kind, in.ConfigSetID)
	}))
	r.handle(api.MethodGitWorktreeOpenFolder, typed(func(in struct {
		AppID int64  `json:"appID"`
		Path  string `json:"path"`
	}) (types.Ok, error) {
		return o.GitWorktreeOpenFolder(in.AppID, in.Path)
	}))
	r.handle(api.MethodGitWorktreeOpenEditor, typed(func(in struct {
		AppID  int64  `json:"appID"`
		Path   string `json:"path"`
		Editor string `json:"editor"`
	}) (types.Ok, error) {
		return o.GitWorktreeOpenInEditor(in.AppID, in.Path, in.Editor)
	}))
	r.handle(api.MethodGitWorktreeLinks, typed(func(in struct {
		AppID  int64  `json:"appID"`
		Path   string `json:"path"`
		Create bool   `json:"create"`
	}) ([]types.WorktreeLinkState, error) {
		return o.GitWorktreeLinks(in.AppID, in.Path, in.Create)
	}))
	r.handle(api.MethodWorktreeLinksGet, typed(func(in struct {
		AppID int64 `json:"appID"`
	}) ([]string, error) {
		return o.WorktreeLinksGet(in.AppID)
	}))
	r.handle(api.MethodWorktreeLinksSave, typed(func(in struct {
		AppID int64    `json:"appID"`
		Paths []string `json:"paths"`
	}) ([]string, error) {
		return o.WorktreeLinksSave(in.AppID, in.Paths)
	}))
}
