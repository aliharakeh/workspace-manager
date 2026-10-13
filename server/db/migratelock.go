package db

import (
	"os"
	"path/filepath"
)

// Open runs the startup migrations under a file lock, because more than one
// process opens the database: the daemon and the GUI. The schema
// upgrade checks for a column and then alters the table, which must not happen
// twice at once.
func withMigrationLock(fn func() error) error {
	lockPath := filepath.Join(DataDir(), "migrate.lock")
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	if err := lockFile(f); err != nil {
		return err
	}
	defer func() { _ = unlockFile(f) }()
	return fn()
}
