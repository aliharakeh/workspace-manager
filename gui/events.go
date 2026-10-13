package main

import (
	"context"

	"workspace-manager/api"
	"workspace-manager/api/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func EventsEmitRunner(ctx context.Context, appID int64, event any) {
	runtime.EventsEmit(ctx, "runnerEvent", map[string]any{
		"appId": appID,
		"event": event,
	})
}

func EventsEmitAppAI(ctx context.Context, event any) {
	runtime.EventsEmit(ctx, "appAIEvent", event)
}

func EventsEmitBlueprint(ctx context.Context, event types.BlueprintLogEvent) {
	runtime.EventsEmit(ctx, "blueprintEvent", event)
}

// EventsEmitDataChanged tells the frontend that an entity was written, so it
// reloads it: the writer may be another client (an agent, another window).
func EventsEmitDataChanged(ctx context.Context, event api.DataChanged) {
	runtime.EventsEmit(ctx, "dataChanged", event)
}
