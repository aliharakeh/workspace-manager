package mcp

import (
	"fmt"
	"path/filepath"

	"workspace-manager/shared/api"
)

func appTool() mcpTool {
	return mcpTool{
		Name:        "wm_app",
		Description: "Manage apps: list, inspect (with config sets and runner status), create, rename/move, delete, reorder.",
		Schema: schema("App management", map[string]any{
			"action":    map[string]any{"type": "string", "description": "list, get, create, update, delete, reorder", "enum": []string{"list", "get", "create", "update", "delete", "reorder"}},
			"app":       strProp(appRefDesc),
			"workspace": strProp(wsRefDesc + " (for list/create/reorder)"),
			"name":      strProp("new app name (create/update)"),
			"path":      strProp("project folder (create: required; update: new path)"),
			"ids":       arrProp("ordered app ids or names (reorder)", "string"),
		}, "action"),
		Handler: func(args map[string]any) (any, error) {
			switch argStr(args, "action") {
			case "list":
				if w, ok := argOpt(args, "workspace"); ok {
					r, err := resolveWorkspace(w)
					if err != nil {
						return nil, err
					}
					return daoCall(api.MethodAppsList, map[string]any{"workspaceID": r.ID})
				}
				apps, err := allApps()
				if err != nil {
					return nil, err
				}
				if apps == nil {
					apps = []map[string]any{}
				}
				return apps, nil
			case "get":
				app, err := resolveApp(argStr(args, "app"))
				if err != nil {
					return nil, err
				}
				sets, err := daoCall(api.MethodConfigSetsList, map[string]any{"appID": appIDOf(app)})
				if err != nil {
					return nil, err
				}
				var detail any
				if set, err := resolveConfigSet(app, ""); err == nil {
					detail, _ = daoCall(api.MethodConfigSetsDetail, map[string]any{"id": set.ID})
				}
				status, err := daoCall(api.MethodRunnerStatus, map[string]any{"appID": appIDOf(app)})
				if err != nil {
					return nil, err
				}
				return map[string]any{
					"app": app, "configSets": sets,
					"activeConfigSetDetail": detail, "status": status,
				}, nil
			case "create":
				name, err := need(args, "name", "the app name")
				if err != nil {
					return nil, err
				}
				path, err := need(args, "path", "the project folder")
				if err != nil {
					return nil, err
				}
				if abs, err := filepath.Abs(path); err == nil {
					path = abs
				}
				ws, err := resolveWorkspace(argStr(args, "workspace"))
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodAppsCreate, map[string]any{
					"workspaceID": ws.ID,
					"body":        map[string]any{"name": name, "project_path": path},
				})
			case "update":
				app, err := resolveApp(argStr(args, "app"))
				if err != nil {
					return nil, err
				}
				body := map[string]any{}
				if name, ok := argOpt(args, "name"); ok {
					body["name"] = name
				}
				if path, ok := argOpt(args, "path"); ok {
					if abs, err := filepath.Abs(path); err == nil {
						path = abs
					}
					body["project_path"] = path
				}
				if len(body) == 0 {
					return nil, fmt.Errorf("nothing to update: pass \"name\" and/or \"path\"")
				}
				return daoCall(api.MethodAppsUpdate, map[string]any{"id": appIDOf(app), "body": body})
			case "delete":
				app, err := resolveApp(argStr(args, "app"))
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodAppsDelete, map[string]any{"id": appIDOf(app)})
			case "reorder":
				ids := argStrList(args, "ids")
				if len(ids) == 0 {
					return nil, fmt.Errorf("pass \"ids\": the ordered app ids or names")
				}
				var wsID int64
				if w, ok := argOpt(args, "workspace"); ok {
					r, err := resolveWorkspace(w)
					if err != nil {
						return nil, err
					}
					wsID = r.ID
				}
				var out []int64
				for _, ref := range ids {
					a, err := resolveApp(ref)
					if err != nil {
						return nil, err
					}
					if wsID == 0 {
						if id, _ := a["workspace_id"].(float64); id != 0 {
							wsID = int64(id)
						}
					} else if id, _ := a["workspace_id"].(float64); int64(id) != wsID {
						name, _ := a["name"].(string)
						return nil, fmt.Errorf("app %s is not in workspace %d", name, wsID)
					}
					out = append(out, appIDOf(a))
				}
				return daoCall(api.MethodAppsReorder, map[string]any{"workspaceID": wsID, "ids": out})
			}
			return nil, fmt.Errorf("unknown action %q", argStr(args, "action"))
		},
	}
}
