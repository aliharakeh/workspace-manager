package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"workspace-manager/internal/api"
	"workspace-manager/internal/api/lib"
	"workspace-manager/internal/api/types"
	"workspace-manager/internal/ai"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Every method below is a proxy to the wmd daemon, which owns the database,
// the runner, the git state and the blueprints. Only the file dialogs, the AI
// methods and openExternal run in this process; see app.go for the call
// helper.

func (a *App) WorkspacesList() ([]types.Workspace, error) {
	var out []types.Workspace
	err := a.call(api.MethodWorkspacesList, nil, &out)
	return out, err
}

func (a *App) WorkspacesCreate(body types.WorkspaceCreateInput) (types.Workspace, error) {
	var out types.Workspace
	err := a.call(api.MethodWorkspacesCreate, body, &out)
	return out, err
}

func (a *App) WorkspacesUpdate(id int64, body types.WorkspaceUpdateInput) (types.Workspace, error) {
	var out types.Workspace
	err := a.call(api.MethodWorkspacesUpdate, map[string]any{"id": id, "body": body}, &out)
	return out, err
}

func (a *App) WorkspacesDelete(id int64) (types.Ok, error) {
	var out types.Ok
	err := a.call(api.MethodWorkspacesDelete, map[string]any{"id": id}, &out)
	return out, err
}

func (a *App) AppsList(workspaceID int64) ([]types.App, error) {
	var out []types.App
	err := a.call(api.MethodAppsList, map[string]any{"workspaceID": workspaceID}, &out)
	return out, err
}

func (a *App) AppsGet(id int64) (types.App, error) {
	var out types.App
	err := a.call(api.MethodAppsGet, map[string]any{"id": id}, &out)
	return out, err
}

func (a *App) AppsCreate(workspaceID int64, body types.AppCreateInput) (types.App, error) {
	var out types.App
	err := a.call(api.MethodAppsCreate, map[string]any{"workspaceID": workspaceID, "body": body}, &out)
	return out, err
}

func (a *App) AppsUpdate(id int64, body types.AppUpdateInput) (types.App, error) {
	var out types.App
	err := a.call(api.MethodAppsUpdate, map[string]any{"id": id, "body": body}, &out)
	return out, err
}

// AppsReorder saves the order of a workspace's apps; ids lists all of them.
func (a *App) AppsReorder(workspaceID int64, ids []int64) (types.Ok, error) {
	var out types.Ok
	err := a.call(api.MethodAppsReorder, map[string]any{"workspaceID": workspaceID, "ids": ids}, &out)
	return out, err
}

func (a *App) AppsDelete(id int64) (types.Ok, error) {
	var out types.Ok
	err := a.call(api.MethodAppsDelete, map[string]any{"id": id}, &out)
	return out, err
}

func (a *App) AppsOpenFolder(id int64) (types.Ok, error) {
	var out types.Ok
	err := a.call(api.MethodAppsOpenFolder, map[string]any{"id": id}, &out)
	return out, err
}

// AppsEditors lists the code editors installed on this machine. It has no
// error to report, so a failure leaves the list empty.
func (a *App) AppsEditors() []types.Editor {
	var out []types.Editor
	if err := a.call(api.MethodAppsEditors, nil, &out); err != nil {
		log.Println("workspace-manager: listing editors:", err)
		return []types.Editor{}
	}
	return out
}

func (a *App) AppsOpenInEditor(id int64, editor string) (types.Ok, error) {
	var out types.Ok
	err := a.call(api.MethodAppsOpenInEditor, map[string]any{"id": id, "editor": editor}, &out)
	return out, err
}

func (a *App) ConfigSetsList(appID int64) ([]types.ConfigSet, error) {
	var out []types.ConfigSet
	err := a.call(api.MethodConfigSetsList, map[string]any{"appID": appID}, &out)
	return out, err
}

func (a *App) ConfigSetsGetDetail(id int64) (types.ConfigSetDetail, error) {
	var out types.ConfigSetDetail
	err := a.call(api.MethodConfigSetsDetail, map[string]any{"id": id}, &out)
	return out, err
}

func (a *App) ConfigSetsCreate(appID int64, body types.ConfigSetCreateInput) (types.ConfigSet, error) {
	var out types.ConfigSet
	err := a.call(api.MethodConfigSetsCreate, map[string]any{"appID": appID, "body": body}, &out)
	return out, err
}

func (a *App) ConfigSetsUpdate(id int64, body types.ConfigSetUpdateInput) (types.ConfigSet, error) {
	var out types.ConfigSet
	err := a.call(api.MethodConfigSetsUpdate, map[string]any{"id": id, "body": body}, &out)
	return out, err
}

func (a *App) ConfigSetsDelete(id int64) (types.Ok, error) {
	var out types.Ok
	err := a.call(api.MethodConfigSetsDelete, map[string]any{"id": id}, &out)
	return out, err
}

func (a *App) ConfigSetsActivate(id int64) (types.ConfigSetActivateResult, error) {
	var out types.ConfigSetActivateResult
	err := a.call(api.MethodConfigSetsActivate, map[string]any{"id": id}, &out)
	return out, err
}

func (a *App) ConfigSetsCopyFrom(id, sourceID int64, parts *types.CopyParts) (types.ConfigSet, error) {
	var out types.ConfigSet
	err := a.call(api.MethodConfigSetsCopyFrom, map[string]any{"id": id, "sourceID": sourceID, "parts": parts}, &out)
	return out, err
}

func (a *App) EnvVarsList(appID int64) ([]types.EnvVar, error) {
	var out []types.EnvVar
	err := a.call(api.MethodEnvVarsList, map[string]any{"appID": appID}, &out)
	return out, err
}

func (a *App) EnvVarsCreate(appID int64, body types.EnvVarCreateInput) (types.EnvVar, error) {
	var out types.EnvVar
	err := a.call(api.MethodEnvVarsCreate, map[string]any{"appID": appID, "body": body}, &out)
	return out, err
}

func (a *App) EnvVarsUpdate(id int64, body types.EnvVarUpdateInput) (types.EnvVar, error) {
	var out types.EnvVar
	err := a.call(api.MethodEnvVarsUpdate, map[string]any{"id": id, "body": body}, &out)
	return out, err
}

func (a *App) EnvVarsDelete(id int64) (types.Ok, error) {
	var out types.Ok
	err := a.call(api.MethodEnvVarsDelete, map[string]any{"id": id}, &out)
	return out, err
}

// EnvVarsImport picks the file in this process (the dialog belongs to the
// window) and lets the server import it into the app's active config set.
func (a *App) EnvVarsImport(appID int64) (types.ImportEnvResult, error) {
	app, err := a.appByID(appID)
	if err != nil {
		return types.ImportEnvResult{}, err
	}
	picked, err := PickFile(a.ctx, app.ProjectPath)
	if err != nil {
		return types.ImportEnvResult{}, err
	}
	if picked.Cancelled {
		return types.ImportEnvResult{Cancelled: true}, nil
	}
	var out types.ImportEnvResult
	err = a.call(api.MethodEnvVarsImport, map[string]any{"appID": appID, "path": picked.Path}, &out)
	return out, err
}

func (a *App) TemplatesList(appID int64) ([]types.Template, error) {
	var out []types.Template
	err := a.call(api.MethodTemplatesList, map[string]any{"appID": appID}, &out)
	return out, err
}

func (a *App) TemplatesCreate(appID int64, body types.TemplateCreateInput) (types.Template, error) {
	var out types.Template
	err := a.call(api.MethodTemplatesCreate, map[string]any{"appID": appID, "body": body}, &out)
	return out, err
}

func (a *App) TemplatesUpdate(id int64, body types.TemplateUpdateInput) (types.Template, error) {
	var out types.Template
	err := a.call(api.MethodTemplatesUpdate, map[string]any{"id": id, "body": body}, &out)
	return out, err
}

func (a *App) TemplatesDelete(id int64) (types.Ok, error) {
	var out types.Ok
	err := a.call(api.MethodTemplatesDelete, map[string]any{"id": id}, &out)
	return out, err
}

func (a *App) PackageScriptsList(appID int64) (types.PackageScripts, error) {
	var out types.PackageScripts
	err := a.call(api.MethodPackageScriptsList, map[string]any{"appID": appID}, &out)
	return out, err
}

// RunConfigGet returns the active config set's dev, prod, build or setup commands.
func (a *App) RunConfigGet(appID int64, kind string) (types.RunConfig, error) {
	var out types.RunConfig
	err := a.call(api.MethodRunConfigGet, map[string]any{"appID": appID, "kind": kind}, &out)
	return out, err
}

func (a *App) RunConfigSave(appID int64, kind string, body types.RunConfigSaveInput) (types.RunConfig, error) {
	var out types.RunConfig
	err := a.call(api.MethodRunConfigSave, map[string]any{"appID": appID, "kind": kind, "body": body}, &out)
	return out, err
}

// RunnerStatus reports an app's current runner session. It has no error to
// report, so a failure reads as "not running".
func (a *App) RunnerStatus(appID int64) types.StatusEvent {
	var out types.StatusEvent
	if err := a.call(api.MethodRunnerStatus, map[string]any{"appID": appID}, &out); err != nil {
		log.Println("workspace-manager: runner status:", err)
		return types.StatusEvent{Processes: []types.ProcessState{}}
	}
	return out
}

func (a *App) RunnerWorkspaceStatus(workspaceID int64) ([]types.StatusEvent, error) {
	var out []types.StatusEvent
	err := a.call(api.MethodRunnerWorkspaceStatus, map[string]any{"workspaceID": workspaceID}, &out)
	return out, err
}

// RunnerOutput returns the recent terminal output of one command of the app's
// current run, so a terminal opened late can catch up before live events.
func (a *App) RunnerOutput(appID, commandID int64) types.RunnerOutput {
	var out types.RunnerOutput
	if err := a.call(api.MethodRunnerOutput, map[string]any{"appID": appID, "commandID": commandID}, &out); err != nil {
		log.Println("workspace-manager: runner output:", err)
		return types.RunnerOutput{}
	}
	return out
}

// RunnerResize sets the terminal size of a running command.
func (a *App) RunnerResize(appID, commandID int64, cols, rows int) types.Ok {
	var out types.Ok
	err := a.call(api.MethodRunnerResize, map[string]any{"appID": appID, "commandID": commandID, "cols": cols, "rows": rows}, &out)
	if err != nil {
		log.Println("workspace-manager: resizing terminal:", err)
		return types.Ok{}
	}
	return out
}

func (a *App) RunnerRun(appID int64) (types.StatusEvent, error) {
	var out types.StatusEvent
	err := a.call(api.MethodRunnerRun, map[string]any{"appID": appID}, &out)
	return out, err
}

func (a *App) RunnerDev(appID int64) (types.StatusEvent, error) {
	var out types.StatusEvent
	err := a.call(api.MethodRunnerDev, map[string]any{"appID": appID}, &out)
	return out, err
}

func (a *App) RunnerProd(appID int64) (types.StatusEvent, error) {
	var out types.StatusEvent
	err := a.call(api.MethodRunnerProd, map[string]any{"appID": appID}, &out)
	return out, err
}

// RunnerBuild runs the active config set's build commands. It shares the app's
// session with RunnerRun: it fails while the app is running or building, and
// RunnerStop stops it.
func (a *App) RunnerBuild(appID int64) (types.StatusEvent, error) {
	var out types.StatusEvent
	err := a.call(api.MethodRunnerBuild, map[string]any{"appID": appID}, &out)
	return out, err
}

// RunnerSetup runs the active config set's setup commands. Like RunnerBuild it
// shares the app's session: it fails while the app is running, building or
// being set up, and RunnerStop stops it.
func (a *App) RunnerSetup(appID int64) (types.StatusEvent, error) {
	var out types.StatusEvent
	err := a.call(api.MethodRunnerSetup, map[string]any{"appID": appID}, &out)
	return out, err
}

func (a *App) RunnerStop(appID int64) (types.StatusEvent, error) {
	var out types.StatusEvent
	err := a.call(api.MethodRunnerStop, map[string]any{"appID": appID}, &out)
	return out, err
}

func (a *App) RunnerReload(appID int64) (types.StatusEvent, error) {
	var out types.StatusEvent
	err := a.call(api.MethodRunnerReload, map[string]any{"appID": appID}, &out)
	return out, err
}

func (a *App) GitInfo(appID int64) (types.GitInfo, error) {
	var out types.GitInfo
	err := a.call(api.MethodGitInfo, map[string]any{"appID": appID}, &out)
	return out, err
}

func (a *App) GitFetchAll(appID int64) (types.GitOutput, error) {
	var out types.GitOutput
	err := a.call(api.MethodGitFetchAll, map[string]any{"appID": appID}, &out)
	return out, err
}

// GitBranchCreate creates a local branch at base (default HEAD), not checked out.
func (a *App) GitBranchCreate(appID int64, name string, base string) (types.Ok, error) {
	var out types.Ok
	err := a.call(api.MethodGitBranchCreate, map[string]any{"appID": appID, "name": name, "base": base}, &out)
	return out, err
}

func (a *App) GitBranchRename(appID int64, oldName string, newName string) (types.Ok, error) {
	var out types.Ok
	err := a.call(api.MethodGitBranchRename, map[string]any{"appID": appID, "oldName": oldName, "newName": newName}, &out)
	return out, err
}

// GitBranchCheckout switches the app's own folder to a local branch. It is
// refused while the app has a runner session, which works in that folder.
func (a *App) GitBranchCheckout(appID int64, name string) (types.Ok, error) {
	var out types.Ok
	err := a.call(api.MethodGitBranchCheckout, map[string]any{"appID": appID, "name": name}, &out)
	return out, err
}

// GitBranchPull pulls a local branch's upstream in the worktree that has it
// checked out. Merge conflicts come back in the result, not as an error. It is
// refused while the app has a runner session.
func (a *App) GitBranchPull(appID int64, name string) (types.GitPullResult, error) {
	var out types.GitPullResult
	err := a.call(api.MethodGitBranchPull, map[string]any{"appID": appID, "name": name}, &out)
	return out, err
}

// GitBranchPush pushes a local branch to its upstream, or publishes it when it
// has none; force uses --force-with-lease.
func (a *App) GitBranchPush(appID int64, name string, force bool) (types.GitOutput, error) {
	var out types.GitOutput
	err := a.call(api.MethodGitBranchPush, map[string]any{"appID": appID, "name": name, "force": force}, &out)
	return out, err
}

// GitFileOpenInEditor opens a file of one of the repo's worktrees (e.g. a
// conflicted one) in an editor (an id from AppsEditors).
func (a *App) GitFileOpenInEditor(appID int64, file string, editor string) (types.Ok, error) {
	var out types.Ok
	err := a.call(api.MethodGitFileOpenInEditor, map[string]any{"appID": appID, "file": file, "editor": editor}, &out)
	return out, err
}

// GitBranchDelete deletes a local branch; force also deletes an unmerged one.
func (a *App) GitBranchDelete(appID int64, name string, force bool) (types.Ok, error) {
	var out types.Ok
	err := a.call(api.MethodGitBranchDelete, map[string]any{"appID": appID, "name": name, "force": force}, &out)
	return out, err
}

func (a *App) GitWorktreeAdd(appID int64, body types.GitWorktreeAddInput) (types.GitWorktreeAddResult, error) {
	var out types.GitWorktreeAddResult
	err := a.call(api.MethodGitWorktreeAdd, map[string]any{"appID": appID, "body": body}, &out)
	return out, err
}

func (a *App) GitWorktreeRemove(appID int64, path string, force bool) (types.Ok, error) {
	var out types.Ok
	err := a.call(api.MethodGitWorktreeRemove, map[string]any{"appID": appID, "path": path, "force": force}, &out)
	return out, err
}

func (a *App) GitWorktreePrune(appID int64) (types.GitOutput, error) {
	var out types.GitOutput
	err := a.call(api.MethodGitWorktreePrune, map[string]any{"appID": appID}, &out)
	return out, err
}

// GitWorktreeStart runs the app's run, build or setup config (by kind) inside
// the worktree at path, with config set configSetID (0: the app's active set).
// It is the app's one runner session, like RunnerRun.
func (a *App) GitWorktreeStart(appID int64, path string, kind string, configSetID int64) (types.StatusEvent, error) {
	var out types.StatusEvent
	err := a.call(api.MethodGitWorktreeStart, map[string]any{"appID": appID, "path": path, "kind": kind, "configSetID": configSetID}, &out)
	return out, err
}

// WorktreeLinksGet returns the app's shared worktree paths: files and folders,
// relative to the app folder, that its worktrees link to instead of copying.
func (a *App) WorktreeLinksGet(appID int64) ([]string, error) {
	var out []string
	err := a.call(api.MethodWorktreeLinksGet, map[string]any{"appID": appID}, &out)
	return out, err
}

// WorktreeLinksSave replaces the app's shared worktree paths and returns them
// cleaned up.
func (a *App) WorktreeLinksSave(appID int64, paths []string) ([]string, error) {
	var out []string
	err := a.call(api.MethodWorktreeLinksSave, map[string]any{"appID": appID, "paths": paths}, &out)
	return out, err
}

// GitWorktreeLinks reports each shared path in the worktree at path; with
// create it first links the ones that are missing there. Existing files and
// folders are never replaced.
func (a *App) GitWorktreeLinks(appID int64, path string, create bool) ([]types.WorktreeLinkState, error) {
	var out []types.WorktreeLinkState
	err := a.call(api.MethodGitWorktreeLinks, map[string]any{"appID": appID, "path": path, "create": create}, &out)
	return out, err
}

// GitWorktreeOpenFolder opens the app's folder inside the worktree at path.
func (a *App) GitWorktreeOpenFolder(appID int64, path string) (types.Ok, error) {
	var out types.Ok
	err := a.call(api.MethodGitWorktreeOpenFolder, map[string]any{"appID": appID, "path": path}, &out)
	return out, err
}

// GitWorktreeOpenInEditor opens the app's folder inside the worktree at path in
// an editor (an id from AppsEditors).
func (a *App) GitWorktreeOpenInEditor(appID int64, path string, editor string) (types.Ok, error) {
	var out types.Ok
	err := a.call(api.MethodGitWorktreeOpenEditor, map[string]any{"appID": appID, "path": path, "editor": editor}, &out)
	return out, err
}

func (a *App) PortsList() (types.PortsListResult, error) {
	var out types.PortsListResult
	err := a.call(api.MethodPortsList, nil, &out)
	return out, err
}

func (a *App) PortsKill(pid int64) (types.PortsKillResult, error) {
	var out types.PortsKillResult
	err := a.call(api.MethodPortsKill, map[string]any{"pid": pid}, &out)
	return out, err
}

func (a *App) ReadyUrlPatternsList() ([]types.ReadyUrlPattern, error) {
	var out []types.ReadyUrlPattern
	err := a.call(api.MethodReadyUrlPatternsList, nil, &out)
	return out, err
}

func (a *App) ReadyUrlPatternsCreate(body types.ReadyUrlPatternCreateInput) (types.ReadyUrlPattern, error) {
	var out types.ReadyUrlPattern
	err := a.call(api.MethodReadyUrlPatternsCreate, body, &out)
	return out, err
}

func (a *App) ReadyUrlPatternsUpdate(id int64, body types.ReadyUrlPatternUpdateInput) (types.ReadyUrlPattern, error) {
	var out types.ReadyUrlPattern
	err := a.call(api.MethodReadyUrlPatternsUpdate, map[string]any{"id": id, "body": body}, &out)
	return out, err
}

func (a *App) ReadyUrlPatternsDelete(id int64) (types.Ok, error) {
	var out types.Ok
	err := a.call(api.MethodReadyUrlPatternsDelete, map[string]any{"id": id}, &out)
	return out, err
}

func (a *App) SettingsGet() (map[string]string, error) {
	var out map[string]string
	err := a.call(api.MethodSettingsGet, nil, &out)
	return out, err
}

func (a *App) SettingsSet(key string, value string) (map[string]string, error) {
	var out map[string]string
	err := a.call(api.MethodSettingsSet, map[string]any{"key": key, "value": value}, &out)
	return out, err
}

// aiConnectionInfo maps a stored connection to what the UI sees (no API key).
func aiConnectionInfo(name string, cfg ai.AIProviderConfig) types.AIConnectionInfo {
	return types.AIConnectionInfo{
		Name:        name,
		Provider:    cfg.Provider,
		BaseURL:     cfg.BaseURL,
		Model:       cfg.Model,
		HasAPIKey:   cfg.APIKey != "",
		Temperature: cfg.Temperature,

		OpenCodeSession: cfg.OpenCodeSession,
	}
}

func aiConfigInfo(store ai.AIStore) types.AIConfigInfo {
	info := types.AIConfigInfo{
		Providers: make([]types.AIConnectionInfo, 0, len(store.Providers)),
		Active:    store.Active,
	}
	names := make([]string, 0, len(store.Providers))
	for name := range store.Providers {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		info.Providers = append(info.Providers, aiConnectionInfo(name, store.Providers[name]))
	}
	return info
}

// The AI methods run in this process (package gui/ai). They read the app and
// its config set from the server and write through it; the daemon has no AI.

func (a *App) AIConfigGet() (types.AIConfigInfo, error) {
	store, err := ai.LoadAIStore()
	if err != nil {
		return types.AIConfigInfo{}, err
	}
	return aiConfigInfo(store), nil
}

// AIConfigSave upserts one connection keyed by its normalized name.
func (a *App) AIConfigSave(body types.AIProviderConfigInput) (types.AIConfigInfo, error) {
	store, err := ai.LoadAIStore()
	if err != nil {
		return types.AIConfigInfo{}, err
	}
	if err := store.UpsertAIConnection(body.Name, body.Provider, ai.AIProviderConfig{
		BaseURL:     body.BaseURL,
		APIKey:      body.APIKey,
		Model:       body.Model,
		Temperature: body.Temperature,

		OpenCodeSession: body.OpenCodeSession,
	}, body.ClearAPIKey); err != nil {
		return types.AIConfigInfo{}, err
	}
	if err := ai.SaveAIStore(store); err != nil {
		return types.AIConfigInfo{}, err
	}
	return aiConfigInfo(store), nil
}

func (a *App) AIConfigDelete(name string) (types.AIConfigInfo, error) {
	store, err := ai.LoadAIStore()
	if err != nil {
		return types.AIConfigInfo{}, err
	}
	if !store.DeleteAIConnection(name) {
		return types.AIConfigInfo{}, fmt.Errorf("Connection not found")
	}
	if err := ai.SaveAIStore(store); err != nil {
		return types.AIConfigInfo{}, err
	}
	return aiConfigInfo(store), nil
}

func (a *App) AIConfigActivate(body types.AIActivateInput) (types.AIConfigInfo, error) {
	store, err := ai.LoadAIStore()
	if err != nil {
		return types.AIConfigInfo{}, err
	}
	if err := store.ActivateAIConnection(body.Name); err != nil {
		return types.AIConfigInfo{}, err
	}
	if err := ai.SaveAIStore(store); err != nil {
		return types.AIConfigInfo{}, err
	}
	return aiConfigInfo(store), nil
}

func (a *App) AIChat(body types.AIChatInput) (types.AIChatResult, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	text, err := ai.ChatAI(ctx, body.System, body.Prompt)
	if err != nil {
		return types.AIChatResult{}, err
	}
	return types.AIChatResult{Text: text}, nil
}

func (a *App) AIAppChat(body types.AppAIChatInput) (types.AppAIChatResult, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	app, err := a.appByID(body.AppID)
	if err != nil {
		return types.AppAIChatResult{}, err
	}
	var detail types.ConfigSetDetail
	if err := a.call(api.MethodConfigSetsDetail, map[string]any{"id": body.ConfigSetID}, &detail); err != nil {
		return types.AppAIChatResult{}, err
	}
	if detail.AppID != app.ID {
		return types.AppAIChatResult{}, fmt.Errorf("Config set not found")
	}
	return ai.AppChatAI(ctx, app.Name, app.ProjectPath, detail, body.History, body.Instruction, func(ev types.AppAIStreamEvent) {
		if a.ctx == nil {
			return
		}
		payload := map[string]any{"type": ev.Type}
		if ev.Text != "" {
			payload["text"] = ev.Text
		}
		if ev.Call != nil {
			payload["call"] = map[string]any{
				"name":   ev.Call.Name,
				"input":  ev.Call.Input,
				"output": ev.Call.Output,
			}
		}
		EventsEmitAppAI(a.ctx, payload)
	})
}

// AITest runs one minimal generation against an unsaved connection payload so
// the UI can verify credentials before saving. Nothing is persisted.
func (a *App) AITest(body types.AITestInput) (types.AITestResult, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	slug := ai.NormalizeProvider(body.Provider)
	text, err := ai.TestAI(ctx, slug, ai.AIProviderConfig{
		BaseURL:     body.BaseURL,
		APIKey:      body.APIKey,
		Model:       body.Model,
		Temperature: body.Temperature,

		OpenCodeSession: body.OpenCodeSession,
	})
	if err != nil {
		return types.AITestResult{}, err
	}
	return types.AITestResult{Ok: true, Text: text}, nil
}

// AIApplyPatch writes a staged AI patch in one call, so it cannot half-apply:
// the config set's env vars, templates and command lists. Values excluded from
// AI are never touched.
func (a *App) AIApplyPatch(body types.AIApplyPatchInput) (types.Ok, error) {
	var out types.Ok
	if err := a.call(api.MethodConfigSetsApplyChanges, map[string]any{
		"appID": body.AppID, "configSetID": body.ConfigSetID, "patch": body.Patch,
	}, &out); err != nil {
		return types.Ok{}, err
	}
	return out, nil
}

// appByID reads one app through the server. The in-process file dialogs and
// the env import need its project path, which only the server owns.
func (a *App) appByID(id int64) (types.App, error) {
	var app types.App
	err := a.call(api.MethodAppsGet, map[string]any{"id": id}, &app)
	return app, err
}

// FsValidatePath stays in this process: it only checks a path, so it works
// without the server.
func (a *App) FsValidatePath(path string) (types.ValidatePathResult, error) {
	if path == "" {
		return types.ValidatePathResult{}, fmt.Errorf("path is required")
	}
	ok, resolved, errMsg := lib.ValidateProjectPath(path)
	if !ok {
		return types.ValidatePathResult{Ok: false, Error: errMsg}, nil
	}
	return types.ValidatePathResult{Ok: true, Path: resolved}, nil
}

func (a *App) FsPickFolder(body types.FsPickFolderInput) (types.PickFolderResult, error) {
	start := ""
	if body.StartDir != nil {
		start = strings.TrimSpace(*body.StartDir)
	}
	picked, err := PickFolder(a.ctx, start)
	if err != nil {
		return types.PickFolderResult{}, err
	}
	if picked.Cancelled {
		return types.PickFolderResult{Cancelled: true}, nil
	}
	return types.PickFolderResult{Path: picked.Path}, nil
}

func (a *App) FsPickFile(body types.FsPickFileInput) (types.PickFileResult, error) {
	dir := ""
	if body.StartDir != nil {
		dir = strings.TrimSpace(*body.StartDir)
	}
	var projectRoot string
	if body.AppID != nil {
		app, err := a.appByID(*body.AppID)
		if err != nil {
			return types.PickFileResult{}, err
		}
		projectRoot = app.ProjectPath
		if dir == "" {
			dir = app.ProjectPath
		}
	}
	picked, err := PickFile(a.ctx, dir)
	if err != nil {
		return types.PickFileResult{}, err
	}
	if picked.Cancelled {
		return types.PickFileResult{Cancelled: true}, nil
	}
	data, err := os.ReadFile(picked.Path)
	if err != nil {
		return types.PickFileResult{}, fmt.Errorf("Failed to read selected file")
	}
	if projectRoot != "" {
		rel, ok := lib.ToProjectRelative(projectRoot, picked.Path)
		if !ok {
			return types.PickFileResult{}, fmt.Errorf("Selected file must be inside the app project directory")
		}
		return types.PickFileResult{Path: picked.Path, RelativePath: rel, Content: string(data)}, nil
	}
	return types.PickFileResult{Path: picked.Path, Content: string(data)}, nil
}

func (a *App) FsPickAppFile(appID int64) (types.PickFileResult, error) {
	app, err := a.appByID(appID)
	if err != nil {
		return types.PickFileResult{}, err
	}
	picked, err := PickFile(a.ctx, app.ProjectPath)
	if err != nil {
		return types.PickFileResult{}, err
	}
	if picked.Cancelled {
		return types.PickFileResult{Cancelled: true}, nil
	}
	rel, ok := lib.ToProjectRelative(app.ProjectPath, picked.Path)
	if !ok {
		return types.PickFileResult{}, fmt.Errorf("Selected file must be inside the app project directory")
	}
	data, err := os.ReadFile(picked.Path)
	if err != nil {
		return types.PickFileResult{}, err
	}
	return types.PickFileResult{Path: picked.Path, RelativePath: rel, Content: string(data)}, nil
}

// FsReadAppFile reads a file of the app's project, which the server does so it
// can resolve the path against the app it owns.
func (a *App) FsReadAppFile(appID int64, path string) (types.ReadAppFileResult, error) {
	var out types.ReadAppFileResult
	err := a.call(api.MethodFsReadAppFile, map[string]any{"appID": appID, "path": path}, &out)
	return out, err
}

func (a *App) OpenExternal(url string) (types.Ok, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return types.Ok{}, fmt.Errorf("url is required")
	}
	runtime.BrowserOpenURL(a.ctx, url)
	return types.Ok{Ok: true}, nil
}

func (a *App) BlueprintsList() ([]types.Blueprint, error) {
	var out []types.Blueprint
	err := a.call(api.MethodBlueprintsList, nil, &out)
	return out, err
}

func (a *App) BlueprintsCreate(body types.BlueprintInput) (types.Blueprint, error) {
	var out types.Blueprint
	err := a.call(api.MethodBlueprintsCreate, body, &out)
	return out, err
}

func (a *App) BlueprintsUpdate(id int64, body types.BlueprintInput) (types.Blueprint, error) {
	var out types.Blueprint
	err := a.call(api.MethodBlueprintsUpdate, map[string]any{"id": id, "body": body}, &out)
	return out, err
}

func (a *App) BlueprintsDelete(id int64) (types.Ok, error) {
	var out types.Ok
	err := a.call(api.MethodBlueprintsDelete, map[string]any{"id": id}, &out)
	return out, err
}

// BlueprintsAIPropose asks the AI for a blueprint, or for a change to the one
// in body.Draft. It saves nothing: the UI loads the result into the editor.
func (a *App) BlueprintsAIPropose(body types.BlueprintAIInput) (types.BlueprintAIResult, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return ai.ProposeBlueprintAI(ctx, body.Instruction, body.Draft)
}

// BlueprintsCreateApp runs a blueprint to completion (logs stream on the
// "blueprintEvent" event) and returns the created app.
func (a *App) BlueprintsCreateApp(body types.BlueprintRunInput) (types.BlueprintRunResult, error) {
	var out types.BlueprintRunResult
	err := a.call(api.MethodBlueprintsCreateApp, body, &out)
	return out, err
}

// BlueprintsResolve answers a run that paused on a failed command: action is
// "skip" (continue with the next command) or "abort".
func (a *App) BlueprintsResolve(runID, action string) types.Ok {
	var out types.Ok
	if err := a.call(api.MethodBlueprintsResolve, map[string]any{"runID": runID, "action": action}, &out); err != nil {
		log.Println("workspace-manager: resolving blueprint run:", err)
		return types.Ok{}
	}
	return out
}

// BlueprintsSendInput writes keystrokes to the terminal of the command a run is
// executing.
func (a *App) BlueprintsSendInput(runID, data string) (types.Ok, error) {
	var out types.Ok
	err := a.call(api.MethodBlueprintsSendInput, map[string]any{"runID": runID, "data": data}, &out)
	return out, err
}

// BlueprintsResize sets the size of a run's terminal.
func (a *App) BlueprintsResize(runID string, cols, rows int) types.Ok {
	var out types.Ok
	if err := a.call(api.MethodBlueprintsResize, map[string]any{"runID": runID, "cols": cols, "rows": rows}, &out); err != nil {
		log.Println("workspace-manager: resizing blueprint terminal:", err)
		return types.Ok{}
	}
	return out
}

func (a *App) BlueprintsCancel(runID string) types.Ok {
	var out types.Ok
	if err := a.call(api.MethodBlueprintsCancel, map[string]any{"runID": runID}, &out); err != nil {
		log.Println("workspace-manager: cancelling blueprint run:", err)
		return types.Ok{}
	}
	return out
}
