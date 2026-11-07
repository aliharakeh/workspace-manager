package daemon

import (
	"context"
	"os"
	"testing"

	"workspace-manager/daemon/db"
	"workspace-manager/shared/api"
)

func TestServeLocksBeforeOpeningDatabase(t *testing.T) {
	t.Setenv("WM_DATA_DIR", t.TempDir())
	lock, err := acquireLock(api.LockPath())
	if err != nil {
		t.Fatal(err)
	}
	defer releaseLock(lock)
	if err := Serve(context.Background()); err == nil {
		t.Fatal("a competing daemon started")
	}
	if _, err := os.Stat(db.DBPath()); !os.IsNotExist(err) {
		t.Fatalf("competing startup touched the database: %v", err)
	}
}
