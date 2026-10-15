package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

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
	return a.ops.WorkspacesList()
}

func (a *App) WorkspacesCreate(body types.WorkspaceCreateInput) (types.Workspace, error) {
	return a.ops.WorkspacesCreate(body)
}

func (a *App) WorkspacesUpdate(id int64, body types.WorkspaceUpdateInput) (types.Workspace, error) {
	return a.ops.WorkspacesUpdate(id, body)
}

func (a *App) WorkspacesDelete(id int64) (types.Ok, error) {
	return a.ops.WorkspacesDelete(id)
}

func (a *App) AppsList(workspaceID int64) ([]types.App, error) {
	return a.ops.AppsList(workspaceID)
}

func (a *App) AppsGet(id int64) (types.App, error) {
	return a.ops.AppsGet(id)
}

func (a *App) AppsCreate(workspaceID int64, body types.AppCreateInput) (types.App, error) {
	return a.ops.AppsCreate(workspaceID, body)
}

func (a *App) AppsUpdate(id int64, body types.AppUpdateInput) (types.App, error) {
	return a.ops.AppsUpdate(id, body)
}

// AppsReorder saves the order of a workspace's apps; ids lists all of them.
func (a *App) AppsReorder(workspaceID int64, ids []int64) (types.Ok, error) {
	return a.ops.AppsReorder(workspaceID, ids)
}

func (a *App) AppsDelete(id int64) (types.Ok, error) {
	return a.ops.AppsDelete(id)
}

func (a *App) AppsOpenFolder(id int64) (types.Ok, error) {
	return a.ops.AppsOpenFolder(id)
}

// AppsEditors lists the code editors installed on this machine.
func (a *App) AppsEditors() []types.Editor {
	return a.ops.AppsEditors()
}

func (a *App) AppsOpenInEditor(id int64, editor string) (types.Ok, error) {
	return a.ops.AppsOpenInEditor(id, editor)
}

func (a *App) ConfigSetsList(appID int64) ([]types.ConfigSet, error) {
	return a.ops.ConfigSetsList(appID)
}

func (a *App) ConfigSetsGetDetail(id int64) (types.ConfigSetDetail, error) {
	return a.ops.ConfigSetsGetDetail(id)
}

func (a *App) ConfigSetsCreate(appID int64, body types.ConfigSetCreateInput) (types.ConfigSet, error) {
	return a.ops.ConfigSetsCreate(appID, body)
}

func (a *App) ConfigSetsUpdate(id int64, body types.ConfigSetUpdateInput) (types.ConfigSet, error) {
	return a.ops.ConfigSetsUpdate(id, body)
}

func (a *App) ConfigSetsDelete(id int64) (types.Ok, error) {
	return a.ops.ConfigSetsDelete(id)
}

func (a *App) ConfigSetsActivate(id int64) (types.ConfigSetActivateResult, error) {
	return a.ops.ConfigSetsActivate(id)
}

func (a *App) ConfigSetsCopyFrom(id, sourceID int64, parts *types.CopyParts) (types.ConfigSet, error) {
	return a.ops.ConfigSetsCopyFrom(id, sourceID, parts)
}

func (a *App) EnvVarsList(appID int64) ([]types.EnvVar, error) {
	return a.ops.EnvVarsList(appID)
}

func (a *App) EnvVarsCreate(appID int64, body types.EnvVarCreateInput) (types.EnvVar, error) {
	return a.ops.EnvVarsCreate(appID, body)
}

func (a *App) EnvVarsUpdate(id int64, body types.EnvVarUpdateInput) (types.EnvVar, error) {
	return a.ops.EnvVarsUpdate(id, body)
}

func (a *App) EnvVarsDelete(id int64) (types.Ok, error) {
	return a.ops.EnvVarsDelete(id)
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
	return a.ops.EnvVarsImport(appID, picked.Path)
}

func (a *App) TemplatesList(appID int64) ([]types.Template, error) {
	return a.ops.TemplatesList(appID)
}

func (a *App) TemplatesCreate(appID int64, body types.TemplateCreateInput) (types.Template, error) {
	return a.ops.TemplatesCreate(appID, body)
}

func (a *App) TemplatesUpdate(id int64, body types.TemplateUpdateInput) (types.Template, error) {
	return a.ops.TemplatesUpdate(id, body)
}

func (a *App) TemplatesDelete(id int64) (types.Ok, error) {
	return a.ops.TemplatesDelete(id)
}

func (a *App) PackageScriptsList(appID int64) (types.PackageScripts, error) {
	return a.ops.PackageScriptsList(appID)
}

// RunConfigGet returns the active config set's dev, prod, build or setup commands.
func (a *App) RunConfigGet(appID int64, kind string) (types.RunConfig, error) {
	return a.ops.RunConfigGet(appID, kind)
}

func (a *App) RunConfigSave(appID int64, kind string, body types.RunConfigSaveInput) (types.RunConfig, error) {
	return a.ops.RunConfigSave(appID, kind, body)
}

// RunnerStatus reports an app's current runner session.
func (a *App) RunnerStatus(appID int64) types.StatusEvent {
	return a.ops.RunnerStatus(appID)
}

func (a *App) RunnerWorkspaceStatus(workspaceID int64) ([]types.StatusEvent, error) {
	return a.ops.RunnerWorkspaceStatus(workspaceID)
}

// RunnerOutput returns the recent terminal output of one command of the app's
// current run, so a terminal opened late can catch up before live events.
func (a *App) RunnerOutput(appID, commandID int64) types.RunnerOutput {
	return a.ops.RunnerOutput(appID, commandID)
}

// RunnerResize sets the terminal size of a running command.
func (a *App) RunnerResize(appID, commandID int64, cols, rows int) types.Ok {
	return a.ops.RunnerResize(appID, commandID, cols, rows)
}

func (a *App) RunnerRun(appID int64) (types.StatusEvent, error) {
	return a.ops.RunnerRun(appID)
}

func (a *App) RunnerDev(appID int64) (types.StatusEvent, error) {
	return a.ops.RunnerRun(appID)
}

func (a *App) RunnerProd(appID int64) (types.StatusEvent, error) {
	return a.ops.RunnerProd(appID)
}

// RunnerBuild runs the active config set's build commands. It shares the app's
// session with RunnerRun: it fails while the app is running or building, and
// RunnerStop stops it.
func (a *App) RunnerBuild(appID int64) (types.StatusEvent, error) {
	return a.ops.RunnerBuild(appID)
}

// RunnerSetup runs the active config set's setup commands. Like RunnerBuild it
// shares the app's session: it fails while the app is running, building or
// being set up, and RunnerStop stops it.
func (a *App) RunnerSetup(appID int64) (types.StatusEvent, error) {
	return a.ops.RunnerSetup(appID)
}

func (a *App) RunnerStop(appID int64) (types.StatusEvent, error) {
	return a.ops.RunnerStop(appID)
}

func (a *App) RunnerReload(appID int64) (types.StatusEvent, error) {
	return a.ops.RunnerReload(appID)
}

func (a *App) GitInfo(appID int64) (types.GitInfo, error) {
	return a.ops.GitInfo(appID)
}

func (a *App) GitFetchAll(appID int64) (types.GitOutput, error) {
	return a.ops.GitFetchAll(appID)
}

// GitBranchCreate creates a local branch at base (default HEAD), not checked out.
func (a *App) GitBranchCreate(appID int64, name string, base string) (types.Ok, error) {
	return a.ops.GitBranchCreate(appID, name, base)
}

func (a *App) GitBranchRename(appID int64, oldName string, newName string) (types.Ok, error) {
	return a.ops.GitBranchRename(appID, oldName, newName)
}

// GitBranchCheckout switches the app's own folder to a local branch. It is
// refused while the app has a runner session, which works in that folder.
func (a *App) GitBranchCheckout(appID int64, name string) (types.Ok, error) {
	return a.ops.GitBranchCheckout(appID, name)
}

// GitBranchPull pulls a local branch's upstream in the worktree that has it
// checked out. Merge conflicts come back in the result, not as an error. It is
// refused while the app has a runner session.
func (a *App) GitBranchPull(appID int64, name string) (types.GitPullResult, error) {
	return a.ops.GitBranchPull(appID, name)
}

// GitBranchPush pushes a local branch to its upstream, or publishes it when it
// has none; force uses --force-with-lease.
func (a *App) GitBranchPush(appID int64, name string, force bool) (types.GitOutput, error) {
	return a.ops.GitBranchPush(appID, name, force)
}

// GitFileOpenInEditor opens a file of one of the repo's worktrees (e.g. a
// conflicted one) in an editor (an id from AppsEditors).
func (a *App) GitFileOpenInEditor(appID int64, file string, editor string) (types.Ok, error) {
	return a.ops.GitFileOpenInEditor(appID, file, editor)
}

// GitBranchDelete deletes a local branch; force also deletes an unmerged one.
func (a *App) GitBranchDelete(appID int64, name string, force bool) (types.Ok, error) {
	return a.ops.GitBranchDelete(appID, name, force)
}

func (a *App) GitWorktreeAdd(appID int64, body types.GitWorktreeAddInput) (types.GitWorktreeAddResult, error) {
	return a.ops.GitWorktreeAdd(appID, body)
}

func (a *App) GitWorktreeRemove(appID int64, path string, force bool) (types.Ok, error) {
	return a.ops.GitWorktreeRemove(appID, path, force)
}

func (a *App) GitWorktreePrune(appID int64) (types.GitOutput, error) {
	return a.ops.GitWorktreePrune(appID)
}

// GitWorktreeStart runs the app's run, build or setup config (by kind) inside
// the worktree at path, with config set configSetID (0: the app's active set).
// It is the app's one runner session, like RunnerRun.
func (a *App) GitWorktreeStart(appID int64, path string, kind string, configSetID int64) (types.StatusEvent, error) {
	return a.ops.GitWorktreeStart(appID, path, kind, configSetID)
}

// WorktreeLinksGet returns the app's shared worktree paths: files and folders,
// relative to the app folder, that its worktrees link to instead of copying.
func (a *App) WorktreeLinksGet(appID int64) ([]string, error) {
	return a.ops.WorktreeLinksGet(appID)
}

// WorktreeLinksSave replaces the app's shared worktree paths and returns them
// cleaned up.
func (a *App) WorktreeLinksSave(appID int64, paths []string) ([]string, error) {
	return a.ops.WorktreeLinksSave(appID, paths)
}

// GitWorktreeLinks reports each shared path in the worktree at path; with
// create it first links the ones that are missing there. Existing files and
// folders are never replaced.
func (a *App) GitWorktreeLinks(appID int64, path string, create bool) ([]types.WorktreeLinkState, error) {
	return a.ops.GitWorktreeLinks(appID, path, create)
}

// GitWorktreeOpenFolder opens the app's folder inside the worktree at path.
func (a *App) GitWorktreeOpenFolder(appID int64, path string) (types.Ok, error) {
	return a.ops.GitWorktreeOpenFolder(appID, path)
}

// GitWorktreeOpenInEditor opens the app's folder inside the worktree at path in
// an editor (an id from AppsEditors).
func (a *App) GitWorktreeOpenInEditor(appID int64, path string, editor string) (types.Ok, error) {
	return a.ops.GitWorktreeOpenInEditor(appID, path, editor)
}

func (a *App) PortsList() (types.PortsListResult, error) {
	return a.ops.PortsList()
}

func (a *App) PortsKill(pid int64) (types.PortsKillResult, error) {
	return a.ops.PortsKill(pid)
}

func (a *App) ReadyUrlPatternsList() ([]types.ReadyUrlPattern, error) {
	return a.ops.ReadyUrlPatternsList()
}

func (a *App) ReadyUrlPatternsCreate(body types.ReadyUrlPatternCreateInput) (types.ReadyUrlPattern, error) {
	return a.ops.ReadyUrlPatternsCreate(body)
}

func (a *App) ReadyUrlPatternsUpdate(id int64, body types.ReadyUrlPatternUpdateInput) (types.ReadyUrlPattern, error) {
	return a.ops.ReadyUrlPatternsUpdate(id, body)
}

func (a *App) ReadyUrlPatternsDelete(id int64) (types.Ok, error) {
	return a.ops.ReadyUrlPatternsDelete(id)
}

func (a *App) SettingsGet() (map[string]string, error) {
	return a.ops.SettingsGet()
}

func (a *App) SettingsSet(key string, value string) (map[string]string, error) {
	return a.ops.SettingsSet(key, value)
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
	detail, err := a.ops.ConfigSetsGetDetail(body.ConfigSetID)
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
	if err := a.ops.ConfigSetsApplyChanges(body.AppID, body.ConfigSetID, body.Patch); err != nil {
		return types.Ok{}, err
	}
	return types.Ok{Ok: true}, nil
}

// appByID reads one app. The file dialogs and the env import need its project
// path.
func (a *App) appByID(id int64) (types.App, error) {
	return a.ops.AppsGet(id)
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
	return a.ops.FsReadAppFile(appID, path)
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
	return a.ops.BlueprintsList()
}

func (a *App) BlueprintsCreate(body types.BlueprintInput) (types.Blueprint, error) {
	return a.ops.BlueprintsCreate(body)
}

func (a *App) BlueprintsUpdate(id int64, body types.BlueprintInput) (types.Blueprint, error) {
	return a.ops.BlueprintsUpdate(id, body)
}

func (a *App) BlueprintsDelete(id int64) (types.Ok, error) {
	return a.ops.BlueprintsDelete(id)
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
	return a.ops.BlueprintsCreateApp(body)
}

// BlueprintsResolve answers a run that paused on a failed command: action is
// "skip" (continue with the next command) or "abort".
func (a *App) BlueprintsResolve(runID, action string) types.Ok {
	return a.ops.BlueprintsResolve(runID, action)
}

// BlueprintsSendInput writes keystrokes to the terminal of the command a run is
// executing.
func (a *App) BlueprintsSendInput(runID, data string) (types.Ok, error) {
	return a.ops.BlueprintsSendInput(runID, data)
}

// BlueprintsResize sets the size of a run's terminal.
func (a *App) BlueprintsResize(runID string, cols, rows int) types.Ok {
	return a.ops.BlueprintsResize(runID, cols, rows)
}

func (a *App) BlueprintsCancel(runID string) types.Ok {
	return a.ops.BlueprintsCancel(runID)
}
