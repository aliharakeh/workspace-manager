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

// seedWorkspace builds a workspace with two apps under one root, each with a
// visible and a hidden env var, two templates, commands, a second config set
// and a worktree link.
func seedWorkspace(t *testing.T) (*ops.Ops, *recorder, types.Workspace, string) {
	t.Helper()
	t.Setenv("WM_DATA_DIR", t.TempDir())
	d, err := db.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	o := ops.New(d, func(int64, any) {}, func(string, string) {})
	rec := &recorder{}
	o.Pub = rec

	root := t.TempDir()
	icon := "rocket"
	ws, err := o.WorkspacesCreate(types.WorkspaceCreateInput{Name: "Shop", Icon: &icon})
	must(t, err)
	hidden, shown := false, true
	for _, name := range []string{"api", "web"} {
		dir := filepath.Join(root, name)
		must(t, os.MkdirAll(dir, 0o755))
		app, err := o.AppsCreate(ws.ID, types.AppCreateInput{Name: name, ProjectPath: dir})
		must(t, err)
		v1, v2 := "http://localhost", "hunter2"
		_, err = o.EnvVarsCreate(app.ID, types.EnvVarCreateInput{Key: "URL", Value: &v1, IncludeInAI: &shown})
		must(t, err)
		_, err = o.EnvVarsCreate(app.ID, types.EnvVarCreateInput{Key: "TOKEN", Value: &v2, IncludeInAI: &hidden})
		must(t, err)
		c1, c2 := "plain", "password=hunter2"
		_, err = o.TemplatesCreate(app.ID, types.TemplateCreateInput{FilePath: "a.txt", Content: &c1, IncludeInAI: &shown})
		must(t, err)
		_, err = o.TemplatesCreate(app.ID, types.TemplateCreateInput{FilePath: ".env", Content: &c2, IncludeInAI: &hidden})
		must(t, err)
		set, err := o.DB.ResolveActive(o.Ctx(), app.ID)
		must(t, err)
		label := "serve"
		_, err = o.DB.UpsertRunConfig(o.Ctx(), set.ID, types.KindDev, nil, []types.RunCommandInput{{Label: &label, Command: "npm run dev"}})
		must(t, err)
		_, err = o.DB.UpsertRunConfig(o.Ctx(), set.ID, types.KindSetup, nil, []types.RunCommandInput{{Command: "npm i"}})
		must(t, err)
		_, err = o.ConfigSetsCreate(app.ID, types.ConfigSetCreateInput{Name: "staging", Activate: &shown})
		must(t, err)
		_, err = o.WorktreeLinksSave(app.ID, []string{"node_modules"})
		must(t, err)
	}
	return o, rec, ws, root
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	o, rec, ws, root := seedWorkspace(t)
	seeded, err := o.AppsList(ws.ID)
	must(t, err)
	must(t, o.DB.SetAppExcludeFromStartAllT(o.Ctx(), seeded[1].ID, true))
	bundle, err := o.WorkspacesExport(ws.ID, types.WorkspaceExportOptions{Secrets: types.SecretsInclude}, true)
	must(t, err)
	if bundle.Format != types.WorkspaceBundleFormat || bundle.Version != 1 || bundle.Secrets != types.SecretsInclude {
		t.Fatalf("bad header: %+v", bundle)
	}
	if len(bundle.Apps) != 2 || bundle.Apps[0].Path != "api" || bundle.Apps[0].Absolute {
		t.Fatalf("apps should be relative to the common root: %+v", bundle.Apps)
	}

	// Import under a new base folder, with a different name.
	newBase := t.TempDir()
	must(t, os.MkdirAll(filepath.Join(newBase, "api"), 0o755))
	rec.changed = nil
	res, err := o.WorkspacesImport(bundle, types.WorkspaceImportOptions{Name: "Shop copy", BaseFolder: newBase})
	must(t, err)
	if res.WorkspaceID == 0 || len(res.AppIDs) != 2 || res.Name != "Shop copy" {
		t.Fatalf("result: %+v", res)
	}
	if res.Apps[0].FolderMissing || !res.Apps[1].FolderMissing {
		t.Errorf("folder flags: %+v", res.Apps)
	}
	if got := strings.Join(rec.changed, ","); got != "workspace,app" {
		t.Errorf("announced %q, want workspace,app", got)
	}

	apps, err := o.AppsList(res.WorkspaceID)
	must(t, err)
	if len(apps) != 2 || apps[0].Name != "api" || apps[1].Name != "web" {
		t.Fatalf("apps and their order: %+v", apps)
	}
	if apps[0].ExcludeFromStartAll || !apps[1].ExcludeFromStartAll {
		t.Errorf("exclude from Start all should round-trip: %v, %v", apps[0].ExcludeFromStartAll, apps[1].ExcludeFromStartAll)
	}
	if apps[0].ProjectPath != filepath.Join(newBase, "api") {
		t.Errorf("remapped folder: %s", apps[0].ProjectPath)
	}
	if apps[0].ActiveConfigSetName == nil || *apps[0].ActiveConfigSetName != "staging" {
		t.Errorf("active config set: %v", apps[0].ActiveConfigSetName)
	}
	// The imported workspace is a copy: the original is untouched.
	if orig, _ := o.AppsList(ws.ID); orig[0].ProjectPath != filepath.Join(root, "api") {
		t.Errorf("original changed: %+v", orig)
	}

	sets, err := o.ConfigSetsList(apps[0].ID)
	must(t, err)
	if len(sets) != 2 {
		t.Fatalf("config sets: %+v", sets)
	}
	for _, s := range sets {
		if s.Name != "Default" {
			continue
		}
		d, err := o.ConfigSetsGetDetail(s.ID)
		must(t, err)
		vals := map[string]types.EnvVar{}
		for _, v := range d.EnvVars {
			vals[v.Key] = v
		}
		if vals["URL"].Value != "http://localhost" || !vals["URL"].IncludeInAI || vals["TOKEN"].Value != "hunter2" || vals["TOKEN"].IncludeInAI {
			t.Errorf("env vars: %+v", vals)
		}
		if len(d.Templates) != 2 {
			t.Errorf("templates: %+v", d.Templates)
		}
		if d.DevConfig == nil || len(d.DevConfig.Commands) != 1 || d.DevConfig.Commands[0].Command != "npm run dev" || *d.DevConfig.Commands[0].Label != "serve" {
			t.Errorf("dev: %+v", d.DevConfig)
		}
		if d.SetupConfig == nil || d.SetupConfig.Mode != "sequential" || len(d.SetupConfig.Commands) != 1 {
			t.Errorf("setup: %+v", d.SetupConfig)
		}
		if d.BuildConfig == nil || d.ProdConfig == nil {
			t.Errorf("all four command lists should exist")
		}
	}
	links, _ := o.WorktreeLinksGet(apps[0].ID)
	if len(links) != 1 || links[0] != "node_modules" {
		t.Errorf("worktree links: %v", links)
	}
}

func bundleLeaks(b types.WorkspaceBundle) bool {
	for _, a := range b.Apps {
		for _, cs := range a.ConfigSets {
			for _, v := range cs.EnvVars {
				if strings.Contains(v.Value, "hunter2") {
					return true
				}
			}
			for _, tpl := range cs.Templates {
				if strings.Contains(tpl.Content, "hunter2") {
					return true
				}
			}
		}
	}
	return false
}

func defaultSet(t *testing.T, b types.WorkspaceBundle) types.BundleConfigSet {
	t.Helper()
	for _, cs := range b.Apps[0].ConfigSets {
		if cs.Name == "Default" {
			return cs
		}
	}
	t.Fatal("no Default config set")
	return types.BundleConfigSet{}
}

func TestExportSecretModes(t *testing.T) {
	o, _, ws, _ := seedWorkspace(t)

	// Placeholder is the default: keys and paths stay, secret values do not.
	b, err := o.WorkspacesExport(ws.ID, types.WorkspaceExportOptions{}, true)
	must(t, err)
	if b.Secrets != types.SecretsPlaceholder || bundleLeaks(b) {
		t.Fatalf("default export: secrets=%s leaks=%v", b.Secrets, bundleLeaks(b))
	}
	cs := defaultSet(t, b)
	vars := map[string]types.BundleEnvVar{}
	for _, v := range cs.EnvVars {
		vars[v.Key] = v
	}
	if len(vars) != 2 || !vars["TOKEN"].Redacted || vars["TOKEN"].Value != types.SecretPlaceholder {
		t.Errorf("env vars: %+v", cs.EnvVars)
	}
	if vars["URL"].Value != "http://localhost" || vars["URL"].Redacted {
		t.Errorf("a visible value should be kept: %+v", vars["URL"])
	}
	tpls := map[string]types.BundleTemplate{}
	for _, tp := range cs.Templates {
		tpls[tp.FilePath] = tp
	}
	if len(tpls) != 2 || !tpls[".env"].Redacted || tpls["a.txt"].Redacted || tpls["a.txt"].Content != "plain" {
		t.Errorf("templates: %+v", cs.Templates)
	}

	// Exclude drops the secret items.
	b, err = o.WorkspacesExport(ws.ID, types.WorkspaceExportOptions{Secrets: types.SecretsExclude}, true)
	must(t, err)
	cs = defaultSet(t, b)
	if bundleLeaks(b) || len(cs.EnvVars) != 1 || len(cs.Templates) != 1 || cs.Templates[0].FilePath != "a.txt" {
		t.Errorf("exclude: %+v", cs)
	}

	// An untrusted (agent) caller asking for include gets placeholder.
	b, err = o.WorkspacesExport(ws.ID, types.WorkspaceExportOptions{Secrets: types.SecretsInclude}, false)
	must(t, err)
	if b.Secrets != types.SecretsPlaceholder || bundleLeaks(b) {
		t.Errorf("untrusted include: secrets=%s leaks=%v", b.Secrets, bundleLeaks(b))
	}
	if _, err := o.WorkspacesExport(ws.ID, types.WorkspaceExportOptions{Secrets: "everything"}, true); err == nil {
		t.Error("an unknown secrets mode should be refused")
	}
}

func TestImportBlanksRedactedValues(t *testing.T) {
	o, _, ws, _ := seedWorkspace(t)
	b, err := o.WorkspacesExport(ws.ID, types.WorkspaceExportOptions{}, true)
	must(t, err)
	dry, err := o.WorkspacesImport(b, types.WorkspaceImportOptions{DryRun: true})
	must(t, err)
	// Two config sets per app, both carrying the redacted items.
	if dry.WorkspaceID != 0 || dry.Apps[0].EmptyEnvVars != 1 || dry.Apps[0].EmptyTemplates != 1 {
		t.Errorf("dry run: %+v", dry)
	}
	if list, _ := o.WorkspacesList(); len(list) != 1 {
		t.Fatalf("a dry run must not write: %d workspaces", len(list))
	}

	res, err := o.WorkspacesImport(b, types.WorkspaceImportOptions{})
	must(t, err)
	apps, _ := o.AppsList(res.WorkspaceID)
	sets, _ := o.ConfigSetsList(apps[0].ID)
	seen := 0
	for _, s := range sets {
		det, _ := o.ConfigSetsGetDetail(s.ID)
		for _, v := range det.EnvVars {
			if v.Key == "TOKEN" {
				seen++
				if v.Value != "" || v.IncludeInAI {
					t.Errorf("TOKEN should come in empty and hidden: %+v", v)
				}
			}
		}
		for _, tpl := range det.Templates {
			if tpl.FilePath == ".env" && (tpl.Content != "" || tpl.IncludeInAI) {
				t.Errorf(".env should come in empty and hidden: %+v", tpl)
			}
		}
	}
	if seen == 0 {
		t.Error("TOKEN key should survive the round trip")
	}
}

func TestImportFolderRemapAndAtomicity(t *testing.T) {
	o, _, ws, _ := seedWorkspace(t)
	b, err := o.WorkspacesExport(ws.ID, types.WorkspaceExportOptions{Secrets: types.SecretsExclude}, true)
	must(t, err)

	custom := t.TempDir()
	res, err := o.WorkspacesImport(b, types.WorkspaceImportOptions{Folders: map[string]string{"web": custom}})
	must(t, err)
	apps, _ := o.AppsList(res.WorkspaceID)
	if apps[1].ProjectPath != custom || res.Apps[1].FolderMissing {
		t.Errorf("override ignored: %+v", apps[1])
	}

	// A bad second app refuses the whole import.
	before, _ := o.WorkspacesList()
	bad := b
	bad.Apps = append([]types.BundleApp{}, b.Apps...)
	bad.Apps[1].ConfigSets = append([]types.BundleConfigSet{}, bad.Apps[1].ConfigSets...)
	bad.Apps[1].ConfigSets[0].Templates = []types.BundleTemplate{{FilePath: "../../etc/passwd"}}
	if _, err := o.WorkspacesImport(bad, types.WorkspaceImportOptions{}); err == nil {
		t.Fatal("a template path escaping the app folder should be refused")
	}
	if after, _ := o.WorkspacesList(); len(after) != len(before) {
		t.Errorf("a failed import left a workspace behind")
	}

	// Duplicate config set names are refused too.
	dup := b
	dup.Apps = append([]types.BundleApp{}, b.Apps...)
	dup.Apps[0].ConfigSets = append(append([]types.BundleConfigSet{}, b.Apps[0].ConfigSets...), b.Apps[0].ConfigSets[0])
	if _, err := o.WorkspacesImport(dup, types.WorkspaceImportOptions{}); err == nil {
		t.Error("duplicate config sets should be refused")
	}
	if after, _ := o.WorkspacesList(); len(after) != len(before) {
		t.Errorf("a failed import left a workspace behind")
	}
}

func TestImportRejectsForeignBundles(t *testing.T) {
	o, _, _, _ := seedWorkspace(t)
	for name, b := range map[string]types.WorkspaceBundle{
		"format":  {Format: "other", Version: 1},
		"version": {Format: types.WorkspaceBundleFormat, Version: 99, Workspace: types.BundleWorkspace{Name: "x"}},
		"name":    {Format: types.WorkspaceBundleFormat, Version: 1},
	} {
		if _, err := o.WorkspacesImport(b, types.WorkspaceImportOptions{}); err == nil {
			t.Errorf("%s: should be refused", name)
		}
	}
}
