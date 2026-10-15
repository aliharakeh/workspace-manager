package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// DataDir is where the database and the AI config live. WM_DATA_DIR overrides
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

// KindOf reports the kind of err, defaulting to KindFailed.
func KindOf(err error) Kind {
	if err == nil {
		return ""
	}
	var e *Err
	if errors.As(err, &e) && e.Kind != "" {
		return e.Kind
	}
	return KindFailed
}
