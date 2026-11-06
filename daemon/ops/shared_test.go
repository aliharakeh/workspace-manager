package ops_test

import (
	"strings"
	"testing"

	"workspace-manager/daemon/ops"
	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"
)

func sp(s string) *string { return &s }

// sharedFixture is a workspace with a "dev2" shared set holding DATABASE_URL
// and a hidden DATABASE_PASSWORD, and one app with a config set to link.
type sharedFixture struct {
	o        *ops.Ops
	appID    int64
	setID    int64 // the app's config set
	shared   types.SharedSet
	url, pwd types.SharedEnvVar
}

func newSharedFixture(t *testing.T) sharedFixture {
	t.Helper()
	o, _, appID := setup(t)
	app, err := o.AppsGet(appID)
	if err != nil {
		t.Fatal(err)
	}
	shared, err := o.SharedSetsCreate(app.WorkspaceID, "dev2")
	if err != nil {
		t.Fatal(err)
	}
	hidden := true
	url, err := o.SharedVarsCreate(shared.ID, types.EnvVarCreateInput{Key: "DATABASE_URL", Value: sp("postgres://dev2")})
	if err != nil {
		t.Fatal(err)
	}
	pwd, err := o.SharedVarsCreate(shared.ID, types.EnvVarCreateInput{Key: "DATABASE_PASSWORD", Value: sp("hunter2"), Private: &hidden})
	if err != nil {
		t.Fatal(err)
	}
	return sharedFixture{o: o, appID: appID, setID: *app.ActiveConfigSetID, shared: shared, url: url, pwd: pwd}
}

func (f sharedFixture) link(t *testing.T, ids ...int64) types.ConfigSet {
	t.Helper()
	cs, err := f.o.ConfigSetsLinkShared(f.setID, types.SharedLinkInput{SharedSetID: f.shared.ID, VarIDs: ids})
	if err != nil {
		t.Fatal(err)
	}
	return cs
}

func (f sharedFixture) vars(t *testing.T) map[string]types.EnvVar {
	t.Helper()
	list, err := f.o.EnvVarsList(f.appID, f.setID)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]types.EnvVar{}
	for _, v := range list {
		out[v.Key] = v
	}
	return out
}

func TestLinkTakesChosenVariablesAsReadOnlyRows(t *testing.T) {
	f := newSharedFixture(t)
	cs := f.link(t, f.url.ID)
	if cs.SharedSetID == nil || *cs.SharedSetID != f.shared.ID || cs.SharedSetName == nil || *cs.SharedSetName != "dev2" {
		t.Fatalf("config set not linked: %+v", cs)
	}
	vars := f.vars(t)
	if len(vars) != 1 || vars["DATABASE_URL"].Value != "postgres://dev2" || vars["DATABASE_URL"].SharedVarID == nil {
		t.Fatalf("unexpected rows: %+v", vars)
	}
	// The row is read-only in the app.
	id := vars["DATABASE_URL"].ID
	if _, err := f.o.EnvVarsUpdate(id, types.EnvVarUpdateInput{Value: sp("x")}); err == nil || api.KindOf(err) != api.KindInvalid {
		t.Fatalf("shared row was editable: %v", err)
	}
	if _, err := f.o.EnvVarsDelete(id); err == nil {
		t.Fatal("shared row was deletable")
	}
	// The app can still add its own variables next to it.
	if _, err := f.o.EnvVarsCreate(f.appID, f.setID, types.EnvVarCreateInput{Key: "PORT", Value: sp("3001")}); err != nil {
		t.Fatal(err)
	}
}

func TestWorkspaceEditsReachLinkedRowsImmediately(t *testing.T) {
	f := newSharedFixture(t)
	f.link(t, f.url.ID, f.pwd.ID)

	if _, err := f.o.SharedVarsUpdate(f.url.ID, types.EnvVarUpdateInput{Value: sp("postgres://new")}); err != nil {
		t.Fatal(err)
	}
	if got := f.vars(t)["DATABASE_URL"].Value; got != "postgres://new" {
		t.Fatalf("value did not propagate: %q", got)
	}

	// Options and the selected choice travel with the value.
	opts := []types.EnvVarOption{{Label: "a", Value: "A"}, {Label: "b", Value: "B"}}
	if _, err := f.o.SharedVarsUpdate(f.url.ID, types.EnvVarUpdateInput{Options: &opts, Selected: sp("b")}); err != nil {
		t.Fatal(err)
	}
	if v := f.vars(t)["DATABASE_URL"]; v.Value != "B" || v.Selected != "b" || len(v.Options) != 2 {
		t.Fatalf("choice did not propagate: %+v", v)
	}

	// So does the hidden flag.
	if v := f.vars(t)["DATABASE_PASSWORD"]; !v.Private {
		t.Fatalf("hidden flag lost on link: %+v", v)
	}
	visible := false
	if _, err := f.o.SharedVarsUpdate(f.pwd.ID, types.EnvVarUpdateInput{Private: &visible}); err != nil {
		t.Fatal(err)
	}
	if v := f.vars(t)["DATABASE_PASSWORD"]; v.Private {
		t.Fatalf("hidden flag did not propagate: %+v", v)
	}
}

func TestEditsReachEveryLinkedConfigSet(t *testing.T) {
	f := newSharedFixture(t)
	f.link(t, f.url.ID)
	// A second app and a second config set of the first app, both linked.
	app, _ := f.o.AppsGet(f.appID)
	other, err := f.o.AppsCreate(app.WorkspaceID, types.AppCreateInput{Name: "other", ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.o.ConfigSetsLinkShared(*other.ActiveConfigSetID, types.SharedLinkInput{SharedSetID: f.shared.ID, VarIDs: []int64{f.url.ID}}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.o.SharedVarsUpdate(f.url.ID, types.EnvVarUpdateInput{Value: sp("v2")}); err != nil {
		t.Fatal(err)
	}
	otherVars, _ := f.o.EnvVarsList(other.ID, 0)
	if len(otherVars) != 1 || otherVars[0].Value != "v2" || f.vars(t)["DATABASE_URL"].Value != "v2" {
		t.Fatal("not every linked config set was updated")
	}
	d, err := f.o.SharedSetsGet(f.shared.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range d.Vars {
		if v.ID == f.url.ID && len(v.UsedBy) != 2 {
			t.Fatalf("usage not listed: %+v", v.UsedBy)
		}
		if v.ID == f.pwd.ID && len(v.UsedBy) != 0 {
			t.Fatalf("unused variable reports usage: %+v", v.UsedBy)
		}
	}
}

func TestRenamePropagatesAndRefusesCollisions(t *testing.T) {
	f := newSharedFixture(t)
	f.link(t, f.url.ID)
	if _, err := f.o.SharedVarsUpdate(f.url.ID, types.EnvVarUpdateInput{Key: sp("DB_URL")}); err != nil {
		t.Fatal(err)
	}
	vars := f.vars(t)
	if _, old := vars["DATABASE_URL"]; old || vars["DB_URL"].Value != "postgres://dev2" {
		t.Fatalf("rename did not propagate: %+v", vars)
	}
	// An own variable with the target name blocks the rename and names where.
	if _, err := f.o.EnvVarsCreate(f.appID, f.setID, types.EnvVarCreateInput{Key: "TAKEN", Value: sp("1")}); err != nil {
		t.Fatal(err)
	}
	_, err := f.o.SharedVarsUpdate(f.url.ID, types.EnvVarUpdateInput{Key: sp("TAKEN")})
	if err == nil || api.KindOf(err) != api.KindConflict || !strings.Contains(err.Error(), "app /") {
		t.Fatalf("collision not refused with a location: %v", err)
	}
	if f.vars(t)["DB_URL"].Value == "" {
		t.Fatal("refused rename left the rows changed")
	}
}

func TestDeletingAVariableOrSetCleansTheApps(t *testing.T) {
	f := newSharedFixture(t)
	f.link(t, f.url.ID, f.pwd.ID)
	if _, err := f.o.SharedVarsDelete(f.pwd.ID); err != nil {
		t.Fatal(err)
	}
	if vars := f.vars(t); len(vars) != 1 || vars["DATABASE_URL"].ID == 0 {
		t.Fatalf("variable delete did not cascade: %+v", vars)
	}
	if _, err := f.o.SharedSetsDelete(f.shared.ID); err != nil {
		t.Fatal(err)
	}
	if vars := f.vars(t); len(vars) != 0 {
		t.Fatalf("set delete left rows: %+v", vars)
	}
	sets, _ := f.o.ConfigSetsList(f.appID)
	for _, s := range sets {
		if s.SharedSetID != nil {
			t.Fatalf("config set still linked to a deleted set: %+v", s)
		}
	}
}

func TestLinkConflictWithOwnVariable(t *testing.T) {
	f := newSharedFixture(t)
	if _, err := f.o.EnvVarsCreate(f.appID, f.setID, types.EnvVarCreateInput{Key: "DATABASE_URL", Value: sp("mine")}); err != nil {
		t.Fatal(err)
	}
	_, err := f.o.ConfigSetsLinkShared(f.setID, types.SharedLinkInput{SharedSetID: f.shared.ID, VarIDs: []int64{f.url.ID}})
	if err == nil || api.KindOf(err) != api.KindConflict {
		t.Fatalf("clash not reported as a conflict: %v", err)
	}
	if v := f.vars(t)["DATABASE_URL"]; v.Value != "mine" || v.SharedVarID != nil {
		t.Fatalf("refused link changed the own variable: %+v", v)
	}
	if _, err := f.o.ConfigSetsLinkShared(f.setID, types.SharedLinkInput{SharedSetID: f.shared.ID, VarIDs: []int64{f.url.ID}, Replace: true}); err != nil {
		t.Fatal(err)
	}
	if v := f.vars(t)["DATABASE_URL"]; v.Value != "postgres://dev2" || v.SharedVarID == nil {
		t.Fatalf("replace did not take the shared value: %+v", v)
	}
}

func TestRelinkChangesSelectionAndSource(t *testing.T) {
	f := newSharedFixture(t)
	f.link(t, f.url.ID, f.pwd.ID)
	// Drop one variable from the selection.
	f.link(t, f.url.ID)
	if vars := f.vars(t); len(vars) != 1 || vars["DATABASE_URL"].SharedVarID == nil {
		t.Fatalf("selection not narrowed: %+v", vars)
	}
	// Link to a differently named set: the old rows are replaced.
	app, _ := f.o.AppsGet(f.appID)
	other, err := f.o.SharedSetsCreate(app.WorkspaceID, "prod")
	if err != nil {
		t.Fatal(err)
	}
	pv, err := f.o.SharedVarsCreate(other.ID, types.EnvVarCreateInput{Key: "API_KEY", Value: sp("k")})
	if err != nil {
		t.Fatal(err)
	}
	cs, err := f.o.ConfigSetsLinkShared(f.setID, types.SharedLinkInput{SharedSetID: other.ID, VarIDs: []int64{pv.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if *cs.SharedSetName != "prod" {
		t.Fatalf("source not switched: %+v", cs)
	}
	if vars := f.vars(t); len(vars) != 1 || vars["API_KEY"].SharedVarID == nil {
		t.Fatalf("old rows were not replaced: %+v", vars)
	}
	// A variable of a set the config set is not linked to, or of another workspace, is refused.
	ws2, _ := f.o.WorkspacesCreate(types.WorkspaceCreateInput{Name: "w2"})
	foreign, _ := f.o.SharedSetsCreate(ws2.ID, "dev2")
	fv, _ := f.o.SharedVarsCreate(foreign.ID, types.EnvVarCreateInput{Key: "X", Value: sp("1")})
	if _, err := f.o.ConfigSetsLinkShared(f.setID, types.SharedLinkInput{SharedSetID: foreign.ID, VarIDs: []int64{fv.ID}}); err == nil {
		t.Fatal("linked to a shared set of another workspace")
	}
	if _, err := f.o.ConfigSetsLinkShared(f.setID, types.SharedLinkInput{SharedSetID: other.ID, VarIDs: []int64{f.url.ID}}); err == nil {
		t.Fatal("linked a variable that is not in the chosen shared set")
	}
	if _, err := f.o.ConfigSetsLinkShared(f.setID, types.SharedLinkInput{SharedSetID: other.ID}); err == nil {
		t.Fatal("linked with nothing selected")
	}
}

func TestUnlinkKeepsOrDeletes(t *testing.T) {
	for _, keep := range []bool{true, false} {
		f := newSharedFixture(t)
		f.link(t, f.url.ID)
		cs, err := f.o.ConfigSetsUnlinkShared(f.setID, keep)
		if err != nil {
			t.Fatal(err)
		}
		if cs.SharedSetID != nil {
			t.Fatalf("still linked: %+v", cs)
		}
		vars := f.vars(t)
		if keep {
			v := vars["DATABASE_URL"]
			if v.SharedVarID != nil || v.Value != "postgres://dev2" {
				t.Fatalf("kept row should become the app's own: %+v", v)
			}
			// The app's own now: editable, and no longer follows the workspace.
			if _, err := f.o.EnvVarsUpdate(v.ID, types.EnvVarUpdateInput{Value: sp("mine")}); err != nil {
				t.Fatal(err)
			}
			if _, err := f.o.SharedVarsUpdate(f.url.ID, types.EnvVarUpdateInput{Value: sp("later")}); err != nil {
				t.Fatal(err)
			}
			if f.vars(t)["DATABASE_URL"].Value != "mine" {
				t.Fatal("an unlinked row still followed the workspace")
			}
		} else if len(vars) != 0 {
			t.Fatalf("rows not deleted: %+v", vars)
		}
	}
}

func TestOtherWritesLeaveSharedRowsAlone(t *testing.T) {
	f := newSharedFixture(t)
	f.link(t, f.url.ID)
	// The AI's reviewed patch cannot change or delete a shared row.
	err := f.o.ConfigSetsApplyChanges(f.appID, f.setID, types.AppAIPatch{Env: &types.AppAIEnvPatch{
		Upsert: []types.AppAIEnvUpsert{{Key: "DATABASE_URL", Value: "ai"}},
		Delete: []string{"DATABASE_URL"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if v := f.vars(t)["DATABASE_URL"]; v.Value != "postgres://dev2" || v.SharedVarID == nil {
		t.Fatalf("AI patch changed a shared row: %+v", v)
	}
	// Copying a config set into a linked one keeps the shared rows and skips the keys they own.
	sets, _ := f.o.ConfigSetsList(f.appID)
	src, err := f.o.ConfigSetsCreate(f.appID, types.ConfigSetCreateInput{Name: "src"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.o.EnvVarsCreate(f.appID, src.ID, types.EnvVarCreateInput{Key: "DATABASE_URL", Value: sp("from-src")}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.o.EnvVarsCreate(f.appID, src.ID, types.EnvVarCreateInput{Key: "EXTRA", Value: sp("1")}); err != nil {
		t.Fatal(err)
	}
	_ = sets
	if _, err := f.o.ConfigSetsCopyFrom(f.setID, src.ID, &types.CopyParts{Env: true}); err != nil {
		t.Fatal(err)
	}
	vars := f.vars(t)
	if vars["DATABASE_URL"].Value != "postgres://dev2" || vars["DATABASE_URL"].SharedVarID == nil || vars["EXTRA"].Value != "1" {
		t.Fatalf("copy disturbed the shared rows: %+v", vars)
	}
}

func TestExportImportKeepsSharedLinks(t *testing.T) {
	f := newSharedFixture(t)
	f.link(t, f.url.ID, f.pwd.ID)
	if _, err := f.o.EnvVarsCreate(f.appID, f.setID, types.EnvVarCreateInput{Key: "PORT", Value: sp("3001")}); err != nil {
		t.Fatal(err)
	}
	app, _ := f.o.AppsGet(f.appID)

	// With secrets included, the shared rows are not repeated as app variables.
	bundle, err := f.o.WorkspacesExport(app.WorkspaceID, types.WorkspaceExportOptions{Secrets: types.SecretsInclude}, true)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Version != 2 || len(bundle.Workspace.SharedSets) != 1 {
		t.Fatalf("shared config missing from the bundle: v%d %+v", bundle.Version, bundle.Workspace.SharedSets)
	}
	cs := bundle.Apps[0].ConfigSets[0]
	if cs.SharedSet != "dev2" || len(cs.SharedKeys) != 2 || len(cs.EnvVars) != 1 || cs.EnvVars[0].Key != "PORT" {
		t.Fatalf("link not recorded: %+v", cs)
	}

	res, err := f.o.WorkspacesImport(bundle, types.WorkspaceImportOptions{Name: "copy", Folders: map[string]string{bundle.Apps[0].Key: t.TempDir()}})
	if err != nil {
		t.Fatal(err)
	}
	vars, err := f.o.EnvVarsList(res.AppIDs[0], 0)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]types.EnvVar{}
	for _, v := range vars {
		got[v.Key] = v
	}
	if len(got) != 3 || got["DATABASE_URL"].SharedVarID == nil || got["DATABASE_PASSWORD"].Value != "hunter2" || got["PORT"].SharedVarID != nil {
		t.Fatalf("imported rows wrong: %+v", got)
	}
	// The import is live too: its shared set is its own, not the original's.
	sets, _ := f.o.SharedSetsList(res.WorkspaceID)
	if len(sets) != 1 || sets[0].WorkspaceID == app.WorkspaceID {
		t.Fatalf("import did not create its own shared set: %+v", sets)
	}
	d, _ := f.o.SharedSetsGet(sets[0].ID)
	for _, v := range d.Vars {
		if v.Key == "DATABASE_URL" {
			if _, err := f.o.SharedVarsUpdate(v.ID, types.EnvVarUpdateInput{Value: sp("imported-new")}); err != nil {
				t.Fatal(err)
			}
		}
	}
	after, _ := f.o.EnvVarsList(res.AppIDs[0], 0)
	for _, v := range after {
		if v.Key == "DATABASE_URL" && v.Value != "imported-new" {
			t.Fatal("imported link is not live")
		}
	}
	if f.vars(t)["DATABASE_URL"].Value != "postgres://dev2" {
		t.Fatal("editing the copy changed the original workspace")
	}
}

func TestExportWithoutSharedConfigStaysVersionOne(t *testing.T) {
	o, _, appID := setup(t)
	app, _ := o.AppsGet(appID)
	bundle, err := o.WorkspacesExport(app.WorkspaceID, types.WorkspaceExportOptions{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Version != types.WorkspaceBundleBaseVersion {
		t.Fatalf("a bundle without shared config should stay readable by older builds: v%d", bundle.Version)
	}
}

func TestImportRejectsBrokenSharedLinks(t *testing.T) {
	f := newSharedFixture(t)
	f.link(t, f.url.ID)
	app, _ := f.o.AppsGet(f.appID)
	bundle, err := f.o.WorkspacesExport(app.WorkspaceID, types.WorkspaceExportOptions{Secrets: types.SecretsInclude}, true)
	if err != nil {
		t.Fatal(err)
	}
	folders := map[string]string{bundle.Apps[0].Key: t.TempDir()}
	bundle.Apps[0].ConfigSets[0].SharedKeys = []string{"NOPE"}
	if _, err := f.o.WorkspacesImport(bundle, types.WorkspaceImportOptions{Name: "x", Folders: folders}); err == nil {
		t.Fatal("imported a link to a key the shared set does not have")
	}
	bundle.Apps[0].ConfigSets[0].SharedKeys = []string{"DATABASE_URL"}
	bundle.Apps[0].ConfigSets[0].SharedSet = "ghost"
	if _, err := f.o.WorkspacesImport(bundle, types.WorkspaceImportOptions{Name: "x", Folders: folders}); err == nil {
		t.Fatal("imported a link to a missing shared set")
	}
}
