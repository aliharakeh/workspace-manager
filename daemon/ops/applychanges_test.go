package ops_test

import (
	"testing"

	"workspace-manager/shared/api/types"
	"workspace-manager/daemon/db"
	"workspace-manager/daemon/ops"
)

type recorder struct{ changed []string }

func (*recorder) Publish(string, int64, any) {}
func (r *recorder) Changed(entity string, _ ...int64) {
	r.changed = append(r.changed, entity)
}

func setup(t *testing.T) (*ops.Ops, *recorder, int64) {
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
	ws, err := o.WorkspacesCreate(types.WorkspaceCreateInput{Name: "w"})
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	app, err := o.AppsCreate(ws.ID, types.AppCreateInput{Name: "app", ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatalf("app: %v", err)
	}
	return o, rec, app.ID
}

func TestWritesAreAnnounced(t *testing.T) {
	o, rec, appID := setup(t)
	rec.changed = nil
	name := "renamed"
	if _, err := o.AppsUpdate(appID, types.AppUpdateInput{Name: &name}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(rec.changed) == 0 {
		t.Fatal("a write did not reach the publisher")
	}
}

func TestApplyChangesWritesAndSkipsExcludedValues(t *testing.T) {
	o, rec, appID := setup(t)
	visible, hidden := true, false
	one, secret := "1", "secret"
	for _, v := range []types.EnvVarCreateInput{
		{Key: "PORT", Value: &one, IncludeInAI: &visible},
		{Key: "SECRET", Value: &secret, IncludeInAI: &hidden},
	} {
		if _, err := o.EnvVarsCreate(appID, v); err != nil {
			t.Fatalf("create %s: %v", v.Key, err)
		}
	}
	sets, err := o.ConfigSetsList(appID)
	if err != nil || len(sets) == 0 {
		t.Fatalf("config sets: %v (%d)", err, len(sets))
	}
	rec.changed = nil
	patch := types.AppAIPatch{Env: &types.AppAIEnvPatch{Upsert: []types.AppAIEnvUpsert{
		{Key: "PORT", Value: "2"}, {Key: "SECRET", Value: "leaked"}, {Key: "NEW", Value: "x"},
	}}}
	if err := o.ConfigSetsApplyChanges(appID, sets[0].ID, patch); err != nil {
		t.Fatalf("apply: %v", err)
	}
	vars, err := o.EnvVarsList(appID)
	if err != nil {
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
