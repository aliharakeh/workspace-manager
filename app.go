package main

import (
	"context"
	"log"

	"workspace-manager/internal/api"
	"workspace-manager/internal/api/types"
	"workspace-manager/internal/db"
	"workspace-manager/internal/handlers"
	"workspace-manager/internal/native"
	"workspace-manager/internal/ops"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails-bound API surface. It owns the database, the runner and the
// blueprints in this process: the frontend's calls go through a "gui" Local
// over the operations, and the runner's events go straight to the frontend.
type App struct {
	ctx   context.Context
	db    *db.DB
	ops   *ops.Ops
	local *handlers.Local
}

func NewApp(database *db.DB) (*App, error) {
	a := &App{db: database}
	a.ops = ops.New(database, func(appID int64, event any) {
		if a.ctx != nil {
			EventsEmitRunner(a.ctx, appID, event)
		}
	}, native.Notify)
	a.ops.Pub = a
	local, err := handlers.NewLocal(a.ops)
	if err != nil {
		return nil, err
	}
	a.local = local
	return a, nil
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	_ = runtime.InitializeNotifications(ctx)
}

func (a *App) shutdown(ctx context.Context) {
	log.Println("workspace-manager: shutdown")
	runtime.CleanupNotifications(ctx)
	// Apps the GUI started stop with it; this also restores their templates.
	a.ops.Runner.StopAll(context.Background())
	_ = a.db.Close()
}

// call runs one API method as the GUI and decodes its result into out.
func (a *App) call(method string, params, out any) error {
	return a.local.Call(method, params, out)
}

// Publish implements ops.Publisher: the blueprint runner's log lines go to the
// frontend. Runner output does not pass through here.
func (a *App) Publish(typ string, appID int64, data any) {
	if a.ctx == nil {
		return
	}
	if typ == string(api.EventBlueprint) {
		if ev, ok := data.(types.BlueprintLogEvent); ok {
			EventsEmitBlueprint(a.ctx, ev)
		}
	}
}

// Changed implements ops.Publisher: a write made here tells the frontend what
// to reload.
func (a *App) Changed(entity string, ids ...int64) {
	if a.ctx == nil {
		return
	}
	EventsEmitDataChanged(a.ctx, api.DataChanged{Entity: entity, IDs: ids})
}
