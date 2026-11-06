package daemon_test

import (
	"testing"

	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"
	"workspace-manager/shared/client"
)

func envList(t *testing.T, c *client.Client, appID int64) []types.EnvVar {
	t.Helper()
	var vars []types.EnvVar
	if err := c.Call(api.MethodEnvVarsList, map[string]any{"appID": appID}, &vars); err != nil {
		t.Fatal(err)
	}
	return vars
}

// A variable with options takes the selected option's value: the first one by
// default, and a value edit lands on the selected option.
func TestEnvVarOptionsSelection(t *testing.T) {
	c := startServer(t)
	app := makeApp(t, c)
	opts := []types.EnvVarOption{{Label: "local", Value: "http://localhost"}, {Label: "prod", Value: "https://prod"}}
	var v types.EnvVar
	if err := c.Call(api.MethodEnvVarsCreate, map[string]any{
		"appID": app.ID, "body": types.EnvVarCreateInput{Key: "API_URL", Options: &opts},
	}, &v); err != nil {
		t.Fatal(err)
	}
	if v.Selected != "local" || v.Value != "http://localhost" || len(v.Options) != 2 {
		t.Fatalf("first option should be selected by default: %+v", v)
	}

	prod := "prod"
	if err := c.Call(api.MethodEnvVarsUpdate, map[string]any{"id": v.ID, "body": types.EnvVarUpdateInput{Selected: &prod}}, &v); err != nil {
		t.Fatal(err)
	}
	if v.Selected != "prod" || v.Value != "https://prod" {
		t.Fatalf("selecting prod: %+v", v)
	}

	edited := "https://prod2"
	if err := c.Call(api.MethodEnvVarsUpdate, map[string]any{"id": v.ID, "body": types.EnvVarUpdateInput{Value: &edited}}, &v); err != nil {
		t.Fatal(err)
	}
	if v.Value != edited || v.Options[1].Value != edited || v.Options[0].Value != "http://localhost" {
		t.Fatalf("a value edit should change the selected option only: %+v", v)
	}

	// Replacing the options keeps the choice while its label exists, else the first.
	next := []types.EnvVarOption{{Label: "prod", Value: "p"}, {Label: "dev", Value: "d"}}
	if err := c.Call(api.MethodEnvVarsUpdate, map[string]any{"id": v.ID, "body": types.EnvVarUpdateInput{Options: &next}}, &v); err != nil {
		t.Fatal(err)
	}
	if v.Selected != "prod" || v.Value != "p" {
		t.Fatalf("choice should survive: %+v", v)
	}
	next = []types.EnvVarOption{{Label: "dev", Value: "d"}}
	if err := c.Call(api.MethodEnvVarsUpdate, map[string]any{"id": v.ID, "body": types.EnvVarUpdateInput{Options: &next}}, &v); err != nil {
		t.Fatal(err)
	}
	if v.Selected != "dev" || v.Value != "d" {
		t.Fatalf("choice should fall back to the first option: %+v", v)
	}

	// Bad input is refused.
	missing := "nope"
	dup := []types.EnvVarOption{{Label: "a"}, {Label: "a"}}
	empty := []types.EnvVarOption{{Label: " "}}
	for _, body := range []types.EnvVarUpdateInput{{Selected: &missing}, {Options: &dup}, {Options: &empty}} {
		if err := c.Call(api.MethodEnvVarsUpdate, map[string]any{"id": v.ID, "body": body}, nil); err == nil {
			t.Fatalf("accepted %+v", body)
		}
	}

	// An empty list turns it back into a plain value that keeps the current one.
	none := []types.EnvVarOption{}
	if err := c.Call(api.MethodEnvVarsUpdate, map[string]any{"id": v.ID, "body": types.EnvVarUpdateInput{Options: &none}}, &v); err != nil {
		t.Fatal(err)
	}
	if len(v.Options) != 0 || v.Selected != "" || v.Value != "d" {
		t.Fatalf("plain again: %+v", v)
	}
}

// Option values of a hidden variable never reach agents, and agents cannot
// change the options or the choice of one.
func TestHiddenEnvOptionsStayHidden(t *testing.T) {
	agent := startServer(t)
	app := makeApp(t, agent)
	gui, err := client.Dial(client.Options{Kind: "gui", Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	defer gui.Close()
	hidden := true
	opts := []types.EnvVarOption{{Label: "a", Value: "secret-a"}, {Label: "b", Value: "secret-b"}}
	var v types.EnvVar
	if err := gui.Call(api.MethodEnvVarsCreate, map[string]any{
		"appID": app.ID, "body": types.EnvVarCreateInput{Key: "SECRET", Options: &opts, Private: &hidden},
	}, &v); err != nil {
		t.Fatal(err)
	}
	for _, e := range envList(t, agent, app.ID) {
		if e.Value != "" {
			t.Fatal("agent saw the hidden value")
		}
		for _, o := range e.Options {
			if o.Value != "" || o.Label == "" {
				t.Fatalf("agent saw an option value or lost a label: %+v", e.Options)
			}
		}
	}
	var detail types.ConfigSetDetail
	if err := agent.Call(api.MethodConfigSetsDetail, map[string]any{"id": v.ConfigSetID}, &detail); err != nil {
		t.Fatal(err)
	}
	for _, o := range detail.EnvVars[0].Options {
		if o.Value != "" {
			t.Fatal("config detail exposed an option value")
		}
	}
	b, val := "b", "x"
	for _, body := range []types.EnvVarUpdateInput{{Selected: &b}, {Options: &opts}, {Value: &val, Selected: &b}} {
		if err := agent.Call(api.MethodEnvVarsUpdate, map[string]any{"id": v.ID, "body": body}, nil); err == nil {
			t.Fatalf("agent changed a hidden choice: %+v", body)
		}
	}
	// A plain value replacement is still allowed and lands on the selected option.
	if err := agent.Call(api.MethodEnvVarsUpdate, map[string]any{"id": v.ID, "body": types.EnvVarUpdateInput{Value: &val}}, nil); err != nil {
		t.Fatal(err)
	}
	got := envList(t, gui, app.ID)[0]
	if got.Selected != "a" || got.Value != "x" || got.Options[0].Value != "x" || got.Options[1].Value != "secret-b" {
		t.Fatalf("stored: %+v", got)
	}
}

// Copying a config set keeps the options and the choice.
func TestCopyKeepsEnvOptions(t *testing.T) {
	c := startServer(t)
	app := makeApp(t, c)
	opts := []types.EnvVarOption{{Label: "x", Value: "1"}, {Label: "y", Value: "2"}}
	sel := "y"
	if err := c.Call(api.MethodEnvVarsCreate, map[string]any{
		"appID": app.ID, "body": types.EnvVarCreateInput{Key: "K", Options: &opts, Selected: &sel},
	}, nil); err != nil {
		t.Fatal(err)
	}
	var active types.ConfigSet
	var sets []types.ConfigSet
	if err := c.Call(api.MethodConfigSetsList, map[string]any{"appID": app.ID}, &sets); err != nil || len(sets) != 1 {
		t.Fatalf("sets: %v %+v", err, sets)
	}
	active = sets[0]
	noActivate := false
	var copy types.ConfigSet
	if err := c.Call(api.MethodConfigSetsCreate, map[string]any{
		"appID": app.ID, "body": map[string]any{
			"name": "copy", "activate": noActivate, "copy_from_id": active.ID, "parts": map[string]any{"env": true},
		},
	}, &copy); err != nil {
		t.Fatal(err)
	}
	var vars []types.EnvVar
	if err := c.Call(api.MethodEnvVarsList, map[string]any{"appID": app.ID, "configSetID": copy.ID}, &vars); err != nil || len(vars) != 1 {
		t.Fatalf("copied vars: %v %+v", err, vars)
	}
	if vars[0].Selected != "y" || vars[0].Value != "2" || len(vars[0].Options) != 2 {
		t.Fatalf("copy lost the options: %+v", vars[0])
	}
}
