package db

import (
	"path/filepath"

	"workspace-manager/internal/api"
)

const dbFile = "workspace-manager.sqlite"

// DataDir is where the database and the daemon's files live. It is shared with
// the clients, so WM_DATA_DIR isolates both the daemon and the clients that
// talk to it.
func DataDir() string {
	return api.DataDir()
}

func DBPath() string {
	return filepath.Join(DataDir(), dbFile)
}
