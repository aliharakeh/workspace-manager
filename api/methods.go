// Package api is the contract between the GUI and the MCP server: the request
// and response types, the method registry, and the error model. Both
// depend on it; the GUI does not import the MCP server, nor the other way round.
package api

// Method names are "group.operation". Each one matches a server handler.
const (
	// Server lifecycle and streams.
	// Workspaces.
	MethodWorkspacesList   = "workspaces.list"
	MethodWorkspacesCreate = "workspaces.create"
	MethodWorkspacesUpdate = "workspaces.update"
	MethodWorkspacesDelete = "workspaces.delete"

	// Apps.
	MethodAppsList         = "apps.list"
	MethodAppsGet          = "apps.get"
	MethodAppsCreate       = "apps.create"
	MethodAppsUpdate       = "apps.update"
	MethodAppsReorder      = "apps.reorder"
	MethodAppsDelete       = "apps.delete"
	MethodAppsOpenFolder   = "apps.openFolder"
	MethodAppsEditors      = "apps.editors"
	MethodAppsOpenInEditor = "apps.openInEditor"

	// Config sets.
	MethodConfigSetsList         = "configSets.list"
	MethodConfigSetsDetail       = "configSets.detail"
	MethodConfigSetsCreate       = "configSets.create"
	MethodConfigSetsUpdate       = "configSets.update"
	MethodConfigSetsDelete       = "configSets.delete"
	MethodConfigSetsActivate     = "configSets.activate"
	MethodConfigSetsCopyFrom     = "configSets.copyFrom"
	MethodConfigSetsApplyChanges = "configSets.applyChanges"

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
	MethodRunnerDev             = "runner.dev"
	MethodRunnerProd            = "runner.prod"
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
	MethodBlueprintsCreateApp = "blueprints.createApp"
	MethodBlueprintsPlan      = "blueprints.plan"
	MethodBlueprintsResolve   = "blueprints.resolve"
	MethodBlueprintsSendInput = "blueprints.sendInput"
	MethodBlueprintsResize    = "blueprints.resize"
	MethodBlueprintsCancel    = "blueprints.cancel"
)

// Method describes one method: its group, and who may call it.
type Method struct {
	Name string
	// Agent is true when an agent client (the MCP server) may call the method
	// as well as the GUI. git and applying AI changes are GUI-only, as is the runner
	// terminal resize.
	Agent bool
}

// Methods lists every server method. The MCP server builds its tools from this
// registry, so what it can call and what the server accepts cannot drift
// apart.
var Methods = []Method{

	{Name: MethodWorkspacesList, Agent: true},
	{Name: MethodWorkspacesCreate, Agent: true},
	{Name: MethodWorkspacesUpdate, Agent: true},
	{Name: MethodWorkspacesDelete, Agent: true},

	{Name: MethodAppsList, Agent: true},
	{Name: MethodAppsGet, Agent: true},
	{Name: MethodAppsCreate, Agent: true},
	{Name: MethodAppsUpdate, Agent: true},
	{Name: MethodAppsReorder, Agent: true},
	{Name: MethodAppsDelete, Agent: true},
	{Name: MethodAppsOpenFolder, Agent: false},
	{Name: MethodAppsEditors, Agent: false},
	{Name: MethodAppsOpenInEditor, Agent: false},

	{Name: MethodConfigSetsList, Agent: true},
	{Name: MethodConfigSetsDetail, Agent: true},
	{Name: MethodConfigSetsCreate, Agent: true},
	{Name: MethodConfigSetsUpdate, Agent: true},
	{Name: MethodConfigSetsDelete, Agent: true},
	{Name: MethodConfigSetsActivate, Agent: true},
	{Name: MethodConfigSetsCopyFrom, Agent: true},
	// Applies changes the GUI reviewed (the app AI's staged edits) in one call.
	{Name: MethodConfigSetsApplyChanges, Agent: false},

	{Name: MethodEnvVarsList, Agent: true},
	{Name: MethodEnvVarsCreate, Agent: true},
	{Name: MethodEnvVarsUpdate, Agent: true},
	{Name: MethodEnvVarsDelete, Agent: true},
	{Name: MethodEnvVarsImport, Agent: true},

	{Name: MethodTemplatesList, Agent: true},
	{Name: MethodTemplatesCreate, Agent: true},
	{Name: MethodTemplatesUpdate, Agent: true},
	{Name: MethodTemplatesDelete, Agent: true},

	{Name: MethodPackageScriptsList, Agent: true},

	{Name: MethodRunConfigGet, Agent: true},
	{Name: MethodRunConfigSave, Agent: true},

	{Name: MethodRunnerStatus, Agent: false},
	{Name: MethodRunnerWorkspaceStatus, Agent: false},
	{Name: MethodRunnerOutput, Agent: false},
	{Name: MethodRunnerResize, Agent: false},
	{Name: MethodRunnerRun, Agent: false},
	{Name: MethodRunnerDev, Agent: false},
	{Name: MethodRunnerProd, Agent: false},
	{Name: MethodRunnerBuild, Agent: false},
	{Name: MethodRunnerSetup, Agent: false},
	{Name: MethodRunnerStop, Agent: false},
	{Name: MethodRunnerReload, Agent: false},

	// Git is GUI-only: agents (the MCP server, the AI) never touch the repo.
	{Name: MethodGitInfo, Agent: false},
	{Name: MethodGitFetchAll, Agent: false},
	{Name: MethodGitBranchCreate, Agent: false},
	{Name: MethodGitBranchRename, Agent: false},
	{Name: MethodGitBranchCheckout, Agent: false},
	{Name: MethodGitBranchPull, Agent: false},
	{Name: MethodGitBranchPush, Agent: false},
	{Name: MethodGitBranchDelete, Agent: false},
	{Name: MethodGitFileOpenInEditor, Agent: false},
	{Name: MethodGitWorktreeAdd, Agent: false},
	{Name: MethodGitWorktreeRemove, Agent: false},
	{Name: MethodGitWorktreePrune, Agent: false},
	{Name: MethodGitWorktreeStart, Agent: false},
	{Name: MethodGitWorktreeOpenFolder, Agent: false},
	{Name: MethodGitWorktreeOpenEditor, Agent: false},
	{Name: MethodGitWorktreeLinks, Agent: false},
	{Name: MethodWorktreeLinksGet, Agent: false},
	{Name: MethodWorktreeLinksSave, Agent: false},

	{Name: MethodPortsList, Agent: true},
	{Name: MethodPortsKill, Agent: false},

	{Name: MethodReadyUrlPatternsList, Agent: true},
	{Name: MethodReadyUrlPatternsCreate, Agent: true},
	{Name: MethodReadyUrlPatternsUpdate, Agent: true},
	{Name: MethodReadyUrlPatternsDelete, Agent: true},

	{Name: MethodSettingsGet, Agent: true},
	{Name: MethodSettingsSet, Agent: true},

	{Name: MethodFsValidatePath, Agent: true},
	{Name: MethodFsReadAppFile, Agent: true},
	{Name: MethodOpenExternal, Agent: false},

	{Name: MethodBlueprintsList, Agent: true},
	{Name: MethodBlueprintsCreate, Agent: true},
	{Name: MethodBlueprintsUpdate, Agent: true},
	{Name: MethodBlueprintsDelete, Agent: true},
	{Name: MethodBlueprintsCreateApp, Agent: false},
	{Name: MethodBlueprintsPlan, Agent: true},
	{Name: MethodBlueprintsResolve, Agent: false},
	{Name: MethodBlueprintsSendInput, Agent: false},
	// Resizing is only useful to a terminal UI; an agent client never sends one.
	{Name: MethodBlueprintsResize, Agent: false},
	{Name: MethodBlueprintsCancel, Agent: false},
}
