package api

// Event types pushed to subscribers.
type EventType string

const (
	EventRunner      EventType = "runnerEvent"
	EventBlueprint   EventType = "blueprintEvent"
	EventAppAI       EventType = "appAIEvent"
	EventDataChanged EventType = "dataChanged"
)

// Event is a pushed notification. Data holds the payload clients already know
// today, so the GUI's event payloads stay unchanged.
type Event struct {
	Type EventType `json:"type"`
	App  int64     `json:"appId,omitempty"`
	Data  any      `json:"data,omitempty"`
}

// SubscribeInput filters pushed events. With no types, a client gets all of
// them; with no apps, events for every app.
type SubscribeInput struct {
	Types []EventType `json:"types,omitempty"`
	Apps  []int64     `json:"apps,omitempty"`
}

// Matches reports whether an event passes the filter.
func (f SubscribeInput) Matches(ev Event) bool {
	if len(f.Types) > 0 {
		ok := false
		for _, t := range f.Types {
			if t == ev.Type {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if len(f.Apps) > 0 && ev.App != 0 {
		ok := false
		for _, id := range f.Apps {
			if id == ev.App {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}
