package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"workspace-manager/shared/api/types"
)

// TestStartsDaemonAndServesTools runs the real wm-mcp binary against a fresh
// data folder with no daemon running: the first tool call must start wmd (found
// next to wm-mcp) and answer through it.
func TestStartsDaemonAndServesTools(t *testing.T) {
	bin := t.TempDir()
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	for _, pkg := range []string{"wmd", "wm-mcp"} {
		out, err := exec.Command("go", "build", "-o", filepath.Join(bin, pkg+suffix), "../"+pkg).CombinedOutput()
		if err != nil {
			t.Fatalf("build %s: %v\n%s", pkg, err, out)
		}
	}
	data := t.TempDir()

	cmd := exec.Command(filepath.Join(bin, "wm-mcp"+suffix))
	cmd.Env = append(os.Environ(), "WM_DATA_DIR="+data)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// Stop the daemon the server started, then the server.
		_ = send(stdin, bufio.NewReader(stdout), 99, "tools/call", map[string]any{
			"name": "wm_call", "arguments": map[string]any{"method": "server.stop"},
		})
		_ = stdin.Close()
		_ = cmd.Wait()
	})
	out := bufio.NewReader(stdout)

	init := send(stdin, out, 1, "initialize", map[string]any{"protocolVersion": "2025-06-18"})
	if init["result"] == nil {
		t.Fatalf("initialize: %v", init)
	}

	list := send(stdin, out, 2, "tools/list", nil)
	if !strings.Contains(toJSON(list), `"wm_overview"`) {
		t.Fatalf("tools/list has no wm_overview: %v", list)
	}
	if strings.Contains(toJSON(list), `"git`) {
		t.Errorf("tools/list exposes git: %v", list)
	}

	// The first data call starts the daemon.
	create := mcpCall(t, stdin, out, 3, "wm_call", map[string]any{
		"method": "workspaces.create", "params": map[string]any{"name": "mcp-ws"},
	})
	if !strings.Contains(create, "mcp-ws") {
		t.Fatalf("workspaces.create: %s", create)
	}
	overview := mcpCall(t, stdin, out, 4, "wm_overview", map[string]any{})
	if !strings.Contains(overview, "mcp-ws") {
		t.Fatalf("wm_overview misses the workspace: %s", overview)
	}

	// Choose a config set on the run itself, leaving the app's active set alone.
	call := func(name string, args map[string]any, target any) {
		t.Helper()
		resp := send(stdin, out, 10, "tools/call", map[string]any{"name": name, "arguments": args})
		result, ok := resp["result"].(map[string]any)
		if !ok || result["isError"] == true {
			t.Fatalf("%s: %v", name, resp)
		}
		if target != nil {
			content := result["content"].([]any)[0].(map[string]any)["text"].(string)
			if err := json.Unmarshal([]byte(content), target); err != nil {
				t.Fatalf("decode %s: %v", name, err)
			}
		}
	}
	var app types.App
	call("wm_app", map[string]any{"action": "create", "name": "demo", "workspace": "mcp-ws", "path": t.TempDir()}, &app)
	var secret types.EnvVar
	call("wm_env", map[string]any{"action": "set", "key": "SECRET", "value": "original-secret", "private": true}, &secret)
	if secret.Value != "" || !secret.Private {
		t.Fatal("creating a hidden variable exposed its value")
	}
	// Omitting hidden must preserve the flag rather than making the variable visible.
	call("wm_env", map[string]any{"action": "set", "key": "SECRET", "value": "replacement-secret"}, &secret)
	if secret.Value != "" || !secret.Private {
		t.Fatal("updating a hidden variable exposed its value")
	}
	var vars []types.EnvVar
	call("wm_env", map[string]any{"action": "list"}, &vars)
	if len(vars) != 1 || vars[0].Value != "" || !vars[0].Private {
		t.Fatal("listing a hidden variable exposed its value")
	}
	refused := send(stdin, out, 11, "tools/call", map[string]any{
		"name": "wm_env", "arguments": map[string]any{"action": "set", "key": "SECRET", "private": false},
	})
	if !strings.Contains(toJSON(refused), `"isError":true`) {
		t.Fatal("MCP could make a hidden variable visible")
	}
	// Shared config: an app config set links to a workspace set, the linked
	// variable is read-only through wm_env and follows edits made with wm_shared.
	call("wm_shared", map[string]any{"action": "create_set", "workspace": "mcp-ws", "name": "dev"}, nil)
	call("wm_shared", map[string]any{"action": "set", "workspace": "mcp-ws", "shared": "dev", "key": "DATABASE_URL", "value": "postgres://one"}, nil)
	call("wm_shared", map[string]any{"action": "link", "app": "demo", "shared": "dev"}, nil)
	call("wm_env", map[string]any{"action": "list"}, &vars)
	linkedValue := func() (string, bool) {
		for _, v := range vars {
			if v.Key == "DATABASE_URL" {
				return v.Value, v.SharedVarID != nil
			}
		}
		return "", false
	}
	if value, shared := linkedValue(); value != "postgres://one" || !shared {
		t.Fatalf("link did not add the shared variable: %+v", vars)
	}
	editShared := send(stdin, out, 12, "tools/call", map[string]any{
		"name": "wm_env", "arguments": map[string]any{"action": "set", "key": "DATABASE_URL", "value": "mine"},
	})
	if !strings.Contains(toJSON(editShared), `"isError":true`) {
		t.Fatal("wm_env edited a shared variable")
	}
	call("wm_shared", map[string]any{"action": "set", "workspace": "mcp-ws", "shared": "dev", "key": "DATABASE_URL", "value": "postgres://two"}, nil)
	call("wm_env", map[string]any{"action": "list"}, &vars)
	if value, _ := linkedValue(); value != "postgres://two" {
		t.Fatalf("a workspace edit did not reach the linked config set: %+v", vars)
	}
	call("wm_shared", map[string]any{"action": "unlink", "app": "demo"}, nil)
	call("wm_env", map[string]any{"action": "list"}, &vars)
	if _, shared := linkedValue(); shared || len(vars) != 1 {
		t.Fatalf("unlink did not remove the shared variable: %+v", vars)
	}

	// Dynamic environment script: saved, read back, kept on a partial update, cleared.
	var script types.EnvProvider
	call("wm_env_script", map[string]any{"action": "set", "command": "node gen.js", "timeout": 60}, &script)
	call("wm_env_script", map[string]any{"action": "get"}, &script)
	if script.Command != "node gen.js" || script.Timeout != 60 {
		t.Fatalf("env script not saved: %+v", script)
	}
	call("wm_env_script", map[string]any{"action": "set", "timeout": 90}, &script)
	call("wm_env_script", map[string]any{"action": "get"}, &script)
	if script.Command != "node gen.js" || script.Timeout != 90 {
		t.Fatalf("a partial update lost the command: %+v", script)
	}
	call("wm_env_script", map[string]any{"action": "set", "workspace": "mcp-ws", "shared": "dev", "command": "node shared.js", "directory": t.TempDir()}, nil)
	relative := send(stdin, out, 13, "tools/call", map[string]any{
		"name": "wm_env_script", "arguments": map[string]any{"action": "set", "workspace": "mcp-ws", "shared": "dev", "directory": "relative"},
	})
	if !strings.Contains(toJSON(relative), `"isError":true`) {
		t.Fatal("a shared script accepted a relative directory")
	}
	call("wm_env_script", map[string]any{"action": "clear"}, &script)
	call("wm_env_script", map[string]any{"action": "get"}, &script)
	if script.Command != "" {
		t.Fatalf("clear did not disable the script: %+v", script)
	}

	var active types.ConfigSet
	call("wm_config", map[string]any{"action": "create", "name": "selected"}, &active)
	for _, kind := range []string{"dev", "prod", "build", "setup"} {
		call("wm_config", map[string]any{
			"action": "save_commands", "kind": kind,
			"commands": []any{map[string]any{"command": "echo selected-" + kind}},
		}, nil)
	}
	// Create an empty active set so a start using it would fail.
	var guiSet types.ConfigSet
	call("wm_config", map[string]any{"action": "create", "name": "gui"}, &guiSet)
	for _, kind := range []string{"dev", "prod", "build", "setup", "run"} {
		ref := "selected"
		if kind == "run" {
			ref = fmt.Sprint(active.ID)
		}
		var status types.StatusEvent
		call("wm_run", map[string]any{"action": kind, "configSet": ref}, &status)
		if status.ConfigSetID != active.ID || !status.Running {
			t.Fatalf("%s used the wrong config set: %+v", kind, status)
		}
		deadline := time.Now().Add(20 * time.Second)
		for status.Running {
			if time.Now().After(deadline) {
				t.Fatalf("%s did not finish", kind)
			}
			time.Sleep(20 * time.Millisecond)
			call("wm_run", map[string]any{"action": "status"}, &status)
		}
		var current types.App
		call("wm_call", map[string]any{"method": "apps.get", "params": map[string]any{"id": app.ID}}, &current)
		if current.ActiveConfigSetID == nil || *current.ActiveConfigSetID != guiSet.ID {
			t.Fatalf("%s changed the active set: %+v", kind, current)
		}
	}

	// GUI-only methods are refused to an agent.
	res := send(stdin, out, 5, "tools/call", map[string]any{
		"name": "wm_call", "arguments": map[string]any{"method": "settings.get"},
	})
	if !strings.Contains(toJSON(res), `"isError":true`) {
		t.Errorf("settings.get should be refused to MCP clients: %v", res)
	}
}

func send(w interface{ Write([]byte) (int, error) }, r *bufio.Reader, id int, method string, params any) map[string]any {
	req := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		req["params"] = params
	}
	b, _ := json.Marshal(req)
	_, _ = w.Write(append(b, '\n'))
	line, err := r.ReadBytes('\n')
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	var resp map[string]any
	_ = json.Unmarshal(line, &resp)
	return resp
}

func mcpCall(t *testing.T, w interface{ Write([]byte) (int, error) }, r *bufio.Reader, id int, name string, args map[string]any) string {
	t.Helper()
	resp := send(w, r, id, "tools/call", map[string]any{"name": name, "arguments": args})
	if strings.Contains(toJSON(resp), `"isError":true`) {
		t.Fatalf("%s failed: %v", name, resp)
	}
	return toJSON(resp)
}

func toJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
