package ops

import (
	"fmt"
	"regexp"
	"strings"

	"workspace-manager/shared/api/lib"
	"workspace-manager/shared/api/types"
	"workspace-manager/shared/native"
)

func (o *Ops) WorkspacesList() ([]types.Workspace, error) {
	return o.DB.ListWorkspacesT(o.Ctx())
}

var workspaceColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// normalizeWorkspaceColor returns the color to store: nil (none) for nil or
// "", otherwise a lower-case "#rrggbb". The UI puts it straight into CSS, so
// anything else is rejected.
func normalizeWorkspaceColor(color *string) (*string, error) {
	if color == nil || *color == "" {
		return nil, nil
	}
	if !workspaceColorRe.MatchString(*color) {
		return nil, fmt.Errorf("color must be a hex value like #3b82f6")
	}
	c := strings.ToLower(*color)
	return &c, nil
}

func (o *Ops) WorkspacesCreate(body types.WorkspaceCreateInput) (types.Workspace, error) {
	name := strings.TrimSpace(body.Name)
	if name == "" {
		return types.Workspace{}, fmt.Errorf("name is required")
	}
	color, err := normalizeWorkspaceColor(body.Color)
	if err != nil {
		return types.Workspace{}, err
	}
	ws, err := o.DB.CreateWorkspaceT(o.Ctx(), name, body.Icon, color)
	if err == nil {
		o.changed("workspace", ws.ID)
	}
	return ws, err
}

func (o *Ops) WorkspacesUpdate(id int64, body types.WorkspaceUpdateInput) (types.Workspace, error) {
	if body.Name != nil && strings.TrimSpace(*body.Name) == "" {
		return types.Workspace{}, fmt.Errorf("name cannot be empty")
	}
	color, err := normalizeWorkspaceColor(body.Color)
	if err != nil {
		return types.Workspace{}, err
	}
	var name *string
	if body.Name != nil {
		n := strings.TrimSpace(*body.Name)
		name = &n
	}
	ws, err := o.DB.UpdateWorkspaceT(o.Ctx(), id, name, body.Icon, body.Icon != nil, color, body.Color != nil)
	if err == nil {
		o.changed("workspace", ws.ID)
	}
	return ws, err
}

func (o *Ops) WorkspacesDelete(id int64) (types.Ok, error) {
	n, err := o.DB.DeleteWorkspace(o.Ctx(), id)
	if err != nil {
		return types.Ok{}, err
	}
	if n == 0 {
		return types.Ok{}, fmt.Errorf("Workspace not found")
	}
	o.changed("workspace", id)
	return types.Ok{Ok: true}, nil
}

func (o *Ops) AppsList(workspaceID int64) ([]types.App, error) {
	if _, err := o.DB.GetWorkspaceT(o.Ctx(), workspaceID); err != nil {
		return nil, err
	}
	return o.DB.ListAppsByWorkspaceT(o.Ctx(), workspaceID)
}

func (o *Ops) AppsGet(id int64) (types.App, error) {
	return o.DB.GetAppT(o.Ctx(), id)
}

func (o *Ops) AppsCreate(workspaceID int64, body types.AppCreateInput) (types.App, error) {
	if _, err := o.DB.GetWorkspaceT(o.Ctx(), workspaceID); err != nil {
		return types.App{}, err
	}
	if strings.TrimSpace(body.Name) == "" {
		return types.App{}, fmt.Errorf("name is required")
	}
	if strings.TrimSpace(body.ProjectPath) == "" {
		return types.App{}, fmt.Errorf("project_path is required")
	}
	ok, resolved, errMsg := lib.ValidateProjectPath(body.ProjectPath)
	if !ok {
		return types.App{}, fmt.Errorf("%s", errMsg)
	}
	app, err := o.DB.CreateAppT(o.Ctx(), workspaceID, strings.TrimSpace(body.Name), resolved)
	if err == nil {
		o.changed("app", app.ID)
	}
	return app, err
}

func (o *Ops) AppsUpdate(id int64, body types.AppUpdateInput) (types.App, error) {
	if body.Name != nil && strings.TrimSpace(*body.Name) == "" {
		return types.App{}, fmt.Errorf("name cannot be empty")
	}
	var name, projectPath *string
	if body.Name != nil {
		n := strings.TrimSpace(*body.Name)
		name = &n
	}
	if body.ProjectPath != nil {
		ok, resolved, errMsg := lib.ValidateProjectPath(*body.ProjectPath)
		if !ok {
			return types.App{}, fmt.Errorf("%s", errMsg)
		}
		projectPath = &resolved
	}
	app, err := o.DB.UpdateAppT(o.Ctx(), id, name, projectPath)
	if err == nil && body.ExcludeFromStartAll != nil {
		if err = o.DB.SetAppExcludeFromStartAllT(o.Ctx(), id, *body.ExcludeFromStartAll); err == nil {
			app.ExcludeFromStartAll = *body.ExcludeFromStartAll
		}
	}
	if err == nil {
		o.changed("app", app.ID)
	}
	return app, err
}

// AppsReorder saves the order of a workspace's apps; ids lists all of them.
func (o *Ops) AppsReorder(workspaceID int64, ids []int64) (types.Ok, error) {
	if err := o.DB.ReorderAppsT(o.Ctx(), workspaceID, ids); err != nil {
		return types.Ok{}, err
	}
	o.changed("app")
	return types.Ok{Ok: true}, nil
}

func (o *Ops) AppsDelete(id int64) (types.Ok, error) {
	n, err := o.DB.DeleteApp(o.Ctx(), id)
	if err != nil {
		return types.Ok{}, err
	}
	if n == 0 {
		return types.Ok{}, fmt.Errorf("App not found")
	}
	o.changed("app", id)
	return types.Ok{Ok: true}, nil
}

func (o *Ops) AppsOpenFolder(id int64) (types.Ok, error) {
	app, err := o.DB.GetAppT(o.Ctx(), id)
	if err != nil {
		return types.Ok{}, err
	}
	ok, resolved, errMsg := lib.ValidateProjectPath(app.ProjectPath)
	if !ok {
		return types.Ok{}, fmt.Errorf("%s", errMsg)
	}
	if err := native.OpenFolder(resolved); err != nil {
		return types.Ok{}, err
	}
	return types.Ok{Ok: true}, nil
}

func (o *Ops) AppsEditors() []types.Editor {
	out := []types.Editor{}
	for _, e := range native.Editors() {
		out = append(out, types.Editor{ID: e.ID, Name: e.Name})
	}
	return out
}

func (o *Ops) AppsOpenInEditor(id int64, editor string) (types.Ok, error) {
	app, err := o.DB.GetAppT(o.Ctx(), id)
	if err != nil {
		return types.Ok{}, err
	}
	ok, resolved, errMsg := lib.ValidateProjectPath(app.ProjectPath)
	if !ok {
		return types.Ok{}, fmt.Errorf("%s", errMsg)
	}
	if err := native.OpenInEditor(editor, resolved); err != nil {
		return types.Ok{}, err
	}
	return types.Ok{Ok: true}, nil
}

// appProjectDir returns the validated project path of an app.
func (o *Ops) appProjectDir(appID int64) (string, error) {
	app, err := o.DB.GetAppT(o.Ctx(), appID)
	if err != nil {
		return "", err
	}
	ok, resolved, errMsg := lib.ValidateProjectPath(app.ProjectPath)
	if !ok {
		return "", fmt.Errorf("%s", errMsg)
	}
	return resolved, nil
}

// WorkspacesReorder saves the order of the workspaces; ids lists all of them.
func (o *Ops) WorkspacesReorder(ids []int64) (types.Ok, error) {
	if err := o.DB.ReorderWorkspacesT(o.Ctx(), ids); err != nil {
		return types.Ok{}, err
	}
	o.changed("workspace", ids...)
	return types.Ok{Ok: true}, nil
}

// WorkspacesSetPinned pins a workspace to the top of the list, or unpins it.
func (o *Ops) WorkspacesSetPinned(id int64, pinned bool) (types.Workspace, error) {
	ws, err := o.DB.SetWorkspacePinnedT(o.Ctx(), id, pinned)
	if err == nil {
		o.changed("workspace", ws.ID)
	}
	return ws, err
}
