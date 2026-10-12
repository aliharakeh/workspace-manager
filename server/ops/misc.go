package ops

import (
	"fmt"
	"strings"

	"workspace-manager/api/lib"
	"workspace-manager/api/types"
	"workspace-manager/server/db"
	"workspace-manager/server/native"
	"workspace-manager/server/services"
)

func (o *Ops) PortsList() (types.PortsListResult, error) {
	procs, err := native.ListListeningProcesses()
	if err != nil {
		return types.PortsListResult{}, err
	}
	out := make([]types.ListeningProcess, 0, len(procs))
	for _, p := range procs {
		out = append(out, types.ListeningProcess{Port: p.Port, PID: p.PID, Name: p.Name})
	}
	return types.PortsListResult{Min: native.UserPortMin, Max: native.UserPortMax, Processes: out}, nil
}

func (o *Ops) PortsKill(pid int64) (types.PortsKillResult, error) {
	if err := native.KillPid(int(pid)); err != nil {
		return types.PortsKillResult{}, err
	}
	return types.PortsKillResult{Ok: true, PID: pid}, nil
}

func (o *Ops) ReadyUrlPatternsList() ([]types.ReadyUrlPattern, error) {
	return o.DB.ListReadyURLPatternsT(o.Ctx())
}

func (o *Ops) ReadyUrlPatternsCreate(body types.ReadyUrlPatternCreateInput) (types.ReadyUrlPattern, error) {
	label := strings.TrimSpace(body.Label)
	pattern := strings.TrimSpace(body.Pattern)
	if label == "" {
		return types.ReadyUrlPattern{}, fmt.Errorf("label is required")
	}
	if pattern == "" {
		return types.ReadyUrlPattern{}, fmt.Errorf("pattern is required")
	}
	flags := "i"
	if body.Flags != nil {
		flags = *body.Flags
	}
	normalized, err := db.NormalizeFlags(flags)
	if err != nil {
		return types.ReadyUrlPattern{}, err
	}
	if err := services.ValidateReadyURLPattern(pattern, normalized); err != nil {
		return types.ReadyUrlPattern{}, err
	}
	row, err := o.DB.CreateReadyURLPatternUser(o.Ctx(), label, pattern, normalized)
	if err != nil {
		return types.ReadyUrlPattern{}, err
	}
	services.InvalidateReadyURLPatternsCache()
	o.changed("readyUrlPattern", row.ID)
	return types.ReadyUrlPattern{ID: row.ID, Key: row.Key, Label: row.Label, Pattern: row.Pattern, Flags: row.Flags, SortOrder: row.SortOrder, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (o *Ops) ReadyUrlPatternsUpdate(id int64, body types.ReadyUrlPatternUpdateInput) (types.ReadyUrlPattern, error) {
	existing, err := o.DB.GetReadyUrlPattern(o.Ctx(), id)
	if err != nil {
		return types.ReadyUrlPattern{}, fmt.Errorf("Pattern not found")
	}
	label, pattern, flags := existing.Label, existing.Pattern, existing.Flags
	if body.Label != nil {
		label = strings.TrimSpace(*body.Label)
	}
	if body.Pattern != nil {
		pattern = strings.TrimSpace(*body.Pattern)
	}
	if body.Flags != nil {
		flags, err = db.NormalizeFlags(*body.Flags)
		if err != nil {
			return types.ReadyUrlPattern{}, err
		}
	}
	if label == "" {
		return types.ReadyUrlPattern{}, fmt.Errorf("label cannot be empty")
	}
	if pattern == "" {
		return types.ReadyUrlPattern{}, fmt.Errorf("pattern cannot be empty")
	}
	if err := services.ValidateReadyURLPattern(pattern, flags); err != nil {
		return types.ReadyUrlPattern{}, err
	}
	row, err := o.DB.UpdateReadyUrlPattern(o.Ctx(), db.UpdateReadyUrlPatternParams{Label: label, Pattern: pattern, Flags: flags, ID: id})
	if err != nil {
		return types.ReadyUrlPattern{}, err
	}
	services.InvalidateReadyURLPatternsCache()
	o.changed("readyUrlPattern", row.ID)
	return types.ReadyUrlPattern{ID: row.ID, Key: row.Key, Label: row.Label, Pattern: row.Pattern, Flags: row.Flags, SortOrder: row.SortOrder, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (o *Ops) ReadyUrlPatternsDelete(id int64) (types.Ok, error) {
	n, err := o.DB.DeleteReadyUrlPattern(o.Ctx(), id)
	if err != nil {
		return types.Ok{}, err
	}
	if n == 0 {
		return types.Ok{}, fmt.Errorf("Pattern not found")
	}
	services.InvalidateReadyURLPatternsCache()
	o.changed("readyUrlPattern", id)
	return types.Ok{Ok: true}, nil
}

func (o *Ops) SettingsGet() (map[string]string, error) {
	return o.DB.SettingsMap(o.Ctx())
}

func (o *Ops) SettingsSet(key string, value string) (map[string]string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}
	if err := o.DB.UpsertSetting(o.Ctx(), db.UpsertSettingParams{Key: key, Value: value}); err != nil {
		return nil, err
	}
	o.changed("settings")
	return o.DB.SettingsMap(o.Ctx())
}

func (o *Ops) FsValidatePath(path string) (types.ValidatePathResult, error) {
	if path == "" {
		return types.ValidatePathResult{}, fmt.Errorf("path is required")
	}
	ok, resolved, errMsg := lib.ValidateProjectPath(path)
	if !ok {
		return types.ValidatePathResult{Ok: false, Error: errMsg}, nil
	}
	return types.ValidatePathResult{Ok: true, Path: resolved}, nil
}

func (o *Ops) FsReadAppFile(appID int64, path string) (types.ReadAppFileResult, error) {
	app, err := o.DB.GetAppT(o.Ctx(), appID)
	if err != nil {
		return types.ReadAppFileResult{}, err
	}
	if strings.TrimSpace(path) == "" {
		return types.ReadAppFileResult{}, fmt.Errorf("path is required")
	}
	content, rel, errMsg := lib.ReadProjectFile(app.ProjectPath, path)
	if errMsg != "" {
		return types.ReadAppFileResult{}, fmt.Errorf("%s", errMsg)
	}
	return types.ReadAppFileResult{Ok: true, Content: content, RelativePath: rel}, nil
}

// BlueprintsList returns every blueprint.
func (o *Ops) BlueprintsList() ([]types.Blueprint, error) {
	return o.DB.ListBlueprintsT(o.Ctx())
}

func (o *Ops) BlueprintsCreate(body types.BlueprintInput) (types.Blueprint, error) {
	bp, err := o.DB.CreateBlueprintT(o.Ctx(), body)
	if err == nil {
		o.changed("blueprint", bp.ID)
	}
	return bp, err
}

func (o *Ops) BlueprintsUpdate(id int64, body types.BlueprintInput) (types.Blueprint, error) {
	bp, err := o.DB.UpdateBlueprintT(o.Ctx(), id, body)
	if err == nil {
		o.changed("blueprint", bp.ID)
	}
	return bp, err
}

func (o *Ops) BlueprintsDelete(id int64) (types.Ok, error) {
	n, err := o.DB.DeleteBlueprint(o.Ctx(), id)
	if err != nil {
		return types.Ok{}, err
	}
	if n == 0 {
		return types.Ok{}, fmt.Errorf("Blueprint not found")
	}
	o.changed("blueprint", id)
	return types.Ok{Ok: true}, nil
}

// BlueprintsCreateApp runs a blueprint to completion (logs stream on the
// "blueprintEvent" event) and returns the created app.
func (o *Ops) BlueprintsCreateApp(body types.BlueprintRunInput) (types.BlueprintRunResult, error) {
	res, err := o.Blueprints.Run(o.Ctx(), body)
	if err == nil {
		o.changed("app", res.App.ID)
	}
	return res, err
}

// BlueprintsResolve answers a run that paused on a failed command: action is
// "skip" (continue with the next command) or "abort".
func (o *Ops) BlueprintsResolve(runID, action string) types.Ok {
	o.Blueprints.Resolve(runID, action)
	return types.Ok{Ok: true}
}

// BlueprintsSendInput writes keystrokes to the terminal of the command a run is
// executing.
func (o *Ops) BlueprintsSendInput(runID, data string) (types.Ok, error) {
	if err := o.Blueprints.Write(runID, data); err != nil {
		return types.Ok{}, err
	}
	return types.Ok{Ok: true}, nil
}

// BlueprintsResize sets the size of a run's terminal.
func (o *Ops) BlueprintsResize(runID string, cols, rows int) types.Ok {
	o.Blueprints.Resize(runID, cols, rows)
	return types.Ok{Ok: true}
}

func (o *Ops) BlueprintsCancel(runID string) types.Ok {
	o.Blueprints.Cancel(runID)
	return types.Ok{Ok: true}
}
