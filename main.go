package main

import (
	"embed"
	"log"

	"workspace-manager/gui"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := gui.NewApp()

	err := wails.Run(&options.App{
		Title:  "Workspace Manager",
		Width:  1280,
		Height: 860,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: true,
		},
		OnStartup:  app.Startup,
		OnShutdown: app.Shutdown,
		Bind:       []interface{}{app},
	})
	if err != nil {
		log.Println("workspace-manager: wails.Run:", err)
	}
}
