package types

// Diagnostic is a detected problem, grouped by command, severity and message.
// Revision changes on repeats and context updates; it is the incremental cursor.
type Diagnostic struct {
	ID               int64  `json:"id"`
	CommandID        int64  `json:"commandId"`
	Severity         string `json:"severity"`
	Source           string `json:"source"`
	Message          string `json:"message"`
	File             string `json:"file,omitempty"`
	Line             int    `json:"line,omitempty"`
	Column           int    `json:"column,omitempty"`
	Count            int64  `json:"count"`
	FirstSeen        int64  `json:"firstSeen"`
	LastSeen         int64  `json:"lastSeen"`
	Revision         int64  `json:"revision"`
	Context          string `json:"context,omitempty"`
	ContextTruncated bool   `json:"contextTruncated"`
}

type DiagnosticsQuery struct {
	AppID     int64  `json:"appID"`
	SessionID string `json:"sessionId"`
	CommandID int64  `json:"commandId"`
	Severity  string `json:"severity"`
	Cursor    int64  `json:"cursor"`
	Limit     int    `json:"limit"`
	ID        int64  `json:"id"`
}

type DiagnosticsResult struct {
	AppID            int64        `json:"appId"`
	SessionID        string       `json:"sessionId"`
	ConfigSetID      int64        `json:"configSetId"`
	Kind             string       `json:"kind"`
	Items            []Diagnostic `json:"items"`
	ErrorGroups      int          `json:"errorGroups"`
	WarningGroups    int          `json:"warningGroups"`
	Occurrences      int64        `json:"occurrences"`
	NextCursor       int64        `json:"nextCursor"`
	HasMore          bool         `json:"hasMore"`
	Dropped          int64        `json:"dropped"`
	InputTruncated   bool         `json:"inputTruncated"`
	PersistenceError string       `json:"persistenceError,omitempty"`
	Detection        string       `json:"detection"`
}
