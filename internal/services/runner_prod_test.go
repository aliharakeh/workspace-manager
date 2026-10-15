package services

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"workspace-manager/internal/api/types"
)

func TestRunnerProdReadinessAndReload(t *testing.T) {
	ctx := context.Background()
	d := newBlueprintTestDB(t)
	if err := d.EnsureReadyURLPatternsSeeded(ctx); err != nil {
		t.Fatal(err)
	}
	InvalidateReadyURLPatternsCache()
	t.Cleanup(InvalidateReadyURLPatternsCache)
	ws, err := d.CreateWorkspaceT(ctx, "ws", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	app, err := d.CreateAppT(ctx, ws.ID, "prod", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	set, err := d.ResolveActive(ctx, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.UpsertRunConfig(ctx, set.ID, types.KindDev, nil, []types.RunCommandInput{{Command: "echo dev-only"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.UpsertRunConfig(ctx, set.ID, types.KindProd, nil, []types.RunCommandInput{{Command: "echo Local: http://localhost:8080/"}}); err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(d, nil, nil)
	status, err := runner.Start(ctx, app.ID, types.KindProd)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if status.Kind != types.KindProd {
			t.Fatalf("kind=%q", status.Kind)
		}
		deadline := time.Now().Add(20 * time.Second)
		for runner.GetStatus(app.ID).Running {
			if time.Now().After(deadline) {
				_, _ = runner.Stop(ctx, app.ID)
				t.Fatal("prod did not finish")
			}
			time.Sleep(50 * time.Millisecond)
		}
		out := runner.GetOutput(app.ID, status.Processes[0].CommandID)
		raw, err := base64.StdEncoding.DecodeString(out.Data)
		if err != nil || !strings.Contains(string(raw), "Detected URL") || strings.Contains(string(raw), "dev-only") {
			t.Fatalf("prod output: %q, %v", raw, err)
		}
		if i == 0 {
			status, err = runner.Reload(ctx, app.ID)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}
