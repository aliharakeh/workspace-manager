package main

import (
	"embed"
	"log"

	"workspace-manager/core/db"


	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	database, err := db.Open()
	if err != nil {
		log.Fatal(err)
	}
	app, err := NewApp(database)
	if err != nil {
		log.Fatal(err)
	}

	err = wails.Run(&options.App{
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
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind:       []interface{}{app},
	})
	if err != nil {
		log.Println("workspace-manager: wails.Run:", err)
	}
}
