package types

type Workspace struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Icon      *string `json:"icon"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type App struct {
	ID                  int64   `json:"id"`
	WorkspaceID         int64   `json:"workspace_id"`
	Name                string  `json:"name"`
	ProjectPath         string  `json:"project_path"`
	ActiveConfigSetID   *int64  `json:"active_config_set_id"`
	ActiveConfigSetName *string `json:"active_config_set_name"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
}

type ConfigSet struct {
	ID        int64  `json:"id"`
	AppID     int64  `json:"app_id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type EnvVar struct {
	ID          int64  `json:"id"`
	ConfigSetID int64  `json:"config_set_id"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type Template struct {
	ID          int64  `json:"id"`
	ConfigSetID int64  `json:"config_set_id"`
	FilePath    string `json:"file_path"`
	Content     string `json:"content"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type RunCommand struct {
	ID          int64   `json:"id"`
	RunConfigID int64   `json:"run_config_id"`
	Label       *string `json:"label"`
	Command     string  `json:"command"`
	SortOrder   int64   `json:"sort_order"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type RunConfig struct {
	ID          int64        `json:"id"`
	ConfigSetID int64        `json:"config_set_id"`
	Mode        string       `json:"mode"`
	CreatedAt   string       `json:"created_at"`
	UpdatedAt   string       `json:"updated_at"`
	Commands    []RunCommand `json:"commands"`
}

type ConfigSetDetail struct {
	ConfigSet
	EnvVars   []EnvVar   `json:"env_vars"`
	Templates []Template `json:"templates"`
	RunConfig *RunConfig `json:"run_config"`
}

type CopyParts struct {
	Env       any `json:"env,omitempty"`
	Templates any `json:"templates,omitempty"`
	Run       any `json:"run,omitempty"`
}

type ReadyUrlPattern struct {
	ID        int64   `json:"id"`
	Key       *string `json:"key"`
	Label     string  `json:"label"`
	Pattern   string  `json:"pattern"`
	Flags     string  `json:"flags"`
	SortOrder int64   `json:"sort_order"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type ProcessState struct {
	CommandID int64    `json:"commandId"`
	Label     string   `json:"label"`
	Command   string   `json:"command"`
	Status    string   `json:"status"`
	ExitCode  *int64   `json:"exitCode"`
	PID       *int64   `json:"pid"`
	URLs      []string `json:"urls"`
}

type StatusEvent struct {
	Type      string         `json:"type"`
	SessionID string         `json:"sessionId"`
	AppID     int64          `json:"appId"`
	Running   bool           `json:"running"`
	Processes []ProcessState `json:"processes"`
	Error     string         `json:"error,omitempty"`
	Ts        int64          `json:"ts"`
}

type LogEvent struct {
	Type      string `json:"type"`
	AppID     int64  `json:"appId,omitempty"`
	CommandID int64  `json:"commandId"`
	Stream    string `json:"stream"`
	Text      string `json:"text"`
	Ts        int64  `json:"ts"`
}

type RunnerLogsSnapshot struct {
	Status StatusEvent `json:"status"`
	Logs   []LogEvent  `json:"logs"`
}

type ListeningProcess struct {
	Port int64  `json:"port"`
	PID  int64  `json:"pid"`
	Name string `json:"name"`
}

type Ok struct {
	Ok bool `json:"ok"`
}

type PortsListResult struct {
	Min       int64              `json:"min"`
	Max       int64              `json:"max"`
	Processes []ListeningProcess `json:"processes"`
}

type PortsKillResult struct {
	Ok  bool  `json:"ok"`
	PID int64 `json:"pid"`
}

type ValidatePathResult struct {
	Ok    bool   `json:"ok"`
	Path  string `json:"path,omitempty"`
	Error string `json:"error,omitempty"`
}

type PickFolderResult struct {
	Cancelled bool   `json:"cancelled"`
	Path      string `json:"path,omitempty"`
}

type PickFileResult struct {
	Cancelled    bool   `json:"cancelled"`
	Path         string `json:"path,omitempty"`
	RelativePath string `json:"relative_path,omitempty"`
	Content      string `json:"content,omitempty"`
}

type ReadAppFileResult struct {
	Ok           bool   `json:"ok"`
	Content      string `json:"content"`
	RelativePath string `json:"relative_path"`
}

type ImportTemplateResult struct {
	ID       int64  `json:"id"`
	FilePath string `json:"file_path"`
	Created  bool   `json:"created"`
}

type ImportEnvResult struct {
	Cancelled bool                  `json:"cancelled"`
	Path      string                `json:"path,omitempty"`
	Format    string                `json:"format,omitempty"`
	Imported  int                   `json:"imported,omitempty"`
	Vars      []EnvVar              `json:"vars,omitempty"`
	Template  *ImportTemplateResult `json:"template"`
}

type RunCommandInput struct {
	Label   *string `json:"label"`
	Command string  `json:"command"`
}

type WorkspaceCreateInput struct {
	Name string  `json:"name"`
	Icon *string `json:"icon"`
}

type WorkspaceUpdateInput struct {
	Name *string `json:"name"`
	Icon *string `json:"icon"`
}

type AppCreateInput struct {
	Name        string `json:"name"`
	ProjectPath string `json:"project_path"`
}

type AppUpdateInput struct {
	Name        *string `json:"name"`
	ProjectPath *string `json:"project_path"`
}

type ConfigSetUpdateInput struct {
	Name string `json:"name"`
}

type ConfigSetCreateInput struct {
	Name       string     `json:"name"`
	CopyFromID *int64     `json:"copy_from_id"`
	Activate   *bool      `json:"activate"`
	Parts      *CopyParts `json:"parts"`
}

type ConfigSetActivateResult struct {
	ID    int64  `json:"id"`
	AppID int64  `json:"app_id"`
	Name  string `json:"name"`
	App   App    `json:"app"`
}

type EnvVarCreateInput struct {
	Key   string  `json:"key"`
	Value *string `json:"value"`
}

type EnvVarUpdateInput struct {
	Key   *string `json:"key"`
	Value *string `json:"value"`
}

type TemplateCreateInput struct {
	FilePath string  `json:"file_path"`
	Content  *string `json:"content"`
}

type TemplateUpdateInput struct {
	FilePath *string `json:"file_path"`
	Content  *string `json:"content"`
}

type RunConfigSaveInput struct {
	Mode     *string           `json:"mode"`
	Commands []RunCommandInput `json:"commands"`
}

type ReadyUrlPatternCreateInput struct {
	Label   string  `json:"label"`
	Pattern string  `json:"pattern"`
	Flags   *string `json:"flags"`
}

type ReadyUrlPatternUpdateInput struct {
	Label   *string `json:"label"`
	Pattern *string `json:"pattern"`
	Flags   *string `json:"flags"`
}

type SettingsSetInput struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type FsPickFolderInput struct {
	StartDir *string `json:"startDir"`
}

type FsPickFileInput struct {
	StartDir *string `json:"startDir"`
	AppID    *int64  `json:"appId"`
}

// AIProviderConfigInput is the save payload for one AI connection. The API key
// is write-only: empty keeps the stored key, ClearAPIKey removes it. It is
// never persisted or returned to the UI. Name is the user-chosen connection
// name; one provider may have several connections under different names.
type AIProviderConfigInput struct {
	Name        string   `json:"name"`
	Provider    string   `json:"provider"`
	BaseURL     string   `json:"baseURL,omitempty"`
	APIKey      string   `json:"apiKey,omitempty"`
	Model       string   `json:"model,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
	ClearAPIKey bool     `json:"clearApiKey,omitempty"`
}

// AIConnectionInfo is one saved connection as the frontend sees it: no secret,
// plus a flag for whether a key is stored.
type AIConnectionInfo struct {
	Name        string   `json:"name"`
	Provider    string   `json:"provider"`
	BaseURL     string   `json:"baseURL,omitempty"`
	Model       string   `json:"model,omitempty"`
	HasAPIKey   bool     `json:"hasApiKey"`
	Temperature *float64 `json:"temperature,omitempty"`
}

// AIConfigInfo is the full config payload: every saved connection plus the
// active (default) connection name.
type AIConfigInfo struct {
	Providers []AIConnectionInfo `json:"providers"`
	Active    string             `json:"active"`
}

type AIActivateInput struct {
	Name string `json:"name"`
}

type AIChatInput struct {
	System string `json:"system,omitempty"`
	Prompt string `json:"prompt"`
}

// AITestInput carries an unsaved connection payload for a test request.
type AITestInput struct {
	Provider    string   `json:"provider"`
	BaseURL     string   `json:"baseURL,omitempty"`
	APIKey      string   `json:"apiKey,omitempty"`
	Model       string   `json:"model,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
}

type AIChatResult struct {
	Text string `json:"text"`
}

type AppAIChatTurn struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type AppAIEnvUpsert struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type AppAIEnvPatch struct {
	Upsert []AppAIEnvUpsert `json:"upsert,omitempty"`
	Delete []string         `json:"delete,omitempty"`
}

type AppAITemplatePatch struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

type AppAIRunCommand struct {
	Label   *string `json:"label"`
	Command string  `json:"command"`
}

type AppAIRunPatch struct {
	Mode     *string           `json:"mode,omitempty"`
	Commands []AppAIRunCommand `json:"commands,omitempty"`
}

type AppAIPatch struct {
	Message   string               `json:"message"`
	Env       *AppAIEnvPatch       `json:"env,omitempty"`
	Templates []AppAITemplatePatch `json:"templates,omitempty"`
	Run       *AppAIRunPatch       `json:"run,omitempty"`
}

type AppAIChatInput struct {
	AppID       int64           `json:"appId"`
	ConfigSetID int64           `json:"configSetId"`
	History     []AppAIChatTurn `json:"history"`
	Instruction string          `json:"instruction"`
}

type AppAIToolCall struct {
	Name   string `json:"name"`
	Input  any    `json:"input"`
	Output any    `json:"output"`
}

type AppAIChatResult struct {
	Text      string          `json:"text"`
	Patch     AppAIPatch      `json:"patch"`
	ToolCalls []AppAIToolCall `json:"toolCalls,omitempty"`
}

type AppAIStreamEvent struct {
	Type  string         `json:"type"`
	Text  string         `json:"text,omitempty"`
	Call  *AppAIToolCall `json:"call,omitempty"`
	Error string         `json:"error,omitempty"`
}

// AITestResult echoes whether the connection worked plus the model's reply.
type AITestResult struct {
	Ok   bool   `json:"ok"`
	Text string `json:"text"`
}
