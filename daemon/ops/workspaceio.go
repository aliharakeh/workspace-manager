package ops

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"workspace-manager/daemon/db"
	"workspace-manager/shared/api"
	"workspace-manager/shared/api/lib"
	"workspace-manager/shared/api/types"
)

var bundleKinds = []string{types.KindSetup, types.KindDev, types.KindBuild, types.KindProd}

// WorkspacesExport bundles a workspace. trusted says whether the caller may
// receive secrets (the GUI): for any other client, a request for "include" is
// exported as "placeholder", so private=false values never reach an
// agent.
func (o *Ops) WorkspacesExport(id int64, opts types.WorkspaceExportOptions, trusted bool) (types.WorkspaceBundle, error) {
	mode := opts.Secrets
	switch mode {
	case "":
		mode = types.SecretsPlaceholder
	case types.SecretsExclude, types.SecretsPlaceholder, types.SecretsInclude:
	default:
		return types.WorkspaceBundle{}, api.Errf(api.KindInvalid, "secrets must be exclude, placeholder or include")
	}
	if mode == types.SecretsInclude && !trusted {
		mode = types.SecretsPlaceholder
	}
	ws, err := o.DB.GetWorkspaceT(o.Ctx(), id)
	if err != nil {
		return types.WorkspaceBundle{}, err
	}
	apps, err := o.DB.ListAppsByWorkspaceT(o.Ctx(), id)
	if err != nil {
		return types.WorkspaceBundle{}, err
	}
	paths := make([]string, len(apps))
	for i, a := range apps {
		paths[i] = a.ProjectPath
	}
	root := commonRoot(paths)
	bundle := types.WorkspaceBundle{
		Format: types.WorkspaceBundleFormat, Version: types.WorkspaceBundleBaseVersion,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Secrets:    mode,
		SourceRoot: filepath.ToSlash(root),
		Workspace:  types.BundleWorkspace{Name: ws.Name, Icon: ws.Icon, Color: ws.Color},
		Apps:       []types.BundleApp{},
	}
	sharedSets, err := o.DB.ListWorkspaceConfigSets(o.Ctx(), id)
	if err != nil {
		return types.WorkspaceBundle{}, err
	}
	for _, s := range sharedSets {
		rows, err := o.DB.ListWorkspaceEnvVars(o.Ctx(), s.ID)
		if err != nil {
			return types.WorkspaceBundle{}, err
		}
		bs := types.BundleSharedSet{Name: s.Name, Vars: []types.BundleEnvVar{}}
		for _, r := range rows {
			if bv, ok := bundleEnvVar(sharedVarAsEnv(r), mode); ok {
				bs.Vars = append(bs.Vars, bv)
			}
		}
		bundle.Workspace.SharedSets = append(bundle.Workspace.SharedSets, bs)
	}
	if len(bundle.Workspace.SharedSets) > 0 {
		bundle.Version = types.WorkspaceBundleVersion
	}
	used := map[string]bool{}
	for _, a := range apps {
		ba := types.BundleApp{Key: uniqueKey(slug(a.Name), used), Name: a.Name, ExcludeFromStartAll: a.ExcludeFromStartAll}
		if rel, ok := relativeTo(root, a.ProjectPath); ok && root != "" {
			ba.Path = rel
		} else {
			ba.Path, ba.Absolute = filepath.ToSlash(a.ProjectPath), true
		}
		if a.ActiveConfigSetName != nil {
			ba.ActiveConfigSet = *a.ActiveConfigSetName
		}
		sets, err := o.DB.ListConfigSetsByAppT(o.Ctx(), a.ID)
		if err != nil {
			return types.WorkspaceBundle{}, err
		}
		for _, set := range sets {
			d, err := o.DB.ConfigSetDetailT(o.Ctx(), set)
			if err != nil {
				return types.WorkspaceBundle{}, err
			}
			ba.ConfigSets = append(ba.ConfigSets, bundleConfigSet(d, mode))
		}
		if ba.ConfigSets == nil {
			ba.ConfigSets = []types.BundleConfigSet{}
		}
		links, err := o.DB.ListWorktreeLinksT(o.Ctx(), a.ID)
		if err != nil {
			return types.WorkspaceBundle{}, err
		}
		if links == nil {
			links = []string{}
		}
		ba.WorktreeLinks = links
		bundle.Apps = append(bundle.Apps, ba)
	}
	return bundle, nil
}

// bundleEnvVar is a variable as a bundle writes it under a secrets mode; false
// means it is left out.
func bundleEnvVar(v types.EnvVar, mode string) (types.BundleEnvVar, bool) {
	secret := v.Private
	switch {
	case secret && mode == types.SecretsExclude:
		return types.BundleEnvVar{}, false
	case secret && mode == types.SecretsPlaceholder:
		return types.BundleEnvVar{Key: v.Key, Value: types.SecretPlaceholder, Redacted: true}, true
	}
	return types.BundleEnvVar{Key: v.Key, Value: v.Value, Private: v.Private, Options: v.Options, Selected: v.Selected}, true
}

func bundleConfigSet(d types.ConfigSetDetail, mode string) types.BundleConfigSet {
	out := types.BundleConfigSet{
		Name: d.Name, EnvVars: []types.BundleEnvVar{}, Templates: []types.BundleTemplate{},
		Commands: map[string]types.BundleCommandList{},
	}
	for _, v := range d.EnvVars {
		bv, ok := bundleEnvVar(v, mode)
		if !ok {
			continue
		}
		if v.SharedVarID != nil {
			// A shared row is recorded as a link, not copied: the workspace owns it.
			out.SharedKeys = append(out.SharedKeys, v.Key)
			continue
		}
		out.EnvVars = append(out.EnvVars, bv)
	}
	if len(out.SharedKeys) > 0 && d.SharedSetName != nil {
		out.SharedSet = *d.SharedSetName
	} else {
		out.SharedKeys = nil
	}
	for _, t := range d.Templates {
		secret := t.Private
		switch {
		case secret && mode == types.SecretsExclude:
			continue
		case secret && mode == types.SecretsPlaceholder:
			out.Templates = append(out.Templates, types.BundleTemplate{FilePath: t.FilePath, Content: types.SecretPlaceholder, Redacted: true})
		default:
			out.Templates = append(out.Templates, types.BundleTemplate{FilePath: t.FilePath, Content: t.Content, Private: t.Private})
		}
	}
	cfgs := map[string]*types.RunConfig{
		types.KindSetup: d.SetupConfig, types.KindDev: d.DevConfig,
		types.KindBuild: d.BuildConfig, types.KindProd: d.ProdConfig,
	}
	for _, kind := range bundleKinds {
		list := types.BundleCommandList{Mode: db.DefaultMode(kind), Commands: []types.BundleCommand{}}
		if c := cfgs[kind]; c != nil {
			list.Mode = c.Mode
			for _, cmd := range c.Commands {
				list.Commands = append(list.Commands, types.BundleCommand{Label: cmd.Label, Command: cmd.Command})
			}
		}
		out.Commands[kind] = list
	}
	return out
}

// WorkspacesImport creates a workspace from a bundle in one transaction, or
// with DryRun reports what it would create.
func (o *Ops) WorkspacesImport(bundle types.WorkspaceBundle, opts types.WorkspaceImportOptions) (types.WorkspaceImportResult, error) {
	if bundle.Format != types.WorkspaceBundleFormat {
		return types.WorkspaceImportResult{}, api.Errf(api.KindInvalid, "not a workspace bundle")
	}
	if bundle.Version < 1 || bundle.Version > types.WorkspaceBundleVersion {
		return types.WorkspaceImportResult{}, api.Errf(api.KindInvalid, "unsupported bundle version %d (this build reads up to %d)", bundle.Version, types.WorkspaceBundleVersion)
	}
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		name = strings.TrimSpace(bundle.Workspace.Name)
	}
	if name == "" {
		return types.WorkspaceImportResult{}, api.Errf(api.KindInvalid, "name is required")
	}
	color, err := normalizeWorkspaceColor(bundle.Workspace.Color)
	if err != nil {
		return types.WorkspaceImportResult{}, api.Errf(api.KindInvalid, "%v", err)
	}
	base := strings.TrimSpace(opts.BaseFolder)
	if base == "" {
		base = filepath.FromSlash(bundle.SourceRoot)
	}

	if err := validateBundleShared(bundle.Workspace.SharedSets, bundle.Apps); err != nil {
		return types.WorkspaceImportResult{}, api.Errf(api.KindInvalid, "%v", err)
	}

	type plannedApp struct {
		src    types.BundleApp
		folder string
	}
	planned := make([]plannedApp, 0, len(bundle.Apps))
	result := types.WorkspaceImportResult{Name: name, DryRun: opts.DryRun, Apps: []types.WorkspaceImportApp{}, AppIDs: []int64{}}
	keys := map[string]bool{}
	for i, a := range bundle.Apps {
		a.Name = strings.TrimSpace(a.Name)
		if a.Name == "" {
			return result, api.Errf(api.KindInvalid, "app %d has no name", i+1)
		}
		if a.Key == "" {
			a.Key = strconv.Itoa(i + 1)
		}
		if keys[a.Key] {
			return result, api.Errf(api.KindInvalid, "app key %q is used twice", a.Key)
		}
		keys[a.Key] = true
		folder, err := importFolder(a, base, opts.Folders[a.Key])
		if err != nil {
			return result, api.Errf(api.KindInvalid, "%s: %v", a.Name, err)
		}
		if err := validateBundleApp(&a); err != nil {
			return result, api.Errf(api.KindInvalid, "%s: %v", a.Name, err)
		}
		planned = append(planned, plannedApp{a, folder})
		info := types.WorkspaceImportApp{Key: a.Key, Name: a.Name, Folder: folder, ConfigSets: len(a.ConfigSets)}
		if st, err := os.Stat(folder); err != nil || !st.IsDir() {
			info.FolderMissing = true
		}
		for _, cs := range a.ConfigSets {
			info.EnvVars += len(cs.EnvVars)
			info.EnvVars += len(cs.SharedKeys)
			info.Templates += len(cs.Templates)
			for _, v := range cs.EnvVars {
				if v.Redacted {
					info.EmptyEnvVars++
				}
			}
			for _, t := range cs.Templates {
				if t.Redacted {
					info.EmptyTemplates++
				}
			}
			for _, l := range cs.Commands {
				info.Commands += len(l.Commands)
			}
		}
		result.Apps = append(result.Apps, info)
	}
	if opts.DryRun {
		return result, nil
	}

	ctx := o.Ctx()
	tx, err := o.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer func() { _ = tx.Rollback() }()
	q := o.DB.WithTx(tx)
	ws, err := q.CreateWorkspace(ctx, db.CreateWorkspaceParams{Name: name, Icon: bundle.Workspace.Icon, Color: color})
	if err != nil {
		return result, err
	}
	shared, err := importSharedSets(ctx, q, ws.ID, bundle.Workspace.SharedSets)
	if err != nil {
		return result, err
	}
	for _, p := range planned {
		app, err := q.CreateApp(ctx, db.CreateAppParams{WorkspaceID: ws.ID, Name: p.src.Name, ProjectPath: p.folder})
		if err != nil {
			return result, err
		}
		if p.src.ExcludeFromStartAll {
			if err := q.SetAppExcludeFromStartAll(ctx, db.SetAppExcludeFromStartAllParams{ExcludeFromStartAll: 1, ID: app.ID}); err != nil {
				return result, err
			}
		}
		sets := p.src.ConfigSets
		if len(sets) == 0 {
			sets = []types.BundleConfigSet{{Name: "Default"}}
		}
		active := sets[0].Name
		var activeID int64
		for _, cs := range sets {
			set, err := q.CreateConfigSet(ctx, db.CreateConfigSetParams{AppID: app.ID, Name: cs.Name})
			if err != nil {
				return result, err
			}
			if cs.Name == p.src.ActiveConfigSet || activeID == 0 && cs.Name == active {
				activeID = set.ID
			}
			if err := importConfigSet(ctx, q, set.ID, cs); err != nil {
				return result, err
			}
			if err := linkImportedShared(ctx, q, set.ID, cs, shared); err != nil {
				return result, err
			}
		}
		if _, err := q.SetActiveConfigSet(ctx, db.SetActiveConfigSetParams{ActiveConfigSetID: &activeID, ID: app.ID}); err != nil {
			return result, err
		}
		for i, link := range p.src.WorktreeLinks {
			if err := q.CreateWorktreeLink(ctx, db.CreateWorktreeLinkParams{AppID: app.ID, Path: link, SortOrder: int64(i)}); err != nil {
				return result, err
			}
		}
		result.AppIDs = append(result.AppIDs, app.ID)
	}
	if err := tx.Commit(); err != nil {
		return result, err
	}
	result.WorkspaceID = ws.ID
	o.changed("workspace", ws.ID)
	o.changed("app", result.AppIDs...)
	return result, nil
}

func importConfigSet(ctx context.Context, q *db.Queries, setID int64, cs types.BundleConfigSet) error {
	for _, v := range cs.EnvVars {
		value, private, opts := v.Value, v.Private, v.Options
		if v.Redacted {
			value, private, opts = "", true, nil
		}
		value, optsJSON, selected := db.EnvStore(value, opts, v.Selected)
		if _, err := q.CreateEnvVar(ctx, db.CreateEnvVarParams{ConfigSetID: setID, Key: v.Key, Value: value, Private: db.BoolInt(private), Options: optsJSON, Selected: selected}); err != nil {
			return err
		}
	}
	for _, t := range cs.Templates {
		content, private := t.Content, t.Private
		if t.Redacted {
			content, private = "", true
		}
		if _, err := q.CreateTemplate(ctx, db.CreateTemplateParams{ConfigSetID: setID, FilePath: t.FilePath, Content: content, Private: db.BoolInt(private)}); err != nil {
			return err
		}
	}
	for _, kind := range bundleKinds {
		list, ok := cs.Commands[kind]
		mode := list.Mode
		if !ok || mode == "" {
			mode = db.DefaultMode(kind)
		}
		cfg, err := q.CreateRunConfig(ctx, db.CreateRunConfigParams{ConfigSetID: setID, Kind: kind, Mode: mode})
		if err != nil {
			return err
		}
		for i, c := range list.Commands {
			if _, err := q.CreateRunCommand(ctx, db.CreateRunCommandParams{RunConfigID: cfg.ID, Label: c.Label, Command: c.Command, SortOrder: int64(i)}); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateBundleApp checks and normalizes what import writes into the
// database, since a bundle is untrusted input.
func validateBundleApp(a *types.BundleApp) error {
	seenSet := map[string]bool{}
	for si := range a.ConfigSets {
		cs := &a.ConfigSets[si]
		cs.Name = strings.TrimSpace(cs.Name)
		if cs.Name == "" {
			return fmt.Errorf("a config set has no name")
		}
		if seenSet[cs.Name] {
			return fmt.Errorf("config set %q is listed twice", cs.Name)
		}
		seenSet[cs.Name] = true
		seen := map[string]bool{}
		for i := range cs.EnvVars {
			k := strings.TrimSpace(cs.EnvVars[i].Key)
			if k == "" || seen[k] {
				return fmt.Errorf("config set %q: env var key %q is empty or repeated", cs.Name, k)
			}
			seen[k] = true
			cs.EnvVars[i].Key = k
			opts, err := db.NormalizeEnvOptions(cs.EnvVars[i].Options)
			if err != nil {
				return fmt.Errorf("config set %q: env var %q: %w", cs.Name, k, err)
			}
			cs.EnvVars[i].Options = opts
		}
		seen = map[string]bool{}
		for i := range cs.Templates {
			p := path.Clean(strings.ReplaceAll(strings.TrimSpace(cs.Templates[i].FilePath), "\\", "/"))
			if p == "." || p == ".." || strings.HasPrefix(p, "../") || strings.HasPrefix(p, "/") || filepath.VolumeName(p) != "" {
				return fmt.Errorf("config set %q: template path %q must stay inside the app folder", cs.Name, cs.Templates[i].FilePath)
			}
			if seen[p] {
				return fmt.Errorf("config set %q: template %q is listed twice", cs.Name, p)
			}
			seen[p] = true
			cs.Templates[i].FilePath = p
		}
		for kind, l := range cs.Commands {
			known := false
			for _, k := range bundleKinds {
				known = known || k == kind
			}
			if !known {
				return fmt.Errorf("config set %q: unknown command list %q", cs.Name, kind)
			}
			if l.Mode != "" && l.Mode != "parallel" && l.Mode != "sequential" {
				return fmt.Errorf("config set %q: unknown mode %q", cs.Name, l.Mode)
			}
		}
	}
	links, err := lib.NormalizeWorktreeLinks(a.WorktreeLinks)
	if err != nil {
		return err
	}
	a.WorktreeLinks = links
	return nil
}

// importFolder decides where an imported app lives: an explicit override, an
// absolute path from the bundle, or the bundle's relative path under base.
func importFolder(a types.BundleApp, base, override string) (string, error) {
	if o := strings.TrimSpace(override); o != "" {
		return filepath.Abs(o)
	}
	rel := filepath.FromSlash(strings.TrimSpace(a.Path))
	if rel == "" {
		return "", fmt.Errorf("the bundle has no folder for this app")
	}
	if a.Absolute {
		return filepath.Clean(rel), nil
	}
	if filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" {
		return "", fmt.Errorf("folder %q should be relative", a.Path)
	}
	if base == "" {
		return "", fmt.Errorf("choose a base folder")
	}
	full := filepath.Join(base, rel)
	if r, err := filepath.Rel(base, full); err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("folder %q leaves the base folder", a.Path)
	}
	return filepath.Abs(full)
}

// commonRoot returns the deepest folder holding every path (the parent of the
// one path when there is one), or "" when they share none.
func commonRoot(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	root := filepath.Dir(filepath.Clean(paths[0]))
	for {
		all := true
		for _, p := range paths {
			if _, ok := relativeTo(root, p); !ok {
				all = false
				break
			}
		}
		if all {
			return root
		}
		parent := filepath.Dir(root)
		if parent == root {
			return ""
		}
		root = parent
	}
}

// relativeTo gives p relative to root with forward slashes, when p is root or
// inside it.
func relativeTo(root, p string) (string, bool) {
	if root == "" {
		return "", false
	}
	rel, err := filepath.Rel(root, filepath.Clean(p))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

func slug(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case b.Len() > 0 && !strings.HasSuffix(b.String(), "-"):
			b.WriteByte('-')
		}
	}
	if s := strings.Trim(b.String(), "-"); s != "" {
		return s
	}
	return "app"
}

func uniqueKey(k string, used map[string]bool) string {
	key := k
	for n := 2; used[key]; n++ {
		key = fmt.Sprintf("%s-%d", k, n)
	}
	used[key] = true
	return key
}

// importedShared is a shared set created by an import: its id and variables by key.
type importedShared struct {
	id   int64
	vars map[string]db.WorkspaceEnvVar
}

// validateBundleShared checks the shared sets of a bundle and every link to
// them, since a bundle is untrusted input. A link must name a set of the
// bundle and keys it holds, and a key cannot also be an own variable.
func validateBundleShared(sets []types.BundleSharedSet, apps []types.BundleApp) error {
	byName := map[string]map[string]bool{}
	for i, s := range sets {
		name := strings.TrimSpace(s.Name)
		if name == "" {
			return fmt.Errorf("shared set %d has no name", i+1)
		}
		if _, dup := byName[name]; dup {
			return fmt.Errorf("shared set %q is used twice", name)
		}
		keys := map[string]bool{}
		for _, v := range s.Vars {
			if strings.TrimSpace(v.Key) == "" {
				return fmt.Errorf("shared set %q has a variable without a key", name)
			}
			if keys[v.Key] {
				return fmt.Errorf("shared set %q has %s twice", name, v.Key)
			}
			keys[v.Key] = true
		}
		byName[name] = keys
	}
	for _, a := range apps {
		for _, cs := range a.ConfigSets {
			if cs.SharedSet == "" {
				if len(cs.SharedKeys) > 0 {
					return fmt.Errorf("%s / %s lists shared variables but no shared set", a.Name, cs.Name)
				}
				continue
			}
			keys, ok := byName[cs.SharedSet]
			if !ok {
				return fmt.Errorf("%s / %s links to shared set %q, which the bundle does not have", a.Name, cs.Name, cs.SharedSet)
			}
			own := map[string]bool{}
			for _, v := range cs.EnvVars {
				own[v.Key] = true
			}
			seen := map[string]bool{}
			for _, k := range cs.SharedKeys {
				if !keys[k] {
					return fmt.Errorf("%s / %s takes %s, which shared set %q does not have", a.Name, cs.Name, k, cs.SharedSet)
				}
				if own[k] || seen[k] {
					return fmt.Errorf("%s / %s has %s twice", a.Name, cs.Name, k)
				}
				seen[k] = true
			}
		}
	}
	return nil
}

// importSharedSets creates a bundle's shared sets for a new workspace.
func importSharedSets(ctx context.Context, q *db.Queries, workspaceID int64, sets []types.BundleSharedSet) (map[string]importedShared, error) {
	out := map[string]importedShared{}
	for _, s := range sets {
		name := strings.TrimSpace(s.Name)
		row, err := q.CreateWorkspaceConfigSet(ctx, db.CreateWorkspaceConfigSetParams{WorkspaceID: workspaceID, Name: name})
		if err != nil {
			return nil, err
		}
		is := importedShared{id: row.ID, vars: map[string]db.WorkspaceEnvVar{}}
		for _, v := range s.Vars {
			value, private, opts := v.Value, v.Private, v.Options
			if v.Redacted {
				value, private, opts = "", true, nil
			}
			value, optsJSON, selected := db.EnvStore(value, opts, v.Selected)
			vr, err := q.CreateWorkspaceEnvVar(ctx, db.CreateWorkspaceEnvVarParams{
				WorkspaceConfigSetID: row.ID, Key: strings.TrimSpace(v.Key), Value: value, Private: db.BoolInt(private), Options: optsJSON, Selected: selected,
			})
			if err != nil {
				return nil, err
			}
			is.vars[vr.Key] = vr
		}
		out[name] = is
	}
	return out, nil
}

// linkImportedShared links an imported config set to its shared set and takes
// the variables its bundle lists.
func linkImportedShared(ctx context.Context, q *db.Queries, setID int64, cs types.BundleConfigSet, shared map[string]importedShared) error {
	if cs.SharedSet == "" {
		return nil
	}
	is := shared[strings.TrimSpace(cs.SharedSet)]
	if err := q.SetConfigSetSharedSet(ctx, db.SetConfigSetSharedSetParams{SharedSetID: &is.id, ID: setID}); err != nil {
		return err
	}
	for _, key := range cs.SharedKeys {
		v := is.vars[key]
		vid := v.ID
		if _, err := q.CreateSharedEnvVar(ctx, db.CreateSharedEnvVarParams{
			ConfigSetID: setID, Key: v.Key, Value: v.Value, Private: v.Private, Options: v.Options, Selected: v.Selected, SharedVarID: &vid,
		}); err != nil {
			return err
		}
	}
	return nil
}
