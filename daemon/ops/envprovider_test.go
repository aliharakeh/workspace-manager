package ops_test

import (
	"testing"
	"workspace-manager/shared/api/types"
)

func TestProviderBundleWithoutDeclaredKeys(t *testing.T) {
	f := newSharedFixture(t)
	f.link(t, f.url.ID)
	p := types.EnvProvider{Command: "echo private-command", Directory: t.TempDir(), Timeout: 30}
	if _, err := f.o.EnvProviderSave(true, f.shared.ID, p); err != nil {
		t.Fatal(err)
	}
	if _, err := f.o.EnvProviderSave(false, f.setID, p); err != nil {
		t.Fatal("script should not require existing local variable keys", err)
	}
	app, _ := f.o.AppsGet(f.appID)
	for _, trusted := range []bool{false, true} {
		b, err := f.o.WorkspacesExport(app.WorkspaceID, types.WorkspaceExportOptions{Secrets: types.SecretsInclude}, trusted)
		if err != nil {
			t.Fatal(err)
		}
		if (b.Workspace.SharedSets[0].EnvProvider != nil) != trusted {
			t.Fatal("provider export privacy mismatch")
		}
		if trusted {
			if b.Version != 3 {
				t.Fatal("dynamic config requires bundle version 3")
			}
			result, err := f.o.WorkspacesImport(b, types.WorkspaceImportOptions{Name: "copy", BaseFolder: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			sets, err := f.o.SharedSetsList(result.WorkspaceID)
			if err != nil || len(sets) != 1 {
				t.Fatal(err)
			}
			got, err := f.o.EnvProviderGet(true, sets[0].ID)
			if err != nil || got.Command != p.Command {
				t.Fatal("provider lost on import")
			}
		}
	}
}

func TestProviderRunSavesLinkableVariables(t *testing.T) {
	f := newSharedFixture(t)
	p := types.EnvProvider{Command: `echo {"GEN_TOKEN":"abc"}`, Directory: t.TempDir(), Timeout: 30}
	if _, err := f.o.EnvProviderSave(true, f.shared.ID, p); err != nil {
		t.Fatal(err)
	}
	if _, err := f.o.EnvProviderRun(true, f.shared.ID); err != nil {
		t.Fatal(err)
	}
	detail, err := f.o.SharedSetsGet(f.shared.ID)
	if err != nil {
		t.Fatal(err)
	}
	var id int64
	for _, v := range detail.Vars {
		if v.Key == "GEN_TOKEN" && v.Value == "abc" && v.Private {
			id = v.ID
		}
	}
	if id == 0 {
		t.Fatal("script output was not saved as a shared variable")
	}
	if _, err := f.o.ConfigSetsLinkShared(f.setID, types.SharedLinkInput{SharedSetID: f.shared.ID, VarIDs: []int64{id}}); err != nil {
		t.Fatal("generated variable should be linkable:", err)
	}
}

func TestProviderOutputOverridesManualVariables(t *testing.T) {
	f := newSharedFixture(t)
	f.link(t, f.url.ID)
	manual, err := f.o.EnvVarsCreate(f.appID, f.setID, types.EnvVarCreateInput{Key: "MANUAL", Value: ptr("old")})
	if err != nil {
		t.Fatal(err)
	}
	sharedManual, err := f.o.SharedVarsCreate(f.shared.ID, types.EnvVarCreateInput{Key: "SHARED_MANUAL", Value: ptr("old")})
	if err != nil {
		t.Fatal(err)
	}
	cmd := `echo {"SHARED_MANUAL":"new"}`
	_, _ = f.o.EnvVarsCreate(f.appID, f.setID, types.EnvVarCreateInput{Key: "SHARED_MANUAL", Value: ptr("old")})
	if _, err := f.o.EnvProviderSave(true, f.shared.ID, types.EnvProvider{Command: cmd, Directory: t.TempDir(), Timeout: 30}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.o.EnvProviderRun(true, f.shared.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := f.o.SharedVarsGet(sharedManual.ID)
	if got.Value != "new" {
		t.Fatalf("shared variable not overridden: %q", got.Value)
	}
	vars, _ := f.o.EnvVarsList(f.appID, f.setID)
	for _, v := range vars {
		if v.Key == "SHARED_MANUAL" && v.Value != "new" {
			t.Fatalf("linked app variable not overridden: %q", v.Value)
		}
		if v.ID == manual.ID && v.Value != "old" {
			t.Fatal("unrelated variable changed")
		}
	}
}

func ptr(s string) *string { return &s }
