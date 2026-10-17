package gui

import (
	"context"
	"log"
	"sync"

	"workspace-manager/shared/api"
	"workspace-manager/shared/client"
)

// Link is the GUI's connection to the wmd daemon. It dials (starting the daemon
// when it is not running) on first use and again after the connection drops, and
// subscribes to the daemon's events, which the App relays to the frontend.
type Link struct {
	mu      sync.Mutex
	c       *client.Client
	onEvent func(api.Event)
	// onConnect runs after every successful (re)connection but the first, so
	// the UI can reload what it may have missed while disconnected.
	onConnect func()
	first     bool
}

func newLink(onEvent func(api.Event), onConnect func()) *Link {
	return &Link{onEvent: onEvent, onConnect: onConnect, first: true}
}

// client returns the live connection, dialing when there is none.
func (l *Link) client() (*client.Client, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.c != nil {
		select {
		case <-l.c.Done():
			l.c = nil
		default:
			return l.c, nil
		}
	}
	c, err := client.Dial(client.Options{
		Kind:    "gui",
		Version: api.Version,
		Start:   client.StartDaemon(),
	})
	if err != nil {
		return nil, err
	}
	if err := c.Subscribe(api.SubscribeInput{}, l.onEvent); err != nil {
		_ = c.Close()
		return nil, err
	}
	l.c = c
	if l.first {
		l.first = false
		log.Printf("workspace-manager: connected to wmd %s", c.Info().Version)
	} else if l.onConnect != nil {
		go l.onConnect()
	}
	return c, nil
}

// Call runs a daemon method and decodes the result into out.
func (l *Link) Call(method string, params, out any) error {
	c, err := l.client()
	if err != nil {
		return err
	}
	return c.Call(method, params, out)
}

// Close drops the connection. The daemon keeps running, and with it the apps.
func (l *Link) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.c != nil {
		_ = l.c.Close()
		l.c = nil
	}
}

// call is Link.Call with the result type as a parameter.
func call[T any](l *Link, method string, params any) (T, error) {
	var out T
	err := l.Call(method, params, &out)
	return out, err
}

// ctxOrBackground returns ctx, or a background context before startup.
func ctxOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
