package gui

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"workspace-manager/gui/ai"
	"workspace-manager/shared/api"
	"workspace-manager/shared/api/lib"
	"workspace-manager/shared/api/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Every method below is a proxy to the wmd daemon, which owns the database, the
// runner and the blueprints, except: git (the git package), the AI methods,
// the file dialogs and OpenExternal, which run in this process.

// m is a daemon call's parameters.
type m = map[string]any

func (a *App) WorkspacesList() ([]types.Workspace, error) {
	return call[[]types.Workspace](a.link, api.MethodWorkspacesList, nil)
}

func (a *App) WorkspacesCreate(body types.WorkspaceCreateInput) (types.Workspace, error) {
	return call[types.Workspace](a.link, api.MethodWorkspacesCreate, body)
}

func (a *App) WorkspacesUpdate(id int64, body types.WorkspaceUpdateInput) (types.Workspace, error) {
	return call[types.Workspace](a.link, api.MethodWorkspacesUpdate, m{"id": id, "body": body})
}

func (a *App) WorkspacesDelete(id int64) (types.Ok, error) {
	return call[types.Ok](a.link, api.MethodWorkspacesDelete, m{"id": id})
}

func (a *App) AppsList(workspaceID int64) ([]types.App, error) {
	return call[[]types.App](a.link, api.MethodAppsList, m{"workspaceID": workspaceID})
}

func (a *App) AppsGet(id int64) (types.App, error) {
	return call[types.App](a.link, api.MethodAppsGet, m{"id": id})
}

func (a *App) AppsCreate(workspaceID int64, body types.AppCreateInput) (types.App, error) {
	return call[types.App](a.link, api.MethodAppsCreate, m{"workspaceID": workspaceID, "body": body})
}

func (a *App) AppsUpdate(id int64, body types.AppUpdateInput) (types.App, error) {
	return call[types.App](a.link, api.MethodAppsUpdate, m{"id": id, "body": body})
}

// WorkspacesReorder saves the order of the workspaces; ids lists all of them.
func (a *App) WorkspacesReorder(ids []int64) (types.Ok, error) {
	return call[types.Ok](a.link, api.MethodWorkspacesReorder, m{"ids": ids})
}

// WorkspacesSetPinned pins a workspace to the top of the list, or unpins it.
func (a *App) WorkspacesSetPinned(id int64, pinned bool) (types.Workspace, error) {
	return call[types.Workspace](a.link, api.MethodWorkspacesSetPinned, m{"id": id, "pinned": pinned})
}

// AppsReorder saves the order of a workspace's apps; ids lists all of them.
func (a *App) AppsReorder(workspaceID int64, ids []int64) (types.Ok, error) {
	return call[types.Ok](a.link, api.MethodAppsReorder, m{"workspaceID": workspaceID, "ids": ids})
}

func (a *App) AppsDelete(id int64) (types.Ok, error) {
	return call[types.Ok](a.link, api.MethodAppsDelete, m{"id": id})
}

func (a *App) AppsOpenFolder(id int64) (types.Ok, error) {
	return call[types.Ok](a.link, api.MethodAppsOpenFolder, m{"id": id})
}

// AppsEditors lists the code editors installed on this machine.
func (a *App) AppsEditors() []types.Editor {
	editors, err := call[[]types.Editor](a.link, api.MethodAppsEditors, nil)
	if err != nil || editors == nil {
		return []types.Editor{}
	}
	return editors
}

func (a *App) AppsOpenInEditor(id int64, editor string) (types.Ok, error) {
	return call[types.Ok](a.link, api.MethodAppsOpenInEditor, m{"id": id, "editor": editor})
}

func (a *App) ConfigSetsList(appID int64) ([]types.ConfigSet, error) {
	return call[[]types.ConfigSet](a.link, api.MethodConfigSetsList, m{"appID": appID})
}

func (a *App) ConfigSetsGetDetail(id int64) (types.ConfigSetDetail, error) {
	return call[types.ConfigSetDetail](a.link, api.MethodConfigSetsDetail, m{"id": id})
}

func (a *App) ConfigSetsCreate(appID int64, body types.ConfigSetCreateInput) (types.ConfigSet, error) {
	return call[types.ConfigSet](a.link, api.MethodConfigSetsCreate, m{"appID": appID, "body": body})
}

func (a *App) ConfigSetsUpdate(id int64, body types.ConfigSetUpdateInput) (types.ConfigSet, error) {
	return call[types.ConfigSet](a.link, api.MethodConfigSetsUpdate, m{"id": id, "body": body})
}

func (a *App) ConfigSetsDelete(id int64) (types.Ok, error) {
	return call[types.Ok](a.link, api.MethodConfigSetsDelete, m{"id": id})
}

func (a *App) ConfigSetsActivate(id int64) (types.ConfigSetActivateResult, error) {
	return call[types.ConfigSetActivateResult](a.link, api.MethodConfigSetsActivate, m{"id": id})
}

func (a *App) ConfigSetsCopyFrom(id, sourceID int64, parts *types.CopyParts) (types.ConfigSet, error) {
	return call[types.ConfigSet](a.link, api.MethodConfigSetsCopyFrom, m{"id": id, "sourceID": sourceID, "parts": parts})
}

func (a *App) EnvVarsList(appID int64) ([]types.EnvVar, error) {
	return call[[]types.EnvVar](a.link, api.MethodEnvVarsList, m{"appID": appID})
}

func (a *App) EnvVarsCreate(appID int64, body types.EnvVarCreateInput) (types.EnvVar, error) {
	return call[types.EnvVar](a.link, api.MethodEnvVarsCreate, m{"appID": appID, "body": body})
}

func (a *App) EnvVarsUpdate(id int64, body types.EnvVarUpdateInput) (types.EnvVar, error) {
	return call[types.EnvVar](a.link, api.MethodEnvVarsUpdate, m{"id": id, "body": body})
}

func (a *App) EnvVarsDelete(id int64) (types.Ok, error) {
	return call[types.Ok](a.link, api.MethodEnvVarsDelete, m{"id": id})
}

// EnvVarsImport picks the file in this process (the dialog belongs to the
// window) and lets the daemon import it into the app's active config set.
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
	return call[types.ImportEnvResult](a.link, api.MethodEnvVarsImport, m{"appID": appID, "path": picked.Path})
}

func (a *App) TemplatesList(appID int64) ([]types.Template, error) {
	return call[[]types.Template](a.link, api.MethodTemplatesList, m{"appID": appID})
}

func (a *App) TemplatesCreate(appID int64, body types.TemplateCreateInput) (types.Template, error) {
	return call[types.Template](a.link, api.MethodTemplatesCreate, m{"appID": appID, "body": body})
}

func (a *App) TemplatesUpdate(id int64, body types.TemplateUpdateInput) (types.Template, error) {
	return call[types.Template](a.link, api.MethodTemplatesUpdate, m{"id": id, "body": body})
}

func (a *App) TemplatesDelete(id int64) (types.Ok, error) {
	return call[types.Ok](a.link, api.MethodTemplatesDelete, m{"id": id})
}

func (a *App) PackageScriptsList(appID int64) (types.PackageScripts, error) {
	return call[types.PackageScripts](a.link, api.MethodPackageScriptsList, m{"appID": appID})
}

// RunConfigGet returns the active config set's dev, prod, build or setup commands.
func (a *App) RunConfigGet(appID int64, kind string) (types.RunConfig, error) {
	return call[types.RunConfig](a.link, api.MethodRunConfigGet, m{"appID": appID, "kind": kind})
}

func (a *App) RunConfigSave(appID int64, kind string, body types.RunConfigSaveInput) (types.RunConfig, error) {
	return call[types.RunConfig](a.link, api.MethodRunConfigSave, m{"appID": appID, "kind": kind, "body": body})
}

// RunnerStatus reports an app's current runner session.
func (a *App) RunnerStatus(appID int64) types.StatusEvent {
	st, err := call[types.StatusEvent](a.link, api.MethodRunnerStatus, m{"appID": appID})
	if err != nil {
		return types.StatusEvent{AppID: appID}
	}
	return st
}

func (a *App) RunnerWorkspaceStatus(workspaceID int64) ([]types.StatusEvent, error) {
	return call[[]types.StatusEvent](a.link, api.MethodRunnerWorkspaceStatus, m{"workspaceID": workspaceID})
}

// RunnerOutput returns the recent terminal output of one command of the app's
// current run, so a terminal opened late can catch up before live events.
func (a *App) RunnerOutput(appID, commandID int64) types.RunnerOutput {
	out, err := call[types.RunnerOutput](a.link, api.MethodRunnerOutput, m{"appID": appID, "commandID": commandID})
	if err != nil {
		return types.RunnerOutput{}
	}
	return out
}

// RunnerResize sets the terminal size of a running command.
func (a *App) RunnerResize(appID, commandID int64, cols, rows int) types.Ok {
	out, _ := call[types.Ok](a.link, api.MethodRunnerResize, m{"appID": appID, "commandID": commandID, "cols": cols, "rows": rows})
	return out
}

func (a *App) RunnerRun(appID int64) (types.StatusEvent, error) {
	return call[types.StatusEvent](a.link, api.MethodRunnerRun, m{"appID": appID})
}

func (a *App) RunnerDev(appID int64) (types.StatusEvent, error) {
	return call[types.StatusEvent](a.link, api.MethodRunnerDev, m{"appID": appID})
}

func (a *App) RunnerProd(appID int64) (types.StatusEvent, error) {
	return call[types.StatusEvent](a.link, api.MethodRunnerProd, m{"appID": appID})
}

// RunnerBuild runs the active config set's build commands. It shares the app's
// session with RunnerRun: it fails while the app is running or building, and
// RunnerStop stops it.
func (a *App) RunnerBuild(appID int64) (types.StatusEvent, error) {
	return call[types.StatusEvent](a.link, api.MethodRunnerBuild, m{"appID": appID})
}

// RunnerSetup runs the active config set's setup commands. Like RunnerBuild it
// shares the app's session: it fails while the app is running, building or
// being set up, and RunnerStop stops it.
func (a *App) RunnerSetup(appID int64) (types.StatusEvent, error) {
	return call[types.StatusEvent](a.link, api.MethodRunnerSetup, m{"appID": appID})
}

func (a *App) RunnerStop(appID int64) (types.StatusEvent, error) {
	return call[types.StatusEvent](a.link, api.MethodRunnerStop, m{"appID": appID})
}

func (a *App) RunnerReload(appID int64) (types.StatusEvent, error) {
	return call[types.StatusEvent](a.link, api.MethodRunnerReload, m{"appID": appID})
}

func (a *App) GitInfo(appID int64) (types.GitInfo, error) {
	return a.git.Info(appID)
}

func (a *App) GitFetchAll(appID int64) (types.GitOutput, error) {
	return a.git.FetchAll(appID)
}

// GitBranchCreate creates a local branch at base (default HEAD), not checked out.
func (a *App) GitBranchCreate(appID int64, name string, base string) (types.Ok, error) {
	return a.git.BranchCreate(appID, name, base)
}

func (a *App) GitBranchRename(appID int64, oldName string, newName string) (types.Ok, error) {
	return a.git.BranchRename(appID, oldName, newName)
}

// GitBranchCheckout switches the app's own folder to a local branch. It is
// refused while the app has a runner session, which works in that folder.
func (a *App) GitBranchCheckout(appID int64, name string) (types.Ok, error) {
	return a.git.BranchCheckout(appID, name)
}

// GitBranchPull pulls a local branch's upstream in the worktree that has it
// checked out. Merge conflicts come back in the result, not as an error. It is
// refused while the app has a runner session.
func (a *App) GitBranchPull(appID int64, name string) (types.GitPullResult, error) {
	return a.git.BranchPull(appID, name)
}

// GitBranchPush pushes a local branch to its upstream, or publishes it when it
// has none; force uses --force-with-lease.
func (a *App) GitBranchPush(appID int64, name string, force bool) (types.GitOutput, error) {
	return a.git.BranchPush(appID, name, force)
}

// GitFileOpenInEditor opens a file of one of the repo's worktrees (e.g. a
// conflicted one) in an editor (an id from AppsEditors).
func (a *App) GitFileOpenInEditor(appID int64, file string, editor string) (types.Ok, error) {
	return a.git.FileOpenInEditor(appID, file, editor)
}

// GitBranchDelete deletes a local branch; force also deletes an unmerged one.
func (a *App) GitBranchDelete(appID int64, name string, force bool) (types.Ok, error) {
	return a.git.BranchDelete(appID, name, force)
}

func (a *App) GitWorktreeAdd(appID int64, body types.GitWorktreeAddInput) (types.GitWorktreeAddResult, error) {
	return a.git.WorktreeAdd(appID, body)
}

func (a *App) GitWorktreeRemove(appID int64, path string, force bool) (types.Ok, error) {
	return a.git.WorktreeRemove(appID, path, force)
}

func (a *App) GitWorktreePrune(appID int64) (types.GitOutput, error) {
	return a.git.WorktreePrune(appID)
}

// GitWorktreeStart runs the app's run, build or setup config (by kind) inside
// the worktree at path, with config set configSetID (0: the app's active set).
// It is the app's one runner session, like RunnerRun.
func (a *App) GitWorktreeStart(appID int64, path string, kind string, configSetID int64) (types.StatusEvent, error) {
	return a.git.WorktreeStart(appID, path, kind, configSetID)
}

// WorktreeLinksGet returns the app's shared worktree paths: files and folders,
// relative to the app folder, that its worktrees link to instead of copying.
func (a *App) WorktreeLinksGet(appID int64) ([]string, error) {
	return call[[]string](a.link, api.MethodWorktreeLinksGet, m{"appID": appID})
}

// WorktreeLinksSave replaces the app's shared worktree paths and returns them
// cleaned up.
func (a *App) WorktreeLinksSave(appID int64, paths []string) ([]string, error) {
	return call[[]string](a.link, api.MethodWorktreeLinksSave, m{"appID": appID, "paths": paths})
}

// GitWorktreeLinks reports each shared path in the worktree at path; with
// create it first links the ones that are missing there. Existing files and
// folders are never replaced.
func (a *App) GitWorktreeLinks(appID int64, path string, create bool) ([]types.WorktreeLinkState, error) {
	return a.git.WorktreeLinks(appID, path, create)
}

// GitWorktreeOpenFolder opens the app's folder inside the worktree at path.
func (a *App) GitWorktreeOpenFolder(appID int64, path string) (types.Ok, error) {
	return a.git.WorktreeOpenFolder(appID, path)
}

// GitWorktreeOpenInEditor opens the app's folder inside the worktree at path in
// an editor (an id from AppsEditors).
func (a *App) GitWorktreeOpenInEditor(appID int64, path string, editor string) (types.Ok, error) {
	return a.git.WorktreeOpenInEditor(appID, path, editor)
}

func (a *App) PortsList() (types.PortsListResult, error) {
	return call[types.PortsListResult](a.link, api.MethodPortsList, nil)
}

func (a *App) PortsKill(pid int64) (types.PortsKillResult, error) {
	return call[types.PortsKillResult](a.link, api.MethodPortsKill, m{"pid": pid})
}

func (a *App) ReadyUrlPatternsList() ([]types.ReadyUrlPattern, error) {
	return call[[]types.ReadyUrlPattern](a.link, api.MethodReadyUrlPatternsList, nil)
}

func (a *App) ReadyUrlPatternsCreate(body types.ReadyUrlPatternCreateInput) (types.ReadyUrlPattern, error) {
	return call[types.ReadyUrlPattern](a.link, api.MethodReadyUrlPatternsCreate, body)
}

func (a *App) ReadyUrlPatternsUpdate(id int64, body types.ReadyUrlPatternUpdateInput) (types.ReadyUrlPattern, error) {
	return call[types.ReadyUrlPattern](a.link, api.MethodReadyUrlPatternsUpdate, m{"id": id, "body": body})
}

func (a *App) ReadyUrlPatternsDelete(id int64) (types.Ok, error) {
	return call[types.Ok](a.link, api.MethodReadyUrlPatternsDelete, m{"id": id})
}

func (a *App) SettingsGet() (map[string]string, error) {
	return call[map[string]string](a.link, api.MethodSettingsGet, nil)
}

func (a *App) SettingsSet(key string, value string) (map[string]string, error) {
	return call[map[string]string](a.link, api.MethodSettingsSet, types.SettingsSetInput{Key: key, Value: value})
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
	detail, err := a.ConfigSetsGetDetail(body.ConfigSetID)
	if err != nil {
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
	if _, err := call[types.Ok](a.link, api.MethodConfigSetsApplyChanges, m{"appID": body.AppID, "configSetID": body.ConfigSetID, "patch": body.Patch}); err != nil {
		return types.Ok{}, err
	}
	return types.Ok{Ok: true}, nil
}

// appByID reads one app. The file dialogs and the env import need its project
// path.
func (a *App) appByID(id int64) (types.App, error) {
	return call[types.App](a.link, api.MethodAppsGet, m{"id": id})
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

// FsReadAppFile reads a file of the app's project, which the daemon does so it
// can resolve the path against the app it owns.
func (a *App) FsReadAppFile(appID int64, path string) (types.ReadAppFileResult, error) {
	return call[types.ReadAppFileResult](a.link, api.MethodFsReadAppFile, m{"appID": appID, "path": path})
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
	return call[[]types.Blueprint](a.link, api.MethodBlueprintsList, nil)
}

func (a *App) BlueprintsCreate(body types.BlueprintInput) (types.Blueprint, error) {
	return call[types.Blueprint](a.link, api.MethodBlueprintsCreate, body)
}

func (a *App) BlueprintsUpdate(id int64, body types.BlueprintInput) (types.Blueprint, error) {
	return call[types.Blueprint](a.link, api.MethodBlueprintsUpdate, m{"id": id, "body": body})
}

func (a *App) BlueprintsDelete(id int64) (types.Ok, error) {
	return call[types.Ok](a.link, api.MethodBlueprintsDelete, m{"id": id})
}

// BlueprintsAIPropose asks the AI for a blueprint, or for a change to the one
// in body.Draft. It saves nothing: the UI loads the result into the editor.
func (a *App) BlueprintsAIPropose(body types.BlueprintAIInput) (types.BlueprintAIResult, error) {
	return ai.ProposeBlueprintAI(ctxOrBackground(a.ctx), body.Instruction, body.Draft)
}

// BlueprintsCreateApp runs a blueprint to completion (logs stream on the
// "blueprintEvent" event) and returns the created app.
func (a *App) BlueprintsCreateApp(body types.BlueprintRunInput) (types.BlueprintRunResult, error) {
	return call[types.BlueprintRunResult](a.link, api.MethodBlueprintsCreateApp, body)
}

// BlueprintsResolve answers a run that paused on a failed command: action is
// "skip" (continue with the next command) or "abort".
func (a *App) BlueprintsResolve(runID, action string) types.Ok {
	out, _ := call[types.Ok](a.link, api.MethodBlueprintsResolve, m{"runID": runID, "action": action})
	return out
}

// BlueprintsSendInput writes keystrokes to the terminal of the command a run is
// executing.
func (a *App) BlueprintsSendInput(runID, data string) (types.Ok, error) {
	return call[types.Ok](a.link, api.MethodBlueprintsSendInput, m{"runID": runID, "data": data})
}

// BlueprintsResize sets the size of a run's terminal.
func (a *App) BlueprintsResize(runID string, cols, rows int) types.Ok {
	out, _ := call[types.Ok](a.link, api.MethodBlueprintsResize, m{"runID": runID, "cols": cols, "rows": rows})
	return out
}

func (a *App) BlueprintsCancel(runID string) types.Ok {
	out, _ := call[types.Ok](a.link, api.MethodBlueprintsCancel, m{"runID": runID})
	return out
}
