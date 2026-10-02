package main

import (
	"context"

	"workspace-manager/db"
	"workspace-manager/native"
	"workspace-manager/services"
	"workspace-manager/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx        context.Context
	db         *db.DB
	runner     *services.Runner
	blueprints *services.BlueprintRunner
}

func NewApp(database *db.DB) *App {
	return &App{db: database}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	_ = runtime.InitializeNotifications(ctx)
	a.blueprints = services.NewBlueprintRunner(a.db, func(event types.BlueprintLogEvent) {
		EventsEmitBlueprint(a.ctx, event)
	})
	a.runner = services.NewRunner(a.db, func(appID int64, event any) {
		if a.ctx == nil {
			return
		}
		EventsEmitRunner(a.ctx, appID, event)
	}, func(title, body string) {
		native.Notify(a.ctx, title, body)
	})
}

func (a *App) shutdown(ctx context.Context) {
	if a.runner != nil {
		a.runner.StopAll(ctx)
	}
	runtime.CleanupNotifications(ctx)
	if a.db != nil {
		_ = a.db.Close()
	}
}
