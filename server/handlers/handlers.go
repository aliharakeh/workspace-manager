// Package handlers maps API methods to server operations, applying the
// client-kind rules: AI methods are GUI-only, and env var values and template
// contents excluded from AI are hidden from CLI clients.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"workspace-manager/api"
	"workspace-manager/api/rpc"
	"workspace-manager/server/ops"
)

// Client kinds known to the server.
const (
	ClientCLI = "cli"
	ClientGUI = "gui"
)

// Registry is the method table the daemon serves.
type Registry struct {
	Ops *ops.Ops
	AI  AIOps

	srv     rpc.Registerer
	done    map[string]bool
	publish func(t api.EventType, appID int64, data any)
}

// RegisterAll wires every domain into a server and returns the registry, so
// the caller can check the method table afterwards. publish, when set, carries
// streamed payloads such as AI progress to subscribers.
func (r *Registry) RegisterAll(srv rpc.Registerer, publish func(t api.EventType, appID int64, data any)) *Registry {
	r.srv = srv
	r.done = map[string]bool{}
	if publish == nil {
		publish = func(api.EventType, int64, any) {}
	}
	r.publish = publish
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
	r.aiOps()
	return r
}

// handle registers one method and records it.
func (r *Registry) handle(method string, fn rpc.Handler) {
	r.srv.Handle(method, fn)
	r.done[method] = true
}

// CheckRegistration compares the API registry against the handlers that were
// registered, so a missing or extra handler is caught at startup. hello,
// events.subscribe and server.stop are served by the connection itself, so
// they count as registered.
func (r *Registry) CheckRegistration() error {
	builtin := map[string]bool{
		api.MethodHello:     true,
		api.MethodSubscribe: true,
		api.MethodStop:      true,
	}
	missing := []string{}
	for _, m := range api.Methods {
		if !r.done[m.Name] && !builtin[m.Name] {
			missing = append(missing, m.Name)
		}
	}
	extra := []string{}
	for name := range r.done {
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
func typed[In any, Out any](fn func(In) (Out, error)) rpc.Handler {
	return func(_ context.Context, raw Params) (any, error) {
		in, err := decode[In](raw)
		if err != nil {
			return nil, err
		}
		return fn(in)
	}
}

// typedCtx is typed with the request context, for calls that can be cancelled.
func typedCtx[In any, Out any](fn func(context.Context, In) (Out, error)) rpc.Handler {
	return func(ctx context.Context, raw Params) (any, error) {
		in, err := decode[In](raw)
		if err != nil {
			return nil, err
		}
		return fn(ctx, in)
	}
}

// withKind wraps a handler that needs to know whether the caller is the CLI or
// the GUI, which decides whether AI-excluded values are shown.
func withKind(fn func(kind string, raw Params) (any, error)) rpc.Handler {
	return func(ctx context.Context, raw Params) (any, error) {
		return fn(clientKind(ctx), raw)
	}
}

// clientKind reads the client kind the daemon attached to the request context.
func clientKind(ctx context.Context) string {
	if v, ok := ctx.Value(rpc.ClientKindContextKey{}).(string); ok && v != "" {
		return v
	}
	return ClientGUI
}
