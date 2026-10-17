package main

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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
