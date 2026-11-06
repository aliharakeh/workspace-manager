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

func TestAgentCanDeleteAndReplaceHiddenItems(t *testing.T) {
	agent := startServer(t)
	app := makeApp(t, agent)
	gui, err := client.Dial(client.Options{Kind: "gui", Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	defer gui.Close()
	hidden, visible := false, true
	secret := "s"
	var v types.EnvVar
	if err := gui.Call(api.MethodEnvVarsCreate, map[string]any{
		"appID": app.ID, "body": types.EnvVarCreateInput{Key: "K", Value: &secret, IncludeInAI: &hidden},
	}, &v); err != nil {
		t.Fatal(err)
	}
	content := "original"
	var tpl types.Template
	if err := gui.Call(api.MethodTemplatesCreate, map[string]any{
		"appID": app.ID, "body": types.TemplateCreateInput{FilePath: ".env.x", Content: &content, IncludeInAI: &hidden},
	}, &tpl); err != nil {
		t.Fatal(err)
	}
	// Replacing content works and keeps the template hidden.
	next, renamed := "new", "other.txt"
	var out types.Template
	if err := agent.Call(api.MethodTemplatesUpdate, map[string]any{
		"id": tpl.ID, "body": types.TemplateUpdateInput{Content: &next, IncludeInAI: &hidden},
	}, &out); err != nil || out.Content != "" || out.IncludeInAI {
		t.Fatalf("agent could not replace hidden template content: %v", err)
	}
	for _, body := range []types.TemplateUpdateInput{
		{IncludeInAI: &visible},
		{Content: &next, IncludeInAI: &visible},
		{FilePath: &renamed, Content: &next},
	} {
		if err := agent.Call(api.MethodTemplatesUpdate, map[string]any{"id": tpl.ID, "body": body}, nil); err == nil {
			t.Fatal("agent could unhide or rename a hidden template")
		}
	}
	var stored types.Template
	if err := gui.Call(api.MethodTemplatesUpdate, map[string]any{"id": tpl.ID, "body": types.TemplateUpdateInput{}}, &stored); err != nil || stored.Content != next || stored.IncludeInAI {
		t.Fatalf("stored template wrong: %v", err)
	}
	// The list still names the hidden template, with its content blanked.
	var listed []types.Template
	if err := agent.Call(api.MethodTemplatesList, map[string]any{"appID": app.ID}, &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].FilePath != ".env.x" || listed[0].Content != "" || listed[0].IncludeInAI {
		t.Fatalf("hidden template should list with its file name and no content: %+v", listed)
	}
	if err := agent.Call(api.MethodTemplatesDelete, map[string]any{"id": tpl.ID}, nil); err != nil {
		t.Fatalf("agent could not delete hidden template: %v", err)
	}
	if err := agent.Call(api.MethodEnvVarsDelete, map[string]any{"id": v.ID}, nil); err != nil {
		t.Fatalf("agent could not delete hidden variable: %v", err)
	}
}
