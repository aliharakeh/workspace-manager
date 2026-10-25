package gui

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

func EventsEmitRunner(app *application.App, appID int64, event any) {
	app.Event.Emit("runnerEvent", map[string]any{
		"appId": appID,
		"event": event,
	})
}

func EventsEmitAppAI(app *application.App, event any) {
	app.Event.Emit("appAIEvent", event)
}

func EventsEmitBlueprint(app *application.App, event any) {
	app.Event.Emit("blueprintEvent", event)
}

// EventsEmitDataChanged tells the frontend that an entity was written, so it
// reloads it.
func EventsEmitDataChanged(app *application.App, event any) {
	app.Event.Emit("dataChanged", event)
}
