// Package daemon runs the background server: it owns the database, the runner,
// the blueprints and the AI, and serves the API over a local socket.
package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"workspace-manager/api"
	"workspace-manager/api/rpc"
	"workspace-manager/server/db"
	"workspace-manager/server/handlers"
	"workspace-manager/server/native"
	"workspace-manager/server/ops"
)

// Version identifies the server build; clients compare it on connect.
var Version = "dev"

// IdleExit is how long the server stays up with no client activity and nothing
// running. Run sessions count as activity, so a started app keeps it alive.
var IdleExit = 10 * time.Minute

// Daemon is the running server.
type Daemon struct {
	db     *db.DB
	ops    *ops.Ops
	server *rpc.Server
	logger *os.File
	lock   *os.File

	mu    sync.Mutex
	last  time.Time
	stop  chan struct{}
	once  sync.Once
}

// New prepares the server: database, operations, AI and the method table.
func New(database *db.DB, logw *os.File) (*Daemon, error) {
	d := &Daemon{db: database, last: time.Now(), logger: logw}

	d.ops = ops.New(database, func(appID int64, event any) {
		d.broadcast(api.EventRunner, appID, event)
	}, native.Notify)
	d.ops.Pub = d


	d.server = rpc.NewServer(d.hello, d.permit)
	reg := &handlers.Registry{Ops: d.ops}
	reg.RegisterAll(d.server)
	if err := reg.CheckRegistration(); err != nil {
		return nil, err
	}
	d.server.Register(api.MethodStop, d.handleStop)
	return d, nil
}

// Publisher: the ops layer reports writes here so clients can refresh.
func (d *Daemon) Publish(typ string, appID int64, data any) {
	d.broadcast(api.EventType(typ), appID, data)
}

// Changed pushes a lightweight "this entity changed" event.
func (d *Daemon) Changed(entity string, ids ...int64) {
	d.broadcast(api.EventDataChanged, 0, api.DataChanged{Entity: entity, IDs: ids})
}

func (d *Daemon) broadcast(t api.EventType, appID int64, data any) {
	d.touch()
	d.server.Broadcast(api.Event{Type: t, App: appID, Data: data})
}

func (d *Daemon) touch() {
	d.mu.Lock()
	d.last = time.Now()
	d.mu.Unlock()
}

// hello answers the handshake and records the connecting client's kind.
func (d *Daemon) hello(raw json.RawMessage) (api.ClientInfo, any, error) {
	var in api.ClientInfo
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &in)
	}
	if in.Kind == "" {
		// Least privilege: a client that does not say what it is is treated as
		// an agent, never the GUI.
		in.Kind = handlers.ClientAgent
	}
	d.touch()
	info := api.ServerInfo{
		Version:   Version,
		StartedAt: time.Now().UnixMilli(),
		DataDir:   api.DataDir(),
		Socket:    api.SocketPath(),
	}
	return in, map[string]any{"ok": true, "server": info}, nil
}

// permit decides what a client may call: methods marked Agent are for both
// clients, everything else is GUI-only, and server.stop is local shutdown.
// Only hello runs before a client has a kind; the rpc layer refuses the rest.
func (d *Daemon) permit(kind, method string) bool {
	if method == api.MethodHello || method == api.MethodSubscribe || method == api.MethodStop {
		return true
	}
	for _, m := range api.Methods {
		if m.Name == method {
			if m.Agent {
				return true
			}
			return kind == handlers.ClientGUI
		}
	}
	return false
}

// handleStop asks the daemon to stop; a client uses it to shut the daemon down.
func (d *Daemon) handleStop(context.Context, json.RawMessage) (any, error) {
	go func() {
		// Let the reply reach the client before the socket goes away.
		time.Sleep(100 * time.Millisecond)
		d.Stop()
	}()
	return map[string]any{"ok": true}, nil
}

// Stop ends the server gracefully; the main loop stops the runner.
func (d *Daemon) Stop() {
	d.once.Do(func() { close(d.stop) })
}

// Run serves until ctx is cancelled, Stop is called, or it is idle with
// nothing running.
func (d *Daemon) Run(ctx context.Context) error {
	if v := os.Getenv("WM_DAEMON_IDLE"); v != "" {
		if secs, err := time.ParseDuration(v); err == nil && secs > 0 {
			IdleExit = secs
		}
	}
	socket := api.SocketPath()
	lock, err := acquireLock(api.LockPath())
	if err != nil {
		return err
	}
	d.lock = lock
	defer releaseLock(lock)

	ln, err := rpc.Listen(socket)
	if err != nil {
		return fmt.Errorf("opening the socket: %w", err)
	}
	defer func() { _ = os.Remove(socket) }()
	d.stop = make(chan struct{})

	serveCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		select {
		case <-d.stop:
			cancel()
		case <-ctx.Done():
		}
	}()
	go d.watchIdle(serveCtx, cancel)

	d.logf("daemon %s listening on %s", Version, socket)
	err = d.server.Serve(serveCtx, ln)
	d.ops.Runner.StopAll(context.Background())
	_ = d.server.Close()
	d.logf("daemon stopped")
	return err
}

// watchIdle cancels the context once nothing is running, no client is
// connected, and no client has spoken for IdleExit.
func (d *Daemon) watchIdle(ctx context.Context, cancel context.CancelFunc) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if d.hasRunningApps() {
				d.touch()
				continue
			}
			// A connected client (the desktop app, say) needs the server even
			// while nothing runs, so it counts as activity.
			if d.server.ClientCount() > 0 {
				d.touch()
				continue
			}
			d.mu.Lock()
			idle := time.Since(d.last)
			d.mu.Unlock()
			if idle >= IdleExit {
				d.logf("idle for %s with nothing running, exiting", idle.Round(time.Second))
				cancel()
				return
			}
		}
	}
}

func (d *Daemon) hasRunningApps() bool {
	workspaces, err := d.db.ListWorkspacesT(context.Background())
	if err != nil {
		return false
	}
	for _, ws := range workspaces {
		apps, err := d.db.ListAppsByWorkspaceT(context.Background(), ws.ID)
		if err != nil {
			continue
		}
		for _, app := range apps {
			if d.ops.Runner.GetStatus(app.ID).Running {
				return true
			}
		}
	}
	return false
}

// logf writes to the daemon log file when one is open.
func (d *Daemon) logf(format string, args ...any) {
	if d.logger == nil {
		return
	}
	line := time.Now().Format(time.RFC3339) + " " + fmt.Sprintf(format, args...) + "\n"
	_, _ = d.logger.WriteString(line)
}

// acquireLock keeps one daemon per data folder. A second daemon fails
// immediately instead of fighting over the socket.
func acquireLock(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := tryLock(f); err != nil {
		_ = f.Close()
		return nil, errors.New("another workspace-manager daemon is already running for this data folder")
	}
	return f, nil
}

func releaseLock(f *os.File) {
	if f == nil {
		return
	}
	_ = unlock(f)
	_ = f.Close()
}
