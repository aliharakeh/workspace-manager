package services

import (
	"context"
	"os"

	"workspace-manager/api"
	"workspace-manager/core/db"
	"workspace-manager/core/native"
)

// Session owners recorded in app_sessions.
const (
	OwnerGUI   = "gui"
	OwnerAgent = "agent"
)

// ClaimSession records that claim's owner runs the app, so the GUI's runner and
// an agent working through the MCP server never apply templates at once. It
// fails with a conflict while another live session holds the app. A row left by
// a process that is gone is stale: it is cleared and its templates restored
// first. A row of this very process is stale too when ownSessionLive says so
// (the runner no longer has it).
func ClaimSession(ctx context.Context, d *db.DB, claim db.SessionClaim, ownSessionLive func(db.SessionClaim) bool) error {
	for attempt := 0; attempt < 2; attempt++ {
		ok, err := d.TryInsertAppSession(ctx, claim)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		held, found, err := d.GetAppSession(ctx, claim.AppID)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		live := native.PidAlive(held.PID)
		if held.PID == os.Getpid() && ownSessionLive != nil {
			live = ownSessionLive(held)
		}
		if live {
			return api.Errf(api.KindConflict, "App is already in use by %s (%s)", held.Owner, held.Kind)
		}
		_ = RestoreTemplates(ctx, d, held.AppID, held.ConfigSetID, held.Dir, held.SessionID)
		if err := d.DeleteAppSession(ctx, held.AppID, held.SessionID); err != nil {
			return err
		}
	}
	return api.Errf(api.KindConflict, "App is already in use")
}

// ReleaseSession drops the app's claim if it still belongs to sessionID.
func ReleaseSession(ctx context.Context, d *db.DB, appID int64, sessionID string) {
	_ = d.DeleteAppSession(ctx, appID, sessionID)
}

// ForeignSession returns the live session another process holds on the app,
// (zero, false) when none. A row of a dead process is not live.
func ForeignSession(ctx context.Context, d *db.DB, appID int64) (db.SessionClaim, bool) {
	held, found, err := d.GetAppSession(ctx, appID)
	if err != nil || !found || held.PID == os.Getpid() || !native.PidAlive(held.PID) {
		return db.SessionClaim{}, false
	}
	return held, true
}
