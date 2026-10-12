// Package api is the contract between the server and its clients: the request
// and response types, the method registry, and the error model. The CLI and
// the GUI both depend on it; neither imports server code.
package api

// Method names are "group.operation". Each one matches a server handler.
const (
	// Server lifecycle and streams.
	MethodHello    = "hello"
	MethodStop     = "server.stop"
	MethodSubscribe = "events.subscribe"

	// Workspaces.
	MethodWorkspacesList   = "workspaces.list"
	MethodWorkspacesCreate = "workspaces.create"
	MethodWorkspacesUpdate = "workspaces.update"
	MethodWorkspacesDelete = "workspaces.delete"

	// Apps.
	MethodAppsList          = "apps.list"
	MethodAppsGet           = "apps.get"
	MethodAppsCreate        = "apps.create"
	MethodAppsUpdate        = "apps.update"
	MethodAppsReorder       = "apps.reorder"
	MethodAppsDelete        = "apps.delete"
	MethodAppsOpenFolder    = "apps.openFolder"
	MethodAppsEditors       = "apps.editors"
	MethodAppsOpenInEditor  = "apps.openInEditor"

	// Config sets.
	MethodConfigSetsList       = "configSets.list"
	MethodConfigSetsDetail     = "configSets.detail"
	MethodConfigSetsCreate     = "configSets.create"
	MethodConfigSetsUpdate     = "configSets.update"
	MethodConfigSetsDelete     = "configSets.delete"
	MethodConfigSetsActivate   = "configSets.activate"
	MethodConfigSetsCopyFrom   = "configSets.copyFrom"

	// Env vars.
	MethodEnvVarsList   = "envVars.list"
	MethodEnvVarsCreate = "envVars.create"
	MethodEnvVarsUpdate = "envVars.update"
	MethodEnvVarsDelete = "envVars.delete"
	MethodEnvVarsImport = "envVars.import"

	// Templates.
	MethodTemplatesList   = "templates.list"
	MethodTemplatesCreate = "templates.create"
	MethodTemplatesUpdate = "templates.update"
	MethodTemplatesDelete = "templates.delete"

	// Package scripts.
	MethodPackageScriptsList = "packageScripts.list"

	// Run, build and setup command lists.
	MethodRunConfigGet  = "runConfig.get"
	MethodRunConfigSave = "runConfig.save"

	// Runner.
	MethodRunnerStatus          = "runner.status"
	MethodRunnerWorkspaceStatus = "runner.workspaceStatus"
	MethodRunnerOutput          = "runner.output"
	MethodRunnerResize          = "runner.resize"
	MethodRunnerRun             = "runner.run"
	MethodRunnerBuild           = "runner.build"
	MethodRunnerSetup           = "runner.setup"
	MethodRunnerStop            = "runner.stop"
	MethodRunnerReload          = "runner.reload"

	// Git: repository info, branches and worktrees.
	MethodGitInfo               = "git.info"
	MethodGitFetchAll           = "git.fetchAll"
	MethodGitBranchCreate       = "git.branch.create"
	MethodGitBranchRename       = "git.branch.rename"
	MethodGitBranchCheckout     = "git.branch.checkout"
	MethodGitBranchPull         = "git.branch.pull"
	MethodGitBranchPush         = "git.branch.push"
	MethodGitBranchDelete       = "git.branch.delete"
	MethodGitFileOpenInEditor   = "git.file.openInEditor"
	MethodGitWorktreeAdd        = "git.worktree.add"
	MethodGitWorktreeRemove     = "git.worktree.remove"
	MethodGitWorktreePrune      = "git.worktree.prune"
	MethodGitWorktreeStart      = "git.worktree.start"
	MethodGitWorktreeOpenFolder = "git.worktree.openFolder"
	MethodGitWorktreeOpenEditor = "git.worktree.openInEditor"
	MethodGitWorktreeLinks      = "git.worktree.links"
	MethodWorktreeLinksGet      = "git.worktreeLinks.get"
	MethodWorktreeLinksSave     = "git.worktreeLinks.save"

	// Ports.
	MethodPortsList = "ports.list"
	MethodPortsKill = "ports.kill"

	// Ready URL patterns.
	MethodReadyUrlPatternsList   = "readyUrlPatterns.list"
	MethodReadyUrlPatternsCreate = "readyUrlPatterns.create"
	MethodReadyUrlPatternsUpdate = "readyUrlPatterns.update"
	MethodReadyUrlPatternsDelete = "readyUrlPatterns.delete"

	// Settings.
	MethodSettingsGet = "settings.get"
	MethodSettingsSet = "settings.set"

	// AI connections and agents. GUI clients only; the CLI has no AI commands.
	MethodAIConfigGet    = "ai.config.get"
	MethodAIConfigSave   = "ai.config.save"
	MethodAIConfigDelete = "ai.config.delete"
	MethodAIConfigActivate = "ai.config.activate"
	MethodAIChat         = "ai.chat"
	MethodAIAppChat      = "ai.appChat"
	MethodAITest         = "ai.test"
	MethodAIApplyPatch   = "ai.applyPatch"

	// Filesystem helpers.
	MethodFsValidatePath = "fs.validatePath"
	MethodFsReadAppFile  = "fs.readAppFile"

	// Opening a URL in the system browser.
	MethodOpenExternal = "openExternal"

	// Blueprints.
	MethodBlueprintsList      = "blueprints.list"
	MethodBlueprintsCreate    = "blueprints.create"
	MethodBlueprintsUpdate    = "blueprints.update"
	MethodBlueprintsDelete    = "blueprints.delete"
	MethodBlueprintsAIPropose = "blueprints.aiPropose"
	MethodBlueprintsCreateApp = "blueprints.createApp"
	MethodBlueprintsResolve   = "blueprints.resolve"
	MethodBlueprintsSendInput = "blueprints.sendInput"
	MethodBlueprintsResize    = "blueprints.resize"
	MethodBlueprintsCancel    = "blueprints.cancel"
)

// Method describes one method: its group, and who may call it.
type Method struct {
	Name string
	// CLI is true when the method is available to the CLI. AI methods are
	// GUI-only, as is the raw blueprint terminal.
	CLI bool
}

// Methods lists every server method. The CLI builds its commands from this
// registry, so help output and what the server accepts cannot drift apart.
var Methods = []Method{
	{Name: MethodHello, CLI: true},
	{Name: MethodStop, CLI: true},
	{Name: MethodSubscribe, CLI: true},

	{Name: MethodWorkspacesList, CLI: true},
	{Name: MethodWorkspacesCreate, CLI: true},
	{Name: MethodWorkspacesUpdate, CLI: true},
	{Name: MethodWorkspacesDelete, CLI: true},

	{Name: MethodAppsList, CLI: true},
	{Name: MethodAppsGet, CLI: true},
	{Name: MethodAppsCreate, CLI: true},
	{Name: MethodAppsUpdate, CLI: true},
	{Name: MethodAppsReorder, CLI: true},
	{Name: MethodAppsDelete, CLI: true},
	{Name: MethodAppsOpenFolder, CLI: true},
	{Name: MethodAppsEditors, CLI: true},
	{Name: MethodAppsOpenInEditor, CLI: true},

	{Name: MethodConfigSetsList, CLI: true},
	{Name: MethodConfigSetsDetail, CLI: true},
	{Name: MethodConfigSetsCreate, CLI: true},
	{Name: MethodConfigSetsUpdate, CLI: true},
	{Name: MethodConfigSetsDelete, CLI: true},
	{Name: MethodConfigSetsActivate, CLI: true},
	{Name: MethodConfigSetsCopyFrom, CLI: true},

	{Name: MethodEnvVarsList, CLI: true},
	{Name: MethodEnvVarsCreate, CLI: true},
	{Name: MethodEnvVarsUpdate, CLI: true},
	{Name: MethodEnvVarsDelete, CLI: true},
	{Name: MethodEnvVarsImport, CLI: true},

	{Name: MethodTemplatesList, CLI: true},
	{Name: MethodTemplatesCreate, CLI: true},
	{Name: MethodTemplatesUpdate, CLI: true},
	{Name: MethodTemplatesDelete, CLI: true},

	{Name: MethodPackageScriptsList, CLI: true},

	{Name: MethodRunConfigGet, CLI: true},
	{Name: MethodRunConfigSave, CLI: true},

	{Name: MethodRunnerStatus, CLI: true},
	{Name: MethodRunnerWorkspaceStatus, CLI: true},
	{Name: MethodRunnerOutput, CLI: true},
	{Name: MethodRunnerResize, CLI: false},
	{Name: MethodRunnerRun, CLI: true},
	{Name: MethodRunnerBuild, CLI: true},
	{Name: MethodRunnerSetup, CLI: true},
	{Name: MethodRunnerStop, CLI: true},
	{Name: MethodRunnerReload, CLI: true},

	{Name: MethodGitInfo, CLI: true},
	{Name: MethodGitFetchAll, CLI: true},
	{Name: MethodGitBranchCreate, CLI: true},
	{Name: MethodGitBranchRename, CLI: true},
	{Name: MethodGitBranchCheckout, CLI: true},
	{Name: MethodGitBranchPull, CLI: true},
	{Name: MethodGitBranchPush, CLI: true},
	{Name: MethodGitBranchDelete, CLI: true},
	{Name: MethodGitFileOpenInEditor, CLI: true},
	{Name: MethodGitWorktreeAdd, CLI: true},
	{Name: MethodGitWorktreeRemove, CLI: true},
	{Name: MethodGitWorktreePrune, CLI: true},
	{Name: MethodGitWorktreeStart, CLI: true},
	{Name: MethodGitWorktreeOpenFolder, CLI: true},
	{Name: MethodGitWorktreeOpenEditor, CLI: true},
	{Name: MethodGitWorktreeLinks, CLI: true},
	{Name: MethodWorktreeLinksGet, CLI: true},
	{Name: MethodWorktreeLinksSave, CLI: true},

	{Name: MethodPortsList, CLI: true},
	{Name: MethodPortsKill, CLI: true},

	{Name: MethodReadyUrlPatternsList, CLI: true},
	{Name: MethodReadyUrlPatternsCreate, CLI: true},
	{Name: MethodReadyUrlPatternsUpdate, CLI: true},
	{Name: MethodReadyUrlPatternsDelete, CLI: true},

	{Name: MethodSettingsGet, CLI: true},
	{Name: MethodSettingsSet, CLI: true},

	{Name: MethodAIConfigGet, CLI: false},
	{Name: MethodAIConfigSave, CLI: false},
	{Name: MethodAIConfigDelete, CLI: false},
	{Name: MethodAIConfigActivate, CLI: false},
	{Name: MethodAIChat, CLI: false},
	{Name: MethodAIAppChat, CLI: false},
	{Name: MethodAITest, CLI: false},
	{Name: MethodAIApplyPatch, CLI: false},

	{Name: MethodFsValidatePath, CLI: true},
	{Name: MethodFsReadAppFile, CLI: true},
	{Name: MethodOpenExternal, CLI: true},

	{Name: MethodBlueprintsList, CLI: true},
	{Name: MethodBlueprintsCreate, CLI: true},
	{Name: MethodBlueprintsUpdate, CLI: true},
	{Name: MethodBlueprintsDelete, CLI: true},
	{Name: MethodBlueprintsAIPropose, CLI: false},
	{Name: MethodBlueprintsCreateApp, CLI: true},
	{Name: MethodBlueprintsResolve, CLI: true},
	{Name: MethodBlueprintsSendInput, CLI: true},
	// Resizing is only useful to a terminal UI; the CLI never sends one.
	{Name: MethodBlueprintsResize, CLI: false},
	{Name: MethodBlueprintsCancel, CLI: true},
}

// CLIMethods returns the methods the CLI exposes, in registry order.
func CLIMethods() []Method {
	out := make([]Method, 0, len(Methods))
	for _, m := range Methods {
		if m.CLI {
			out = append(out, m)
		}
	}
	return out
}
