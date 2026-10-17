// Command wmd is the background server: it owns the database, the runner and the
// blueprints, and serves clients over a local socket. Clients start
// it on demand; it exits when idle with nothing running.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"workspace-manager/shared/api"
	"workspace-manager/daemon"
	"workspace-manager/daemon/db"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "wmd:", err)
		os.Exit(1)
	}
}

func run() error {
	dataDir := api.DataDir()
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	logFile, err := os.OpenFile(filepath.Join(dataDir, "wmd.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = logFile.Close() }()

	database, err := db.Open()
	if err != nil {
		return err
	}
	defer func() { _ = database.Close() }()

	d, err := daemon.New(database, logFile)
	if err != nil {
		return err
	}

	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	return d.Run(ctx)
}
