package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// DataDir is where the database, the daemon socket and the AI config live. WM_DATA_DIR overrides
// it, which gives tests and sandboxes their own data.
func DataDir() string {
	if dir := os.Getenv("WM_DATA_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(base, "workspace-manager")
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "workspace-manager")
	default:
		base := os.Getenv("XDG_DATA_HOME")
		if base == "" {
			base = filepath.Join(home, ".local", "share")
		}
		return filepath.Join(base, "workspace-manager")
	}
}

// Kind classifies an error so a caller can react without parsing the
// message.
type Kind string

const (
	KindInvalid     Kind = "invalid"
	KindNotFound    Kind = "not_found"
	KindConflict    Kind = "conflict"
	KindUnavailable Kind = "unavailable"
	KindFailed      Kind = "failed"
)

// Err is an error with a kind. Handlers and ops return these.
type Err struct {
	Kind    Kind
	Message string
}

func (e *Err) Error() string { return e.Message }

// Errf builds a typed error.
func Errf(kind Kind, format string, args ...any) *Err {
	return &Err{Kind: kind, Message: fmt.Sprintf(format, args...)}
}

// KindOf reports the kind of err, defaulting to KindFailed. It understands both
// local typed errors and errors that came back over the wire.
func KindOf(err error) Kind {
	if err == nil {
		return ""
	}
	var e *Err
	if errors.As(err, &e) && e.Kind != "" {
		return e.Kind
	}
	var k interface{ ErrorKind() string }
	if errors.As(err, &k) {
		if kind := k.ErrorKind(); kind != "" {
			return Kind(kind)
		}
	}
	return KindFailed
}

// SocketPath is where the daemon listens and clients connect.
func SocketPath() string {
	return filepath.Join(DataDir(), "wmd.sock")
}

// LockPath is the file that keeps one daemon per data folder.
func LockPath() string {
	return filepath.Join(DataDir(), "wmd.lock")
}

// ServerInfo is what hello returns. Clients check Version on connect.
type ServerInfo struct {
	Version   string `json:"version"`
	StartedAt int64  `json:"startedAt"`
	DataDir   string `json:"dataDir"`
	Socket    string `json:"socket"`
}

// ClientInfo identifies a connecting client. Kind is "agent" or "gui"; the
// server uses it to decide what the client may call and whether to hide
// AI-excluded values.
type ClientInfo struct {
	Kind    string `json:"kind"`
	Name    string `json:"name,omitempty"`
	Version string `json:"version"`
	// Token authenticates a connection over the loopback TCP endpoint; the
	// unix socket does not need one.
	Token string `json:"token,omitempty"`
}

// Version is the build version the daemon reports in hello and the GUI sends.
var Version = "dev"

// EndpointPath is the file where the daemon publishes its loopback TCP
// endpoint for clients that cannot use the unix socket (Node on Windows, the
// VS Code extension).
func EndpointPath() string {
	return filepath.Join(DataDir(), "wmd.endpoint")
}

// Endpoint is the content of the endpoint file: where to connect, and the
// token every connection must present in hello. The file is readable only by
// the user, which is what keeps other local users and web pages out.
type Endpoint struct {
	Addr  string `json:"addr"`
	Token string `json:"token"`
}
