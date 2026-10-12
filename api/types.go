package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// DataDir is where the database, the socket and the AI config live.
// WM_DATA_DIR overrides it, which gives tests and sandboxed agents their own
// server and data.
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

// ClientInfo identifies a connecting client. Kind is "cli" or "gui"; the
// server uses it to decide whether to hide AI-excluded values.
type ClientInfo struct {
	Kind    string `json:"kind"`
	Name    string `json:"name,omitempty"`
	Version string `json:"version"`
}

// Kind classifies an error so a client can react without parsing the message.
// The CLI maps kinds to exit codes.
type Kind string

const (
	KindInvalid     Kind = "invalid"
	KindNotFound    Kind = "not_found"
	KindConflict    Kind = "conflict"
	KindUnavailable Kind = "unavailable"
	KindFailed      Kind = "failed"
)

// Exit codes used by the CLI.
const (
	ExitOK          = 0
	ExitError       = 1
	ExitUsage       = 2
	ExitNotFound    = 3
	ExitConflict    = 4
	ExitUnavailable = 5
)

// Err is an error with a kind. Handlers return these and the server sends the
// kind alongside the message.
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

// ExitCode maps a kind to the CLI's exit code.
func (k Kind) ExitCode() int {
	switch k {
	case KindNotFound:
		return ExitNotFound
	case KindConflict:
		return ExitConflict
	case KindUnavailable:
		return ExitUnavailable
	default:
		return ExitError
	}
}
