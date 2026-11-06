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
	hidden, visible := true, false
	key, original := "SECRET", "original-secret"
	var created types.EnvVar
	if err := gui.Call(api.MethodEnvVarsCreate, map[string]any{
		"appID": app.ID,
		"body":  types.EnvVarCreateInput{Key: key, Value: &original, Private: &hidden},
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
		if result.Value != "" || !result.Private {
			t.Fatal("update exposed the hidden value or changed its visibility")
		}
		var stored []types.EnvVar
		if err := gui.Call(api.MethodEnvVarsList, map[string]any{"appID": app.ID}, &stored); err != nil {
			t.Fatal(err)
		}
		if len(stored) != 1 || stored[0].Value != value || !stored[0].Private {
			t.Fatal("replacement was not stored with the hidden flag")
		}
	}
	// The MCP upsert also sends the existing key and may explicitly keep it hidden.
	replacement := "last-secret"
	if err := agent.Call(api.MethodEnvVarsUpdate, map[string]any{
		"id": created.ID, "body": types.EnvVarUpdateInput{Key: &key, Value: &replacement, Private: &hidden},
	}, nil); err != nil {
		t.Fatal(err)
	}
	renamed := "RENAMED"
	for _, body := range []types.EnvVarUpdateInput{
		{Private: &visible},
		{Value: &replacement, Private: &visible},
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
	if len(vars) != 1 || vars[0].Value != "" || !vars[0].Private || vars[0].Key != key {
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
		"id": created.ID, "body": types.EnvVarUpdateInput{Private: &visible},
	}, &result); err != nil || result.Value != replacement || result.Private {
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
	hidden, visible := true, false
	secret := "s"
	var v types.EnvVar
	if err := gui.Call(api.MethodEnvVarsCreate, map[string]any{
		"appID": app.ID, "body": types.EnvVarCreateInput{Key: "K", Value: &secret, Private: &hidden},
	}, &v); err != nil {
		t.Fatal(err)
	}
	content := "original"
	var tpl types.Template
	if err := gui.Call(api.MethodTemplatesCreate, map[string]any{
		"appID": app.ID, "body": types.TemplateCreateInput{FilePath: ".env.x", Content: &content, Private: &hidden},
	}, &tpl); err != nil {
		t.Fatal(err)
	}
	// Replacing content works and keeps the template hidden.
	next, renamed := "new", "other.txt"
	var out types.Template
	if err := agent.Call(api.MethodTemplatesUpdate, map[string]any{
		"id": tpl.ID, "body": types.TemplateUpdateInput{Content: &next, Private: &hidden},
	}, &out); err != nil || out.Content != "" || !out.Private {
		t.Fatalf("agent could not replace hidden template content: %v", err)
	}
	for _, body := range []types.TemplateUpdateInput{
		{Private: &visible},
		{Content: &next, Private: &visible},
		{FilePath: &renamed, Content: &next},
	} {
		if err := agent.Call(api.MethodTemplatesUpdate, map[string]any{"id": tpl.ID, "body": body}, nil); err == nil {
			t.Fatal("agent could unhide or rename a hidden template")
		}
	}
	var stored types.Template
	if err := gui.Call(api.MethodTemplatesUpdate, map[string]any{"id": tpl.ID, "body": types.TemplateUpdateInput{}}, &stored); err != nil || stored.Content != next || !stored.Private {
		t.Fatalf("stored template wrong: %v", err)
	}
	// The list still names the hidden template, with its content blanked.
	var listed []types.Template
	if err := agent.Call(api.MethodTemplatesList, map[string]any{"appID": app.ID}, &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].FilePath != ".env.x" || listed[0].Content != "" || !listed[0].Private {
		t.Fatalf("hidden template should list with its file name and no content: %+v", listed)
	}
	if err := agent.Call(api.MethodTemplatesDelete, map[string]any{"id": tpl.ID}, nil); err != nil {
		t.Fatalf("agent could not delete hidden template: %v", err)
	}
	if err := agent.Call(api.MethodEnvVarsDelete, map[string]any{"id": v.ID}, nil); err != nil {
		t.Fatalf("agent could not delete hidden variable: %v", err)
	}
}

func TestAgentSeesSharedHiddenValuesBlanked(t *testing.T) {
	agent := startServer(t)
	app := makeApp(t, agent)
	gui, err := client.Dial(client.Options{Kind: "gui", Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	defer gui.Close()
	hidden := true
	secret, plain := "s3cret", "visible"
	var set types.SharedSet
	if err := gui.Call(api.MethodSharedSetsCreate, map[string]any{"workspaceID": app.WorkspaceID, "name": "dev"}, &set); err != nil {
		t.Fatal(err)
	}
	var pwd, url types.SharedEnvVar
	if err := gui.Call(api.MethodSharedVarsCreate, map[string]any{
		"setID": set.ID, "body": types.EnvVarCreateInput{Key: "PASSWORD", Value: &secret, Private: &hidden},
	}, &pwd); err != nil {
		t.Fatal(err)
	}
	if err := gui.Call(api.MethodSharedVarsCreate, map[string]any{
		"setID": set.ID, "body": types.EnvVarCreateInput{Key: "URL", Value: &plain},
	}, &url); err != nil {
		t.Fatal(err)
	}

	// An agent reads the set with the hidden value blanked and the visible one intact.
	var detail types.SharedSetDetail
	if err := agent.Call(api.MethodSharedSetsGet, map[string]any{"id": set.ID}, &detail); err != nil {
		t.Fatal(err)
	}
	for _, v := range detail.Vars {
		if v.Key == "PASSWORD" && v.Value != "" || v.Key == "URL" && v.Value != plain {
			t.Fatalf("agent saw %q = %q", v.Key, v.Value)
		}
	}
	// It cannot unhide, rename or reuse the key of a hidden variable, but may replace its value.
	visible, renamed := false, "RENAMED"
	for _, body := range []types.EnvVarUpdateInput{{Private: &visible}, {Key: &renamed, Value: &plain}} {
		if err := agent.Call(api.MethodSharedVarsUpdate, map[string]any{"id": pwd.ID, "body": body}, nil); err == nil {
			t.Fatal("agent changed the visibility or name of a hidden shared variable")
		}
	}
	if err := agent.Call(api.MethodSharedVarsCreate, map[string]any{
		"setID": set.ID, "body": types.EnvVarCreateInput{Key: "PASSWORD", Value: &plain},
	}, nil); err == nil {
		t.Fatal("agent overwrote a hidden shared variable by creating its key")
	}
	next := "rotated"
	var updated types.SharedEnvVar
	if err := agent.Call(api.MethodSharedVarsUpdate, map[string]any{"id": pwd.ID, "body": types.EnvVarUpdateInput{Value: &next}}, &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Value != "" || !updated.Private {
		t.Fatal("update exposed the hidden value or unhid it")
	}

	// Linked into an app, the agent still sees the hidden row blanked and cannot edit it.
	var sets []types.ConfigSet
	if err := agent.Call(api.MethodConfigSetsList, map[string]any{"appID": app.ID}, &sets); err != nil {
		t.Fatal(err)
	}
	if err := agent.Call(api.MethodConfigSetsLinkShared, map[string]any{
		"id": sets[0].ID, "body": types.SharedLinkInput{SharedSetID: set.ID},
	}, nil); err != nil {
		t.Fatal(err)
	}
	var vars []types.EnvVar
	if err := agent.Call(api.MethodEnvVarsList, map[string]any{"appID": app.ID}, &vars); err != nil {
		t.Fatal(err)
	}
	var stored []types.EnvVar
	if err := gui.Call(api.MethodEnvVarsList, map[string]any{"appID": app.ID}, &stored); err != nil {
		t.Fatal(err)
	}
	if len(vars) != 2 || len(stored) != 2 {
		t.Fatalf("link did not create the rows: %d %d", len(vars), len(stored))
	}
	for i, v := range vars {
		if v.Key == "PASSWORD" && v.Value != "" {
			t.Fatal("agent saw the shared hidden value in the app")
		}
		if v.SharedVarID == nil {
			t.Fatalf("row %s is not marked shared", v.Key)
		}
		if stored[i].Key == "PASSWORD" && stored[i].Value != next {
			t.Fatalf("the replaced value did not reach the app row: %q", stored[i].Value)
		}
		if err := agent.Call(api.MethodEnvVarsUpdate, map[string]any{"id": v.ID, "body": types.EnvVarUpdateInput{Value: &plain}}, nil); err == nil {
			t.Fatalf("agent edited the shared row %s in the app", v.Key)
		}
	}
}
