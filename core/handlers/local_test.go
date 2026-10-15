package handlers_test

import (
	"testing"

	"workspace-manager/api"
	"workspace-manager/api/types"
	"workspace-manager/core/db"
	"workspace-manager/core/handlers"
	"workspace-manager/core/ops"
)

type recorder struct{ changed []string }

func (*recorder) Publish(string, int64, any) {}
func (r *recorder) Changed(entity string, _ ...int64) {
	r.changed = append(r.changed, entity)
}

// setup opens a fresh database and returns a GUI and an agent client over the
// same operations, plus the id of an app with a project folder.
func setup(t *testing.T) (gui, agent *handlers.Local, rec *recorder, appID int64) {
	t.Helper()
	t.Setenv("WM_DATA_DIR", t.TempDir())
	d, err := db.Open()
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	o := ops.New(d, func(int64, any) {}, func(string, string) {})
	rec = &recorder{}
	o.Pub = rec
	if gui, err = handlers.NewLocal(o, handlers.ClientGUI); err != nil {
		t.Fatalf("gui local: %v", err)
	}
	if agent, err = handlers.NewLocal(o, "anything-else"); err != nil {
		t.Fatalf("agent local: %v", err)
	}
	var ws types.Workspace
	if err := gui.Call(api.MethodWorkspacesCreate, types.WorkspaceCreateInput{Name: "w"}, &ws); err != nil {
		t.Fatalf("workspace: %v", err)
	}
	var app types.App
	if err := gui.Call(api.MethodAppsCreate, map[string]any{
		"workspaceID": ws.ID,
		"body":        types.AppCreateInput{Name: "app", ProjectPath: t.TempDir()},
	}, &app); err != nil {
		t.Fatalf("app: %v", err)
	}
	return gui, agent, rec, app.ID
}

func TestAnyKindButGUIIsAnAgent(t *testing.T) {
	_, agent, _, _ := setup(t)
	if agent.Kind() != handlers.ClientAgent {
		t.Fatalf("kind %q, want agent", agent.Kind())
	}
}

func TestAgentCannotRunOrUseGit(t *testing.T) {
	_, agent, _, appID := setup(t)
	for _, m := range []string{
		api.MethodRunnerDev, api.MethodRunnerStop, api.MethodGitInfo,
		api.MethodGitWorktreeStart, api.MethodConfigSetsApplyChanges,
		api.MethodBlueprintsCreateApp, api.MethodPortsKill, api.MethodOpenExternal,
	} {
		if err := agent.Call(m, map[string]any{"appID": appID}, nil); err == nil {
			t.Errorf("an agent called %s", m)
		}
	}
	if err := agent.Call(api.MethodAppsGet, map[string]any{"id": appID}, nil); err != nil {
		t.Errorf("an agent cannot read an app: %v", err)
	}
}

func TestHiddenValuesAreBlankedForAgents(t *testing.T) {
	gui, agent, _, appID := setup(t)
	hidden, visible := false, true
	secret, plain := "secret", "plain"
	for _, v := range []types.EnvVarCreateInput{
		{Key: "SECRET", Value: &secret, IncludeInAI: &hidden},
		{Key: "PLAIN", Value: &plain, IncludeInAI: &visible},
	} {
		if err := gui.Call(api.MethodEnvVarsCreate, map[string]any{"appID": appID, "body": v}, nil); err != nil {
			t.Fatalf("create %s: %v", v.Key, err)
		}
	}
	values := func(c *handlers.Local) map[string]string {
		var vars []types.EnvVar
		if err := c.Call(api.MethodEnvVarsList, map[string]any{"appID": appID}, &vars); err != nil {
			t.Fatalf("list: %v", err)
		}
		out := map[string]string{}
		for _, v := range vars {
			out[v.Key] = v.Value
		}
		return out
	}
	if got := values(agent); got["SECRET"] != "" || got["PLAIN"] != "plain" {
		t.Fatalf("agent sees %v", got)
	}
	if got := values(gui); got["SECRET"] != "secret" {
		t.Fatalf("gui sees %v", got)
	}
}

func TestApplyChangesWritesAndSkipsHidden(t *testing.T) {
	gui, _, rec, appID := setup(t)
	visible, hidden := true, false
	one, secret := "1", "secret"
	for _, v := range []types.EnvVarCreateInput{
		{Key: "PORT", Value: &one, IncludeInAI: &visible},
		{Key: "SECRET", Value: &secret, IncludeInAI: &hidden},
	} {
		if err := gui.Call(api.MethodEnvVarsCreate, map[string]any{"appID": appID, "body": v}, nil); err != nil {
			t.Fatalf("create %s: %v", v.Key, err)
		}
	}
	var sets []types.ConfigSet
	if err := gui.Call(api.MethodConfigSetsList, map[string]any{"appID": appID}, &sets); err != nil || len(sets) == 0 {
		t.Fatalf("config sets: %v (%d)", err, len(sets))
	}
	rec.changed = nil
	patch := types.AppAIPatch{Env: &types.AppAIEnvPatch{Upsert: []types.AppAIEnvUpsert{
		{Key: "PORT", Value: "2"}, {Key: "SECRET", Value: "leaked"}, {Key: "NEW", Value: "x"},
	}}}
	if err := gui.Call(api.MethodConfigSetsApplyChanges,
		map[string]any{"appID": appID, "configSetID": sets[0].ID, "patch": patch}, nil); err != nil {
		t.Fatalf("apply: %v", err)
	}
	var vars []types.EnvVar
	if err := gui.Call(api.MethodEnvVarsList, map[string]any{"appID": appID}, &vars); err != nil {
		t.Fatalf("list: %v", err)
	}
	got := map[string]string{}
	for _, v := range vars {
		got[v.Key] = v.Value
	}
	if got["PORT"] != "2" || got["NEW"] != "x" || got["SECRET"] != "secret" {
		t.Fatalf("env after apply: %v", got)
	}
	if len(rec.changed) == 0 || rec.changed[0] != "envVar" {
		t.Fatalf("the write was not announced: %v", rec.changed)
	}
}

func TestAgentWritesAreAnnounced(t *testing.T) {
	_, agent, rec, appID := setup(t)
	if err := agent.Call(api.MethodAppsUpdate, map[string]any{"id": appID, "body": map[string]any{"name": "renamed"}}, nil); err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(rec.changed) == 0 {
		t.Fatal("an agent write did not reach the publisher")
	}
}
