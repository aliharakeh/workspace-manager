// Command wm is the workspace-manager CLI. It is a client of the wmd daemon,
// which it starts on demand. Every non-AI feature is available here, so an
// agent can drive the whole app without touching the database directly.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"workspace-manager/api"
)

// version is the CLI build version, sent in the handshake.
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	root := newRootCmd()
	root.SetArgs(args)
	err := root.Execute()
	if err == nil {
		return api.ExitOK
	}
	// Usage errors are recognized by cobra, not by a kind.
	var usageErr usageError
	if errors.As(err, &usageErr) {
		emitError(err, api.ExitUsage)
		return api.ExitUsage
	}
	kind := api.KindOf(err)
	emitError(err, kind.ExitCode())
	return kind.ExitCode()
}

// usageError marks a command-line mistake, which exits with the usage code.
type usageError struct{ err error }

func (u usageError) Error() string { return u.err.Error() }
func (u usageError) Unwrap() error { return u.err }

// usage wraps an error as a usage problem.
func usage(format string, args ...any) error {
	return usageError{err: fmt.Errorf(format, args...)}
}

// emitError writes a machine-readable error to stderr, so an agent can react
// without parsing prose.
func emitError(err error, code int) {
	if jsonOutput {
		_ = json.NewEncoder(os.Stderr).Encode(map[string]any{
			"error": map[string]any{"kind": string(api.KindOf(err)), "message": err.Error(), "exitCode": code},
		})
		return
	}
	fmt.Fprintln(os.Stderr, "wm:", err)
}
