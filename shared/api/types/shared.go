package types

// SharedSet is a config set a workspace owns. App config sets link to one and
// take chosen variables from it as read-only, live copies.
type SharedSet struct {
	ID          int64  `json:"id"`
	WorkspaceID int64  `json:"workspace_id"`
	Name        string `json:"name"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// SharedUsage is one app config set that uses a shared variable.
type SharedUsage struct {
	AppID         int64  `json:"app_id"`
	AppName       string `json:"app_name"`
	ConfigSetID   int64  `json:"config_set_id"`
	ConfigSetName string `json:"config_set_name"`
}

// SharedEnvVar is a variable of a shared set. It carries the same value, hidden
// flag and options as an app's EnvVar, plus the app config sets that use it.
type SharedEnvVar struct {
	ID          int64          `json:"id"`
	SetID       int64          `json:"set_id"`
	Key         string         `json:"key"`
	Value       string         `json:"value"`
	Private bool           `json:"private"`
	Options     []EnvVarOption `json:"options"`
	Selected    string         `json:"selected"`
	UsedBy      []SharedUsage  `json:"used_by"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}

// SharedSetDetail is a shared set with its variables.
type SharedSetDetail struct {
	SharedSet
	Vars []SharedEnvVar `json:"vars"`
}

// SharedLinkInput links an app config set to a shared set and picks the
// variables it takes. Replace lets a chosen variable overwrite an own variable
// with the same key; without it such a clash is a conflict.
type SharedLinkInput struct {
	SharedSetID int64   `json:"shared_set_id"`
	VarIDs      []int64 `json:"var_ids"`
	Replace     bool    `json:"replace"`
}
