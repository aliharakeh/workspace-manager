// Package api is the contract between the daemon and its clients: the method
// registry, event types, the handshake types and the error model. The request
// and response payloads live in the types package.
package api

// Method names are "group.operation". Each one matches a server handler.
const (
	// Server lifecycle and streams.
	MethodHello     = "hello"
	MethodStop      = "server.stop"
	MethodSubscribe = "events.subscribe"

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
	MethodTemplatesRender = "templates.render"

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

	// Worktree support for the GUI's git code (git runs in the GUI process).
	MethodWorktreeLinksGet  = "worktreeLinks.get"
	MethodWorktreeLinksSave = "worktreeLinks.save"
	MethodRunnerStartIn     = "runner.startIn"

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
	MethodFsReadAppFile  = "fs.readAppFile"


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

// Method describes one method: its group, and who may call it.
type Method struct {
	Name string
	// Agent is true when an agent client may call the method as well as the GUI.
	// Settings, worktree support, applying AI changes and terminal resizes are
	// GUI-only.
	Agent bool
}

// Methods lists every daemon method. The daemon refuses to start when a method
// has no handler or a handler has no method, so the two cannot drift apart.
var Methods = []Method{
	{Name: MethodHello, Agent: true},
	{Name: MethodStop, Agent: true},
	{Name: MethodSubscribe, Agent: true},

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
	{Name: MethodAppsOpenFolder, Agent: true},
	{Name: MethodAppsEditors, Agent: true},
	{Name: MethodAppsOpenInEditor, Agent: true},

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
	// Rendering shows hidden env values, so it is for the GUI and editor clients.
	{Name: MethodTemplatesRender, Agent: false},

	{Name: MethodPackageScriptsList, Agent: true},

	{Name: MethodRunConfigGet, Agent: true},
	{Name: MethodRunConfigSave, Agent: true},

	{Name: MethodRunnerStatus, Agent: true},
	{Name: MethodRunnerWorkspaceStatus, Agent: true},
	{Name: MethodRunnerOutput, Agent: true},
	{Name: MethodRunnerResize, Agent: false},
	{Name: MethodRunnerRun, Agent: true},
	{Name: MethodRunnerDev, Agent: true},
	{Name: MethodRunnerProd, Agent: true},
	{Name: MethodRunnerBuild, Agent: true},
	{Name: MethodRunnerSetup, Agent: true},
	{Name: MethodRunnerStop, Agent: true},
	{Name: MethodRunnerReload, Agent: true},

	// Git runs in the GUI process; these serve its worktree support.
	{Name: MethodWorktreeLinksGet, Agent: false},
	{Name: MethodWorktreeLinksSave, Agent: false},
	{Name: MethodRunnerStartIn, Agent: false},

	{Name: MethodPortsList, Agent: true},
	{Name: MethodPortsKill, Agent: true},

	{Name: MethodReadyUrlPatternsList, Agent: true},
	{Name: MethodReadyUrlPatternsCreate, Agent: true},
	{Name: MethodReadyUrlPatternsUpdate, Agent: true},
	{Name: MethodReadyUrlPatternsDelete, Agent: true},

	{Name: MethodSettingsGet, Agent: false},
	{Name: MethodSettingsSet, Agent: false},

	{Name: MethodFsReadAppFile, Agent: true},

	{Name: MethodBlueprintsList, Agent: true},
	{Name: MethodBlueprintsCreate, Agent: true},
	{Name: MethodBlueprintsUpdate, Agent: true},
	{Name: MethodBlueprintsDelete, Agent: true},
	{Name: MethodBlueprintsCreateApp, Agent: true},
	{Name: MethodBlueprintsResolve, Agent: true},
	{Name: MethodBlueprintsSendInput, Agent: true},
	// Resizing is only useful to a terminal UI; an agent client never sends one.
	{Name: MethodBlueprintsResize, Agent: false},
	{Name: MethodBlueprintsCancel, Agent: true},
}

// AgentMethods returns the methods an agent client may call, in registry order.
func AgentMethods() []Method {
	out := make([]Method, 0, len(Methods))
	for _, m := range Methods {
		if m.Agent {
			out = append(out, m)
		}
	}
	return out
}
