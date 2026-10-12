package main

import (
	"encoding/json"

	"workspace-manager/api/types"
)

// runnerEvent is a runner payload pushed by the daemon:
// {"type":"runnerEvent","appId":N,"data":{...}} where data is a StatusEvent or
// a LogEvent. RawMessage keeps both decodable.
type runnerEvent struct {
	AppID int64           `json:"appId"`
	Data  json.RawMessage `json:"data"`
}

// parseRunnerStatus extracts a status event from a pushed runner payload.
func parseRunnerStatus(data []byte) (types.StatusEvent, bool) {
	var re runnerEvent
	if err := json.Unmarshal(data, &re); err != nil || len(re.Data) == 0 {
		return types.StatusEvent{}, false
	}
	var ev types.StatusEvent
	if err := json.Unmarshal(re.Data, &ev); err != nil {
		return types.StatusEvent{}, false
	}
	if ev.Type != "status" {
		return types.StatusEvent{}, false
	}
	if ev.AppID == 0 {
		ev.AppID = re.AppID
	}
	return ev, true
}

// parseRunnerLog extracts a log chunk from a pushed runner payload.
func parseRunnerLog(data []byte) (types.LogEvent, bool) {
	var re runnerEvent
	if err := json.Unmarshal(data, &re); err != nil || len(re.Data) == 0 {
		return types.LogEvent{}, false
	}
	var ev types.LogEvent
	if err := json.Unmarshal(re.Data, &ev); err != nil {
		return types.LogEvent{}, false
	}
	if ev.Type != "log" {
		return types.LogEvent{}, false
	}
	if ev.AppID == 0 {
		ev.AppID = re.AppID
	}
	return ev, true
}
