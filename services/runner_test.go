package services

import (
	"context"
	"encoding/base64"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"workspace-manager/types"
)

func TestRunnerStreamsTerminalOutput(t *testing.T) {
	ctx := context.Background()
	d := newBlueprintTestDB(t)
	if err := d.EnsureReadyURLPatternsSeeded(ctx); err != nil {
		t.Fatal(err)
	}
	InvalidateReadyURLPatternsCache()
	t.Cleanup(InvalidateReadyURLPatternsCache)

	ws, _ := d.CreateWorkspaceT(ctx, "ws", nil)
	app, err := d.CreateAppT(ctx, ws.ID, "demo", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	set, err := d.ResolveActive(ctx, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	command := `echo   Local:   http://localhost:5173/ & echo second line`
	if runtime.GOOS != "windows" {
		command = `echo "  Local:   http://localhost:5173/"; echo second line`
	}
	mode := "sequential"
	if _, err := d.UpsertRunConfig(ctx, set.ID, &mode, []types.RunCommandInput{{Command: command}}); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var events []types.LogEvent
	runner := NewRunner(d, func(_ int64, ev any) {
		if log, ok := ev.(types.LogEvent); ok {
			mu.Lock()
			events = append(events, log)
			mu.Unlock()
		}
	}, nil)
	status, err := runner.Start(ctx, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	commandID := status.Processes[0].CommandID
	deadline := time.Now().Add(20 * time.Second)
	for runner.GetStatus(app.ID).Running {
		if time.Now().After(deadline) {
			t.Fatal("run did not finish")
		}
		time.Sleep(50 * time.Millisecond)
	}

	out := runner.GetOutput(app.ID, commandID)
	raw, err := base64.StdEncoding.DecodeString(out.Data)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{"second line", "Detected URL", "http://localhost:5173/", "Process exited with code 0"} {
		if !strings.Contains(text, want) {
			t.Fatalf("output is missing %q:\n%q", want, text)
		}
	}
	if out.End != int64(len(raw)) || out.SessionID == "" {
		t.Fatalf("end=%d len=%d session=%q", out.End, len(raw), out.SessionID)
	}

	// The live events rebuild exactly what a late UI gets from the snapshot.
	mu.Lock()
	defer mu.Unlock()
	var rebuilt []byte
	for _, ev := range events {
		chunk, err := base64.StdEncoding.DecodeString(ev.Data)
		if err != nil {
			t.Fatal(err)
		}
		if ev.Offset != int64(len(rebuilt)) {
			t.Fatalf("event offset %d, expected %d", ev.Offset, len(rebuilt))
		}
		rebuilt = append(rebuilt, chunk...)
	}
	if string(rebuilt) != text {
		t.Fatalf("events and snapshot differ:\n%q\n%q", rebuilt, text)
	}
}
