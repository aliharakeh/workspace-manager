package daemon_test

import (
	"testing"

	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"
	"workspace-manager/shared/client"
)

func TestAgentCanReplaceHiddenEnvValue(t *testing.T) {
	agent := startServer(t)
	app := makeApp(t, agent)
	gui, err := client.Dial(client.Options{Kind: "gui", Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	defer gui.Close()
	hidden, visible := false, true
	key, original := "SECRET", "original-secret"
	var created types.EnvVar
	if err := gui.Call(api.MethodEnvVarsCreate, map[string]any{
		"appID": app.ID,
		"body":  types.EnvVarCreateInput{Key: key, Value: &original, IncludeInAI: &hidden},
	}, &created); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"replacement", ""} {
		var result types.EnvVar
		if err := agent.Call(api.MethodEnvVarsUpdate, map[string]any{
			"id": created.ID, "body": types.EnvVarUpdateInput{Value: &value},
		}, &result); err != nil {
			t.Fatal(err)
		}
		if result.Value != "" || result.IncludeInAI {
			t.Fatal("update exposed the hidden value or changed its visibility")
		}
		var stored []types.EnvVar
		if err := gui.Call(api.MethodEnvVarsList, map[string]any{"appID": app.ID}, &stored); err != nil {
			t.Fatal(err)
		}
		if len(stored) != 1 || stored[0].Value != value || stored[0].IncludeInAI {
			t.Fatal("replacement was not stored with the hidden flag")
		}
	}
	// The MCP upsert also sends the existing key and may explicitly keep it hidden.
	replacement := "last-secret"
	if err := agent.Call(api.MethodEnvVarsUpdate, map[string]any{
		"id": created.ID, "body": types.EnvVarUpdateInput{Key: &key, Value: &replacement, IncludeInAI: &hidden},
	}, nil); err != nil {
		t.Fatal(err)
	}
	renamed := "RENAMED"
	for _, body := range []types.EnvVarUpdateInput{
		{IncludeInAI: &visible},
		{Value: &replacement, IncludeInAI: &visible},
		{Key: &renamed, Value: &replacement},
	} {
		if err := agent.Call(api.MethodEnvVarsUpdate, map[string]any{"id": created.ID, "body": body}, nil); err == nil {
			t.Fatal("agent could unhide or rename a hidden variable")
		}
	}
	if err := agent.Call(api.MethodEnvVarsDelete, map[string]any{"id": created.ID}, nil); err == nil {
		t.Fatal("agent could delete a hidden variable")
	}
	var vars []types.EnvVar
	if err := agent.Call(api.MethodEnvVarsList, map[string]any{"appID": app.ID}, &vars); err != nil {
		t.Fatal(err)
	}
	if len(vars) != 1 || vars[0].Value != "" || vars[0].IncludeInAI || vars[0].Key != key {
		t.Fatal("list exposed or changed the hidden variable")
	}
	var detail types.ConfigSetDetail
	if err := agent.Call(api.MethodConfigSetsDetail, map[string]any{"id": created.ConfigSetID}, &detail); err != nil {
		t.Fatal(err)
	}
	if len(detail.EnvVars) != 1 || detail.EnvVars[0].Value != "" {
		t.Fatal("config detail exposed the hidden value")
	}
	// The GUI can still reveal the flag and read the last replacement.
	var result types.EnvVar
	if err := gui.Call(api.MethodEnvVarsUpdate, map[string]any{
		"id": created.ID, "body": types.EnvVarUpdateInput{IncludeInAI: &visible},
	}, &result); err != nil || result.Value != replacement || !result.IncludeInAI {
		t.Fatalf("GUI update failed: %v", err)
	}
}
