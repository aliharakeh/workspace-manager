package mcp

import (
	"encoding/json"
	"fmt"

	"workspace-manager/shared/api"
)

// providerOf reads a config set's or shared set's dynamic environment script.
func providerOf(shared bool, id int64) (map[string]any, error) {
	res, err := daoCall(api.MethodEnvProviderGet, map[string]any{"shared": shared, "id": id})
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(res)
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

func envScriptTool() mcpTool {
	return mcpTool{
		Name: "wm_env_script",
		Description: "The dynamic environment script of an app config set or of a workspace shared set: a command that prints one JSON object of strings, merged into the runtime environment of every start. " +
			"get reads it, set saves it (omitted fields keep their stored value), clear disables it, run executes it now with the saved configuration, stop ends the script that is running (no target needed; a script also stops by itself at its timeout). " +
			"Target an app config set with app/set, or a shared set with shared (+ workspace). " +
			"A script never runs on start or reload: run the shared script first, then the app script. Results stay in the daemon's memory and are never returned or stored. " +
			"App scripts run in the app's project folder; a shared script runs in its directory (absolute, or relative to the first linked app's folder; empty = that folder). Keep credentials out of the command.",
		Schema: schema("Dynamic environment script", map[string]any{
			"action":    map[string]any{"type": "string", "description": "get, set, clear, run, stop", "enum": []string{"get", "set", "clear", "run", "stop"}},
			"app":       strProp(appRefDesc + " (app config set)"),
			"set":       strProp(setRefDesc + " (app config set; works on any set without activating it)"),
			"configSet": strProp("alias of set"),
			"shared":    strProp("shared config set id or name: targets the shared set's script instead of an app config set"),
			"workspace": strProp("workspace id or name (shared sets only; omit when the app or the only workspace tells it)"),
			"command":   strProp("the script command, run with cmd.exe on Windows and sh elsewhere, no interactive input (set)"),
			"directory": strProp("working directory (set); shared script only: absolute, or relative to the first linked app folder; empty = that folder"),
			"timeout":   numProp("timeout in seconds, 1 to 600, 30 by default (set)"),
		}, "action"),
		Handler: func(args map[string]any) (any, error) {
			action := argStr(args, "action")
			if action == "stop" {
				return daoCall(api.MethodEnvProviderStop, map[string]any{})
			}
			shared := argStr(args, "shared") != ""
			var id int64
			if shared {
				var app map[string]any
				if argStr(args, "app") != "" {
					a, err := resolveApp(argStr(args, "app"))
					if err != nil {
						return nil, err
					}
					app = a
				}
				wsID, err := workspaceID(app, args)
				if err != nil {
					// A single workspace needs no argument.
					all, aerr := allWorkspaces()
					if aerr != nil || len(all) != 1 {
						return nil, err
					}
					f, _ := all[0]["id"].(float64)
					wsID = int64(f)
				}
				set, err := resolveSharedSet(wsID, argStr(args, "shared"))
				if err != nil {
					return nil, err
				}
				id = set.ID
			} else {
				app, err := resolveApp(argStr(args, "app"))
				if err != nil {
					return nil, err
				}
				if id, err = setIDArg(app, args); err != nil {
					return nil, err
				}
				if id == 0 {
					active, _ := app["active_config_set_id"].(float64)
					if id = int64(active); id == 0 {
						return nil, fmt.Errorf("this app has no active config set: pass \"set\"")
					}
				}
			}

			switch action {
			case "get":
				return providerOf(shared, id)
			case "run":
				return daoCall(api.MethodEnvProviderRun, map[string]any{"shared": shared, "id": id})
			case "clear":
				cur, err := providerOf(shared, id)
				if err != nil {
					return nil, err
				}
				cur["command"] = ""
				return daoCall(api.MethodEnvProviderSave, map[string]any{"shared": shared, "id": id, "body": cur})
			case "set":
				cur, err := providerOf(shared, id)
				if err != nil {
					return nil, err
				}
				if v, ok := args["command"].(string); ok {
					cur["command"] = v
				}
				if v, ok := args["directory"].(string); ok {
					cur["directory"] = v
				}
				if v, ok := argNum(args, "timeout"); ok {
					cur["timeout"] = v
				}
				if c, _ := cur["command"].(string); c == "" {
					return nil, fmt.Errorf("missing the script: pass \"command\" (or use clear to disable it)")
				}
				return daoCall(api.MethodEnvProviderSave, map[string]any{"shared": shared, "id": id, "body": cur})
			}
			return nil, fmt.Errorf("unknown action %q", action)
		},
	}
}
