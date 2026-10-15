package api

// Event types pushed to subscribers.
type EventType string

const (
	EventRunner      EventType = "runnerEvent"
	EventBlueprint   EventType = "blueprintEvent"
	EventDataChanged EventType = "dataChanged"
)

// Event is a pushed notification. Data holds the payload clients already know
// today, so the GUI's event payloads stay unchanged.
type Event struct {
	Type EventType `json:"type"`
	App  int64     `json:"appId,omitempty"`
	Data any       `json:"data,omitempty"`
}

// DataChanged is the payload of EventDataChanged: the entity a write touched
// and the rows involved, so a client knows what to reload. For an app's own
// data the ids are app ids (workspace ids for a workspace); an empty list means
// the write is not tied to one row, so reload the entity.
type DataChanged struct {
	Entity string  `json:"entity"`
	IDs    []int64 `json:"ids"`
}
