package daemon

import (
	"context"
	"os"
	"path/filepath"

	"workspace-manager/daemon/db"
	"workspace-manager/shared/api"
)

// Serve acquires ownership of the data folder before opening the database.
func Serve(ctx context.Context) error {
	lock, err := acquireLock(api.LockPath())
	if err != nil {
		return err
	}
	defer releaseLock(lock)

	logFile, err := os.OpenFile(filepath.Join(api.DataDir(), "wmd.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer logFile.Close()
	database, err := db.Open()
	if err != nil {
		return err
	}
	defer database.Close()
	d, err := New(database, logFile)
	if err != nil {
		return err
	}
	return d.run(ctx)
}
