package ops_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"workspace-manager/daemon/db"
	"workspace-manager/daemon/ops"
	"workspace-manager/shared/api/types"
)

// testLogo is a 1×1 PNG as a data URL.
const testLogo = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="

func logoFiles(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(db.LogoDir())
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

func TestWorkspaceLogoFile(t *testing.T) {
	o, _, ws, _ := seedWorkspace(t)
	if ws.LogoPath == "" || filepath.Dir(ws.LogoPath) != db.LogoDir() {
		t.Fatalf("create should store the logo in the logos folder: %q", ws.LogoPath)
	}
	if _, err := os.Stat(ws.LogoPath); err != nil {
		t.Fatalf("logo file: %v", err)
	}
	got, err := o.WorkspacesLogo(ws.ID)
	must(t, err)
	if !strings.HasPrefix(got, "data:image/png;base64,") {
		t.Errorf("workspaces.logo: %.40q", got)
	}

	for _, bad := range []string{
		"rocket",
		"data:image/svg+xml;base64,PHN2Zz4=",
		"data:image/png;base64,aGVsbG8=", // not an image
		"data:image/png;base64," + strings.Repeat("A", 2*db.MaxLogo),
	} {
		if _, err := o.WorkspacesUpdate(ws.ID, types.WorkspaceUpdateInput{LogoData: &bad}); err == nil {
			t.Errorf("logo %.40q should be refused", bad)
		}
	}
	if files := logoFiles(t); len(files) != 1 {
		t.Errorf("refused logos should leave no files: %v", files)
	}

	// A name-only update keeps the logo; a new one replaces the old file.
	name := "Shop 2"
	same, err := o.WorkspacesUpdate(ws.ID, types.WorkspaceUpdateInput{Name: &name})
	must(t, err)
	if same.LogoPath != ws.LogoPath {
		t.Errorf("a name update should keep the logo: %q", same.LogoPath)
	}
	logo := testLogo
	replaced, err := o.WorkspacesUpdate(ws.ID, types.WorkspaceUpdateInput{LogoData: &logo})
	must(t, err)
	if replaced.LogoPath == ws.LogoPath {
		t.Error("a new logo should get a new file")
	}
	if _, err := os.Stat(ws.LogoPath); !os.IsNotExist(err) {
		t.Errorf("the old logo file should be removed: %v", err)
	}

	empty := ""
	cleared, err := o.WorkspacesUpdate(ws.ID, types.WorkspaceUpdateInput{LogoData: &empty})
	must(t, err)
	if cleared.LogoPath != "" || len(logoFiles(t)) != 0 {
		t.Errorf("an empty logo should clear it: %q %v", cleared.LogoPath, logoFiles(t))
	}

	again, err := o.WorkspacesUpdate(ws.ID, types.WorkspaceUpdateInput{LogoData: &logo})
	must(t, err)
	_, err = o.WorkspacesDelete(again.ID)
	must(t, err)
	if files := logoFiles(t); len(files) != 0 {
		t.Errorf("deleting the workspace should remove its logo: %v", files)
	}
}

func TestImportKeepsValidLogoAndDropsBadOne(t *testing.T) {
	o, _, ws, _ := seedWorkspace(t)
	b, err := o.WorkspacesExport(ws.ID, types.WorkspaceExportOptions{}, true)
	must(t, err)
	if b.Workspace.Icon == nil || !strings.HasPrefix(*b.Workspace.Icon, "data:image/png;base64,") {
		t.Fatalf("export should carry the logo image: %v", b.Workspace.Icon)
	}

	res, err := o.WorkspacesImport(b, types.WorkspaceImportOptions{Name: "Copy", BaseFolder: t.TempDir()})
	must(t, err)
	imported := findWorkspace(t, o, res.WorkspaceID)
	if imported.LogoPath == "" || imported.LogoPath == ws.LogoPath {
		t.Errorf("import should store its own logo file: %q", imported.LogoPath)
	}

	bad := "rocket"
	b.Workspace.Icon = &bad
	res, err = o.WorkspacesImport(b, types.WorkspaceImportOptions{Name: "Copy 2", BaseFolder: t.TempDir()})
	must(t, err)
	if imported := findWorkspace(t, o, res.WorkspaceID); imported.LogoPath != "" {
		t.Errorf("import should drop an invalid logo: %q", imported.LogoPath)
	}
}

func findWorkspace(t *testing.T, o *ops.Ops, id int64) types.Workspace {
	t.Helper()
	all, err := o.WorkspacesList()
	must(t, err)
	for _, w := range all {
		if w.ID == id {
			return w
		}
	}
	t.Fatalf("workspace %d not found", id)
	return types.Workspace{}
}
