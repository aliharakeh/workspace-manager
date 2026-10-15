// Package api is the contract between the frontend bindings and the Go side:
// the request and response types, the method registry, and the error model.
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
	MethodBlueprintsResolve   = "blueprints.resolve"
	MethodBlueprintsSendInput = "blueprints.sendInput"
	MethodBlueprintsResize    = "blueprints.resize"
	MethodBlueprintsCancel    = "blueprints.cancel"
)

// Method describes one API method.
type Method struct {
	Name string
}

// Methods lists every API method. The handler table is checked against it at
// startup, so a method without a handler, or a handler without a method,
// cannot drift.
var Methods = []Method{

	{Name: MethodWorkspacesList},
	{Name: MethodWorkspacesCreate},
	{Name: MethodWorkspacesUpdate},
	{Name: MethodWorkspacesDelete},

	{Name: MethodAppsList},
	{Name: MethodAppsGet},
	{Name: MethodAppsCreate},
	{Name: MethodAppsUpdate},
	{Name: MethodAppsReorder},
	{Name: MethodAppsDelete},
	{Name: MethodAppsOpenFolder},
	{Name: MethodAppsEditors},
	{Name: MethodAppsOpenInEditor},

	{Name: MethodConfigSetsList},
	{Name: MethodConfigSetsDetail},
	{Name: MethodConfigSetsCreate},
	{Name: MethodConfigSetsUpdate},
	{Name: MethodConfigSetsDelete},
	{Name: MethodConfigSetsActivate},
	{Name: MethodConfigSetsCopyFrom},
	{Name: MethodConfigSetsApplyChanges},

	{Name: MethodEnvVarsList},
	{Name: MethodEnvVarsCreate},
	{Name: MethodEnvVarsUpdate},
	{Name: MethodEnvVarsDelete},
	{Name: MethodEnvVarsImport},

	{Name: MethodTemplatesList},
	{Name: MethodTemplatesCreate},
	{Name: MethodTemplatesUpdate},
	{Name: MethodTemplatesDelete},

	{Name: MethodPackageScriptsList},

	{Name: MethodRunConfigGet},
	{Name: MethodRunConfigSave},

	{Name: MethodRunnerStatus},
	{Name: MethodRunnerWorkspaceStatus},
	{Name: MethodRunnerOutput},
	{Name: MethodRunnerResize},
	{Name: MethodRunnerRun},
	{Name: MethodRunnerDev},
	{Name: MethodRunnerProd},
	{Name: MethodRunnerBuild},
	{Name: MethodRunnerSetup},
	{Name: MethodRunnerStop},
	{Name: MethodRunnerReload},

	{Name: MethodGitInfo},
	{Name: MethodGitFetchAll},
	{Name: MethodGitBranchCreate},
	{Name: MethodGitBranchRename},
	{Name: MethodGitBranchCheckout},
	{Name: MethodGitBranchPull},
	{Name: MethodGitBranchPush},
	{Name: MethodGitBranchDelete},
	{Name: MethodGitFileOpenInEditor},
	{Name: MethodGitWorktreeAdd},
	{Name: MethodGitWorktreeRemove},
	{Name: MethodGitWorktreePrune},
	{Name: MethodGitWorktreeStart},
	{Name: MethodGitWorktreeOpenFolder},
	{Name: MethodGitWorktreeOpenEditor},
	{Name: MethodGitWorktreeLinks},
	{Name: MethodWorktreeLinksGet},
	{Name: MethodWorktreeLinksSave},

	{Name: MethodPortsList},
	{Name: MethodPortsKill},

	{Name: MethodReadyUrlPatternsList},
	{Name: MethodReadyUrlPatternsCreate},
	{Name: MethodReadyUrlPatternsUpdate},
	{Name: MethodReadyUrlPatternsDelete},

	{Name: MethodSettingsGet},
	{Name: MethodSettingsSet},

	{Name: MethodFsValidatePath},
	{Name: MethodFsReadAppFile},
	{Name: MethodOpenExternal},

	{Name: MethodBlueprintsList},
	{Name: MethodBlueprintsCreate},
	{Name: MethodBlueprintsUpdate},
	{Name: MethodBlueprintsDelete},
	{Name: MethodBlueprintsCreateApp},
	{Name: MethodBlueprintsResolve},
	{Name: MethodBlueprintsSendInput},
	// Resizing is only useful to a terminal UI.
	{Name: MethodBlueprintsResize},
	{Name: MethodBlueprintsCancel},
}
