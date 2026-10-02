package services

import (
	"context"
	"encoding/base64"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"workspace-manager/db"
	"workspace-manager/types"
)

func TestRenderBlueprintCommand(t *testing.T) {
	vars := map[string]string{"app_name": "My App", "folder_name": "my-app", "app_dir": `C:\p\my-app`}
	got := renderBlueprintCommand(`bun create vite {{folder_name}} && echo "{{ app_name }}" {{app_dir}} {{other}}`, vars)
	want := `bun create vite my-app && echo "My App" C:\p\my-app {{other}}`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestValidateFolderName(t *testing.T) {
	for _, bad := range []string{"", "  ", ".", "..", "a/b", `a\b`, "a:b", "a*", "dot."} {
		if _, err := validateFolderName(bad); err == nil {
			t.Errorf("expected %q to be rejected", bad)
		}
	}
	got, err := validateFolderName("  my-app ")
	if err != nil || got != "my-app" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func newBlueprintTestDB(t *testing.T) *db.DB {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", "file:"+filepath.ToSlash(filepath.Join(t.TempDir(), "t.sqlite"))+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	schema, err := os.ReadFile("../db/schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(string(schema)); err != nil {
		t.Fatal(err)
	}
	return &db.DB{SQL: sqlDB, Queries: db.New(sqlDB)}
}

func TestBlueprintRun(t *testing.T) {
	ctx := context.Background()
	d := newBlueprintTestDB(t)
	ws, err := d.CreateWorkspaceT(ctx, "ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	var logs []string
	runner := NewBlueprintRunner(d, func(e types.BlueprintLogEvent) { logs = append(logs, e.Text) })
	count := 0
	mk := func(createFolder bool, cmds ...string) types.Blueprint {
		count++
		in := types.BlueprintInput{Name: fmt.Sprintf("bp%d", count), CreateFolder: createFolder}
		for _, c := range cmds {
			in.Commands = append(in.Commands, types.BlueprintCommand{Command: c})
		}
		bp, err := d.CreateBlueprintT(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		return bp
	}
	_, gitErr := exec.LookPath("git")

	t.Run("create folder runs commands inside it, inits git, creates app", func(t *testing.T) {
		parent := t.TempDir()
		bp := mk(true, "echo hi> marker.txt")
		res, err := runner.Run(ctx, types.BlueprintRunInput{
			WorkspaceID: ws.ID, RunID: "r1", BlueprintID: bp.ID, Name: "Demo", ParentPath: parent, FolderName: "demo", CreateFolder: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(parent, "demo")
		if _, err := os.Stat(filepath.Join(dir, "marker.txt")); err != nil {
			t.Fatalf("command did not run inside the app folder: %v", err)
		}
		if gitErr == nil {
			if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
				t.Fatalf("git init did not run: %v", err)
			}
		}
		if res.App.Name != "Demo" || res.App.ProjectPath != dir || res.App.WorkspaceID != ws.ID {
			t.Fatalf("unexpected app: %+v", res.App)
		}
	})

	t.Run("without folder the command creates it in the parent", func(t *testing.T) {
		parent := t.TempDir()
		bp := mk(false, "mkdir {{folder_name}}")
		res, err := runner.Run(ctx, types.BlueprintRunInput{
			WorkspaceID: ws.ID, RunID: "r2", BlueprintID: bp.ID, Name: "Other", ParentPath: parent, FolderName: "other", CreateFolder: false,
		})
		if err != nil {
			t.Fatal(err)
		}
		if res.App.ProjectPath != filepath.Join(parent, "other") {
			t.Fatalf("unexpected path: %s", res.App.ProjectPath)
		}
	})

	t.Run("fails when the command never creates the folder", func(t *testing.T) {
		parent := t.TempDir()
		bp := mk(false, "echo nothing")
		_, err := runner.Run(ctx, types.BlueprintRunInput{
			WorkspaceID: ws.ID, RunID: "r3", BlueprintID: bp.ID, Name: "X", ParentPath: parent, FolderName: "x",
		})
		if err == nil || !strings.Contains(err.Error(), "not created") {
			t.Fatalf("expected not-created error, got %v", err)
		}
	})

	t.Run("failing command keeps the folder and creates no app", func(t *testing.T) {
		parent := t.TempDir()
		bp := mk(true, "exit 3")
		_, err := runner.Run(ctx, types.BlueprintRunInput{
			WorkspaceID: ws.ID, RunID: "r4", BlueprintID: bp.ID, Name: "Bad", ParentPath: parent, FolderName: "bad", CreateFolder: true,
		})
		if err == nil || !strings.Contains(err.Error(), "exit code 3") {
			t.Fatalf("expected exit code error, got %v", err)
		}
		if _, err := os.Stat(filepath.Join(parent, "bad")); err != nil {
			t.Fatalf("folder should be kept: %v", err)
		}
		apps, _ := d.ListAppsByWorkspaceT(ctx, ws.ID)
		for _, a := range apps {
			if a.Name == "Bad" {
				t.Fatal("app must not be created on failure")
			}
		}
	})

	t.Run("one blueprint builds apps in different workspaces", func(t *testing.T) {
		other, err := d.CreateWorkspaceT(ctx, "other", nil)
		if err != nil {
			t.Fatal(err)
		}
		bp := mk(true, "echo hi> marker.txt")
		for i, target := range []int64{ws.ID, other.ID} {
			res, err := runner.Run(ctx, types.BlueprintRunInput{
				WorkspaceID: target, RunID: fmt.Sprintf("multi%d", i), BlueprintID: bp.ID,
				Name: fmt.Sprintf("Multi %d", i), ParentPath: t.TempDir(), FolderName: "multi", CreateFolder: true,
			})
			if err != nil {
				t.Fatal(err)
			}
			if res.App.WorkspaceID != target {
				t.Fatalf("app landed in workspace %d, want %d", res.App.WorkspaceID, target)
			}
		}
	})

	t.Run("rejects an unknown workspace before touching disk", func(t *testing.T) {
		parent := t.TempDir()
		bp := mk(true, "echo hi")
		_, err := runner.Run(ctx, types.BlueprintRunInput{
			WorkspaceID: 9999, RunID: "r8", BlueprintID: bp.ID, Name: "Ghost", ParentPath: parent, FolderName: "ghost", CreateFolder: true,
		})
		if err == nil || !strings.Contains(err.Error(), "Workspace not found") {
			t.Fatalf("expected workspace error, got %v", err)
		}
		if _, err := os.Stat(filepath.Join(parent, "ghost")); err == nil {
			t.Fatal("folder must not be created")
		}
	})

	t.Run("refuses a non-empty target folder", func(t *testing.T) {
		parent := t.TempDir()
		dir := filepath.Join(parent, "busy")
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o644)
		bp := mk(true, "echo hi")
		_, err := runner.Run(ctx, types.BlueprintRunInput{
			WorkspaceID: ws.ID, RunID: "r5", BlueprintID: bp.ID, Name: "Busy", ParentPath: parent, FolderName: "busy", CreateFolder: true,
		})
		if err == nil || !strings.Contains(err.Error(), "not empty") {
			t.Fatalf("expected not-empty error, got %v", err)
		}
	})
}

func TestBlueprintAskOnError(t *testing.T) {
	ctx := context.Background()
	d := newBlueprintTestDB(t)
	ws, _ := d.CreateWorkspaceT(ctx, "ws", nil)
	var runner *BlueprintRunner
	answer := "skip"
	prompts := 0
	runner = NewBlueprintRunner(d, func(e types.BlueprintLogEvent) {
		if e.Stream != "failed" {
			return
		}
		prompts++
		if answer == "cancel" {
			runner.Cancel(e.RunID)
		} else {
			runner.Resolve(e.RunID, answer)
		}
	})
	run := func(id string, cmds ...string) (types.BlueprintRunResult, string, error) {
		in := types.BlueprintInput{Name: "bp-" + id, CreateFolder: true}
		for _, c := range cmds {
			in.Commands = append(in.Commands, types.BlueprintCommand{Command: c})
		}
		bp, err := d.CreateBlueprintT(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		parent := t.TempDir()
		res, err := runner.Run(ctx, types.BlueprintRunInput{
			WorkspaceID: ws.ID, RunID: id, BlueprintID: bp.ID, Name: id, ParentPath: parent,
			FolderName: "app", CreateFolder: true, AskOnError: true,
		})
		return res, filepath.Join(parent, "app"), err
	}

	prompts = 0
	answer = "skip"
	res, dir, err := run("skip", "exit 3", "echo ok> after.txt")
	if err != nil {
		t.Fatalf("skipping should let the run finish: %v", err)
	}
	if prompts != 1 || res.App.ProjectPath != dir {
		t.Fatalf("prompts=%d app=%+v", prompts, res.App)
	}
	if _, err := os.Stat(filepath.Join(dir, "after.txt")); err != nil {
		t.Fatalf("command after the skipped one did not run: %v", err)
	}

	answer = "abort"
	_, dir, err = run("abort", "exit 3", "echo ok> after.txt")
	if err == nil || !strings.Contains(err.Error(), "exit code 3") {
		t.Fatalf("expected exit code error, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "after.txt")); err == nil {
		t.Fatal("run must stop after abort")
	}

	answer = "cancel"
	if _, _, err = run("cancel", "exit 3", "echo ok> after.txt"); err != errBlueprintCancelled {
		t.Fatalf("expected cancelled, got %v", err)
	}
}

func decodeBlueprintData(t *testing.T, e types.BlueprintLogEvent) string {
	t.Helper()
	if e.Stream != "data" {
		return ""
	}
	raw, err := base64.StdEncoding.DecodeString(e.Data)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestBlueprintTerminal(t *testing.T) {
	ctx := context.Background()
	d := newBlueprintTestDB(t)
	ws, _ := d.CreateWorkspaceT(ctx, "ws", nil)

	// Prompts without a newline, reads a line, then writes what it got to got.txt.
	command := `set /p ans=Your name: & call echo %^ans%> got.txt`
	if runtime.GOOS != "windows" {
		command = `printf 'Your name: '; read ans; echo "$ans" > got.txt`
	}
	prompted := make(chan struct{})
	var once sync.Once
	var mu sync.Mutex
	var out strings.Builder
	runner := NewBlueprintRunner(d, func(e types.BlueprintLogEvent) {
		text := decodeBlueprintData(t, e)
		mu.Lock()
		out.WriteString(text)
		// The first match is the echoed "$ command" line; the second is the prompt.
		seen := strings.Count(out.String(), "Your name:") >= 2
		mu.Unlock()
		if seen {
			once.Do(func() { close(prompted) })
		}
	})
	bp, err := d.CreateBlueprintT(ctx, types.BlueprintInput{
		Name: "ask", CreateFolder: true, Commands: []types.BlueprintCommand{{Command: command}},
	})
	if err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	done := make(chan error, 1)
	go func() {
		_, err := runner.Run(ctx, types.BlueprintRunInput{
			WorkspaceID: ws.ID, RunID: "term", BlueprintID: bp.ID, Name: "app", ParentPath: parent,
			FolderName: "app", CreateFolder: true,
		})
		done <- err
	}()

	select {
	case <-prompted:
	case <-time.After(10 * time.Second):
		t.Fatal("prompt was never shown")
	}
	runner.Resize("term", 90, 20)
	if err := runner.Write("term", "bob\r"); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("run did not finish after input")
	}
	got, err := os.ReadFile(filepath.Join(parent, "app", "got.txt"))
	if err != nil || !strings.Contains(string(got), "bob") {
		t.Fatalf("command did not receive the input: %q, %v", got, err)
	}
	if err := runner.Write("term", "late"); err == nil {
		t.Fatal("input to a finished run must fail")
	}
}

func TestBlueprintCRUD(t *testing.T) {
	ctx := context.Background()
	d := newBlueprintTestDB(t)
	ws, _ := d.CreateWorkspaceT(ctx, "ws", nil)
	label := "  Install "
	bp, err := d.CreateBlueprintT(ctx, types.BlueprintInput{
		Name: " Vite ", CreateFolder: true,
		Commands: []types.BlueprintCommand{{Label: &label, Command: " bun install "}, {Command: "  "}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if bp.Name != "Vite" || len(bp.Commands) != 1 || bp.Commands[0].Command != "bun install" || *bp.Commands[0].Label != "Install" || !bp.CreateFolder {
		t.Fatalf("not normalised: %+v", bp)
	}
	if _, err := d.CreateBlueprintT(ctx, types.BlueprintInput{Name: "Vite"}); err == nil {
		t.Fatal("duplicate name should fail")
	}
	// Blueprints are global: they outlive any workspace.
	if _, err := d.DeleteWorkspace(ctx, ws.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := d.GetBlueprintT(ctx, bp.ID); err != nil {
		t.Fatalf("blueprint should survive workspace deletion: %v", err)
	}
	list, err := d.ListBlueprintsT(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("expected the one global blueprint, got %d, %v", len(list), err)
	}
	if n, err := d.DeleteBlueprint(ctx, bp.ID); err != nil || n != 1 {
		t.Fatalf("delete: %d, %v", n, err)
	}
}
