package daemon_test

import (
	"testing"

	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"
)

// Env vars, templates and command lists can be read and written on any config
// set of the app without making it the active one.
func TestOperationsTargetAnyConfigSet(t *testing.T) {
	c := startServer(t)
	app := makeApp(t, c)
	noActivate := false
	var other types.ConfigSet
	if err := c.Call(api.MethodConfigSetsCreate, map[string]any{
		"appID": app.ID, "body": map[string]any{"name": "other", "activate": noActivate},
	}, &other); err != nil {
		t.Fatal(err)
	}
	val, content := "v", "tpl"
	if err := c.Call(api.MethodEnvVarsCreate, map[string]any{
		"appID": app.ID, "configSetID": other.ID, "body": types.EnvVarCreateInput{Key: "K", Value: &val},
	}, nil); err != nil {
		t.Fatal(err)
	}
	if err := c.Call(api.MethodTemplatesCreate, map[string]any{
		"appID": app.ID, "configSetID": other.ID, "body": types.TemplateCreateInput{FilePath: "a.txt", Content: &content},
	}, nil); err != nil {
		t.Fatal(err)
	}
	mode := "parallel"
	if err := c.Call(api.MethodRunConfigSave, map[string]any{
		"appID": app.ID, "configSetID": other.ID, "kind": types.KindDev,
		"body": types.RunConfigSaveInput{Mode: &mode, Commands: []types.RunCommandInput{{Command: "echo hi"}}},
	}, nil); err != nil {
		t.Fatal(err)
	}

	var vars []types.EnvVar
	var tpls []types.Template
	var cfg types.RunConfig
	if err := c.Call(api.MethodEnvVarsList, map[string]any{"appID": app.ID, "configSetID": other.ID}, &vars); err != nil || len(vars) != 1 {
		t.Fatalf("other set vars: %v %+v", err, vars)
	}
	if err := c.Call(api.MethodTemplatesList, map[string]any{"appID": app.ID, "configSetID": other.ID}, &tpls); err != nil || len(tpls) != 1 {
		t.Fatalf("other set templates: %v %+v", err, tpls)
	}
	if err := c.Call(api.MethodRunConfigGet, map[string]any{"appID": app.ID, "configSetID": other.ID, "kind": types.KindDev}, &cfg); err != nil || len(cfg.Commands) != 1 {
		t.Fatalf("other set commands: %v %+v", err, cfg)
	}
	// The active set was never touched.
	vars, tpls = nil, nil
	if err := c.Call(api.MethodEnvVarsList, map[string]any{"appID": app.ID}, &vars); err != nil || len(vars) != 0 {
		t.Fatalf("active set vars changed: %v %+v", err, vars)
	}
	if err := c.Call(api.MethodTemplatesList, map[string]any{"appID": app.ID}, &tpls); err != nil || len(tpls) != 0 {
		t.Fatalf("active set templates changed: %v %+v", err, tpls)
	}
	// A set of another app is refused.
	if err := c.Call(api.MethodEnvVarsList, map[string]any{"appID": app.ID, "configSetID": 999999}, &vars); err == nil {
		t.Fatal("unknown config set accepted")
	}
}
