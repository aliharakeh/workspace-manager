package handlers_test

import (
	"testing"

	"workspace-manager/internal/api"
	"workspace-manager/internal/api/types"
	"workspace-manager/internal/db"
	"workspace-manager/internal/handlers"
	"workspace-manager/internal/ops"
)

type recorder struct{ changed []string }

func (*recorder) Publish(string, int64, any) {}
func (r *recorder) Changed(entity string, _ ...int64) {
	r.changed = append(r.changed, entity)
}

func setup(t *testing.T) (*handlers.Local, *recorder, int64) {
	t.Helper()
	t.Setenv("WM_DATA_DIR", t.TempDir())
	d, err := db.Open()
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	o := ops.New(d, func(int64, any) {}, func(string, string) {})
	rec := &recorder{}
	o.Pub = rec
	l, err := handlers.NewLocal(o)
	if err != nil {
		t.Fatalf("local: %v", err)
	}
	var ws types.Workspace
	if err := l.Call(api.MethodWorkspacesCreate, types.WorkspaceCreateInput{Name: "w"}, &ws); err != nil {
		t.Fatalf("workspace: %v", err)
	}
	var app types.App
	if err := l.Call(api.MethodAppsCreate, map[string]any{
		"workspaceID": ws.ID,
		"body":        types.AppCreateInput{Name: "app", ProjectPath: t.TempDir()},
	}, &app); err != nil {
		t.Fatalf("app: %v", err)
	}
	return l, rec, app.ID
}

func TestUnknownMethodIsRefused(t *testing.T) {
	l, _, _ := setup(t)
	if err := l.Call("nope.nothing", nil, nil); api.KindOf(err) != api.KindInvalid {
		t.Fatalf("want an invalid-method error, got %v", err)
	}
}

func TestWritesAreAnnounced(t *testing.T) {
	l, rec, appID := setup(t)
	rec.changed = nil
	if err := l.Call(api.MethodAppsUpdate, map[string]any{"id": appID, "body": map[string]any{"name": "renamed"}}, nil); err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(rec.changed) == 0 {
		t.Fatal("a write did not reach the publisher")
	}
}

func TestApplyChangesWritesAndSkipsHidden(t *testing.T) {
	l, rec, appID := setup(t)
	visible, hidden := true, false
	one, secret := "1", "secret"
	for _, v := range []types.EnvVarCreateInput{
		{Key: "PORT", Value: &one, IncludeInAI: &visible},
		{Key: "SECRET", Value: &secret, IncludeInAI: &hidden},
	} {
		if err := l.Call(api.MethodEnvVarsCreate, map[string]any{"appID": appID, "body": v}, nil); err != nil {
			t.Fatalf("create %s: %v", v.Key, err)
		}
	}
	var sets []types.ConfigSet
	if err := l.Call(api.MethodConfigSetsList, map[string]any{"appID": appID}, &sets); err != nil || len(sets) == 0 {
		t.Fatalf("config sets: %v (%d)", err, len(sets))
	}
	rec.changed = nil
	patch := types.AppAIPatch{Env: &types.AppAIEnvPatch{Upsert: []types.AppAIEnvUpsert{
		{Key: "PORT", Value: "2"}, {Key: "SECRET", Value: "leaked"}, {Key: "NEW", Value: "x"},
	}}}
	if err := l.Call(api.MethodConfigSetsApplyChanges,
		map[string]any{"appID": appID, "configSetID": sets[0].ID, "patch": patch}, nil); err != nil {
		t.Fatalf("apply: %v", err)
	}
	var vars []types.EnvVar
	if err := l.Call(api.MethodEnvVarsList, map[string]any{"appID": appID}, &vars); err != nil {
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
