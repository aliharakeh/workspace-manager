package ops

import (
	"sync"

	"workspace-manager/shared/api/types"
)

// RunnerStartWorkspace starts the dev run of every app of a workspace. It only
// fans out to the per-app sessions (RunnerRun): an app that already has a
// session going is skipped, one in excludeAppIDs or marked "exclude from Start
// all" is reported as excluded, and a failing app is reported without stopping
// the others. Sequential starts the apps one after another in the workspace's
// order; otherwise they start concurrently.
func (o *Ops) RunnerStartWorkspace(workspaceID int64, excludeAppIDs []int64, sequential bool) (types.WorkspaceRunResult, error) {
	apps, err := o.workspaceApps(workspaceID)
	if err != nil {
		return types.WorkspaceRunResult{}, err
	}
	skip := idSet(excludeAppIDs)
	results := make([]types.WorkspaceRunAppResult, len(apps))
	var wg sync.WaitGroup
	for i, app := range apps {
		results[i] = types.WorkspaceRunAppResult{AppID: app.ID, Name: app.Name}
		switch {
		case skip[app.ID]:
			results[i].Status, results[i].Message = types.WorkspaceRunExcluded, "Excluded for this start"
			continue
		case app.ExcludeFromStartAll:
			results[i].Status, results[i].Message = types.WorkspaceRunExcluded, "Excluded from Start all"
			continue
		case o.Runner.GetStatus(app.ID).Running:
			results[i].Status, results[i].Message = types.WorkspaceRunSkipped, "Already running"
			continue
		}
		start := func(i int, id int64) {
			if _, err := o.RunnerRun(id); err != nil {
				results[i].Status, results[i].Message = types.WorkspaceRunError, err.Error()
				return
			}
			results[i].Status = types.WorkspaceRunStarted
		}
		if sequential {
			start(i, app.ID)
			continue
		}
		wg.Add(1)
		go func(i int, id int64) {
			defer wg.Done()
			start(i, id)
		}(i, app.ID)
	}
	wg.Wait()
	return types.WorkspaceRunResult{WorkspaceID: workspaceID, Results: results}, nil
}

// RunnerStopWorkspace stops every app of a workspace that has a session going
// (a dev, prod, build or setup run alike). Apps without one are skipped; apps in
// excludeAppIDs are left as they are. The persisted "exclude from Start all"
// flag does not apply: Stop all stops whatever is running.
func (o *Ops) RunnerStopWorkspace(workspaceID int64, excludeAppIDs []int64) (types.WorkspaceRunResult, error) {
	apps, err := o.workspaceApps(workspaceID)
	if err != nil {
		return types.WorkspaceRunResult{}, err
	}
	skip := idSet(excludeAppIDs)
	results := make([]types.WorkspaceRunAppResult, len(apps))
	for i, app := range apps {
		results[i] = types.WorkspaceRunAppResult{AppID: app.ID, Name: app.Name}
		switch {
		case skip[app.ID]:
			results[i].Status, results[i].Message = types.WorkspaceRunExcluded, "Excluded for this stop"
		case !o.Runner.GetStatus(app.ID).Running:
			results[i].Status, results[i].Message = types.WorkspaceRunSkipped, "Not running"
		default:
			if _, err := o.RunnerStop(app.ID); err != nil {
				results[i].Status, results[i].Message = types.WorkspaceRunError, err.Error()
			} else {
				results[i].Status = types.WorkspaceRunStopped
			}
		}
	}
	return types.WorkspaceRunResult{WorkspaceID: workspaceID, Results: results}, nil
}

func (o *Ops) workspaceApps(workspaceID int64) ([]types.App, error) {
	if _, err := o.DB.GetWorkspaceT(o.Ctx(), workspaceID); err != nil {
		return nil, err
	}
	return o.DB.ListAppsByWorkspaceT(o.Ctx(), workspaceID)
}

func idSet(ids []int64) map[int64]bool {
	m := make(map[int64]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}
