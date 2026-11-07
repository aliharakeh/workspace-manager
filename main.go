package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"workspace-manager/daemon"
	"workspace-manager/gui"
	"workspace-manager/mcp"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "workspace-manager:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		runGUI()
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: workspace-manager [mcp|daemon]")
	}
	switch args[0] {
	case "mcp":
		return mcp.Run()
	case "daemon":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return daemon.Serve(ctx)
	default:
		return fmt.Errorf("unknown command %q; usage: workspace-manager [mcp|daemon]", args[0])
	}
}

func runGUI() {
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
