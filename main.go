package main

import (
	"embed"
	"log"

	"workspace-manager/gui"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := application.New(application.Options{
		Name: "Workspace Manager",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})
	app.RegisterService(application.NewService(gui.NewApp(app)))
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:          "Workspace Manager",
		Width:          1280,
		Height:         860,
		URL:            "/",
		EnableFileDrop: true,
	})
	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		app.Event.Emit("filesDropped", event.Context().DroppedFiles())
	})
	if err := app.Run(); err != nil {
		log.Println("workspace-manager: app.Run:", err)
	}
}
