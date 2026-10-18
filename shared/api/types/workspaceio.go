package types

// Workspace export and import. A bundle is plain JSON describing a workspace
// and its apps, for moving to another machine or sharing with a teammate.

const (
	// WorkspaceBundleFormat identifies a workspace bundle file.
	WorkspaceBundleFormat = "workspace-manager/workspace"
	// WorkspaceBundleVersion is the bundle layout this build writes and the
	// newest it reads.
	WorkspaceBundleVersion = 1
)

// How secrets are handled on export. "Secret" means an env var or template
// whose include_in_ai is false.
const (
	// SecretsExclude drops secret env vars and templates altogether.
	SecretsExclude = "exclude"
	// SecretsPlaceholder keeps their keys and paths but replaces the values and
	// contents with a placeholder. This is the default.
	SecretsPlaceholder = "placeholder"
	// SecretsInclude writes the real values. Only the GUI may ask for it; an
	// agent client gets placeholder instead.
	SecretsInclude = "include"
)

// SecretPlaceholder replaces a secret value in a bundle. Import ignores the
// text and relies on the item's Redacted flag.
const SecretPlaceholder = "<redacted>"

type WorkspaceExportOptions struct {
	// Secrets is "exclude", "placeholder" (default) or "include".
	Secrets string `json:"secrets"`
}

type WorkspaceBundle struct {
	Format  string `json:"format"`
	Version int    `json:"version"`
	// ExportedAt is an RFC 3339 timestamp.
	ExportedAt string `json:"exported_at"`
	// Secrets is the mode the export actually used (an agent asking for
	// "include" is exported as "placeholder").
	Secrets string `json:"secrets"`
	// SourceRoot is the folder app paths are relative to, on the exporting
	// machine. Empty when the apps share no folder (their paths are absolute).
	SourceRoot string          `json:"source_root"`
	Workspace  BundleWorkspace `json:"workspace"`
	Apps       []BundleApp     `json:"apps"`
}

type BundleWorkspace struct {
	Name  string  `json:"name"`
	Icon  *string `json:"icon"`
	Color *string `json:"color"`
}

type BundleApp struct {
	// Key identifies the app inside the bundle (import folder overrides use it).
	Key  string `json:"key"`
	Name string `json:"name"`
	// Path is the app folder, relative to SourceRoot with forward slashes
	// ("." is the root itself), or absolute when Absolute is true.
	Path     string `json:"path"`
	Absolute bool   `json:"absolute,omitempty"`
	// ActiveConfigSet is the name of the active config set.
	ActiveConfigSet string            `json:"active_config_set"`
	ConfigSets      []BundleConfigSet `json:"config_sets"`
	// WorktreeLinks are the shared paths of the app's git worktrees.
	WorktreeLinks []string `json:"worktree_links"`
}

type BundleConfigSet struct {
	Name      string           `json:"name"`
	EnvVars   []BundleEnvVar   `json:"env_vars"`
	Templates []BundleTemplate `json:"templates"`
	// Commands holds the setup, dev, build and prod lists, by kind.
	Commands map[string]BundleCommandList `json:"commands"`
}

type BundleEnvVar struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	IncludeInAI bool   `json:"include_in_ai"`
	// Redacted marks a value replaced by a placeholder: import stores it empty.
	Redacted bool `json:"redacted,omitempty"`
}

type BundleTemplate struct {
	FilePath    string `json:"file_path"`
	Content     string `json:"content"`
	IncludeInAI bool   `json:"include_in_ai"`
	Redacted    bool   `json:"redacted,omitempty"`
}

type BundleCommandList struct {
	Mode     string          `json:"mode"`
	Commands []BundleCommand `json:"commands"`
}

type BundleCommand struct {
	Label   *string `json:"label"`
	Command string  `json:"command"`
}

type WorkspaceImportOptions struct {
	// Name overrides the workspace name in the bundle.
	Name string `json:"name"`
	// BaseFolder replaces the bundle's source root: relative app paths are
	// joined to it.
	BaseFolder string `json:"base_folder"`
	// Folders maps an app key to the absolute folder to use for it.
	Folders map[string]string `json:"folders"`
	// DryRun reports what would be created and writes nothing.
	DryRun bool `json:"dry_run"`
}

type WorkspaceImportApp struct {
	Key           string `json:"key"`
	Name          string `json:"name"`
	Folder        string `json:"folder"`
	FolderMissing bool   `json:"folder_missing"`
	ConfigSets    int    `json:"config_sets"`
	EnvVars       int    `json:"env_vars"`
	Templates     int    `json:"templates"`
	Commands      int    `json:"commands"`
	// EmptyEnvVars and EmptyTemplates count the redacted items that come in
	// empty and need filling.
	EmptyEnvVars   int `json:"empty_env_vars"`
	EmptyTemplates int `json:"empty_templates"`
}

type WorkspaceImportResult struct {
	// WorkspaceID is 0 for a dry run.
	WorkspaceID int64                `json:"workspace_id"`
	Name        string               `json:"name"`
	DryRun      bool                 `json:"dry_run"`
	Apps        []WorkspaceImportApp `json:"apps"`
	AppIDs      []int64              `json:"app_ids"`
}

// WorkspaceExportFileResult is what the GUI's "Export…" returns after the save
// dialog: where the bundle was written, or Cancelled.
type WorkspaceExportFileResult struct {
	Path      string `json:"path"`
	Cancelled bool   `json:"cancelled"`
	Apps      int    `json:"apps"`
	Secrets   string `json:"secrets"`
}

// WorkspaceBundleFileResult is what the GUI's open dialog returns for an import.
type WorkspaceBundleFileResult struct {
	Path      string           `json:"path"`
	Cancelled bool             `json:"cancelled"`
	Bundle    *WorkspaceBundle `json:"bundle"`
}
