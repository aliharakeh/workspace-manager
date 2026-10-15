package handlers

import (
	"context"
	"encoding/json"

	"workspace-manager/api"
	"workspace-manager/core/ops"
)

type kindKey struct{}

// Local calls API methods in-process, as one kind of client. The GUI is a
// "gui" Local; the MCP server is an "agent" one, which is refused every
// method not marked Agent and sees AI-excluded values blanked.
type Local struct {
	reg  *Registry
	kind string
}

// NewLocal builds the method table over o and checks it against the registry.
// Any kind but the GUI is treated as an agent, the least-privileged kind.
func NewLocal(o *ops.Ops, kind string) (*Local, error) {
	reg := (&Registry{Ops: o}).RegisterAll()
	if err := reg.CheckRegistration(); err != nil {
		return nil, err
	}
	if kind != ClientGUI {
		kind = ClientAgent
	}
	return &Local{reg: reg, kind: kind}, nil
}

// Kind is the client kind this Local calls as.
func (l *Local) Kind() string { return l.kind }

// Call runs one method and decodes its result into out (which may be nil).
func (l *Local) Call(method string, params, out any) error {
	return l.CallContext(context.Background(), method, params, out)
}

// CallContext is Call with a context for cancellable methods.
func (l *Local) CallContext(ctx context.Context, method string, params, out any) error {
	if !l.permit(method) {
		return api.Errf(api.KindInvalid, "method %q is not available to this client", method)
	}
	h, ok := l.reg.handlers[method]
	if !ok {
		return api.Errf(api.KindInvalid, "unknown method %q", method)
	}
	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return api.Errf(api.KindInvalid, "invalid parameters: %v", err)
		}
		raw = b
	}
	res, err := h(context.WithValue(ctx, kindKey{}, l.kind), raw)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

// permit decides whether this client kind may call a method: methods marked
// Agent are for both kinds, everything else is GUI-only.
func (l *Local) permit(method string) bool {
	for _, m := range api.Methods {
		if m.Name == method {
			return m.Agent || l.kind == ClientGUI
		}
	}
	return false
}
