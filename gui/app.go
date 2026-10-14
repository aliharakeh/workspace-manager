package main

import (
	"context"
	"encoding/json"
	"log"
	"sync"

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
	mu  sync.Mutex
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
	a.connectLocked()
}

// connectLocked dials the daemon (starting it on demand) and stores the
// connection. It is safe to call more than once; a live connection is left
// alone. It returns the resulting error, if any, so callers can report it.
func (a *App) connectLocked() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.conn != nil {
		return nil
	}
	c, err := client.Dial(client.Options{
		Kind:    "gui",
		Version: "dev",
		Start:   client.StartDaemon(),
	})
	if err != nil {
		log.Println("workspace-manager: could not reach the server:", err)
		return err
	}
	log.Println("workspace-manager: connected to the server")
	a.conn = c
	go a.forwardEvents()
	return nil
}

// ConnectServer lets the UI retry the connection after a failed startup. It
// is bound to Wails, so the frontend can call it from a "Retry" button.
func (a *App) ConnectServer() error {
	return a.connectLocked()
}

// ServerConnected reports whether a daemon connection is currently open.
func (a *App) ServerConnected() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.conn != nil
}

func (a *App) shutdown(ctx context.Context) {
	log.Println("workspace-manager: shutdown")
	runtime.CleanupNotifications(ctx)
	a.mu.Lock()
	c := a.conn
	a.conn = nil
	a.mu.Unlock()
	if c != nil {
		_ = c.Close()
	}
	if a.db != nil {
		_ = a.db.Close()
	}
}

// client returns the connection to the daemon, nil when none could be opened.
func (a *App) client() *client.Client {
	a.mu.Lock()
	defer a.mu.Unlock()
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
	c := a.client()
	if c == nil {
		return
	}
	events, err := c.SubscribeRaw(api.SubscribeInput{
		Types: []api.EventType{
			api.EventRunner,
			api.EventBlueprint,
			api.EventAppAI,
			api.EventDataChanged,
		},
	})
	if err != nil {
		log.Println("workspace-manager: subscribing to server events:", err)
		a.mu.Lock()
		if a.conn == c {
			a.conn = nil
		}
		a.mu.Unlock()
		_ = c.Close()
		return
	}
	for {
		select {
		case <-c.Done():
			log.Println("workspace-manager: the server connection closed")
			a.mu.Lock()
			if a.conn == c {
				a.conn = nil
			}
			a.mu.Unlock()
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
	case api.EventDataChanged:
		var changed api.DataChanged
		if err := json.Unmarshal(ev.Data, &changed); err == nil {
			EventsEmitDataChanged(a.ctx, changed)
		}
	}
}
