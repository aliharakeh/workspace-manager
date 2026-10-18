package types

// Statuses of an entry in a ConfigSetCompare. Command entries use "changed"
// where env vars and templates use "different".
const (
	DiffOnlyLeft  = "only_left"
	DiffOnlyRight = "only_right"
	DiffSame      = "same"
	DiffDifferent = "different"
	DiffChanged   = "changed"
)

// ConfigSetEnvDiff is one env var key across two config sets. Redacted entries
// (include_in_ai=false, seen by an agent) carry the status only: Left and
// Right are nil.
type ConfigSetEnvDiff struct {
	Key      string  `json:"key"`
	Status   string  `json:"status"`
	Left     *string `json:"left"`
	Right    *string `json:"right"`
	Redacted bool    `json:"redacted"`
}

// ConfigSetTemplateDiff is one template (matched on file_path). Left/Right are
// nil on a side that lacks the template, and on both when Redacted.
type ConfigSetTemplateDiff struct {
	FilePath string  `json:"file_path"`
	Status   string  `json:"status"`
	Left     *string `json:"left"`
	Right    *string `json:"right"`
	Redacted bool    `json:"redacted"`
}

// ConfigSetCommandDiff is one command of a list. Commands match by label when
// they have one (the n-th occurrence of a label with the n-th on the other
// side), otherwise by position among the unlabeled commands.
type ConfigSetCommandDiff struct {
	Label  string  `json:"label"`
	Status string  `json:"status"`
	Left   *string `json:"left"`
	Right  *string `json:"right"`
}

// ConfigSetCommandListDiff compares the command list of one kind.
type ConfigSetCommandListDiff struct {
	Kind      string                 `json:"kind"`
	LeftMode  string                 `json:"left_mode"`
	RightMode string                 `json:"right_mode"`
	Status    string                 `json:"status"` // same or different
	Commands  []ConfigSetCommandDiff `json:"commands"`
}

// ConfigSetCompare is the result of configSets.compare. Both sets must belong
// to the same app (as copyFrom requires), so "copy this part from the other
// set" is always available.
type ConfigSetCompare struct {
	Left      ConfigSet                  `json:"left"`
	Right     ConfigSet                  `json:"right"`
	Env       []ConfigSetEnvDiff         `json:"env"`
	Templates []ConfigSetTemplateDiff    `json:"templates"`
	Commands  []ConfigSetCommandListDiff `json:"commands"`
}
