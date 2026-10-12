package main

import (
	"fmt"
	"strconv"
	"strings"

	"workspace-manager/api"
	"workspace-manager/api/types"
)

// allWorkspaces returns every workspace.
func allWorkspaces() ([]types.Workspace, error) {
	var out []types.Workspace
	if err := call(api.MethodWorkspacesList, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// allApps returns every app, framed by its workspace. A name may be reused
// across workspaces, so lookups check for ambiguity.
func allApps() ([]types.App, error) {
	workspaces, err := allWorkspaces()
	if err != nil {
		return nil, err
	}
	var out []types.App
	for _, w := range workspaces {
		var apps []types.App
		if err := call(api.MethodAppsList, map[string]any{"workspaceID": w.ID}, &apps); err != nil {
			return nil, err
		}
		out = append(out, apps...)
	}
	return out, nil
}

// resolveApp turns an app argument, which is an ID or a name, into an app. A
// name that matches several apps is an error that lists the candidates.
func resolveApp(arg string) (types.App, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return types.App{}, usage("an app is required")
	}
	if id, err := strconv.ParseInt(arg, 10, 64); err == nil {
		var app types.App
		if err := call(api.MethodAppsGet, map[string]any{"id": id}, &app); err != nil {
			return types.App{}, err
		}
		return app, nil
	}
	apps, err := allApps()
	if err != nil {
		return types.App{}, err
	}
	var matches []types.App
	for _, a := range apps {
		if strings.EqualFold(a.Name, arg) {
			matches = append(matches, a)
		}
	}
	switch len(matches) {
	case 0:
		return types.App{}, api.Errf(api.KindNotFound, "no app named %q", arg)
	case 1:
		return matches[0], nil
	default:
		ids := make([]string, 0, len(matches))
		for _, m := range matches {
			ids = append(ids, fmt.Sprintf("%d (%s)", m.ID, m.ProjectPath))
		}
		return types.App{}, api.Errf(api.KindInvalid, "%d apps are named %q: %s", len(matches), arg, strings.Join(ids, ", "))
	}
}

// resolveWorkspace turns a workspace argument into one, defaulting to the only
// workspace when there is exactly one.
func resolveWorkspace(arg string) (types.Workspace, error) {
	arg = strings.TrimSpace(arg)
	workspaces, err := allWorkspaces()
	if err != nil {
		return types.Workspace{}, err
	}
	if arg == "" {
		switch len(workspaces) {
		case 1:
			return workspaces[0], nil
		case 0:
			return types.Workspace{}, api.Errf(api.KindNotFound, "there are no workspaces yet")
		default:
			return types.Workspace{}, usage("several workspaces exist; pass --workspace <id|name>")
		}
	}
	if id, err := strconv.ParseInt(arg, 10, 64); err == nil {
		for _, w := range workspaces {
			if w.ID == id {
				return w, nil
			}
		}
		return types.Workspace{}, api.Errf(api.KindNotFound, "no workspace with id %d", id)
	}
	var matches []types.Workspace
	for _, w := range workspaces {
		if strings.EqualFold(w.Name, arg) {
			matches = append(matches, w)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return types.Workspace{}, api.Errf(api.KindInvalid, "%d workspaces are named %q", len(matches), arg)
	}
	return types.Workspace{}, api.Errf(api.KindNotFound, "no workspace named %q", arg)
}

// parseID reads a numeric id from an argument.
func parseID(arg, what string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(arg), 10, 64)
	if err != nil || id <= 0 {
		return 0, usage("%s id must be a positive number, got %q", what, arg)
	}
	return id, nil
}

// activeConfigSet resolves the config set of an app, which defaults to the
// active one and can be named or numbered.
func activeConfigSet(app types.App, arg string) (types.ConfigSet, error) {
	var sets []types.ConfigSet
	if err := call(api.MethodConfigSetsList, map[string]any{"appID": app.ID}, &sets); err != nil {
		return types.ConfigSet{}, err
	}
	if strings.TrimSpace(arg) == "" {
		if len(sets) == 1 {
			return sets[0], nil
		}
		for _, s := range sets {
			if app.ActiveConfigSetID != nil && s.ID == *app.ActiveConfigSetID {
				return s, nil
			}
		}
		if len(sets) > 0 {
			return sets[0], nil
		}
		return types.ConfigSet{}, api.Errf(api.KindNotFound, "app %s has no config sets", app.Name)
	}
	if id, err := strconv.ParseInt(arg, 10, 64); err == nil {
		for _, s := range sets {
			if s.ID == id {
				return s, nil
			}
		}
		return types.ConfigSet{}, api.Errf(api.KindNotFound, "no config set with id %d on app %s", id, app.Name)
	}
	var matches []types.ConfigSet
	for _, s := range sets {
		if strings.EqualFold(s.Name, arg) {
			matches = append(matches, s)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return types.ConfigSet{}, api.Errf(api.KindInvalid, "%d config sets are named %q", len(matches), arg)
	}
	return types.ConfigSet{}, api.Errf(api.KindNotFound, "no config set named %q on app %s", arg, app.Name)
}
