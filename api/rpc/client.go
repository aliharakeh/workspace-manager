// Package rpc is the wire protocol between the daemon and its clients. One
// JSON message per line over a local socket: {id, method, params} for a
// request, {id, result} or {id, error} for its reply, and {event, data} for a
// pushed notification.
package rpc

import (
	"bufio"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"time"
)

// Error is a failed call as the server reported it.
type Error struct {
	Code    int
	Message string
	Kind    string
}

func (e *Error) Error() string { return e.Message }

// ErrorKind exposes the server's error kind so api.KindOf can map it to an
// exit code.
func (e *Error) ErrorKind() string { return e.Kind }

// Conn is a client connection to the daemon.
type Conn struct {
	nc net.Conn

	mu      sync.Mutex
	w       *bufio.Writer
	nextID  int64
	pending map[int64]chan Message
	events  func(event string, data json.RawMessage)
	closed  bool
	done    chan struct{}
	// eventCh hands notifications to a dedicated dispatcher goroutine, so an
	// event handler may call back into the connection. Handling events on the
	// read loop would deadlock any handler that makes a call, because the
	// reply could only be read by the loop it is blocking.
	eventCh chan notification
}

type notification struct {
	event string
	data  json.RawMessage
}

// Dial connects to the socket at path. The caller should send hello before
// other calls.
func Dial(path string) (*Conn, error) {
	nc, err := net.DialTimeout("unix", path, 5*time.Second)
	if err != nil {
		return nil, err
	}
	c := &Conn{
		nc:      nc,
		w:       bufio.NewWriter(nc),
		pending: map[int64]chan Message{},
		done:    make(chan struct{}),
		eventCh: make(chan notification, 1024),
	}
	go c.readLoop()
	go c.dispatchEvents()
	return c, nil
}

// OnEvent registers the handler for pushed notifications. It must be set
// before calls that produce events.
func (c *Conn) OnEvent(fn func(event string, data json.RawMessage)) {
	c.mu.Lock()
	c.events = fn
	c.mu.Unlock()
}

func (c *Conn) readLoop() {
	r := bufio.NewReader(c.nc)
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 {
			msg := Message{}
			if err := json.Unmarshal(bytesTrim(line), &msg); err == nil {
				if msg.Event != "" {
					select {
					case c.eventCh <- notification{event: msg.Event, data: msg.Data}:
					case <-c.done:
						return
					}
					continue
				}
				if msg.ID != nil {
					c.mu.Lock()
					ch := c.pending[*msg.ID]
					delete(c.pending, *msg.ID)
					c.mu.Unlock()
					if ch != nil {
						ch <- msg
						close(ch)
					}
				}
			}
		}
		if err != nil {
			c.fail(errors.New("connection to the server closed"))
			return
		}
	}
}

// dispatchEvents delivers notifications in order on its own goroutine.
func (c *Conn) dispatchEvents() {
	for {
		select {
		case <-c.done:
			return
		case n := <-c.eventCh:
			c.mu.Lock()
			fn := c.events
			c.mu.Unlock()
			if fn != nil {
				fn(n.event, n.data)
			}
		}
	}
}

// fail answers every pending call and marks the connection closed.
func (c *Conn) fail(err error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	close(c.done)
	pending := c.pending
	c.pending = map[int64]chan Message{}
	c.mu.Unlock()
	for _, ch := range pending {
		ch <- Message{Error: &WireError{Code: CodeInternal, Message: err.Error()}}
		close(ch)
	}
}

// Done is closed when the connection fails or is closed.
func (c *Conn) Done() <-chan struct{} { return c.done }

// Call runs a method and decodes the result into out.
func (c *Conn) Call(method string, params, out any) error {
	msg, err := c.call(method, params)
	if err != nil {
		return err
	}
	if out == nil || len(msg.Result) == 0 {
		return nil
	}
	return json.Unmarshal(msg.Result, out)
}

func (c *Conn) call(method string, params any) (Message, error) {
	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return Message{}, err
		}
		raw = b
	}

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return Message{}, errors.New("connection to the server closed")
	}
	c.nextID++
	id := c.nextID
	ch := make(chan Message, 1)
	c.pending[id] = ch
	line, err := json.Marshal(Message{ID: &id, Method: method, Params: raw})
	if err == nil {
		_, err = c.w.Write(append(line, '\n'))
	}
	if err == nil {
		err = c.w.Flush()
	}
	if err != nil {
		delete(c.pending, id)
	}
	c.mu.Unlock()
	if err != nil {
		return Message{}, err
	}

	msg, ok := <-ch
	if !ok {
		return Message{}, errors.New("connection to the server closed")
	}
	if msg.Error != nil {
		return Message{}, &Error{Code: msg.Error.Code, Message: msg.Error.Message, Kind: msg.Error.Kind}
	}
	return msg, nil
}

// Close closes the connection.
func (c *Conn) Close() error {
	c.fail(errors.New("connection closed"))
	return c.nc.Close()
}

func bytesTrim(b []byte) []byte {
	out := b
	for len(out) > 0 && (out[len(out)-1] == '\n' || out[len(out)-1] == '\r') {
		out = out[:len(out)-1]
	}
	return out
}
