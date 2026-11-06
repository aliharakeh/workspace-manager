package ops

import (
	"context"
	"sort"

	"workspace-manager/daemon/db"
	"workspace-manager/shared/api/types"
)

// mirrorSharedSet makes every config set linked to the shared set carry a row
// for each of the set's variables. It is what keeps a link "to all variables":
// variables added later (by hand or by the set's script) reach the linked sets
// through it. A variable whose key an own variable already uses is skipped,
// unless replace is set, which drops the own variable in favour of the shared
// one. It returns the ids of the apps whose rows changed.
func (o *Ops) mirrorSharedSet(ctx context.Context, q *db.Queries, sharedSetID int64, replace bool) ([]int64, error) {
	sharedVars, err := q.ListWorkspaceEnvVars(ctx, sharedSetID)
	if err != nil {
		return nil, err
	}
	if len(sharedVars) == 0 {
		return nil, nil
	}
	sort.Slice(sharedVars, func(i, j int) bool { return sharedVars[i].Key < sharedVars[j].Key })
	setIDs, err := q.ListConfigSetIDsLinkedToSharedSet(ctx, &sharedSetID)
	if err != nil {
		return nil, err
	}
	var apps []int64
	for _, setID := range setIDs {
		changed, err := o.mirrorInto(ctx, q, setID, sharedVars, replace)
		if err != nil {
			return nil, err
		}
		if changed {
			set, err := q.GetConfigSet(ctx, setID)
			if err != nil {
				return nil, err
			}
			apps = append(apps, set.AppID)
		}
	}
	return apps, nil
}

// mirrorInto adds the missing shared rows to one config set.
func (o *Ops) mirrorInto(ctx context.Context, q *db.Queries, setID int64, sharedVars []db.WorkspaceEnvVar, replace bool) (bool, error) {
	current, err := q.ListEnvVarsByConfigSet(ctx, setID)
	if err != nil {
		return false, err
	}
	linked := map[int64]bool{}
	own := map[string]db.EnvVar{}
	for _, r := range current {
		if r.SharedVarID != nil {
			linked[*r.SharedVarID] = true
		} else {
			own[r.Key] = r
		}
	}
	changed := false
	for _, v := range sharedVars {
		if linked[v.ID] {
			continue
		}
		if cur, clash := own[v.Key]; clash {
			if !replace {
				continue
			}
			if _, err := q.DeleteEnvVar(ctx, cur.ID); err != nil {
				return false, err
			}
		}
		vid := v.ID
		if _, err := q.CreateSharedEnvVar(ctx, db.CreateSharedEnvVarParams{
			ConfigSetID: setID, Key: v.Key, Value: v.Value, Private: v.Private,
			Options: v.Options, Selected: v.Selected, SharedVarID: &vid,
		}); err != nil {
			return false, err
		}
		changed = true
	}
	return changed, nil
}

// SharedLinksRefresh brings every linked config set up to date with its shared
// set: variables that appeared in the shared set since the link (for example
// from its script) are added. It never runs a script. Variables the config set
// already defines itself are left alone.
func (o *Ops) SharedLinksRefresh() (types.Ok, error) {
	ctx := o.Ctx()
	ids, err := o.DB.ListLinkedSharedSetIDs(ctx)
	if err != nil {
		return types.Ok{}, err
	}
	var apps []int64
	for _, id := range ids {
		if id == nil {
			continue
		}
		changed, err := o.mirrorSharedSet(ctx, o.DB.Queries, *id, false)
		if err != nil {
			return types.Ok{}, err
		}
		apps = append(apps, changed...)
	}
	if len(apps) > 0 {
		o.changed("envVar", apps...)
	}
	return types.Ok{Ok: true}, nil
}
