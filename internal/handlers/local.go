package handlers

import (
	"context"
	"encoding/json"

	"workspace-manager/internal/api"
	"workspace-manager/internal/ops"
)

// Local calls API methods in-process.
type Local struct {
	reg *Registry
}

// NewLocal builds the method table over o and checks it against the registry.
func NewLocal(o *ops.Ops) (*Local, error) {
	reg := (&Registry{Ops: o}).RegisterAll()
	if err := reg.CheckRegistration(); err != nil {
		return nil, err
	}
	return &Local{reg: reg}, nil
}

// Call runs one method and decodes its result into out (which may be nil).
func (l *Local) Call(method string, params, out any) error {
	return l.CallContext(context.Background(), method, params, out)
}

// CallContext is Call with a context for cancellable methods.
func (l *Local) CallContext(ctx context.Context, method string, params, out any) error {
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
	res, err := h(ctx, raw)
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
