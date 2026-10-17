// Package rpc is the wire protocol between the daemon and its clients. One
// JSON message per line over a local socket: {id, method, params} for a
// request, {id, result} or {id, error} for its reply, and {event, data} for a
// pushed notification.
package rpc

import "encoding/json"

// Message is one line on the wire. Only one of method/result/error/event is
// set. Error is a pointer so a failed call is distinguishable from a result
// that happens to be null.
type Message struct {
	ID     *int64          `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`

	Result json.RawMessage `json:"result,omitempty"`
	Error  *WireError      `json:"error,omitempty"`

	// Event is set on a pushed notification.
	Event string          `json:"event,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
}

// WireError is the failure half of a reply.
type WireError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Kind    string `json:"kind,omitempty"`
}

// Error implements the error interface so a reply can carry one.
func (e *WireError) Error() string { return e.Message }

// Wire error codes.
const (
	CodeInvalidParams = -32602
	CodeInternal      = -32603
	CodeMethodUnknown = -32601
	CodeDomain        = 1
)
