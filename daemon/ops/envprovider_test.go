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
