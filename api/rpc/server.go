package rpc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"workspace-manager/api"
)

// Handler runs one method. params is the raw JSON the client sent; the result
// is marshalled back to the caller.
type Handler func(ctx context.Context, params json.RawMessage) (any, error)

// Hello tells the server a client has connected. The info selects the client's
// kind ("cli" or "gui"), which decides what it may call.
type Hello func(params json.RawMessage) (info api.ClientInfo, result any, err error)

// Permit decides whether a client kind may call a method. It is called for
// every request, so a client cannot slip past its kind.
type Permit func(kind, method string) bool

// Server accepts connections on a local socket and dispatches messages to
// registered handlers.
type Server struct {
	mu       sync.Mutex
	handlers map[string]Handler
	hello    Hello
	permit   Permit
	conns    map[*serverConn]struct{}
	ln       net.Listener
}

// NewServer returns an empty server. hello handles the handshake; permit may
// be nil, which allows every registered method.
func NewServer(hello Hello, permit Permit) *Server {
	return &Server{
		handlers: map[string]Handler{},
		hello:    hello,
		permit:   permit,
		conns:    map[*serverConn]struct{}{},
	}
}

// Register adds a handler for a method.
func (s *Server) Register(method string, h Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = h
}

// Handle satisfies Registerer, so the handler registry can wire a server.
func (s *Server) Handle(method string, h Handler) { s.Register(method, h) }

func (s *Server) handler(method string) (Handler, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, ok := s.handlers[method]
	return h, ok
}

// Listen opens the socket at path. An existing socket file is removed first:
// the data folder lock is what prevents two live servers, not this file.
func Listen(path string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	// A stale socket from a dead server would block the bind.
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, 0o600); err != nil {
			_ = ln.Close()
			return nil, err
		}
	}
	return ln, nil
}

// Serve accepts connections until ctx is cancelled or the listener closes.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	s.mu.Lock()
	s.ln = ln
	s.mu.Unlock()

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		nc, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		c := &serverConn{server: s, nc: nc, r: bufio.NewReader(nc), w: bufio.NewWriter(nc)}
		s.mu.Lock()
		s.conns[c] = struct{}{}
		s.mu.Unlock()
		go c.serve(ctx)
	}
}

// Broadcast pushes a notification to every connection whose filter accepts it.
func (s *Server) Broadcast(ev api.Event) {
	s.mu.Lock()
	conns := make([]*serverConn, 0, len(s.conns))
	for c := range s.conns {
		conns = append(conns, c)
	}
	s.mu.Unlock()
	for _, c := range conns {
		c.notify(ev)
	}
}

// Close drops every connection and closes the listener.
func (s *Server) Close() error {
	s.mu.Lock()
	conns := make([]*serverConn, 0, len(s.conns))
	for c := range s.conns {
		conns = append(conns, c)
	}
	ln := s.ln
	s.mu.Unlock()
	for _, c := range conns {
		c.close()
	}
	if ln != nil {
		return ln.Close()
	}
	return nil
}

// serverConn is one client connection.
type serverConn struct {
	server *Server
	nc     net.Conn
	r      *bufio.Reader

	mu     sync.Mutex
	w      *bufio.Writer
	closed bool
	kind   string // "cli" or "gui", set by hello
	filter api.SubscribeInput
}

// Kind reports the client kind reported by hello, empty before it.
func (c *serverConn) Kind() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.kind
}

func (c *serverConn) serve(ctx context.Context) {
	defer c.close()
	for {
		line, err := c.r.ReadBytes('\n')
		if len(line) > 0 {
			var msg Message
			if err := json.Unmarshal(bytesTrim(line), &msg); err == nil && msg.Method != "" {
				c.dispatch(ctx, msg)
			}
		}
		if err != nil {
			return
		}
	}
}

func (c *serverConn) dispatch(ctx context.Context, msg Message) {
	id := msg.ID
	reply := func(res any, err error) {
		out := Message{ID: id}
		if err != nil {
			code := CodeDomain
			var we *WireError
			if errors.As(err, &we) {
				code = we.Code
			}
			out.Error = &WireError{Code: code, Message: err.Error(), Kind: string(api.KindOf(err))}
		} else {
			raw, mErr := json.Marshal(res)
			if mErr != nil {
				out.Error = &WireError{Code: CodeInternal, Message: mErr.Error()}
			} else {
				out.Result = raw
			}
		}
		c.write(out)
	}

	if msg.Method == api.MethodHello {
		if c.server.hello == nil {
			reply(nil, errors.New("server has no handshake"))
			return
		}
		info, result, err := c.server.hello(msg.Params)
		if err == nil {
			c.mu.Lock()
			if info.Kind == "" {
				info.Kind = "gui"
			}
			c.kind = info.Kind
			c.mu.Unlock()
		}
		reply(result, err)
		return
	}

	if msg.Method == api.MethodSubscribe {
		var in api.SubscribeInput
		if len(msg.Params) > 0 {
			if err := json.Unmarshal(msg.Params, &in); err != nil {
				reply(nil, api.Errf(api.KindInvalid, "invalid subscribe filter: %v", err))
				return
			}
		}
		c.mu.Lock()
		c.filter = in
		c.mu.Unlock()
		reply(map[string]any{"ok": true}, nil)
		return
	}

	h, ok := c.server.handler(msg.Method)
	if !ok {
		reply(nil, &WireError{Code: CodeMethodUnknown, Message: "unknown method " + msg.Method})
		return
	}
	if c.server.permit != nil && !c.server.permit(c.Kind(), msg.Method) {
		reply(nil, api.Errf(api.KindUnavailable, "method %q is not available to this client", msg.Method))
		return
	}
	// The client kind travels in the context so a handler can tell a CLI
	// caller from the GUI, which decides whether AI-excluded values are shown.
	ctx = context.WithValue(ctx, ClientKindContextKey{}, c.Kind())
	res, err := h(ctx, msg.Params)
	reply(res, err)
}

// ClientKindContextKey carries the connecting client's kind in a request
// context value.
type ClientKindContextKey struct{}

func (c *serverConn) notify(ev api.Event) {
	c.mu.Lock()
	filter, closed := c.filter, c.closed
	c.mu.Unlock()
	if closed || !filter.Matches(ev) {
		return
	}
	raw, err := json.Marshal(ev)
	if err != nil {
		return
	}
	c.writeEvent(ev.Type, raw)
}

func (c *serverConn) write(msg Message) {
	line, err := json.Marshal(msg)
	if err != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.w == nil {
		return
	}
	_, _ = c.w.Write(append(line, '\n'))
	_ = c.w.Flush()
}

func (c *serverConn) writeEvent(t api.EventType, data json.RawMessage) {
	msg := Message{Event: string(t), Data: data}
	line, err := json.Marshal(msg)
	if err != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.w == nil {
		return
	}
	_, _ = c.w.Write(append(line, '\n'))
	_ = c.w.Flush()
}

func (c *serverConn) close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.mu.Unlock()
	_ = c.nc.Close()
	c.server.mu.Lock()
	delete(c.server.conns, c)
	c.server.mu.Unlock()
}
