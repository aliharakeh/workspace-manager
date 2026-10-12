package ops

import (
	"fmt"
	"strings"

	"workspace-manager/api/types"
)

// checkConfigKind rejects anything but the three kinds of command list.
func checkConfigKind(kind string) error {
	switch kind {
	case types.KindRun, types.KindBuild, types.KindSetup:
		return nil
	}
	return fmt.Errorf("kind must be %s, %s or %s", types.KindRun, types.KindBuild, types.KindSetup)
}

// RunConfigGet returns the active config set's command list of a kind: "run"
// (starts the app), "build" (builds it) or "setup" (prepares it).
func (o *Ops) RunConfigGet(appID int64, kind string) (types.RunConfig, error) {
	if err := checkConfigKind(kind); err != nil {
		return types.RunConfig{}, err
	}
	if _, err := o.DB.GetAppT(o.Ctx(), appID); err != nil {
		return types.RunConfig{}, err
	}
	set, err := o.DB.ResolveActive(o.Ctx(), appID)
	if err != nil {
		return types.RunConfig{}, err
	}
	return o.DB.GetOrCreateRunConfig(o.Ctx(), set.ID, kind)
}

func (o *Ops) RunConfigSave(appID int64, kind string, body types.RunConfigSaveInput) (types.RunConfig, error) {
	if err := checkConfigKind(kind); err != nil {
		return types.RunConfig{}, err
	}
	if _, err := o.DB.GetAppT(o.Ctx(), appID); err != nil {
		return types.RunConfig{}, err
	}
	if body.Mode != nil && *body.Mode != "sequential" && *body.Mode != "parallel" {
		return types.RunConfig{}, fmt.Errorf("mode must be sequential or parallel")
	}
	var cmds []types.RunCommandInput
	if body.Commands != nil {
		cmds = make([]types.RunCommandInput, 0, len(body.Commands))
		for _, c := range body.Commands {
			if strings.TrimSpace(c.Command) == "" {
				return types.RunConfig{}, fmt.Errorf("Each command must have a non-empty command string")
			}
			cmds = append(cmds, types.RunCommandInput{Label: c.Label, Command: strings.TrimSpace(c.Command)})
		}
	}
	set, err := o.DB.ResolveActive(o.Ctx(), appID)
	if err != nil {
		return types.RunConfig{}, err
	}
	cfg, err := o.DB.UpsertRunConfig(o.Ctx(), set.ID, kind, body.Mode, cmds)
	if err != nil {
		return types.RunConfig{}, err
	}
	o.changed("runConfig", appID)
	return cfg, nil
}

func (o *Ops) RunnerStatus(appID int64) types.StatusEvent {
	return o.Runner.GetStatus(appID)
}

func (o *Ops) RunnerWorkspaceStatus(workspaceID int64) ([]types.StatusEvent, error) {
	apps, err := o.DB.ListAppsByWorkspaceT(o.Ctx(), workspaceID)
	if err != nil {
		return nil, err
	}
	out := make([]types.StatusEvent, 0, len(apps))
	for _, app := range apps {
		out = append(out, o.Runner.GetStatus(app.ID))
	}
	return out, nil
}

// RunnerOutput returns the recent terminal output of one command of the app's
// current run, so a terminal opened late can catch up before live events.
func (o *Ops) RunnerOutput(appID, commandID int64) types.RunnerOutput {
	return o.Runner.GetOutput(appID, commandID)
}

// RunnerResize sets the terminal size of a running command.
func (o *Ops) RunnerResize(appID, commandID int64, cols, rows int) types.Ok {
	o.Runner.Resize(appID, commandID, cols, rows)
	return types.Ok{Ok: true}
}

func (o *Ops) RunnerRun(appID int64) (types.StatusEvent, error) {
	if _, err := o.DB.GetAppT(o.Ctx(), appID); err != nil {
		return types.StatusEvent{}, err
	}
	return o.Runner.Start(o.Ctx(), appID, types.KindRun)
}

// RunnerBuild runs the active config set's build commands. It shares the app's
// session with RunnerRun: it fails while the app is running or building, and
// RunnerStop stops it.
func (o *Ops) RunnerBuild(appID int64) (types.StatusEvent, error) {
	if _, err := o.DB.GetAppT(o.Ctx(), appID); err != nil {
		return types.StatusEvent{}, err
	}
	return o.Runner.Start(o.Ctx(), appID, types.KindBuild)
}

// RunnerSetup runs the active config set's setup commands. Like RunnerBuild it
// shares the app's session: it fails while the app is running, building or
// being set up, and RunnerStop stops it.
func (o *Ops) RunnerSetup(appID int64) (types.StatusEvent, error) {
	if _, err := o.DB.GetAppT(o.Ctx(), appID); err != nil {
		return types.StatusEvent{}, err
	}
	return o.Runner.Start(o.Ctx(), appID, types.KindSetup)
}

func (o *Ops) RunnerStop(appID int64) (types.StatusEvent, error) {
	return o.Runner.Stop(o.Ctx(), appID)
}

func (o *Ops) RunnerReload(appID int64) (types.StatusEvent, error) {
	if _, err := o.DB.GetAppT(o.Ctx(), appID); err != nil {
		return types.StatusEvent{}, err
	}
	return o.Runner.Reload(o.Ctx(), appID)
}
