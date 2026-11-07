package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"workspace-manager/shared/api"
)

func envTool() mcpTool {
	return mcpTool{
		Name:        "wm_env",
		Description: "Manage an app's environment variables on any config set (the active one by default). A variable can be a choice: give it options (label/value pairs) and the one in use is selected, the first by default; its value is always the selected option's. Private values can be replaced, but are never returned or made public by this MCP server.",
		Schema: schema("Environment variables", map[string]any{
			"action":    map[string]any{"type": "string", "description": "list, set (create or update), delete, import", "enum": []string{"list", "set", "delete", "import"}},
			"app":       strProp(appRefDesc),
			"set":       strProp(setRefDesc + " (all actions work on any set without activating it)"),
			"configSet": strProp("alias of set"),
			"key":       strProp("variable name (set/delete)"),
			"value":     strProp("variable value (set; omit to keep the stored value on update)"),
			"options":   map[string]any{"type": "array", "description": "make the variable a choice: its options in order, replacing any existing ones; [] turns it back into a plain value (set)", "items": map[string]any{"type": "object", "properties": map[string]any{"label": map[string]any{"type": "string"}, "value": map[string]any{"type": "string"}}, "required": []string{"label"}}},
			"selected":  strProp("label of the option in use; omit to keep the current one (the first for new options) (set; needs options)"),
			"private":   boolProp("make the value private: agents never see it (set); omit to keep the current flag; a private variable cannot be made public here"),
			"file":      strProp(".env or YAML file to import (import)"),
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
			switch argStr(args, "action") {
			case "list":
				return daoCall(api.MethodEnvVarsList, map[string]any{"appID": appID, "configSetID": setID})
			case "set":
				key, err := need(args, "key", "the variable name")
				if err != nil {
					return nil, err
				}
				private := argBool(args, "private", false)
				vars, err := daoCall(api.MethodEnvVarsList, map[string]any{"appID": appID, "configSetID": setID})
				if err != nil {
					return nil, err
				}
				raw, _ := json.Marshal(vars)
				var list []map[string]any
				_ = json.Unmarshal(raw, &list)
				_, hasValue := args["value"]
				value, _ := args["value"].(string)
				options, hasOptions := args["options"]
				selected, hasSelected := args["selected"].(string)
				extra := func(body map[string]any) {
					if hasOptions {
						body["options"] = options
					}
					if hasSelected {
						body["selected"] = selected
					}
				}
				for _, v := range list {
					if v["key"] == key {
						if err := refuseShared(v); err != nil {
							return nil, err
						}
						id, _ := v["id"].(float64)
						body := map[string]any{"key": key}
						if _, specified := args["private"]; specified {
							body["private"] = private
						}
						if hasValue {
							body["value"] = value
						}
						extra(body)
						return daoCall(api.MethodEnvVarsUpdate, map[string]any{"id": int64(id), "body": body})
					}
				}
				body := map[string]any{"key": key, "private": private}
				if hasValue {
					body["value"] = value
				}
				extra(body)
				return daoCall(api.MethodEnvVarsCreate, map[string]any{"appID": appID, "configSetID": setID, "body": body})
			case "delete":
				key, err := need(args, "key", "the variable name")
				if err != nil {
					return nil, err
				}
				vars, err := daoCall(api.MethodEnvVarsList, map[string]any{"appID": appID, "configSetID": setID})
				if err != nil {
					return nil, err
				}
				raw, _ := json.Marshal(vars)
				var list []map[string]any
				_ = json.Unmarshal(raw, &list)
				for _, v := range list {
					if v["key"] == key {
						if err := refuseShared(v); err != nil {
							return nil, err
						}
						id, _ := v["id"].(float64)
						return daoCall(api.MethodEnvVarsDelete, map[string]any{"id": int64(id)})
					}
				}
				return nil, api.Errf(api.KindNotFound, "no env var named %q on this app", key)
			case "import":
				file, err := need(args, "file", "the file to import")
				if err != nil {
					return nil, err
				}
				if abs, err := filepath.Abs(file); err == nil {
					file = abs
				}
				if _, err := os.Stat(file); err != nil {
					return nil, api.Errf(api.KindNotFound, "cannot read %s", file)
				}
				return daoCall(api.MethodEnvVarsImport, map[string]any{"appID": appID, "configSetID": setID, "path": file})
			}
			return nil, fmt.Errorf("unknown action %q", argStr(args, "action"))
		},
	}
}
