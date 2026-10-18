package daemon_test

import (
	"encoding/json"
	"strings"
	"testing"

	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"
	"workspace-manager/shared/client"
)

// TestWorkspaceExportKeepsSecretsFromAgents checks that include_in_ai=false
// values reach only the GUI, and only when it asks for "include".
func TestWorkspaceExportKeepsSecretsFromAgents(t *testing.T) {
	agent := startServer(t)
	app := makeApp(t, agent)
	gui, err := client.Dial(client.Options{Kind: "gui", Version: "test"})
	if err != nil {
		t.Fatalf("dial gui: %v", err)
	}
	t.Cleanup(func() { _ = gui.Close() })

	hidden := false
	val := "s3cret"
	if err := gui.Call(api.MethodEnvVarsCreate, map[string]any{
		"appID": app.ID,
		"body":  types.EnvVarCreateInput{Key: "SECRET", Value: &val, IncludeInAI: &hidden},
	}, nil); err != nil {
		t.Fatalf("create hidden var: %v", err)
	}

	export := func(c *client.Client, mode string) types.WorkspaceBundle {
		t.Helper()
		var b types.WorkspaceBundle
		if err := c.Call(api.MethodWorkspacesExport, map[string]any{
			"id": app.WorkspaceID, "options": types.WorkspaceExportOptions{Secrets: mode},
		}, &b); err != nil {
			t.Fatalf("export (%s): %v", mode, err)
		}
		return b
	}
	for _, mode := range []string{"", "placeholder", "include"} {
		b := export(agent, mode)
		raw, _ := json.Marshal(b)
		if strings.Contains(string(raw), val) || b.Secrets != types.SecretsPlaceholder {
			t.Errorf("agent export (%q) leaked or kept mode %q", mode, b.Secrets)
		}
	}
	if raw, _ := json.Marshal(export(gui, "include")); !strings.Contains(string(raw), val) {
		t.Errorf("the GUI's include export should carry the value")
	}
	if raw, _ := json.Marshal(export(gui, "")); strings.Contains(string(raw), val) {
		t.Errorf("the default GUI export must not carry the value")
	}

	b := export(gui, "")
	var res types.WorkspaceImportResult
	if err := gui.Call(api.MethodWorkspacesImport, map[string]any{
		"bundle": b, "options": types.WorkspaceImportOptions{Name: "copy"},
	}, &res); err != nil {
		t.Fatalf("import: %v", err)
	}
	if res.WorkspaceID == 0 || len(res.AppIDs) != 1 {
		t.Errorf("import result: %+v", res)
	}
}
