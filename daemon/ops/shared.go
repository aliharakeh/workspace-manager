package ops

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"workspace-manager/daemon/db"
	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"
)

// Shared config: a workspace owns config sets of env vars. An app config set
// links to one and takes chosen variables as env_vars rows that carry
// shared_var_id. Those rows are live: every write to a workspace variable
// rewrites them in the same transaction, so nothing reading env_vars needs to
// know about sharing. They are read-only in the app.

func sharedSetFrom(row db.WorkspaceConfigSet) types.SharedSet {
	return types.SharedSet{
		ID: row.ID, WorkspaceID: row.WorkspaceID, Name: row.Name,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

// sharedVarAsEnv views a workspace variable as an EnvVar so the option and
// value rules of app variables apply to it unchanged.
func sharedVarAsEnv(row db.WorkspaceEnvVar) types.EnvVar {
	return db.EnvVarFrom(db.EnvVar{
		ID: row.ID, ConfigSetID: row.WorkspaceConfigSetID, Key: row.Key, Value: row.Value,
		Private: row.Private, Options: row.Options, Selected: row.Selected,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	})
}

func sharedVarFrom(row db.WorkspaceEnvVar, usedBy []types.SharedUsage) types.SharedEnvVar {
	e := sharedVarAsEnv(row)
	if usedBy == nil {
		usedBy = []types.SharedUsage{}
	}
	return types.SharedEnvVar{
		ID: row.ID, SetID: row.WorkspaceConfigSetID, Key: row.Key, Value: row.Value,
		Private: e.Private, Options: e.Options, Selected: e.Selected, UsedBy: usedBy,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

// envFields are the stored columns of a variable after its rules are applied.
type envFields struct {
	key, value, options, selected string
	private                       bool
}

// mergeEnvUpdate applies an update to a variable the way every variable is
// edited: a value edit on a choice lands on the selected option, new options
// replace the list, and the selected label must be one of them.
func mergeEnvUpdate(cur types.EnvVar, body types.EnvVarUpdateInput) (envFields, error) {
	key, value, private := cur.Key, cur.Value, cur.Private
	opts, selected := cur.Options, cur.Selected
	if body.Key != nil {
		key = strings.TrimSpace(*body.Key)
		if key == "" {
			return envFields{}, fmt.Errorf("key cannot be empty")
		}
	}
	if body.Value != nil {
		value = *body.Value
	}
	if body.Private != nil {
		private = *body.Private
	}
	if body.Value != nil && body.Options == nil && len(opts) > 0 {
		// A value edit on a choice lands on the selected option.
		_, optsJSON, _ := db.EnvStoreValue(cur, value)
		opts = db.EnvVarFrom(db.EnvVar{Options: optsJSON}).Options
	}
	if body.Options != nil {
		var err error
		if opts, err = db.NormalizeEnvOptions(*body.Options); err != nil {
			return envFields{}, err
		}
		if body.Selected == nil && !hasOptionLabel(opts, selected) {
			selected = "" // the old choice is gone: fall back to the first option
		}
	}
	if body.Selected != nil {
		selected = *body.Selected
		if selected != "" && !hasOptionLabel(opts, selected) {
			return envFields{}, fmt.Errorf("selected %q is not one of the options", selected)
		}
	}
	value, optsJSON, selected := db.EnvStore(value, opts, selected)
	return envFields{key: key, value: value, options: optsJSON, selected: selected, private: private}, nil
}

// newEnvFields validates a new variable the way EnvVarsCreate does.
func newEnvFields(body types.EnvVarCreateInput) (envFields, error) {
	key := strings.TrimSpace(body.Key)
	if key == "" {
		return envFields{}, fmt.Errorf("key is required")
	}
	value := ""
	if body.Value != nil {
		value = *body.Value
	}
	private := false
	if body.Private != nil {
		private = *body.Private
	}
	var opts []types.EnvVarOption
	if body.Options != nil {
		var err error
		if opts, err = db.NormalizeEnvOptions(*body.Options); err != nil {
			return envFields{}, err
		}
	}
	selected := ""
	if body.Selected != nil {
		selected = *body.Selected
	}
	if selected != "" && !hasOptionLabel(opts, selected) {
		return envFields{}, fmt.Errorf("selected %q is not one of the options", selected)
	}
	value, optsJSON, selected := db.EnvStore(value, opts, selected)
	return envFields{key: key, value: value, options: optsJSON, selected: selected, private: private}, nil
}

func (o *Ops) getSharedSet(id int64) (db.WorkspaceConfigSet, error) {
	row, err := o.DB.GetWorkspaceConfigSet(o.Ctx(), id)
	if err == sql.ErrNoRows {
		return row, api.Errf(api.KindNotFound, "Shared config set not found")
	}
	return row, err
}

func (o *Ops) getSharedVar(id int64) (db.WorkspaceEnvVar, error) {
	row, err := o.DB.GetWorkspaceEnvVar(o.Ctx(), id)
	if err == sql.ErrNoRows {
		return row, api.Errf(api.KindNotFound, "Shared variable not found")
	}
	return row, err
}

// appsOf returns the distinct app ids of usage rows, in order.
func appsOf(usage []types.SharedUsage) []int64 {
	seen := map[int64]bool{}
	var ids []int64
	for _, u := range usage {
		if !seen[u.AppID] {
			seen[u.AppID] = true
			ids = append(ids, u.AppID)
		}
	}
	return ids
}

// usageBySet reads who uses each variable of a shared set, keyed by variable id.
func (o *Ops) usageBySet(ctx context.Context, setID int64) (map[int64][]types.SharedUsage, error) {
	rows, err := o.DB.ListSharedUsageBySet(ctx, setID)
	if err != nil {
		return nil, err
	}
	out := map[int64][]types.SharedUsage{}
	for _, r := range rows {
		if r.SharedVarID == nil {
			continue
		}
		out[*r.SharedVarID] = append(out[*r.SharedVarID], types.SharedUsage{
			AppID: r.AppID, AppName: r.AppName, ConfigSetID: r.ConfigSetID, ConfigSetName: r.ConfigSetName,
		})
	}
	return out, nil
}

func (o *Ops) SharedSetsList(workspaceID int64) ([]types.SharedSet, error) {
	if _, err := o.DB.GetWorkspaceT(o.Ctx(), workspaceID); err != nil {
		return nil, err
	}
	rows, err := o.DB.ListWorkspaceConfigSets(o.Ctx(), workspaceID)
	if err != nil {
		return nil, err
	}
	out := make([]types.SharedSet, 0, len(rows))
	for _, r := range rows {
		out = append(out, sharedSetFrom(r))
	}
	return out, nil
}

// SharedSetsGet returns a shared set with its variables and where each is used.
func (o *Ops) SharedSetsGet(id int64) (types.SharedSetDetail, error) {
	set, err := o.getSharedSet(id)
	if err != nil {
		return types.SharedSetDetail{}, err
	}
	rows, err := o.DB.ListWorkspaceEnvVars(o.Ctx(), id)
	if err != nil {
		return types.SharedSetDetail{}, err
	}
	usage, err := o.usageBySet(o.Ctx(), id)
	if err != nil {
		return types.SharedSetDetail{}, err
	}
	vars := make([]types.SharedEnvVar, 0, len(rows))
	for _, r := range rows {
		vars = append(vars, sharedVarFrom(r, usage[r.ID]))
	}
	return types.SharedSetDetail{SharedSet: sharedSetFrom(set), Vars: vars}, nil
}

// SharedVarsGet returns one workspace variable.
func (o *Ops) SharedVarsGet(id int64) (types.SharedEnvVar, error) {
	row, err := o.getSharedVar(id)
	if err != nil {
		return types.SharedEnvVar{}, err
	}
	usage, err := o.usageBySet(o.Ctx(), row.WorkspaceConfigSetID)
	if err != nil {
		return types.SharedEnvVar{}, err
	}
	return sharedVarFrom(row, usage[id]), nil
}

func (o *Ops) SharedSetsCreate(workspaceID int64, name string) (types.SharedSet, error) {
	if _, err := o.DB.GetWorkspaceT(o.Ctx(), workspaceID); err != nil {
		return types.SharedSet{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return types.SharedSet{}, fmt.Errorf("name is required")
	}
	row, err := o.DB.CreateWorkspaceConfigSet(o.Ctx(), db.CreateWorkspaceConfigSetParams{WorkspaceID: workspaceID, Name: name})
	if err != nil {
		return types.SharedSet{}, db.UniqueErr(err, "A shared config set with this name already exists")
	}
	o.changed("sharedSet", workspaceID)
	return sharedSetFrom(row), nil
}

func (o *Ops) SharedSetsRename(id int64, name string) (types.SharedSet, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return types.SharedSet{}, fmt.Errorf("name is required")
	}
	if _, err := o.getSharedSet(id); err != nil {
		return types.SharedSet{}, err
	}
	row, err := o.DB.UpdateWorkspaceConfigSet(o.Ctx(), db.UpdateWorkspaceConfigSetParams{Name: name, ID: id})
	if err != nil {
		return types.SharedSet{}, db.UniqueErr(err, "A shared config set with this name already exists")
	}
	o.changed("sharedSet", row.WorkspaceID)
	// Linked config sets show the set's name.
	if apps, err := o.DB.ListAppsLinkedToSharedSet(o.Ctx(), &id); err == nil {
		o.changed("configSet", apps...)
	}
	return sharedSetFrom(row), nil
}

// SharedSetsDelete removes a shared set. Its variables go with it, and so do
// the rows mirroring them in app config sets, which become unlinked.
func (o *Ops) SharedSetsDelete(id int64) (types.Ok, error) {
	set, err := o.getSharedSet(id)
	if err != nil {
		return types.Ok{}, err
	}
	apps, _ := o.DB.ListAppsLinkedToSharedSet(o.Ctx(), &id)
	if _, err := o.DB.DeleteWorkspaceConfigSet(o.Ctx(), id); err != nil {
		return types.Ok{}, err
	}
	o.changed("sharedSet", set.WorkspaceID)
	o.changed("configSet", apps...)
	o.changed("envVar", apps...)
	return types.Ok{Ok: true}, nil
}

func (o *Ops) SharedVarsCreate(setID int64, body types.EnvVarCreateInput) (types.SharedEnvVar, error) {
	set, err := o.getSharedSet(setID)
	if err != nil {
		return types.SharedEnvVar{}, err
	}
	f, err := newEnvFields(body)
	if err != nil {
		return types.SharedEnvVar{}, err
	}
	row, err := o.DB.CreateWorkspaceEnvVar(o.Ctx(), db.CreateWorkspaceEnvVarParams{
		WorkspaceConfigSetID: setID, Key: f.key, Value: f.value, Private: db.BoolInt(f.private), Options: f.options, Selected: f.selected,
	})
	if err != nil {
		return types.SharedEnvVar{}, db.UniqueErr(err, "Variable key already exists in this shared config set")
	}
	// Every config set linked to this one takes the new variable.
	apps, err := o.mirrorSharedSet(o.Ctx(), o.DB.Queries, setID, false)
	if err != nil {
		return types.SharedEnvVar{}, err
	}
	o.changed("sharedSet", set.WorkspaceID)
	o.changed("envVar", apps...)
	return sharedVarFrom(row, nil), nil
}

// SharedVarsUpdate changes a workspace variable and every app row mirroring it,
// in one transaction. A rename that would collide with a variable already in a
// linked app config set is refused, naming where.
func (o *Ops) SharedVarsUpdate(id int64, body types.EnvVarUpdateInput) (types.SharedEnvVar, error) {
	ctx := o.Ctx()
	existing, err := o.getSharedVar(id)
	if err != nil {
		return types.SharedEnvVar{}, err
	}
	f, err := mergeEnvUpdate(sharedVarAsEnv(existing), body)
	if err != nil {
		return types.SharedEnvVar{}, err
	}
	tx, err := o.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		return types.SharedEnvVar{}, err
	}
	defer func() { _ = tx.Rollback() }()
	q := o.DB.WithTx(tx)
	if f.key != existing.Key {
		clashes, err := q.ListSharedRenameConflicts(ctx, db.ListSharedRenameConflictsParams{SharedVarID: &id, Key: f.key})
		if err != nil {
			return types.SharedEnvVar{}, err
		}
		if len(clashes) > 0 {
			where := make([]string, len(clashes))
			for i, c := range clashes {
				where[i] = fmt.Sprintf("%s / %s", c.AppName, c.ConfigSetName)
			}
			return types.SharedEnvVar{}, api.Errf(api.KindConflict, "Cannot rename to %s: it already exists in %s", f.key, strings.Join(where, ", "))
		}
	}
	row, err := q.UpdateWorkspaceEnvVar(ctx, db.UpdateWorkspaceEnvVarParams{
		Key: f.key, Value: f.value, Private: db.BoolInt(f.private), Options: f.options, Selected: f.selected, ID: id,
	})
	if err != nil {
		return types.SharedEnvVar{}, db.UniqueErr(err, "Variable key already exists in this shared config set")
	}
	if err := q.PropagateSharedEnvVar(ctx, db.PropagateSharedEnvVarParams{
		Key: f.key, Value: f.value, Private: db.BoolInt(f.private), Options: f.options, Selected: f.selected, SharedVarID: &id,
	}); err != nil {
		return types.SharedEnvVar{}, err
	}
	if err := tx.Commit(); err != nil {
		return types.SharedEnvVar{}, err
	}
	usage, err := o.usageBySet(ctx, row.WorkspaceConfigSetID)
	if err != nil {
		return types.SharedEnvVar{}, err
	}
	if set, err := o.getSharedSet(row.WorkspaceConfigSetID); err == nil {
		o.changed("sharedSet", set.WorkspaceID)
	}
	o.changed("envVar", appsOf(usage[id])...)
	return sharedVarFrom(row, usage[id]), nil
}

// SharedVarsDelete removes a workspace variable; the rows mirroring it in app
// config sets are removed with it.
func (o *Ops) SharedVarsDelete(id int64) (types.Ok, error) {
	existing, err := o.getSharedVar(id)
	if err != nil {
		return types.Ok{}, err
	}
	usage, err := o.usageBySet(o.Ctx(), existing.WorkspaceConfigSetID)
	if err != nil {
		return types.Ok{}, err
	}
	if _, err := o.DB.DeleteWorkspaceEnvVar(o.Ctx(), id); err != nil {
		return types.Ok{}, err
	}
	if set, err := o.getSharedSet(existing.WorkspaceConfigSetID); err == nil {
		o.changed("sharedSet", set.WorkspaceID)
	}
	o.changed("envVar", appsOf(usage[id])...)
	return types.Ok{Ok: true}, nil
}

// ConfigSetsLinkShared links an app config set to a shared set of its own
// workspace. The config set takes every variable of the shared set, now and
// later (see mirrorSharedSet). Calling it with another shared set replaces the
// old shared rows. A shared key that an own variable already uses is a conflict
// unless Replace is set.
func (o *Ops) ConfigSetsLinkShared(configSetID int64, in types.SharedLinkInput) (types.ConfigSet, error) {
	ctx := o.Ctx()
	set, err := o.DB.GetConfigSetT(ctx, configSetID)
	if err != nil {
		return types.ConfigSet{}, err
	}
	app, err := o.DB.GetAppT(ctx, set.AppID)
	if err != nil {
		return types.ConfigSet{}, err
	}
	shared, err := o.getSharedSet(in.SharedSetID)
	if err != nil {
		return types.ConfigSet{}, err
	}
	if shared.WorkspaceID != app.WorkspaceID {
		return types.ConfigSet{}, api.Errf(api.KindInvalid, "The shared config set belongs to another workspace")
	}
	sharedVars, err := o.DB.ListWorkspaceEnvVars(ctx, shared.ID)
	if err != nil {
		return types.ConfigSet{}, err
	}
	sort.Slice(sharedVars, func(i, j int) bool { return sharedVars[i].Key < sharedVars[j].Key })

	tx, err := o.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		return types.ConfigSet{}, err
	}
	defer func() { _ = tx.Rollback() }()
	q := o.DB.WithTx(tx)

	if set.SharedSetID == nil || *set.SharedSetID != shared.ID {
		// A different source: its rows are replaced, not merged.
		if err := q.DeleteSharedEnvVarsByConfigSet(ctx, configSetID); err != nil {
			return types.ConfigSet{}, err
		}
		if err := q.SetConfigSetSharedSet(ctx, db.SetConfigSetSharedSetParams{SharedSetID: &shared.ID, ID: configSetID}); err != nil {
			return types.ConfigSet{}, err
		}
	}
	current, err := q.ListEnvVarsByConfigSet(ctx, configSetID)
	if err != nil {
		return types.ConfigSet{}, err
	}
	linked := map[int64]bool{}
	own := map[string]bool{}
	for _, r := range current {
		if r.SharedVarID != nil {
			linked[*r.SharedVarID] = true
		} else {
			own[r.Key] = true
		}
	}
	var clashes []string
	for _, v := range sharedVars {
		if !linked[v.ID] && own[v.Key] {
			clashes = append(clashes, v.Key)
		}
	}
	if len(clashes) > 0 && !in.Replace {
		return types.ConfigSet{}, api.Errf(api.KindConflict, "This config set already has its own %s", strings.Join(clashes, ", "))
	}
	if _, err := o.mirrorInto(ctx, q, configSetID, sharedVars, true); err != nil {
		return types.ConfigSet{}, err
	}
	if err := tx.Commit(); err != nil {
		return types.ConfigSet{}, err
	}
	o.changed("configSet", set.AppID)
	o.changed("envVar", set.AppID)
	o.changed("sharedSet", shared.WorkspaceID)
	return o.DB.GetConfigSetT(ctx, configSetID)
}

// ConfigSetsUnlinkShared stops a config set sharing. With keep, the shared
// variables stay as the app's own; otherwise they are deleted.
func (o *Ops) ConfigSetsUnlinkShared(configSetID int64, keep bool) (types.ConfigSet, error) {
	ctx := o.Ctx()
	set, err := o.DB.GetConfigSetT(ctx, configSetID)
	if err != nil {
		return types.ConfigSet{}, err
	}
	tx, err := o.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		return types.ConfigSet{}, err
	}
	defer func() { _ = tx.Rollback() }()
	q := o.DB.WithTx(tx)
	if keep {
		err = q.DetachSharedEnvVarsByConfigSet(ctx, configSetID)
	} else {
		err = q.DeleteSharedEnvVarsByConfigSet(ctx, configSetID)
	}
	if err != nil {
		return types.ConfigSet{}, err
	}
	if err := q.SetConfigSetSharedSet(ctx, db.SetConfigSetSharedSetParams{SharedSetID: nil, ID: configSetID}); err != nil {
		return types.ConfigSet{}, err
	}
	if err := tx.Commit(); err != nil {
		return types.ConfigSet{}, err
	}
	o.changed("configSet", set.AppID)
	o.changed("envVar", set.AppID)
	if set.SharedSetID != nil {
		if s, err := o.DB.GetWorkspaceConfigSet(ctx, *set.SharedSetID); err == nil {
			o.changed("sharedSet", s.WorkspaceID)
		}
	}
	return o.DB.GetConfigSetT(ctx, configSetID)
}
