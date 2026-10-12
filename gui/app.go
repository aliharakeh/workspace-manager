package main

import (
	"context"
	"encoding/json"
	"log"

	"workspace-manager/api"
	"workspace-manager/api/client"
	"workspace-manager/api/types"
	"workspace-manager/server/db"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails-bound API surface. It is a client of the wmd daemon, which
// owns the database, the runner and the blueprints; only the AI methods run in
// this process.
type App struct {
	ctx context.Context
	// conn is the connection to the daemon, nil when none could be opened.
	conn *client.Client
	// db backs the AI methods, which run in this process. Nothing else may use
	// it: every other operation goes through the server.
	db *db.DB
}

func NewApp(database *db.DB) *App {
	return &App{db: database}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	_ = runtime.InitializeNotifications(ctx)
	c, err := client.Dial(client.Options{
		Kind:    "gui",
		Version: "dev",
		Start:   client.StartDaemon(),
	})
	if err != nil {
		// The app still opens: the dialogs, the settings and the AI work
		// without the server, while the proxy methods report that it is not
		// running.
		log.Println("workspace-manager: could not reach the server:", err)
		return
	}
	log.Println("workspace-manager: connected to the server")
	a.conn = c
	go a.forwardEvents()
}

func (a *App) shutdown(ctx context.Context) {
	log.Println("workspace-manager: shutdown")
	runtime.CleanupNotifications(ctx)
	if a.conn != nil {
		_ = a.conn.Close()
	}
	if a.db != nil {
		_ = a.db.Close()
	}
}

// client returns the connection to the daemon, nil when none could be opened.
func (a *App) client() *client.Client {
	return a.conn
}

// call runs one server method and decodes its result into out.
func (a *App) call(method string, params, out any) error {
	c := a.client()
	if c == nil {
		return api.Errf(api.KindUnavailable, "the server is not running")
	}
	return c.Call(method, params, out)
}

// forwardEvents re-emits the daemon's streams as the Wails events the frontend
// already listens for.
func (a *App) forwardEvents() {
	events, err := a.conn.SubscribeRaw(api.SubscribeInput{
		Types: []api.EventType{api.EventRunner, api.EventBlueprint, api.EventAppAI},
	})
	if err != nil {
		log.Println("workspace-manager: subscribing to server events:", err)
		return
	}
	for {
		select {
		case <-a.conn.Done():
			log.Println("workspace-manager: the server connection closed")
			return
		case raw := <-events:
			a.forwardEvent(raw)
		}
	}
}

// pushedEvent is a daemon notification: the envelope with the payload left
// raw, so each stream decodes its own shape.
type pushedEvent struct {
	Type api.EventType   `json:"type"`
	App  int64           `json:"appId"`
	Data json.RawMessage `json:"data"`
}

// forwardEvent turns one pushed payload into the matching Wails event. The
// runner payload keeps its raw data, so the frontend receives the same
// {appId, event} object it always did.
func (a *App) forwardEvent(raw json.RawMessage) {
	if a.ctx == nil {
		return
	}
	var ev pushedEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return
	}
	switch ev.Type {
	case api.EventRunner:
		EventsEmitRunner(a.ctx, ev.App, ev.Data)
	case api.EventBlueprint:
		var bp types.BlueprintLogEvent
		if err := json.Unmarshal(ev.Data, &bp); err == nil {
			EventsEmitBlueprint(a.ctx, bp)
		}
	case api.EventAppAI:
		var ai types.AppAIStreamEvent
		if err := json.Unmarshal(ev.Data, &ai); err == nil {
			EventsEmitAppAI(a.ctx, ai)
		}
	}
}
