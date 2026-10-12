// Package daemon_test exercises the running server: the protocol, the runner,
// its events and the hidden-value rule, all through a real socket.
package daemon_test

import (
	"context"
	"encoding/json"
	"runtime"
	"strings"
	"testing"
	"time"

	"workspace-manager/api"
	"workspace-manager/api/client"
	"workspace-manager/api/types"
	"workspace-manager/server/daemon"
	"workspace-manager/server/db"
)

// startServer runs a daemon in the test process on a fresh data folder and
// returns a client connected to it.
func startServer(t *testing.T) *client.Client {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("WM_DATA_DIR", dir)
	t.Setenv("WM_DAEMON_IDLE", "2m")

	database, err := db.Open()
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	d, err := daemon.New(database, nil)
	if err != nil {
		t.Fatalf("new daemon: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- d.Run(ctx) }()
	t.Cleanup(func() {
		d.Stop()
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Errorf("the daemon did not stop")
		}
	})

	// Wait for the socket, then connect.
	var c *client.Client
	for i := 0; i < 50; i++ {
		c, err = client.Dial(client.Options{Kind: "cli", Version: "test"})
		if err == nil {
			t.Cleanup(func() { _ = c.Close() })
			return c
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("dial: %v", err)
	return nil
}

// makeApp creates a workspace and an app in the temp data folder.
func makeApp(t *testing.T, c *client.Client) types.App {
	t.Helper()
	var ws types.Workspace
	if err := c.Call(api.MethodWorkspacesCreate, types.WorkspaceCreateInput{Name: "test"}, &ws); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	proj := t.TempDir()
	var app types.App
	if err := c.Call(api.MethodAppsCreate, map[string]any{
		"workspaceID": ws.ID,
		"body":        types.AppCreateInput{Name: "app", ProjectPath: proj},
	}, &app); err != nil {
		t.Fatalf("create app: %v", err)
	}
	return app
}

// waitEvent reads raw events until match returns a value.
func waitEvent[V any](t *testing.T, events <-chan json.RawMessage, match func(json.RawMessage) (V, bool)) V {
	t.Helper()
	deadline := time.After(30 * time.Second)
	for {
		select {
		case raw := <-events:
			if v, ok := match(raw); ok {
				return v
			}
		case <-deadline:
			t.Fatal("timed out waiting for an event")
		}
	}
}

func TestRunnerSessionOverTheProtocol(t *testing.T) {
	// The command prints a ready URL and then stays up until it is stopped,
	// so the session is alive for the conflict check below.
	runCommand := "echo Local: http://localhost:9998"
	switch runtime.GOOS {
	case "windows":
		runCommand += " && timeout /t 60 /nobreak > nul"
	default:
		runCommand += " && sleep 60"
	}

	c := startServer(t)
	app := makeApp(t, c)

	// A command that prints a ready URL and exits.
	if err := c.Call(api.MethodRunConfigSave, map[string]any{
		"appID": app.ID,
		"kind":  types.KindRun,
		"body": types.RunConfigSaveInput{
			Commands: []types.RunCommandInput{{Command: runCommand}},
		},
	}, nil); err != nil {
		t.Fatalf("save run config: %v", err)
	}

	events, err := c.SubscribeRaw(api.SubscribeInput{
		Types: []api.EventType{api.EventRunner},
		Apps:  []int64{app.ID},
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	var started types.StatusEvent
	if err := c.Call(api.MethodRunnerRun, map[string]any{"appID": app.ID}, &started); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !started.Running {
		t.Fatalf("the session did not start: %+v", started)
	}

	// The runner detects the URL in the output and reports it on the next
	// status event.
	status := waitEvent(t, events, func(raw json.RawMessage) (types.StatusEvent, bool) {
		var re struct {
			AppID int64           `json:"appId"`
			Data  json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(raw, &re); err != nil {
			return types.StatusEvent{}, false
		}
		var ev types.StatusEvent
		if err := json.Unmarshal(re.Data, &ev); err != nil || ev.Type != "status" {
			return types.StatusEvent{}, false
		}
		for _, p := range ev.Processes {
			for _, u := range p.URLs {
				if strings.Contains(u, "9998") {
					return ev, true
				}
			}
		}
		return types.StatusEvent{}, false
	})
	if len(status.Processes) == 0 {
		t.Fatalf("no processes in the status event")
	}

	// A second run while the session is alive is a conflict.
	err = c.Call(api.MethodRunnerRun, map[string]any{"appID": app.ID}, nil)
	if kind := api.KindOf(err); kind != api.KindConflict {
		t.Fatalf("second run: want a conflict, got %v (%v)", kind, err)
	}

	// The output buffer holds what the command printed.
	var out types.RunnerOutput
	if err := c.Call(api.MethodRunnerOutput, map[string]any{
		"appID": app.ID, "commandID": started.Processes[0].CommandID,
	}, &out); err != nil {
		t.Fatalf("output: %v", err)
	}
	if out.Data == "" {
		t.Errorf("the output buffer is empty")
	}

	if err := c.Call(api.MethodRunnerStop, map[string]any{"appID": app.ID}, nil); err != nil {
		t.Fatalf("stop: %v", err)
	}
}

func TestCLIClientCannotSeeHiddenValues(t *testing.T) {
	c := startServer(t)
	app := makeApp(t, c)

	hidden := false
	if err := c.Call(api.MethodEnvVarsCreate, map[string]any{
		"appID": app.ID,
		"body":  types.EnvVarCreateInput{Key: "SECRET", IncludeInAI: &hidden},
	}, nil); err != nil {
		t.Fatalf("create hidden var: %v", err)
	}

	// The value is blanked for a CLI client.
	var vars []types.EnvVar
	if err := c.Call(api.MethodEnvVarsList, map[string]any{"appID": app.ID}, &vars); err != nil {
		t.Fatalf("list vars: %v", err)
	}
	for _, v := range vars {
		if v.Key == "SECRET" {
			if v.Value != "" {
				t.Errorf("a hidden value leaked to a CLI client: %q", v.Value)
			}
			return
		}
	}
	t.Fatalf("the hidden var is missing from the list")
}

func TestUnknownMethodFails(t *testing.T) {
	c := startServer(t)
	err := c.Call("nope.notAMethod", nil, nil)
	if err == nil {
		t.Fatal("an unknown method succeeded")
	}
}

func TestDataFolderIsolation(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WM_DATA_DIR", dir)
	if got := api.DataDir(); got != dir {
		t.Fatalf("WM_DATA_DIR is ignored: got %q want %q", got, dir)
	}
	if got := db.DataDir(); got != dir {
		t.Fatalf("the database ignores WM_DATA_DIR: got %q want %q", got, dir)
	}
}

