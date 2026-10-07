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

	ws, _ := d.CreateWorkspaceT(ctx, "ws", nil, nil)
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
	if _, err := d.UpsertRunConfig(ctx, set.ID, types.KindRun, &mode, []types.RunCommandInput{{Command: command}}); err != nil {
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
	status, err := runner.Start(ctx, app.ID, types.KindRun)
	if err != nil {
		t.Fatal(err)
	}
	if status.Kind != types.KindRun {
		t.Fatalf("kind=%q", status.Kind)
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

func TestRunnerBuildRunsBuildCommands(t *testing.T) {
	ctx := context.Background()
	d := newBlueprintTestDB(t)
	if err := d.EnsureReadyURLPatternsSeeded(ctx); err != nil {
		t.Fatal(err)
	}
	InvalidateReadyURLPatternsCache()
	t.Cleanup(InvalidateReadyURLPatternsCache)

	ws, _ := d.CreateWorkspaceT(ctx, "ws", nil, nil)
	app, err := d.CreateAppT(ctx, ws.ID, "demo", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	set, err := d.ResolveActive(ctx, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(d, nil, nil)

	if _, err := runner.Start(ctx, app.ID, types.KindBuild); err == nil || !strings.Contains(err.Error(), "No build commands configured") {
		t.Fatalf("expected a missing build commands error, got %v", err)
	}

	// The run command must not be what a build executes.
	if _, err := d.UpsertRunConfig(ctx, set.ID, types.KindRun, nil, []types.RunCommandInput{{Command: "echo run-only"}}); err != nil {
		t.Fatal(err)
	}
	command := `echo   built   http://localhost:5173/`
	if runtime.GOOS != "windows" {
		command = `echo "built http://localhost:5173/"`
	}
	if _, err := d.UpsertRunConfig(ctx, set.ID, types.KindBuild, nil, []types.RunCommandInput{{Command: command}}); err != nil {
		t.Fatal(err)
	}

	status, err := runner.Start(ctx, app.ID, types.KindBuild)
	if err != nil {
		t.Fatal(err)
	}
	if status.Kind != types.KindBuild || len(status.Processes) != 1 || status.Processes[0].Command != command {
		t.Fatalf("status: %+v", status)
	}
	if _, err := runner.Start(ctx, app.ID, types.KindRun); err == nil || !strings.Contains(err.Error(), "already building") {
		t.Fatalf("expected a run to be refused during a build, got %v", err)
	}
	commandID := status.Processes[0].CommandID
	deadline := time.Now().Add(20 * time.Second)
	for runner.GetStatus(app.ID).Running {
		if time.Now().After(deadline) {
			t.Fatal("build did not finish")
		}
		time.Sleep(50 * time.Millisecond)
	}

	raw, err := base64.StdEncoding.DecodeString(runner.GetOutput(app.ID, commandID).Data)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, "built") || !strings.Contains(text, "Process exited with code 0") {
		t.Fatalf("unexpected build output:\n%q", text)
	}
	if strings.Contains(text, "Detected URL") || strings.Contains(text, "run-only") {
		t.Fatalf("a build must not detect URLs or run the run commands:\n%q", text)
	}
}

func TestRunnerSetupRunsSetupCommands(t *testing.T) {
	ctx := context.Background()
	d := newBlueprintTestDB(t)
	ws, _ := d.CreateWorkspaceT(ctx, "ws", nil, nil)
	app, err := d.CreateAppT(ctx, ws.ID, "demo", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	set, err := d.ResolveActive(ctx, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(d, nil, nil)

	if _, err := runner.Start(ctx, app.ID, types.KindSetup); err == nil || !strings.Contains(err.Error(), "No setup commands configured") {
		t.Fatalf("expected a missing setup commands error, got %v", err)
	}

	if _, err := d.UpsertRunConfig(ctx, set.ID, types.KindBuild, nil, []types.RunCommandInput{{Command: "echo build-only"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.UpsertRunConfig(ctx, set.ID, types.KindSetup, nil, []types.RunCommandInput{{Command: "echo set-up-done"}}); err != nil {
		t.Fatal(err)
	}
	status, err := runner.Start(ctx, app.ID, types.KindSetup)
	if err != nil {
		t.Fatal(err)
	}
	if status.Kind != types.KindSetup || len(status.Processes) != 1 || status.Processes[0].Command != "echo set-up-done" {
		t.Fatalf("status: %+v", status)
	}
	if _, err := runner.Start(ctx, app.ID, types.KindRun); err == nil || !strings.Contains(err.Error(), "being set up") {
		t.Fatalf("expected a run to be refused during a setup, got %v", err)
	}
	commandID := status.Processes[0].CommandID
	deadline := time.Now().Add(20 * time.Second)
	for runner.GetStatus(app.ID).Running {
		if time.Now().After(deadline) {
			t.Fatal("setup did not finish")
		}
		time.Sleep(50 * time.Millisecond)
	}
	raw, err := base64.StdEncoding.DecodeString(runner.GetOutput(app.ID, commandID).Data)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, "set-up-done") || strings.Contains(text, "build-only") {
		t.Fatalf("unexpected setup output:\n%q", text)
	}
}
