package services

import (
	"context"
	"os"
	"testing"

	"workspace-manager/api"
	"workspace-manager/core/db"
)

func sessionTestApp(t *testing.T) (*db.DB, int64) {
	t.Helper()
	t.Setenv("WM_DATA_DIR", t.TempDir())
	d, err := db.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	ctx := context.Background()
	ws, err := d.CreateWorkspaceT(ctx, "w", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	app, err := d.CreateAppT(ctx, ws.ID, "a", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return d, app.ID
}

func TestClaimSessionIsExclusive(t *testing.T) {
	d, appID := sessionTestApp(t)
	ctx := context.Background()
	mine := db.SessionClaim{AppID: appID, SessionID: "one", Owner: OwnerAgent, Kind: "dev", PID: os.Getpid()}
	if err := ClaimSession(ctx, d, mine, func(db.SessionClaim) bool { return true }); err != nil {
		t.Fatalf("first claim: %v", err)
	}
	other := mine
	other.SessionID, other.Owner = "two", OwnerGUI
	err := ClaimSession(ctx, d, other, func(db.SessionClaim) bool { return true })
	if api.KindOf(err) != api.KindConflict {
		t.Fatalf("second claim: want a conflict, got %v", err)
	}
	if _, ok := ForeignSession(ctx, d, appID); ok {
		t.Fatal("a session of this very process is not foreign")
	}
	ReleaseSession(ctx, d, appID, "one")
	if err := ClaimSession(ctx, d, other, nil); err != nil {
		t.Fatalf("claim after release: %v", err)
	}
}

func TestClaimSessionReclaimsAStaleOne(t *testing.T) {
	d, appID := sessionTestApp(t)
	ctx := context.Background()
	// A holder whose process is gone does not keep the app.
	dead := db.SessionClaim{AppID: appID, SessionID: "dead", Owner: OwnerAgent, Kind: "dev", PID: 0x7ffffff0}
	if ok, err := d.TryInsertAppSession(ctx, dead); err != nil || !ok {
		t.Fatalf("insert: %v %v", ok, err)
	}
	if _, ok := ForeignSession(ctx, d, appID); ok {
		t.Fatal("a dead process's session counted as live")
	}
	next := db.SessionClaim{AppID: appID, SessionID: "next", Owner: OwnerGUI, Kind: "dev", PID: os.Getpid()}
	if err := ClaimSession(ctx, d, next, nil); err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	held, found, err := d.GetAppSession(ctx, appID)
	if err != nil || !found || held.SessionID != "next" {
		t.Fatalf("session after reclaim: %+v %v %v", held, found, err)
	}
}

func TestChangeLog(t *testing.T) {
	d, appID := sessionTestApp(t)
	ctx := context.Background()
	before, _ := d.LastChangeID(ctx)
	if err := d.LogChange(ctx, "session", []int64{appID}); err != nil {
		t.Fatal(err)
	}
	got, err := d.ChangesSince(ctx, before)
	if err != nil || len(got) != 1 || got[0].Entity != "session" || len(got[0].IDs) != 1 || got[0].IDs[0] != appID {
		t.Fatalf("changes: %+v %v", got, err)
	}
}
