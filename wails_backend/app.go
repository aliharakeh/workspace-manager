package main

import (
	"context"

	"wails_backend/internal/db"
	"wails_backend/internal/native"
	"wails_backend/internal/services"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx    context.Context
	db     *db.DB
	runner *services.Runner
}

func NewApp(database *db.DB) *App {
	return &App{db: database}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	_ = runtime.InitializeNotifications(ctx)
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
