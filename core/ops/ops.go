// Package ops holds every write operation the server performs, with the same
// validation the Wails bindings used to do. Handlers and the GUI both call it;
// nothing here touches Wails.
package ops

import (
	"context"

	"workspace-manager/api/types"
	"workspace-manager/core/db"
	"workspace-manager/core/services"
)

// Publisher pushes server events to clients. The daemon wires it to the RPC
// server; tests can leave it nil.
type Publisher interface {
	// Publish sends an event to subscribers. appID 0 means not app-specific.
	Publish(typ string, appID int64, data any)
	// Changed tells clients that an entity was written, so a UI can refresh.
	// ids are the app ids (workspace ids for a workspace) the write touched;
	// with none the client reloads the entity.
	Changed(entity string, ids ...int64)
}

// Ops is the server-side API surface.
type Ops struct {
	DB         *db.DB
	Runner     *services.Runner
	Blueprints *services.BlueprintRunner
	Pub        Publisher
}

// New returns the operations bound to a database, with the runner and
// blueprint runner constructed against it. broker receives runner events (the
// daemon forwards them; the GUI process can emit them itself).
func New(d *db.DB, broker func(appID int64, event any), notify func(title, body string)) *Ops {
	o := &Ops{DB: d}
	o.Runner = services.NewRunner(d, broker, notify)
	o.Blueprints = services.NewBlueprintRunner(d, func(event types.BlueprintLogEvent) {
		o.publish("blueprintEvent", 0, event)
	})
	return o
}

func (o *Ops) publish(typ string, appID int64, data any) {
	if o.Pub != nil {
		o.Pub.Publish(typ, appID, data)
	}
}

// changed announces a write so clients can refresh; ids are the app ids
// (workspace ids for a workspace) it touched. A zero id means "unknown", and
// is dropped so clients read an empty list as "reload the entity".
func (o *Ops) changed(entity string, ids ...int64) {
	if o.Pub == nil {
		return
	}
	kept := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id != 0 {
			kept = append(kept, id)
		}
	}
	o.Pub.Changed(entity, kept...)
}

// Ctx is the context operations run with. Most calls only need a background
// context; run sessions and agent calls get their own.
func (o *Ops) Ctx() context.Context { return context.Background() }
