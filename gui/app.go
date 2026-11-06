package gui

import (
	"context"
	"log"

	"workspace-manager/gui/git"
	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// App is the Wails-bound API surface. The data, the runner and the blueprints
// belong to the wmd daemon, reached through link; git, the file dialogs
// and the terminal events' path to the frontend are handled here.
type App struct {
	ctx  context.Context
	app  *application.App
	link *Link
	git  *git.Git
}

func NewApp(app *application.App) *App {
	a := &App{app: app}
	a.link = newLink(a.relay, a.reloadAll)
	a.git = git.New(gitBackend{a})
	return a
}

func (a *App) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	a.ctx = ctx
	// Connect now so the daemon is up and events flow; a failure is retried by
	// the first call.
	if _, err := a.link.client(); err != nil {
		log.Println("workspace-manager: daemon:", err)
	}
	return nil
}

func (a *App) ServiceShutdown() error {
	log.Println("workspace-manager: shutdown")
	// Only the connection ends: the daemon keeps running, and with it the apps
	// it started.
	a.link.Close()
	return nil
}

// relay forwards a daemon event to the frontend under the event names it
// already knows.
func (a *App) relay(ev api.Event) {
	if a.ctx == nil {
		return
	}
	switch ev.Type {
	case api.EventRunner:
		EventsEmitRunner(a.app, ev.App, ev.Data)
	case api.EventBlueprint:
		EventsEmitBlueprint(a.app, ev.Data)
	case api.EventDataChanged:
		EventsEmitDataChanged(a.app, ev.Data)
	}
}

// reloadAll asks the frontend to reload its lists after the connection to the
// daemon was re-established, since events may have been missed meanwhile.
func (a *App) reloadAll() {
	if a.ctx == nil {
		return
	}
	for _, entity := range []string{"workspace", "app"} {
		EventsEmitDataChanged(a.app, api.DataChanged{Entity: entity, IDs: []int64{}})
	}
}

// gitBackend is what the git operations need from the daemon (gitops.Backend).
// It is its own type so Wails does not bind these methods to the frontend.
type gitBackend struct{ a *App }

func (b gitBackend) App(appID int64) (types.App, error) {
	return call[types.App](b.a.link, api.MethodAppsGet, map[string]any{"id": appID})
}

func (b gitBackend) Running(appID int64) bool {
	st, err := call[types.StatusEvent](b.a.link, api.MethodRunnerStatus, map[string]any{"appID": appID})
	return err == nil && st.Running
}

func (b gitBackend) Links(appID int64) ([]string, error) {
	return call[[]string](b.a.link, api.MethodWorktreeLinksGet, map[string]any{"appID": appID})
}

func (b gitBackend) StartIn(appID int64, kind, appDir, worktree string, configSetID int64) (types.StatusEvent, error) {
	return call[types.StatusEvent](b.a.link, api.MethodRunnerStartIn, map[string]any{
		"appID": appID, "kind": kind, "dir": appDir, "worktree": worktree, "configSetID": configSetID,
	})
}

// Changed tells the frontend that a write made here (git) needs a reload.
func (b gitBackend) Changed(entity string, ids ...int64) {
	if b.a.ctx == nil {
		return
	}
	if ids == nil {
		ids = []int64{}
	}
	EventsEmitDataChanged(b.a.app, api.DataChanged{Entity: entity, IDs: ids})
}
