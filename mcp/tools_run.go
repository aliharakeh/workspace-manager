package mcp

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"workspace-manager/shared/api"
)

func runTool() mcpTool {
	return mcpTool{
		Name:        "wm_run",
		Description: "Run, build and stop apps; check runner status; read recent terminal output. Use wm_wait for completion or ready URLs, and wm_diagnostics for detected errors/warnings without downloading logs. Select a configSet for each start without changing the app's active set. An app runs one session at a time.",
		Schema: schema("Runner", map[string]any{
			"action":     map[string]any{"type": "string", "description": "dev, prod, build, setup, stop, reload, status, logs (run is a legacy alias for dev); startAll and stopAll act on every app of a workspace", "enum": []string{"dev", "prod", "run", "build", "setup", "stop", "reload", "status", "logs", "startAll", "stopAll"}},
			"app":        strProp(appRefDesc),
			"configSet":  strProp("config set id or name for dev, prod, run, build or setup; omit for the active set; does not activate it"),
			"workspace":  strProp("workspace name or id (startAll, stopAll)"),
			"exclude":    arrProp("apps to leave out, by name or id (startAll, stopAll); apps flagged \"exclude from Start all\" are always left out of startAll", "string"),
			"sequential": boolProp("startAll: start the apps one after another instead of together"),
			"commandId":  numProp("only this command id (logs)"),
			"tail":       numProp("only the last N bytes of each command's output (logs)"),
		}, "action"),
		Handler: func(args map[string]any) (any, error) {
			if action := argStr(args, "action"); action == "startAll" || action == "stopAll" {
				ws, err := resolveWorkspace(argStr(args, "workspace"))
				if err != nil {
					return nil, err
				}
				exclude := []int64{}
				for _, ref := range argStrList(args, "exclude") {
					app, err := resolveApp(ref)
					if err != nil {
						return nil, err
					}
					exclude = append(exclude, appIDOf(app))
				}
				params := map[string]any{"workspaceID": ws.ID, "excludeAppIds": exclude}
				if action == "stopAll" {
					return daoCall(api.MethodRunnerStopWorkspace, params)
				}
				params["sequential"] = argBool(args, "sequential", false)
				return daoCall(api.MethodRunnerStartWorkspace, params)
			}
			app, err := resolveApp(argStr(args, "app"))
			if err != nil {
				return nil, err
			}
			appID := appIDOf(app)
			switch argStr(args, "action") {
			case "dev", "prod", "run", "build", "setup":
				kind := argStr(args, "action")
				if kind == "run" {
					kind = "dev"
				}
				method := api.MethodRunnerDev
				switch kind {
				case "prod":
					method = api.MethodRunnerProd
				case "build":
					method = api.MethodRunnerBuild
				case "setup":
					method = api.MethodRunnerSetup
				}
				params := map[string]any{"appID": appID}
				if ref, ok := argOpt(args, "configSet"); ok {
					set, err := resolveConfigSet(app, ref)
					if err != nil {
						return nil, err
					}
					params["configSetID"] = set.ID
				}
				return daoCall(method, params)
			case "stop":
				return daoCall(api.MethodRunnerStop, map[string]any{"appID": appID})
			case "reload":
				return daoCall(api.MethodRunnerReload, map[string]any{"appID": appID})
			case "status":
				return daoCall(api.MethodRunnerStatus, map[string]any{"appID": appID})
			case "logs":
				status, err := daoCall(api.MethodRunnerStatus, map[string]any{"appID": appID})
				if err != nil {
					return nil, err
				}
				raw, _ := json.Marshal(status)
				var st map[string]any
				_ = json.Unmarshal(raw, &st)
				procs, _ := st["processes"].([]any)
				if len(procs) == 0 {
					appName, _ := app["name"].(string)
					return nil, api.Errf(api.KindNotFound, "app %s has no active session", appName)
				}
				filter, hasFilter := argNum(args, "commandId")
				tail, _ := argNum(args, "tail")
				var out []map[string]any
				for _, p := range procs {
					pm, _ := p.(map[string]any)
					cid, _ := pm["commandId"].(float64)
					if hasFilter && int64(cid) != filter {
						continue
					}
					res, err := daoCall(api.MethodRunnerOutput, map[string]any{"appID": appID, "commandID": int64(cid)})
					if err != nil {
						return nil, err
					}
					raw, _ := json.Marshal(res)
					var o map[string]any
					_ = json.Unmarshal(raw, &o)
					data, _ := base64.StdEncoding.DecodeString(fmt.Sprint(o["data"]))
					if tail > 0 && int64(len(data)) > tail {
						data = data[int64(len(data))-tail:]
					}
					out = append(out, map[string]any{"commandId": int64(cid), "text": string(data)})
				}
				if hasFilter && len(out) == 0 {
					return nil, api.Errf(api.KindNotFound, "no command %d in the app's session", filter)
				}
				if out == nil {
					out = []map[string]any{}
				}
				return out, nil
			}
			return nil, fmt.Errorf("unknown action %q", argStr(args, "action"))
		},
	}
}
