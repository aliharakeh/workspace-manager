package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"workspace-manager/daemon/db"
	"workspace-manager/shared/api/types"
)

func TestEnvProviderHelper(t *testing.T) {
	if os.Getenv("WM_PROVIDER_HELPER") != "1" {
		return
	}
	if os.Getenv("WM_PROVIDER_SLEEP") == "1" {
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"LOCAL": os.Getenv("TOKEN") + "/local", "TOKEN": "app-override", "new.output-key": "new", "SHARED_INPUT": os.Getenv("UNLINKED")})
	os.Exit(0)
}
func TestEnvProviderJSON(t *testing.T) {
	dir := t.TempDir()
	command := "cat output.json"
	if runtime.GOOS == "windows" {
		command = "type output.json"
	}
	cases := []struct {
		raw string
		ok  bool
	}{
		{`{"TOKEN":"fresh"}`, true}, {`{"TOKEN":""}`, true}, {`{"TOKEN":null}`, false}, {`{"TOKEN":1}`, false}, {`{"TOKEN":"one","TOKEN":"two"}`, true}, {`{}`, true}, {`{"OTHER":"x"}`, true}, {`{"new.output-key":"x"}`, true}, {`{"TOKEN":"x"} {}`, false}, {`[]`, false},
	}
	for _, c := range cases {
		t.Run(c.raw, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(dir, "output.json"), []byte(c.raw), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := runEnvProvider(context.Background(), types.EnvProvider{Command: command, Timeout: 5}, dir, nil)
			if (err == nil) != c.ok {
				t.Fatalf("success=%v, error=%v", c.ok, err)
			}
		})
	}
}
func TestResolveSharedThenLocal(t *testing.T) {
	ctx := context.Background()
	d := newBlueprintTestDB(t)
	ws, _ := d.CreateWorkspaceT(ctx, "ws", nil, nil)
	dir := t.TempDir()
	app, err := d.CreateAppT(ctx, ws.ID, "app", dir)
	if err != nil {
		t.Fatal(err)
	}
	set, _ := d.ResolveActive(ctx, app.ID)
	res, err := d.SQL.Exec(`INSERT INTO workspace_config_sets(workspace_id,name) VALUES (?, 'shared')`, ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	sharedID, _ := res.LastInsertId()
	res, err = d.SQL.Exec(`INSERT INTO workspace_env_vars(workspace_config_set_id,key,value) VALUES (?, 'TOKEN','old')`, sharedID)
	if err != nil {
		t.Fatal(err)
	}
	varID, _ := res.LastInsertId()
	if _, err = d.SQL.Exec(`UPDATE config_sets SET shared_set_id=? WHERE id=?`, sharedID, set.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = d.SQL.Exec(`INSERT INTO env_vars(config_set_id,key,value,shared_var_id) VALUES (?, 'TOKEN','old',?)`, set.ID, varID); err != nil {
		t.Fatal(err)
	}
	if _, err = d.UpsertEnvVarByKey(ctx, set.ID, "LOCAL", "old-local", true); err != nil {
		t.Fatal(err)
	}
	command := "cat shared.json"
	if runtime.GOOS == "windows" {
		command = "type shared.json"
	}
	if err = os.WriteFile(filepath.Join(dir, "shared.json"), []byte(`{"TOKEN":"fresh","UNLINKED":"shared-only"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err = d.SaveEnvProvider(ctx, true, sharedID, types.EnvProvider{Command: command, Directory: dir, Timeout: 5}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WM_PROVIDER_HELPER", "1")
	localCommand := fmt.Sprintf(`"%s" -test.run=^TestEnvProviderHelper$`, os.Args[0])
	if err = d.SaveEnvProvider(ctx, false, set.ID, types.EnvProvider{Command: localCommand, Timeout: 5}); err != nil {
		t.Fatal(err)
	}
	providers := NewEnvProviders(d)
	if _, e := providers.Resolve(ctx, set.ID, dir); e == nil {
		t.Fatal("start must require a manual refresh")
	}
	if e := providers.Run(ctx, true, sharedID); e != nil {
		t.Fatal(e)
	}
	if e := providers.Run(ctx, false, set.ID); e != nil {
		t.Fatal(e)
	}
	env, err := providers.Resolve(ctx, set.ID, dir)
	if err != nil {
		t.Fatal(err)
	}
	if env["TOKEN"] != "app-override" || env["LOCAL"] != "fresh/local" || env["new.output-key"] != "new" || env["SHARED_INPUT"] != "shared-only" || env["UNLINKED"] != "shared-only" {
		t.Fatal("wrong resolved values")
	}
	if _, e := NewEnvProviders(d).Resolve(ctx, set.ID, dir); e == nil {
		t.Fatal("new daemon instance reused private results")
	}
	stored, _ := d.EnvToRecord(ctx, set.ID)
	if stored["TOKEN"] != "old" || stored["LOCAL"] != "old-local" {
		t.Fatal("resolution persisted values")
	}
	// Reload resolution gets a new shared token.
	_ = os.WriteFile(filepath.Join(dir, "shared.json"), []byte(`{"TOKEN":"new"}`), 0600)
	env, err = providers.Resolve(ctx, set.ID, dir)
	if err != nil || env["LOCAL"] != "fresh/local" {
		t.Fatal("start executed script automatically")
	}
	if e := providers.Run(ctx, true, sharedID); e != nil {
		t.Fatal(e)
	}
	if e := providers.Run(ctx, false, set.ID); e != nil {
		t.Fatal(e)
	}
	env, err = providers.Resolve(ctx, set.ID, dir)
	if err != nil || env["LOCAL"] != "new/local" {
		t.Fatalf("fresh resolution failed: %v", err)
	}
	t.Setenv("WM_PROVIDER_SLEEP", "1")
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()
	_, err = runEnvProvider(cancelCtx, types.EnvProvider{Command: localCommand, Timeout: 5}, dir, nil)
	if err == nil || !strings.Contains(err.Error(), "stopped") {
		t.Fatalf("cancellation: %v", err)
	}
}

func TestRunnerDynamicTemplatesAndFailure(t *testing.T) {
	ctx := context.Background()
	d := newBlueprintTestDB(t)
	ws, _ := d.CreateWorkspaceT(ctx, "ws", nil, nil)
	dir := t.TempDir()
	app, err := d.CreateAppT(ctx, ws.ID, "app", dir)
	if err != nil {
		t.Fatal(err)
	}
	set, _ := d.ResolveActive(ctx, app.ID)
	_, err = d.UpsertEnvVarByKey(ctx, set.ID, "TOKEN", "old", true)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "config.txt")
	_ = os.WriteFile(target, []byte("original"), 0600)
	_, err = d.CreateTemplate(ctx, db.CreateTemplateParams{ConfigSetID: set.ID, FilePath: "config.txt", Content: "{{TOKEN}}"})
	if err != nil {
		t.Fatal(err)
	}
	script := "cat output.json"
	command := "printf '%s' \"$TOKEN\" > env.txt; cp config.txt rendered.txt"
	if runtime.GOOS == "windows" {
		script = "type output.json"
		command = "echo %TOKEN%>env.txt & copy /y config.txt rendered.txt"
	}
	_, err = d.UpsertRunConfig(ctx, set.ID, types.KindBuild, nil, []types.RunCommandInput{{Command: command}})
	if err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "output.json"), []byte(`{"TOKEN":"fresh"}`), 0600)
	err = d.SaveEnvProvider(ctx, false, set.ID, types.EnvProvider{Command: script, Timeout: 5})
	if err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(d, nil, nil)
	if e := runner.Providers.Run(ctx, false, set.ID); e != nil {
		t.Fatal(e)
	}
	run := func() {
		t.Helper()
		if _, e := runner.Start(ctx, app.ID, types.KindBuild); e != nil {
			t.Fatal(e)
		}
		deadline := time.Now().Add(10 * time.Second)
		for runner.GetStatus(app.ID).Running {
			if time.Now().After(deadline) {
				t.Fatal("run did not finish")
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	run()
	for _, file := range []string{"env.txt", "rendered.txt"} {
		b, e := os.ReadFile(filepath.Join(dir, file))
		if e != nil || strings.TrimSpace(string(b)) != "fresh" {
			t.Fatalf("%s did not receive resolved value: %v", file, e)
		}
	}
	b, _ := os.ReadFile(target)
	if string(b) != "original" {
		t.Fatal("template not restored")
	}
	_ = os.Remove(filepath.Join(dir, "env.txt"))
	_ = os.WriteFile(filepath.Join(dir, "output.json"), []byte(`invalid SECRET`), 0600)
	if e := runner.Providers.Run(ctx, false, set.ID); e == nil {
		t.Fatal("invalid provider succeeded")
	}
	run()
	if b, e := os.ReadFile(filepath.Join(dir, "env.txt")); e != nil || strings.TrimSpace(string(b)) != "fresh" {
		t.Fatal("failed refresh discarded previous successful result")
	}
	// Configuration changes invalidate the retained result.
	p, _ := d.EnvProvider(ctx, false, set.ID)
	p.Command = "invalid-command"
	_ = d.SaveEnvProvider(ctx, false, set.ID, p)
	_ = os.Remove(filepath.Join(dir, "env.txt"))
	run()
	if _, e := os.Stat(filepath.Join(dir, "env.txt")); !os.IsNotExist(e) {
		t.Fatal("app ran with result from a different provider configuration")
	}

	b, _ = os.ReadFile(target)
	if string(b) != "original" {
		t.Fatal("failed provider changed template")
	}
	status := runner.GetStatus(app.ID)
	raw, _ := base64.StdEncoding.DecodeString(runner.GetOutput(app.ID, status.Processes[0].CommandID).Data)
	if strings.Contains(string(raw), "SECRET") {
		t.Fatal("provider output leaked into terminal buffer")
	}

}

func TestEnvProviderTimeout(t *testing.T) {
	t.Setenv("WM_PROVIDER_HELPER", "1")
	t.Setenv("WM_PROVIDER_SLEEP", "1")
	command := fmt.Sprintf(`"%s" -test.run=^TestEnvProviderHelper$`, os.Args[0])
	start := time.Now()
	_, err := runEnvProvider(context.Background(), types.EnvProvider{Command: command, Timeout: 1}, t.TempDir(), nil)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout: %v", err)
	}
	if time.Since(start) > 10*time.Second {
		t.Fatal("timeout did not terminate provider tree promptly")
	}
}

func TestEnvProviderStop(t *testing.T) {
	ctx := context.Background()
	d := newBlueprintTestDB(t)
	ws, _ := d.CreateWorkspaceT(ctx, "ws", nil, nil)
	app, err := d.CreateAppT(ctx, ws.ID, "app", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	set, _ := d.ResolveActive(ctx, app.ID)
	t.Setenv("WM_PROVIDER_HELPER", "1")
	t.Setenv("WM_PROVIDER_SLEEP", "1")
	command := fmt.Sprintf(`"%s" -test.run=^TestEnvProviderHelper$`, os.Args[0])
	if err = d.SaveEnvProvider(ctx, false, set.ID, types.EnvProvider{Command: command, Timeout: 60}); err != nil {
		t.Fatal(err)
	}
	providers := NewEnvProviders(d)
	if providers.Stop() {
		t.Fatal("Stop reported a script while none was running")
	}
	done := make(chan error, 1)
	go func() { done <- providers.Run(ctx, false, set.ID) }()
	deadline := time.Now().Add(5 * time.Second)
	for !providers.Stop() {
		if time.Now().After(deadline) {
			t.Fatal("script never started")
		}
		time.Sleep(50 * time.Millisecond)
	}
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "stopped") {
			t.Fatalf("expected a stopped script: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Stop did not end the script promptly")
	}
	if len(providers.Values(false, set.ID)) != 0 {
		t.Fatal("a stopped script saved values")
	}
}
