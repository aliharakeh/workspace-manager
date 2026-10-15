package main

import (
	"context"
	"log"
	"time"

	"workspace-manager/api"
	"workspace-manager/api/types"
	"workspace-manager/core/db"
	"workspace-manager/core/handlers"
	"workspace-manager/core/native"
	"workspace-manager/core/ops"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails-bound API surface. It owns the database, the runner and the
// blueprints in this process: the frontend's calls go through a "gui" Local
// over the operations, and the runner's events go straight to the frontend.
// The MCP server shares the database file; writes it makes reach the UI
// through the change log (see watchChanges).
type App struct {
	ctx   context.Context
	db    *db.DB
	ops   *ops.Ops
	local *handlers.Local
	// stopWatch ends watchChanges.
	stopWatch chan struct{}
}

func NewApp(database *db.DB) (*App, error) {
	a := &App{db: database, stopWatch: make(chan struct{})}
	a.ops = ops.New(database, func(appID int64, event any) {
		if a.ctx != nil {
			EventsEmitRunner(a.ctx, appID, event)
		}
	}, native.Notify)
	a.ops.Pub = a
	local, err := handlers.NewLocal(a.ops, handlers.ClientGUI)
	if err != nil {
		return nil, err
	}
	a.local = local
	return a, nil
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	_ = runtime.InitializeNotifications(ctx)
	go a.watchChanges()
}

func (a *App) shutdown(ctx context.Context) {
	log.Println("workspace-manager: shutdown")
	close(a.stopWatch)
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

// watchChanges turns writes the MCP server logged into dataChanged events, so
// the UI refreshes when an agent edits config or takes a run session.
func (a *App) watchChanges() {
	ctx := context.Background()
	_ = a.db.PruneChanges(ctx)
	last, err := a.db.LastChangeID(ctx)
	if err != nil {
		log.Println("workspace-manager: reading the change log:", err)
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-a.stopWatch:
			return
		case <-ticker.C:
		}
		changes, err := a.db.ChangesSince(ctx, last)
		if err != nil {
			continue
		}
		for _, c := range changes {
			last = c.ID
			a.Changed(c.Entity, c.IDs...)
		}
	}
}
