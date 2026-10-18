package types

// Outcomes of one app in a workspace-wide start or stop.
const (
	WorkspaceRunStarted  = "started"
	WorkspaceRunStopped  = "stopped"
	WorkspaceRunSkipped  = "skipped"
	WorkspaceRunExcluded = "excluded"
	WorkspaceRunError    = "error"
)

// WorkspaceRunAppResult is what a workspace-wide start or stop did to one app.
// Status is one of the WorkspaceRun* constants; Message explains a skip, an
// exclusion or an error.
type WorkspaceRunAppResult struct {
	AppID   int64  `json:"appId"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// WorkspaceRunResult is the per-app summary of runner.startWorkspace and
// runner.stopWorkspace, in the workspace's app order.
type WorkspaceRunResult struct {
	WorkspaceID int64                   `json:"workspaceId"`
	Results     []WorkspaceRunAppResult `json:"results"`
}
