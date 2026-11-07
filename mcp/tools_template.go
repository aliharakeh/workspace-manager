package mcp

import (
	"encoding/json"
	"fmt"

	"workspace-manager/shared/api"
)

func templateTool() mcpTool {
	return mcpTool{
		Name:        "wm_template",
		Description: "Manage an app's file templates (generated config files) on any config set (the active one by default).",
		Schema: schema("File templates", map[string]any{
			"action":  map[string]any{"type": "string", "description": "list, add, edit, delete", "enum": []string{"list", "add", "edit", "delete"}},
			"app":     strProp(appRefDesc),
			"set":     strProp(setRefDesc + " (all actions work on any set without activating it)"),
			"file":    strProp("project-relative file path, e.g. .env.local (add/edit/delete)"),
			"content": strProp("template content, with {{placeholders}} (add/edit)"),
			"private": boolProp("make the content private: agents never see it (add/edit)"),
		}, "action"),
		Handler: func(args map[string]any) (any, error) {
			app, err := resolveApp(argStr(args, "app"))
			if err != nil {
				return nil, err
			}
			appID := appIDOf(app)
			setID, err := setIDArg(app, args)
			if err != nil {
				return nil, err
			}
			find := func(file string) (map[string]any, error) {
				res, err := daoCall(api.MethodTemplatesList, map[string]any{"appID": appID, "configSetID": setID})
				if err != nil {
					return nil, err
				}
				raw, _ := json.Marshal(res)
				var list []map[string]any
				_ = json.Unmarshal(raw, &list)
				for _, t := range list {
					if p, _ := t["file_path"].(string); equalFold(p, file) {
						return t, nil
					}
				}
				return nil, api.Errf(api.KindNotFound, "no template for %q on this app", file)
			}
			switch argStr(args, "action") {
			case "list":
				return daoCall(api.MethodTemplatesList, map[string]any{"appID": appID, "configSetID": setID})
			case "add":
				file, err := need(args, "file", "the project-relative file path")
				if err != nil {
					return nil, err
				}
				body := map[string]any{"file_path": file}
				if content, ok := args["content"].(string); ok {
					body["content"] = content
				}
				if _, changed := args["private"]; changed {
					body["private"] = argBool(args, "private", false)
				}
				return daoCall(api.MethodTemplatesCreate, map[string]any{"appID": appID, "configSetID": setID, "body": body})
			case "edit":
				file, err := need(args, "file", "the project-relative file path")
				if err != nil {
					return nil, err
				}
				tpl, err := find(file)
				if err != nil {
					return nil, err
				}
				id, _ := tpl["id"].(float64)
				body := map[string]any{}
				if content, ok := args["content"].(string); ok {
					body["content"] = content
				}
				if _, changed := args["private"]; changed {
					body["private"] = argBool(args, "private", false)
				}
				if len(body) == 0 {
					return nil, fmt.Errorf("nothing to change: pass \"content\" and/or \"private\"")
				}
				return daoCall(api.MethodTemplatesUpdate, map[string]any{"id": int64(id), "body": body})
			case "delete":
				file, err := need(args, "file", "the project-relative file path")
				if err != nil {
					return nil, err
				}
				tpl, err := find(file)
				if err != nil {
					return nil, err
				}
				id, _ := tpl["id"].(float64)
				return daoCall(api.MethodTemplatesDelete, map[string]any{"id": int64(id)})
			}
			return nil, fmt.Errorf("unknown action %q", argStr(args, "action"))
		},
	}
}
