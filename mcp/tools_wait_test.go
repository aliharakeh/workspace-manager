package mcp

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"
)

func waitProcess(id int64, state string, urls ...string) types.ProcessState {
	code := int64(0)
	p := types.ProcessState{CommandID: id, Label: "command", Status: state, URLs: urls}
	if state == "exited" {
		p.ExitCode = &code
	}
	return p
}

func TestWaitMultipleProcesses(t *testing.T) {
	for _, tt := range []struct {
		name      string
		processes []types.ProcessState
		running   bool
		options   waitOptions
		reason    string
		urls      int
	}{
		{"wait for second URL", []types.ProcessState{waitProcess(1, "running", "http://localhost:3000"), waitProcess(2, "running")}, true, waitOptions{}, "", 1},
		{"all ready", []types.ProcessState{waitProcess(1, "running", "http://localhost:3000"), waitProcess(2, "running", "http://localhost:4000")}, true, waitOptions{}, "ready", 2},
		{"any ready", []types.ProcessState{waitProcess(1, "running", "http://localhost:3000"), waitProcess(2, "pending")}, true, waitOptions{Readiness: "any"}, "ready", 1},
		{"failure beats URL", []types.ProcessState{waitProcess(1, "running", "http://localhost:3000"), waitProcess(2, "error")}, true, waitOptions{}, "failed", 1},
		{"finished setup then serving", []types.ProcessState{waitProcess(1, "exited"), waitProcess(2, "running", "http://localhost:4000")}, true, waitOptions{}, "ready", 1},
		{"pending prevents readiness", []types.ProcessState{waitProcess(1, "running", "http://localhost:3000"), waitProcess(2, "pending")}, true, waitOptions{}, "", 1},
		{"selected ready ignores other failure", []types.ProcessState{waitProcess(1, "running", "http://localhost:3000"), waitProcess(2, "error")}, true, waitOptions{CommandIDs: []int64{1}}, "ready", 1},
		{"selected done while others run", []types.ProcessState{waitProcess(1, "exited"), waitProcess(2, "running", "http://localhost:4000")}, true, waitOptions{CommandIDs: []int64{1}, Until: "exit"}, "completed", 0},
		{"subset excludes later URL", []types.ProcessState{waitProcess(1, "running"), waitProcess(2, "running", "http://localhost:4000")}, true, waitOptions{CommandIDs: []int64{1}}, "", 0},
		{"exit waits past ready", []types.ProcessState{waitProcess(1, "running", "http://localhost:3000"), waitProcess(2, "running", "http://localhost:4000")}, true, waitOptions{Until: "exit"}, "", 2},
		{"one exit is not all done", []types.ProcessState{waitProcess(1, "exited"), waitProcess(2, "running")}, true, waitOptions{Until: "exit"}, "", 0},
		{"all done", []types.ProcessState{waitProcess(1, "exited"), waitProcess(2, "exited")}, false, waitOptions{Until: "exit"}, "completed", 0},
		{"ignore stale URL", []types.ProcessState{waitProcess(1, "exited", "http://localhost:3000")}, false, waitOptions{}, "completed", 0},
		{"intentional stop", []types.ProcessState{waitProcess(1, "killed"), waitProcess(2, "pending")}, false, waitOptions{}, "stopped", 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.options.validate(); err != nil {
				t.Fatal(err)
			}
			status := types.StatusEvent{AppID: 1, SessionID: "run", Running: tt.running, Processes: tt.processes}
			result, err := evaluateWait(status, tt.options)
			if err != nil || result.Reason != tt.reason || len(result.URLs) != tt.urls {
				t.Fatalf("got %+v, %v", result, err)
			}
			for _, url := range result.URLs {
				if url.CommandID == 0 || url.Label == "" {
					t.Fatal("URL lost process attribution")
				}
			}
		})
	}
}

func TestWaitValidationAndStartupFailure(t *testing.T) {
	for _, options := range []waitOptions{{TimeoutSeconds: -1}, {TimeoutSeconds: 61}, {Until: "invalid"}, {Readiness: "invalid"}, {CommandIDs: []int64{}}, {CommandIDs: []int64{0}}, {CommandIDs: []int64{1, 1}}} {
		if options.validate() == nil {
			t.Fatalf("accepted %+v", options)
		}
	}
	o := waitOptions{CommandIDs: []int64{99}}
	_ = o.validate()
	if _, err := evaluateWait(types.StatusEvent{Processes: []types.ProcessState{waitProcess(1, "running")}}, o); api.KindOf(err) != api.KindNotFound {
		t.Fatal("unknown command was accepted")
	}
	o = waitOptions{}
	_ = o.validate()
	result, err := evaluateWait(types.StatusEvent{Error: "template failure", Processes: []types.ProcessState{waitProcess(1, "pending")}}, o)
	if err != nil || result.Reason != "failed" {
		t.Fatalf("startup failure: %+v %v", result, err)
	}
	result, err = evaluateWait(types.StatusEvent{Error: "template restore failure", Processes: []types.ProcessState{waitProcess(1, "exited")}}, o)
	if err != nil || result.Reason != "failed" {
		t.Fatalf("cleanup failure: %+v %v", result, err)
	}
}

type fakeWaitConnection struct {
	mu      sync.Mutex
	status  types.StatusEvent
	handler func(api.Event)
	filter  api.SubscribeInput
	done    chan struct{}
	closed  sync.Once
	reads   int
}

func (c *fakeWaitConnection) Call(_ string, _ any, out any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	select {
	case <-c.done:
		return errors.New("closed")
	default:
	}
	c.reads++
	*out.(*types.StatusEvent) = c.status
	return nil
}
func (c *fakeWaitConnection) Subscribe(filter api.SubscribeInput, handler func(api.Event)) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.filter, c.handler = filter, handler
	return nil
}
func (c *fakeWaitConnection) Done() <-chan struct{} { return c.done }
func (c *fakeWaitConnection) Close() error          { c.closed.Do(func() { close(c.done) }); return nil }

func TestWaitEventsTimeoutAndReplacement(t *testing.T) {
	for _, reason := range []string{"ready", "completed", "timeout", "replaced", "disconnected"} {
		t.Run(reason, func(t *testing.T) {
			conn := &fakeWaitConnection{done: make(chan struct{}), status: types.StatusEvent{AppID: 1, SessionID: "first", Running: true, Processes: []types.ProcessState{waitProcess(1, "running"), waitProcess(2, "running")}}}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			defer conn.Close()
			options := waitOptions{}
			_ = options.validate()
			done := make(chan struct{})
			var result waitResult
			var err error
			go func() { result, err = waitForRun(ctx, conn, 1, options); close(done) }()
			// Wait until the initial snapshot has been read before changing state.
			deadline := time.Now().Add(time.Second)
			for {
				conn.mu.Lock()
				reads := conn.reads
				conn.mu.Unlock()
				if reads > 0 {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("wait never subscribed")
				}
				time.Sleep(time.Millisecond)
			}
			conn.mu.Lock()
			if len(conn.filter.Apps) != 1 || conn.filter.Apps[0] != 1 {
				t.Fatal("subscription is not scoped to app")
			}
			switch reason {
			case "ready":
				conn.status.Processes = []types.ProcessState{waitProcess(1, "running", "http://localhost:3000"), waitProcess(2, "running", "http://localhost:4000")}
			case "completed":
				conn.status.Running = false
				conn.status.Processes = []types.ProcessState{waitProcess(1, "exited"), waitProcess(2, "exited")}
			case "replaced":
				conn.status.SessionID = "second"
			}
			handler := conn.handler
			conn.mu.Unlock()
			if reason == "timeout" {
				cancel()
			} else if reason == "disconnected" {
				_ = conn.Close()
			} else {
				handler(api.Event{Type: api.EventRunner, App: 1, Data: map[string]any{"type": "status"}})
			}
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("wait did not return")
			}
			if reason == "replaced" {
				if api.KindOf(err) != api.KindConflict {
					t.Fatalf("replacement: %v", err)
				}
				return
			}
			if reason == "disconnected" {
				if api.KindOf(err) != api.KindUnavailable {
					t.Fatalf("disconnect: %v", err)
				}
				return
			}
			if err != nil || result.Reason != reason {
				t.Fatalf("got %+v %v", result, err)
			}
		})
	}
}
