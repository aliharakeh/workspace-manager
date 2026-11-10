package mcp

import (
	"context"
	"encoding/json"
	"time"

	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"
)

type waitOptions struct {
	SessionID      string  `json:"sessionId"`
	CommandIDs     []int64 `json:"commandIds"`
	Until          string  `json:"until"`
	Readiness      string  `json:"readiness"`
	TimeoutSeconds int     `json:"timeoutSeconds"`
}

type waitURL struct {
	CommandID int64  `json:"commandId"`
	Label     string `json:"label"`
	URL       string `json:"url"`
}

type waitResult struct {
	Reason string            `json:"reason"`
	URLs   []waitURL         `json:"urls"`
	Status types.StatusEvent `json:"status"`
}

func waitTool() mcpTool {
	return mcpTool{
		Name: "wm_wait", Structured: true,
		Description: "Wait for the current run's URLs or process completion without reading logs. Tracks all app commands by default, or selected commandIds. Default readiness=all waits for every selected command to have a URL or exit successfully, with at least one live URL; readiness=any returns on the first live URL. Any selected process failure returns failed. until=exit waits for completion. Returns attributed URLs and the full runner status. Idle output alone is not URL readiness. Does not start, stop or open the app.",
		Schema: schema("Wait for a run", map[string]any{
			"app":            strProp(appRefDesc),
			"sessionId":      strProp("sessionId returned by wm_run; omit to bind to the current run. A replaced session is refused."),
			"commandIds":     map[string]any{"type": "array", "minItems": 1, "uniqueItems": true, "items": map[string]any{"type": "integer", "minimum": 1}, "description": "Wait for these command processes only; omit for every command of the app. Other processes remain running."},
			"until":          map[string]any{"type": "string", "enum": []string{"ready", "exit"}, "description": "ready (default): wait for URLs or exit; exit: wait for selected commands to finish"},
			"readiness":      map[string]any{"type": "string", "enum": []string{"all", "any"}, "description": "all (default): every selected command must be ready or successfully finished; any: one live URL suffices. Select only web commands when workers have no URLs."},
			"timeoutSeconds": map[string]any{"type": "integer", "minimum": 1, "maximum": 60, "description": "Maximum wait, default 30 seconds, max 60. Timeout returns status and never stops processes; call again with the same sessionId."},
		}),
		Handler: func(args map[string]any) (any, error) {
			var options waitOptions
			raw, err := json.Marshal(args)
			if err != nil {
				return nil, err
			}
			if err = json.Unmarshal(raw, &options); err != nil {
				return nil, api.Errf(api.KindInvalid, "invalid wait parameters: %v", err)
			}
			if err = options.validate(); err != nil {
				return nil, err
			}
			app, err := resolveApp(argStr(args, "app"))
			if err != nil {
				return nil, err
			}
			conn, err := dial()
			if err != nil {
				return nil, err
			}
			defer conn.Close()
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(options.TimeoutSeconds)*time.Second)
			defer cancel()
			return waitForRun(ctx, conn, appIDOf(app), options)
		},
	}
}

func (o *waitOptions) validate() error {
	if o.Until == "" {
		o.Until = "ready"
	}
	if o.Readiness == "" {
		o.Readiness = "all"
	}
	if o.TimeoutSeconds == 0 {
		o.TimeoutSeconds = 30
	}
	if o.Until != "ready" && o.Until != "exit" {
		return api.Errf(api.KindInvalid, "until must be ready or exit")
	}
	if o.Readiness != "all" && o.Readiness != "any" {
		return api.Errf(api.KindInvalid, "readiness must be all or any")
	}
	if o.TimeoutSeconds < 1 || o.TimeoutSeconds > 60 {
		return api.Errf(api.KindInvalid, "timeoutSeconds must be 1..60")
	}
	if o.CommandIDs != nil && len(o.CommandIDs) == 0 {
		return api.Errf(api.KindInvalid, "commandIds must not be empty")
	}
	seen := map[int64]bool{}
	for _, id := range o.CommandIDs {
		if id <= 0 || seen[id] {
			return api.Errf(api.KindInvalid, "commandIds must be unique positive ids")
		}
		seen[id] = true
	}
	return nil
}

type waitConnection interface {
	Call(string, any, any) error
	Subscribe(api.SubscribeInput, func(api.Event)) error
	Done() <-chan struct{}
	Close() error
}

func waitForRun(ctx context.Context, conn waitConnection, appID int64, options waitOptions) (waitResult, error) {
	// Closing the dedicated connection also interrupts a stalled status RPC.
	stopClose := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stopClose()
	wake := make(chan struct{}, 1)
	// Subscribe before reading status so a fast exit/readiness cannot fall in a gap.
	err := conn.Subscribe(api.SubscribeInput{Types: []api.EventType{api.EventRunner}, Apps: []int64{appID}}, func(ev api.Event) {
		if ev.Type != api.EventRunner || ev.App != appID {
			return
		}
		payload, ok := ev.Data.(map[string]any)
		if !ok || payload["type"] != "status" {
			return
		}
		select {
		case wake <- struct{}{}:
		default:
		}
	})
	if err != nil {
		return waitResult{}, err
	}
	// Re-read current status on notifications rather than trusting queued, possibly
	// stale status events. A slow fallback covers dropped subscription events.
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	var last waitResult
	for {
		var status types.StatusEvent
		if err = conn.Call(api.MethodRunnerStatus, map[string]any{"appID": appID}, &status); err != nil {
			if ctx.Err() != nil {
				last.Reason = "timeout"
				return last, nil
			}
			return last, err
		}
		if status.SessionID == "" {
			return last, api.Errf(api.KindNotFound, "app has no current run")
		}
		if options.SessionID == "" {
			options.SessionID = status.SessionID
		}
		if options.SessionID != status.SessionID {
			return last, api.Errf(api.KindConflict, "run was replaced; start a new wait with the new sessionId")
		}
		last, err = evaluateWait(status, options)
		if err != nil || last.Reason != "" {
			return last, err
		}
		select {
		case <-ctx.Done():
			last.Reason = "timeout"
			return last, nil
		case <-conn.Done():
			if ctx.Err() != nil {
				last.Reason = "timeout"
				return last, nil
			}
			return last, api.Errf(api.KindUnavailable, "connection to daemon closed while waiting")
		case <-wake:
		case <-tick.C:
		}
	}
}

func evaluateWait(status types.StatusEvent, o waitOptions) (waitResult, error) {
	out := waitResult{Status: status, URLs: []waitURL{}}
	selected := map[int64]bool{}
	for _, id := range o.CommandIDs {
		selected[id] = true
	}
	filtered := len(o.CommandIDs) > 0
	count, finished, ready, killed, failed := 0, 0, 0, false, false
	for _, process := range status.Processes {
		if filtered && !selected[process.CommandID] {
			continue
		}
		delete(selected, process.CommandID)
		count++
		switch process.Status {
		case "error":
			failed = true
		case "killed":
			finished++
			killed = true
		case "exited":
			finished++
			if process.ExitCode == nil || *process.ExitCode != 0 {
				failed = true
			}
		}
		if status.Running && process.Status == "running" && len(process.URLs) > 0 {
			ready++
			for _, url := range process.URLs {
				out.URLs = append(out.URLs, waitURL{CommandID: process.CommandID, Label: process.Label, URL: url})
			}
		}
	}
	if len(selected) > 0 {
		return out, api.Errf(api.KindNotFound, "one or more commandIds are not in this run")
	}
	switch {
	case failed || status.Error != "" && !filtered:
		out.Reason = "failed"
	case killed:
		out.Reason = "stopped"
	case count > 0 && finished == count && (len(o.CommandIDs) > 0 || !status.Running):
		out.Reason = "completed"
	case status.Error != "" && (len(o.CommandIDs) == 0 || !status.Running):
		out.Reason = "failed"
	case !status.Running:
		out.Reason = "stopped"
	case o.Until == "ready" && ready > 0 && (o.Readiness == "any" || ready+finished == count):
		out.Reason = "ready"
	}
	return out, nil
}
