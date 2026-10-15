// Package handlers maps API methods to operations, applying the client-kind
// rules: git, running and applying reviewed changes are GUI-only, and env var
// values and template contents excluded from AI are hidden from agent clients.
// It runs in the caller's process: the GUI and the MCP server each build a
// Local over the same operations.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"workspace-manager/api"
	"workspace-manager/core/ops"
)

// Client kinds. An "agent" client (the MCP server) is not the GUI: GUI-only
// methods are refused and AI-excluded values are hidden.
const (
	ClientAgent = "agent"
	ClientGUI   = "gui"
)

// Handler runs one method. params is the raw JSON of the call; the result is
// marshalled back to the caller.
type Handler func(ctx context.Context, params json.RawMessage) (any, error)

// Registry is the method table.
type Registry struct {
	Ops *ops.Ops

	handlers map[string]Handler
}

// RegisterAll wires every domain and returns the registry, so the caller can
// check the method table afterwards.
func (r *Registry) RegisterAll() *Registry {
	r.handlers = map[string]Handler{}
	r.workspaces()
	r.apps()
	r.configSets()
	r.envVars()
	r.templates()
	r.scripts()
	r.runner()
	r.git()
	r.system()
	r.blueprints()
	return r
}

// handle registers one method.
func (r *Registry) handle(method string, fn Handler) {
	r.handlers[method] = fn
}

// CheckRegistration compares the API registry against the handlers that were
// registered, so a missing or extra handler is caught at startup.
func (r *Registry) CheckRegistration() error {
	missing := []string{}
	for _, m := range api.Methods {
		if _, ok := r.handlers[m.Name]; !ok {
			missing = append(missing, m.Name)
		}
	}
	extra := []string{}
	for name := range r.handlers {
		known := false
		for _, m := range api.Methods {
			if m.Name == name {
				known = true
				break
			}
		}
		if !known {
			extra = append(extra, name)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 {
		return fmt.Errorf("methods without handlers: %v", missing)
	}
	if len(extra) > 0 {
		return fmt.Errorf("handlers without methods: %v", extra)
	}
	return nil
}

// Params is a handler's raw parameters.
type Params = json.RawMessage

// decode unmarshals params, reporting a typed error on bad input.
func decode[In any](raw Params) (In, error) {
	var in In
	if len(raw) == 0 {
		return in, nil
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return in, api.Errf(api.KindInvalid, "invalid parameters: %v", err)
	}
	return in, nil
}

// typed wraps a plain function into a handler.
func typed[In any, Out any](fn func(In) (Out, error)) Handler {
	return func(_ context.Context, raw Params) (any, error) {
		in, err := decode[In](raw)
		if err != nil {
			return nil, err
		}
		return fn(in)
	}
}

// typedCtx is typed with the request context, for calls that can be cancelled.
func typedCtx[In any, Out any](fn func(context.Context, In) (Out, error)) Handler {
	return func(ctx context.Context, raw Params) (any, error) {
		in, err := decode[In](raw)
		if err != nil {
			return nil, err
		}
		return fn(ctx, in)
	}
}

// withKind wraps a handler that needs to know whether the caller is an agent
// client or the GUI, which decides whether AI-excluded values are shown.
func withKind(fn func(kind string, raw Params) (any, error)) Handler {
	return func(ctx context.Context, raw Params) (any, error) {
		return fn(clientKind(ctx), raw)
	}
}

// clientKind reads the client kind Local attached to the request context.
// A request without one is treated as an agent, so nothing is shown to a client
// that never said what it is.
func clientKind(ctx context.Context) string {
	if v, ok := ctx.Value(kindKey{}).(string); ok && v != "" {
		return v
	}
	return ClientAgent
}
