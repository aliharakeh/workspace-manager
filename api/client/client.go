// Package client is what the GUI and the MCP server use to talk to the daemon:
// dial, handshake, then call methods and receive events.
package client

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"workspace-manager/api"
	"workspace-manager/api/rpc"
)

// Options configure a connection.
type Options struct {
	// Kind is "agent" or "gui". It decides what the server allows and whether
	// AI-excluded values are shown. Empty means "agent".
	Kind string
	// Version is the client's build version, sent in hello.
	Version string
	// Socket overrides the default socket path.
	Socket string
	// Start, when set, is invoked if the socket is not reachable, so a client
	// can launch wmd on demand. Tests leave it nil.
	Start func() error
}

// Client is a connection to the daemon.
type Client struct {
	conn *rpc.Conn
	info api.ServerInfo
	kind string
}

// Dial connects, handshakes, and returns a ready client. When the socket is
// unreachable and opts.Start is set, it launches the daemon and retries.
func Dial(opts Options) (*Client, error) {
	kind := opts.Kind
	if kind == "" {
		// Least privilege: an unnamed client is an agent, not the GUI.
		kind = "agent"
	}
	socket := opts.Socket
	if socket == "" {
		socket = api.SocketPath()
	}

	conn, err := dialRetry(socket, opts.Start)
	if err != nil {
		return nil, err
	}

	var hello struct {
		OK     bool           `json:"ok"`
		Server api.ServerInfo `json:"server"`
	}
	if err := conn.Call(api.MethodHello, api.ClientInfo{Kind: kind, Version: opts.Version}, &hello); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &Client{conn: conn, info: hello.Server, kind: kind}, nil
}

func dialRetry(socket string, start func() error) (*rpc.Conn, error) {
	var lastErr error
	started := false
	for attempt := 0; attempt < 20; attempt++ {
		conn, err := rpc.Dial(socket)
		if err == nil {
			return conn, nil
		}
		lastErr = err
		if attempt >= 1 && start != nil && !started {
			started = true
			if err := start(); err != nil {
				return nil, err
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	return nil, api.Errf(api.KindUnavailable, "cannot reach the server at %s: %v", socket, lastErr)
}

// Info is the server's handshake reply.
func (c *Client) Info() api.ServerInfo { return c.info }

// Kind reports what the client told the server it is.
func (c *Client) Kind() string { return c.kind }

// Call runs a method and decodes the result into out.
func (c *Client) Call(method string, params, out any) error {
	return c.conn.Call(method, params, out)
}

// Subscribe registers the event handler and tells the server what to push.
// The handler receives each event's raw JSON payload, unchanged.
func (c *Client) Subscribe(filter api.SubscribeInput, handler func(ev api.Event)) error {
	c.conn.OnEvent(func(_ string, data json.RawMessage) {
		var ev api.Event
		if err := json.Unmarshal(data, &ev); err == nil {
			handler(ev)
		}
	})
	return c.conn.Call(api.MethodSubscribe, filter, nil)
}

// SubscribeRaw is Subscribe with raw payloads, for callers that decode the
// event body themselves (a runner status, a terminal chunk, AI progress).
func (c *Client) SubscribeRaw(filter api.SubscribeInput) (<-chan json.RawMessage, error) {
	ch := make(chan json.RawMessage, 64)
	c.conn.OnEvent(func(_ string, data json.RawMessage) {
		select {
		case ch <- append(json.RawMessage(nil), data...):
		default:
		}
	})
	if err := c.conn.Call(api.MethodSubscribe, filter, nil); err != nil {
		return nil, err
	}
	return ch, nil
}

// Done is closed when the connection to the server ends.
func (c *Client) Done() <-chan struct{} { return c.conn.Done() }

// Close disconnects.
func (c *Client) Close() error { return c.conn.Close() }

// StartDaemon returns a function that launches wmd detached, so it outlives the
// client that started it. The binary is looked up next to the current
// executable first, then on PATH. The daemon gets the client's data folder, so
// WM_DATA_DIR isolates both ends of the connection.
func StartDaemon() func() error {
	return func() error {
		exe, err := findDaemon()
		if err != nil {
			return err
		}
		cmd := exec.Command(exe, "daemon")
		cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
		cmd.Env = append(os.Environ(), "WM_DATA_DIR="+api.DataDir())
		detach(cmd)
		if err := cmd.Start(); err != nil {
			return err
		}
		go func() { _ = cmd.Wait() }()
		return nil
	}
}

// findDaemon locates the wmd binary: next to this executable, then on PATH.
// The sibling wins, because that is the build shipped with this client.
func findDaemon() (string, error) {
	self, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(self), "wmd"+exeSuffix())
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate, nil
		}
	}
	if p, lookErr := exec.LookPath("wmd" + exeSuffix()); lookErr == nil {
		return p, nil
	}
	return "", api.Errf(api.KindUnavailable, "wmd was not found next to this binary or on PATH")
}
